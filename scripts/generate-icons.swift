#!/usr/bin/env swift

import AppKit
import CoreGraphics
import CoreText

// MARK: - Color Palette (based on AccentColor #1E88E5)

let primaryBlue = NSColor(red: 0.118, green: 0.533, blue: 0.898, alpha: 1.0)   // #1E88E5
let darkBlue    = NSColor(red: 0.051, green: 0.278, blue: 0.631, alpha: 1.0)   // #0D47A1
let lightBlue   = NSColor(red: 0.392, green: 0.710, blue: 0.965, alpha: 1.0)   // #64B5F6
let deepNavy    = NSColor(red: 0.039, green: 0.086, blue: 0.157, alpha: 1.0)   // #0A1628

// MARK: - Path Setup

let scriptPath = URL(fileURLWithPath: CommandLine.arguments[0]).deletingLastPathComponent()
let projectRoot = scriptPath.deletingLastPathComponent()

let tvAssetsBase = projectRoot
    .appendingPathComponent("apps/apple-tv/MediaPlayer/Assets.xcassets/AppIcon.brandassets")

let iosAssetsBase = projectRoot
    .appendingPathComponent("apps/ios-mobile/MediaServerMobile/Assets.xcassets/AppIcon.appiconset")

// MARK: - Helper Functions

func createContext(width: Int, height: Int, opaque: Bool = false) -> CGContext {
    let colorSpace = CGColorSpaceCreateDeviceRGB()
    let bitmapInfo: UInt32 = opaque
        ? CGImageAlphaInfo.noneSkipLast.rawValue
        : CGImageAlphaInfo.premultipliedLast.rawValue
    let ctx = CGContext(
        data: nil,
        width: width,
        height: height,
        bitsPerComponent: 8,
        bytesPerRow: width * 4,
        space: colorSpace,
        bitmapInfo: bitmapInfo
    )!
    ctx.setShouldAntialias(true)
    ctx.setAllowsAntialiasing(true)
    if !opaque {
        ctx.clear(CGRect(x: 0, y: 0, width: width, height: height))
    }
    return ctx
}

func savePNG(_ ctx: CGContext, to path: URL) {
    guard let image = ctx.makeImage() else {
        print("  ERROR: Failed to create image for \(path.lastPathComponent)")
        return
    }
    let rep = NSBitmapImageRep(cgImage: image)
    guard let data = rep.representation(using: .png, properties: [:]) else {
        print("  ERROR: Failed to create PNG data for \(path.lastPathComponent)")
        return
    }
    do {
        try data.write(to: path)
        print("  Created: \(path.lastPathComponent) (\(ctx.width)x\(ctx.height))")
    } catch {
        print("  ERROR: Failed to write \(path.lastPathComponent): \(error)")
    }
}

func drawRadialGradient(ctx: CGContext, center: CGPoint, innerRadius: CGFloat, outerRadius: CGFloat,
                        colors: [CGColor], locations: [CGFloat]) {
    let colorSpace = CGColorSpaceCreateDeviceRGB()
    guard let gradient = CGGradient(colorsSpace: colorSpace, colors: colors as CFArray, locations: locations) else { return }
    ctx.drawRadialGradient(gradient, startCenter: center, startRadius: innerRadius,
                           endCenter: center, endRadius: outerRadius, options: [.drawsAfterEndLocation])
}

func drawLinearGradient(ctx: CGContext, rect: CGRect, colors: [CGColor], locations: [CGFloat],
                        startPoint: CGPoint, endPoint: CGPoint) {
    let colorSpace = CGColorSpaceCreateDeviceRGB()
    guard let gradient = CGGradient(colorsSpace: colorSpace, colors: colors as CFArray, locations: locations) else { return }
    ctx.drawLinearGradient(gradient, start: startPoint, end: endPoint, options: [.drawsAfterEndLocation])
}

