// Turn Blu-ray picture subtitles into text.
//
//     swift scripts/pgs-to-srt.swift subtitles.sup subtitles.srt
//
// Blu-ray subtitles are pictures of text, which an MP4 cannot carry and Apple's
// player cannot show. This reads a .sup stream, redraws each subtitle as an
// image, reads the text off it with the Mac's built-in text recognition, and
// writes an .srt with the original timings. Nothing needs installing, which is
// why it is a Mac script and not part of the server.
//
// To get a .sup out of a saved subtitle track:
//     ffmpeg -i "Episode.eng.1.mks" -map 0:0 -c copy subtitles.sup

import CoreGraphics
import Foundation
import Vision

// MARK: - Reading the stream

/// One subtitle as it appears on screen: when, and the picture of it.
struct Cue {
    let start: Double
    var end: Double?
    let image: CGImage
}

struct SubtitleObject {
    var width = 0
    var height = 0
    var expected = 0
    var rle = Data()
}

struct Placement {
    let objectID: Int
    let x: Int
    let y: Int
}

func be16(_ data: Data, _ offset: Int) -> Int {
    Int(data[offset]) << 8 | Int(data[offset + 1])
}

func be24(_ data: Data, _ offset: Int) -> Int {
    Int(data[offset]) << 16 | Int(data[offset + 1]) << 8 | Int(data[offset + 2])
}

func be32(_ data: Data, _ offset: Int) -> Int {
    Int(data[offset]) << 24 | Int(data[offset + 1]) << 16 | Int(data[offset + 2]) << 8 | Int(data[offset + 3])
}

/// Palette entries arrive as Y'CbCr plus alpha. Returned as (grey, alpha):
/// colour does not help reading, brightness and coverage do.
func greyAndAlpha(y: UInt8, alpha: UInt8) -> (UInt8, UInt8) {
    // Studio range 16-235 stretched to 0-255.
    let stretched = (Double(y) - 16.0) * 255.0 / 219.0
    return (UInt8(max(0, min(255, stretched))), alpha)
}

/// Unpacks a run-length encoded object into one palette index per pixel.
func decodeRLE(_ rle: Data, width: Int, height: Int) -> [UInt8] {
    var pixels = [UInt8](repeating: 0, count: width * height)
    var i = rle.startIndex
    var x = 0
    var y = 0

    func put(_ colour: UInt8, _ count: Int) {
        guard y < height else { return }
        let n = min(count, width - x)
        if n > 0 {
            let base = y * width + x
            for k in 0..<n { pixels[base + k] = colour }
            x += n
        }
    }

    while i < rle.endIndex, y < height {
        let b = rle[i]
        i += 1
        if b != 0 {
            put(b, 1)
            continue
        }
        guard i < rle.endIndex else { break }
        let flags = rle[i]
        i += 1
        if flags == 0 {
            // End of line.
            x = 0
            y += 1
            continue
        }
        var length = Int(flags & 0x3F)
        if flags & 0x40 != 0 {
            guard i < rle.endIndex else { break }
            length = length << 8 | Int(rle[i])
            i += 1
        }
        var colour: UInt8 = 0
        if flags & 0x80 != 0 {
            guard i < rle.endIndex else { break }
            colour = rle[i]
            i += 1
        }
        put(colour, length)
    }
    return pixels
}

/// Draws a display set's objects onto one black canvas, light text on dark,
/// which is what the recogniser reads best.
func render(_ placements: [Placement], objects: [Int: SubtitleObject], palette: [(UInt8, UInt8)]) -> CGImage? {
    let drawn = placements.compactMap { p -> (Placement, SubtitleObject)? in
        guard let object = objects[p.objectID], object.width > 0, object.height > 0 else { return nil }
        return (p, object)
    }
    guard !drawn.isEmpty else { return nil }

    let margin = 24
    let minX = drawn.map { $0.0.x }.min()!
    let minY = drawn.map { $0.0.y }.min()!
    let maxX = drawn.map { $0.0.x + $0.1.width }.max()!
    let maxY = drawn.map { $0.0.y + $0.1.height }.max()!
    let width = maxX - minX + margin * 2
    let height = maxY - minY + margin * 2

    var canvas = [UInt8](repeating: 0, count: width * height)
    for (placement, object) in drawn {
        let indices = decodeRLE(object.rle, width: object.width, height: object.height)
        let offsetX = placement.x - minX + margin
        let offsetY = placement.y - minY + margin
        for row in 0..<object.height {
            for column in 0..<object.width {
                let (grey, alpha) = palette[Int(indices[row * object.width + column])]
                if alpha == 0 { continue }
                // Over black, so a dark outline disappears and the fill stays.
                canvas[(offsetY + row) * width + offsetX + column] = UInt8(Int(grey) * Int(alpha) / 255)
            }
        }
    }

    guard let provider = CGDataProvider(data: Data(canvas) as CFData) else { return nil }
    return CGImage(
        width: width, height: height, bitsPerComponent: 8, bitsPerPixel: 8, bytesPerRow: width,
        space: CGColorSpaceCreateDeviceGray(), bitmapInfo: CGBitmapInfo(rawValue: CGImageAlphaInfo.none.rawValue),
        provider: provider, decode: nil, shouldInterpolate: false, intent: .defaultIntent
    )
}