/// Draw an equilateral play triangle, optically centered (shifted right slightly)
func drawPlayTriangle(ctx: CGContext, center: CGPoint, size: CGFloat, color: NSColor, shadow: Bool = true) {
    let h = size       // height of triangle
    let w = size * 0.866 // width (equilateral proportion)

    // Optical centering: shift right by ~8% of size
    let offsetX = size * 0.08
    let cx = center.x + offsetX
    let cy = center.y

    let top    = CGPoint(x: cx - w / 2, y: cy + h / 2)
    let bottom = CGPoint(x: cx - w / 2, y: cy - h / 2)
    let right  = CGPoint(x: cx + w / 2, y: cy)

    if shadow {
        ctx.saveGState()
        ctx.setShadow(offset: CGSize(width: 0, height: -2), blur: size * 0.08,
                       color: NSColor(white: 0, alpha: 0.4).cgColor)
    }

    ctx.setFillColor(color.cgColor)
    ctx.beginPath()
    ctx.move(to: top)
    ctx.addLine(to: right)
    ctx.addLine(to: bottom)
    ctx.closePath()
    ctx.fillPath()

    if shadow {
        ctx.restoreGState()
    }
}

/// Draw a frosted glass circle
func drawFrostedCircle(ctx: CGContext, center: CGPoint, radius: CGFloat) {
    // Outer subtle glow
    ctx.saveGState()
    ctx.setShadow(offset: .zero, blur: radius * 0.15,
                   color: NSColor(white: 1, alpha: 0.2).cgColor)
    ctx.setFillColor(NSColor(white: 1, alpha: 0.15).cgColor)
    ctx.fillEllipse(in: CGRect(x: center.x - radius, y: center.y - radius,
                                width: radius * 2, height: radius * 2))
    ctx.restoreGState()

    // Inner brighter fill
    ctx.setFillColor(NSColor(white: 1, alpha: 0.12).cgColor)
    let innerR = radius * 0.92
    ctx.fillEllipse(in: CGRect(x: center.x - innerR, y: center.y - innerR,
                                width: innerR * 2, height: innerR * 2))

    // Rim stroke
    ctx.setStrokeColor(NSColor(white: 1, alpha: 0.25).cgColor)
    ctx.setLineWidth(radius * 0.02)
    ctx.strokeEllipse(in: CGRect(x: center.x - radius, y: center.y - radius,
                                  width: radius * 2, height: radius * 2))
}

func drawText(ctx: CGContext, text: String, fontSize: CGFloat, center: CGPoint, color: NSColor) {
    let font = NSFont.systemFont(ofSize: fontSize, weight: .semibold)
    let attrs: [NSAttributedString.Key: Any] = [
        .font: font,
        .foregroundColor: color
    ]
    let attrStr = NSAttributedString(string: text, attributes: attrs)
    let line = CTLineCreateWithAttributedString(attrStr)
    let bounds = CTLineGetBoundsWithOptions(line, .useOpticalBounds)

    let x = center.x - bounds.width / 2
    let y = center.y - bounds.height / 2

    ctx.saveGState()
    ctx.textPosition = CGPoint(x: x, y: y)
    CTLineDraw(line, ctx)
    ctx.restoreGState()
}

// MARK: - Layer Renderers

/// Back layer: rich blue radial gradient (opaque)
func renderBackLayer(width: Int, height: Int) -> CGContext {
    let ctx = createContext(width: width, height: height, opaque: true)
    let w = CGFloat(width)
    let h = CGFloat(height)
    let center = CGPoint(x: w * 0.5, y: h * 0.55) // slightly above center
    let maxR = max(w, h) * 0.8

    // Base fill
    ctx.setFillColor(deepNavy.cgColor)
    ctx.fill(CGRect(x: 0, y: 0, width: w, height: h))

    // Main blue radial gradient
    drawRadialGradient(ctx: ctx, center: center, innerRadius: 0, outerRadius: maxR,
                       colors: [primaryBlue.cgColor, darkBlue.cgColor, deepNavy.cgColor],
                       locations: [0.0, 0.5, 1.0])

    // Subtle highlight near center
    drawRadialGradient(ctx: ctx, center: CGPoint(x: w * 0.5, y: h * 0.6),
                       innerRadius: 0, outerRadius: maxR * 0.4,
                       colors: [lightBlue.withAlphaComponent(0.3).cgColor,
                                lightBlue.withAlphaComponent(0.0).cgColor],
                       locations: [0.0, 1.0])

    return ctx
}

/// Middle layer: decorative rings (transparent background)
func renderMiddleLayer(width: Int, height: Int) -> CGContext {
    let ctx = createContext(width: width, height: height)
    let w = CGFloat(width)
    let h = CGFloat(height)
    let center = CGPoint(x: w * 0.5, y: h * 0.5)

    // Outer ring
    let outerR = min(w, h) * 0.42
    ctx.setStrokeColor(lightBlue.withAlphaComponent(0.12).cgColor)
    ctx.setLineWidth(outerR * 0.03)
    ctx.strokeEllipse(in: CGRect(x: center.x - outerR, y: center.y - outerR,
                                  width: outerR * 2, height: outerR * 2))

    // Inner ring
    let innerR = min(w, h) * 0.32
    ctx.setStrokeColor(lightBlue.withAlphaComponent(0.08).cgColor)
    ctx.setLineWidth(innerR * 0.02)
    ctx.strokeEllipse(in: CGRect(x: center.x - innerR, y: center.y - innerR,
                                  width: innerR * 2, height: innerR * 2))

    // Small accent dots around outer ring
    let dotCount = 12
    let dotR: CGFloat = outerR * 0.025
    ctx.setFillColor(lightBlue.withAlphaComponent(0.15).cgColor)
    for i in 0..<dotCount {
        let angle = CGFloat(i) * (2.0 * .pi / CGFloat(dotCount))
        let dx = center.x + outerR * cos(angle)
        let dy = center.y + outerR * sin(angle)
        ctx.fillEllipse(in: CGRect(x: dx - dotR, y: dy - dotR, width: dotR * 2, height: dotR * 2))
    }

    return ctx
}

/// Front layer: play button in frosted circle (transparent background)
func renderFrontLayer(width: Int, height: Int, showText: Bool = false) -> CGContext {
    let ctx = createContext(width: width, height: height)
    let w = CGFloat(width)
    let h = CGFloat(height)
    let center = CGPoint(x: w * 0.5, y: showText ? h * 0.55 : h * 0.5)

    let circleR = min(w, h) * 0.30

    drawFrostedCircle(ctx: ctx, center: center, radius: circleR)
    drawPlayTriangle(ctx: ctx, center: center, size: circleR * 0.8, color: .white)

    if showText {
        drawText(ctx: ctx, text: "Media Server", fontSize: h * 0.06,
                 center: CGPoint(x: w * 0.5, y: h * 0.18), color: .white)
    }

    return ctx
}

/// Top Shelf: composite flat image
func renderTopShelf(width: Int, height: Int) -> CGContext {
    let ctx = createContext(width: width, height: height, opaque: true)
    let w = CGFloat(width)
    let h = CGFloat(height)

    // Background gradient
    ctx.setFillColor(deepNavy.cgColor)
    ctx.fill(CGRect(x: 0, y: 0, width: w, height: h))

    let center = CGPoint(x: w * 0.5, y: h * 0.5)
    drawRadialGradient(ctx: ctx, center: center, innerRadius: 0, outerRadius: max(w, h) * 0.6,
                       colors: [primaryBlue.withAlphaComponent(0.6).cgColor,
                                darkBlue.withAlphaComponent(0.3).cgColor,
                                deepNavy.cgColor],
                       locations: [0.0, 0.5, 1.0])

    // Play icon
    let iconCenter = CGPoint(x: w * 0.42, y: h * 0.5)
    let circleR = h * 0.25
    drawFrostedCircle(ctx: ctx, center: iconCenter, radius: circleR)
    drawPlayTriangle(ctx: ctx, center: iconCenter, size: circleR * 0.7, color: .white)

    // Text
    drawText(ctx: ctx, text: "Media Server", fontSize: h * 0.10,
             center: CGPoint(x: w * 0.62, y: h * 0.5), color: .white)

    return ctx
}