/// Walks the segments of a .sup stream and returns each subtitle with the
/// time it goes up and the time the next change takes it down.
func readCues(_ data: Data) -> [Cue] {
    var cues: [Cue] = []
    var palettes: [Int: [(UInt8, UInt8)]] = [:]
    var objects: [Int: SubtitleObject] = [:]

    var time = 0.0
    var paletteID = 0
    var placements: [Placement] = []

    var offset = 0
    while offset + 13 <= data.count, data[offset] == 0x50, data[offset + 1] == 0x47 {
        let pts = Double(be32(data, offset + 2)) / 90_000.0
        let kind = data[offset + 10]
        let size = be16(data, offset + 11)
        let body = offset + 13
        guard body + size <= data.count else { break }

        switch kind {
        case 0x16: // Presentation composition: what is on screen from now.
            time = pts
            placements = []
            if size >= 11 {
                // A new epoch starts with nothing carried over.
                if data[body + 7] == 0x80 {
                    objects.removeAll()
                }
                paletteID = Int(data[body + 9])
                var p = body + 11
                for _ in 0..<Int(data[body + 10]) {
                    guard p + 8 <= body + size else { break }
                    let cropped = data[p + 3] & 0x80 != 0
                    placements.append(Placement(objectID: be16(data, p), x: be16(data, p + 4), y: be16(data, p + 6)))
                    p += cropped ? 16 : 8
                }
            }

        case 0x14: // Palette.
            var entries = palettes[Int(data[body])] ?? [(UInt8, UInt8)](repeating: (0, 0), count: 256)
            var p = body + 2
            while p + 5 <= body + size {
                entries[Int(data[p])] = greyAndAlpha(y: data[p + 1], alpha: data[p + 4])
                p += 5
            }
            palettes[Int(data[body])] = entries

        case 0x15: // Object data, possibly split across several segments.
            guard size >= 4 else { break }
            let id = be16(data, body)
            let first = data[body + 3] & 0x80 != 0
            if first, size >= 11 {
                var object = SubtitleObject()
                object.expected = be24(data, body + 4) - 4
                object.width = be16(data, body + 7)
                object.height = be16(data, body + 9)
                object.rle = data.subdata(in: (body + 11)..<(body + size))
                objects[id] = object
            } else if var object = objects[id] {
                object.rle.append(data.subdata(in: (body + 4)..<(body + size)))
                objects[id] = object
            }

        case 0x80: // End of the display set: act on it.
            // Whatever was showing comes down now, whether or not something
            // replaces it.
            for i in cues.indices.reversed() {
                if cues[i].end != nil { break }
                cues[i].end = time
            }
            if !placements.isEmpty, let palette = palettes[paletteID],
               let image = render(placements, objects: objects, palette: palette) {
                cues.append(Cue(start: time, end: nil, image: image))
            }

        default:
            break
        }
        offset = body + size
    }
    return cues
}

// MARK: - Reading the text

/// Reads the text off a subtitle image, top line first.
func recognise(_ image: CGImage) -> String {
    let request = VNRecognizeTextRequest()
    request.recognitionLevel = .accurate
    request.recognitionLanguages = ["en-US"]
    request.usesLanguageCorrection = true

    do {
        try VNImageRequestHandler(cgImage: image, options: [:]).perform([request])
    } catch {
        return ""
    }

    // Vision's origin is bottom-left. Pieces on the same line come back
    // separately, so group by height before ordering left to right.
    let found = (request.results ?? []).compactMap { observation -> (CGRect, String)? in
        guard let text = observation.topCandidates(1).first?.string else { return nil }
        return (observation.boundingBox, text)
    }
    var lines: [[(CGRect, String)]] = []
    for piece in found.sorted(by: { $0.0.midY > $1.0.midY }) {
        if let last = lines.last?.first, abs(last.0.midY - piece.0.midY) < last.0.height * 0.5 {
            lines[lines.count - 1].append(piece)
        } else {
            lines.append([piece])
        }
    }
    return lines
        .map { $0.sorted { $0.0.minX < $1.0.minX }.map { $0.1 }.joined(separator: " ") }
        .joined(separator: "\n")
}

func timestamp(_ seconds: Double) -> String {
    let ms = Int((seconds * 1000).rounded())
    return String(format: "%02d:%02d:%02d,%03d", ms / 3_600_000, ms / 60_000 % 60, ms / 1000 % 60, ms % 1000)
}

// MARK: - Main

let arguments = CommandLine.arguments
guard arguments.count == 3 else {
    FileHandle.standardError.write("usage: swift pgs-to-srt.swift input.sup output.srt\n".data(using: .utf8)!)
    exit(2)
}

guard let data = FileManager.default.contents(atPath: arguments[1]) else {
    FileHandle.standardError.write("cannot read \(arguments[1])\n".data(using: .utf8)!)
    exit(1)
}

let cues = readCues(data)

// A subtitle that fades in or out arrives as several pictures of the same
// words; fold those back into one entry.
var entries: [(start: Double, end: Double, text: String)] = []
var unread = 0
for cue in cues {
    let text = recognise(cue.image)
    if text.isEmpty {
        unread += 1
        continue
    }
    // With no later change to take it down, leave it up for a reading's worth.
    let end = cue.end ?? cue.start + 4
    if let last = entries.last, last.text == text, cue.start - last.end < 0.1 {
        entries[entries.count - 1].end = end
    } else {
        entries.append((cue.start, end, text))
    }
}

var srt = ""
for (index, entry) in entries.enumerated() {
    srt += "\(index + 1)\n\(timestamp(entry.start)) --> \(timestamp(entry.end))\n\(entry.text)\n\n"
}

do {
    try srt.write(toFile: arguments[2], atomically: true, encoding: .utf8)
} catch {
    FileHandle.standardError.write("cannot write \(arguments[2]): \(error.localizedDescription)\n".data(using: .utf8)!)
    exit(1)
}

print("\(entries.count) subtitles from \(cues.count) pictures, \(unread) with no text found")