/// iOS icon: square, blue gradient + play button (no rounded corners)
func renderIOSIcon(size: Int) -> CGContext {
    let ctx = createContext(width: size, height: size, opaque: true)
    let s = CGFloat(size)
    let center = CGPoint(x: s * 0.5, y: s * 0.5)

    // Background
    ctx.setFillColor(deepNavy.cgColor)
    ctx.fill(CGRect(x: 0, y: 0, width: s, height: s))

    drawRadialGradient(ctx: ctx, center: CGPoint(x: s * 0.5, y: s * 0.55),
                       innerRadius: 0, outerRadius: s * 0.7,
                       colors: [primaryBlue.cgColor, darkBlue.cgColor, deepNavy.cgColor],
                       locations: [0.0, 0.5, 1.0])

    // Subtle highlight
    drawRadialGradient(ctx: ctx, center: CGPoint(x: s * 0.5, y: s * 0.6),
                       innerRadius: 0, outerRadius: s * 0.3,
                       colors: [lightBlue.withAlphaComponent(0.25).cgColor,
                                lightBlue.withAlphaComponent(0.0).cgColor],
                       locations: [0.0, 1.0])

    // Frosted circle + play
    let circleR = s * 0.28
    drawFrostedCircle(ctx: ctx, center: center, radius: circleR)
    drawPlayTriangle(ctx: ctx, center: center, size: circleR * 0.8, color: .white)

    return ctx
}

// MARK: - Contents.json Helpers

struct ImageEntry: Codable {
    let filename: String?
    let idiom: String
    let scale: String?
    let platform: String?
    let size: String?
}

struct ContentsJSON: Codable {
    let images: [ImageEntry]
    let info: InfoEntry

    struct InfoEntry: Codable {
        let author: String
        let version: Int
    }
}

func writeTVContentsJSON(at dir: URL, baseName: String) {
    let contents = ContentsJSON(
        images: [
            ImageEntry(filename: "\(baseName).png", idiom: "tv", scale: "1x", platform: nil, size: nil),
            ImageEntry(filename: "\(baseName)@2x.png", idiom: "tv", scale: "2x", platform: nil, size: nil)
        ],
        info: ContentsJSON.InfoEntry(author: "xcode", version: 1)
    )
    writeContentsJSON(contents, to: dir)
}

func writeIOSContentsJSON(at dir: URL, filename: String) {
    let contents = ContentsJSON(
        images: [
            ImageEntry(filename: filename, idiom: "universal", scale: nil, platform: "ios", size: "1024x1024")
        ],
        info: ContentsJSON.InfoEntry(author: "xcode", version: 1)
    )
    writeContentsJSON(contents, to: dir)
}

func writeContentsJSON(_ contents: ContentsJSON, to dir: URL) {
    let encoder = JSONEncoder()
    encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
    guard let data = try? encoder.encode(contents) else { return }
    let path = dir.appendingPathComponent("Contents.json")
    try? data.write(to: path)
    print("  Updated: \(path.lastPathComponent) in \(dir.lastPathComponent)")
}

// MARK: - Main Generation

print("Generating app icons...")
print()

// --- App Icon (400x240 base, 3 layers) ---
let appIconBase = tvAssetsBase.appendingPathComponent("App Icon.imagestack")

print("App Icon - Back Layer:")
let backDir = appIconBase.appendingPathComponent("Back.imagestacklayer/Content.imageset")
savePNG(renderBackLayer(width: 400, height: 240), to: backDir.appendingPathComponent("back.png"))
savePNG(renderBackLayer(width: 800, height: 480), to: backDir.appendingPathComponent("back@2x.png"))
writeTVContentsJSON(at: backDir, baseName: "back")

print("App Icon - Middle Layer:")
let middleDir = appIconBase.appendingPathComponent("Middle.imagestacklayer/Content.imageset")
savePNG(renderMiddleLayer(width: 400, height: 240), to: middleDir.appendingPathComponent("middle.png"))
savePNG(renderMiddleLayer(width: 800, height: 480), to: middleDir.appendingPathComponent("middle@2x.png"))
writeTVContentsJSON(at: middleDir, baseName: "middle")

print("App Icon - Front Layer:")
let frontDir = appIconBase.appendingPathComponent("Front.imagestacklayer/Content.imageset")
savePNG(renderFrontLayer(width: 400, height: 240), to: frontDir.appendingPathComponent("front.png"))
savePNG(renderFrontLayer(width: 800, height: 480), to: frontDir.appendingPathComponent("front@2x.png"))
writeTVContentsJSON(at: frontDir, baseName: "front")

// --- App Store Icon (1280x768 base, 2 layers) ---
let storeBase = tvAssetsBase.appendingPathComponent("App Icon - App Store.imagestack")

print("App Store Icon - Back Layer:")
let storeBackDir = storeBase.appendingPathComponent("Back.imagestacklayer/Content.imageset")
savePNG(renderBackLayer(width: 1280, height: 768), to: storeBackDir.appendingPathComponent("back.png"))
savePNG(renderBackLayer(width: 2560, height: 1536), to: storeBackDir.appendingPathComponent("back@2x.png"))
writeTVContentsJSON(at: storeBackDir, baseName: "back")

print("App Store Icon - Front Layer:")
let storeFrontDir = storeBase.appendingPathComponent("Front.imagestacklayer/Content.imageset")
savePNG(renderFrontLayer(width: 1280, height: 768, showText: true), to: storeFrontDir.appendingPathComponent("front.png"))
savePNG(renderFrontLayer(width: 2560, height: 1536, showText: true), to: storeFrontDir.appendingPathComponent("front@2x.png"))
writeTVContentsJSON(at: storeFrontDir, baseName: "front")

// --- Top Shelf (1920x720) ---
print("Top Shelf Image:")
let topShelfDir = tvAssetsBase.appendingPathComponent("Top Shelf Image.imageset")
savePNG(renderTopShelf(width: 1920, height: 720), to: topShelfDir.appendingPathComponent("top-shelf.png"))
savePNG(renderTopShelf(width: 3840, height: 1440), to: topShelfDir.appendingPathComponent("top-shelf@2x.png"))
writeTVContentsJSON(at: topShelfDir, baseName: "top-shelf")

// --- Top Shelf Wide (2320x720) ---
print("Top Shelf Image Wide:")
let topShelfWideDir = tvAssetsBase.appendingPathComponent("Top Shelf Image Wide.imageset")
savePNG(renderTopShelf(width: 2320, height: 720), to: topShelfWideDir.appendingPathComponent("top-shelf-wide.png"))
savePNG(renderTopShelf(width: 4640, height: 1440), to: topShelfWideDir.appendingPathComponent("top-shelf-wide@2x.png"))
writeTVContentsJSON(at: topShelfWideDir, baseName: "top-shelf-wide")

// --- iOS Icon (1024x1024) ---
print("iOS App Icon:")
savePNG(renderIOSIcon(size: 1024), to: iosAssetsBase.appendingPathComponent("icon.png"))
writeIOSContentsJSON(at: iosAssetsBase, filename: "icon.png")

print()
print("Done! Generated all icon assets.")
