#!/usr/bin/env python3
"""
Repackage MKV files as MP4 that Apple devices play directly.

    remux-for-streaming.py SRC_DIR OUT_DIR [--match REGEX] [--limit N] [--dry-run]

The video is copied untouched, so quality is identical and a file takes about
as long as it takes to read and write it. Nothing in SRC_DIR is changed: each
"SRC_DIR/a/b.mkv" becomes "OUT_DIR/a/b.mp4", and files already converted are
skipped, so the script can be stopped and run again.

What goes into the MP4, for each audio language (English first):
  - a stereo AAC track, which every device plays
  - the surround track as Dolby Digital, copied if the source has one and
    encoded from the best source otherwise
Lossless tracks (TrueHD, DTS-HD) are left behind: Apple's player cannot play
them.

Text subtitles are carried over. Picture subtitles (Blu-ray PGS, DVD) cannot
live in an MP4, so each is saved beside the output as "b.<lang>.<n>.mks", a
lossless copy to turn into text later.

Needs ffmpeg and ffprobe. Prints one line per file and a summary; exits 1 if
any file failed.
"""

import argparse
import json
import os
import re
import subprocess
import sys

LOSSLESS = {"truehd", "mlp", "flac", "dts", "pcm_s16le", "pcm_s24le", "pcm_bluray"}
PASSTHROUGH = {"ac3", "eac3"}
TEXT_SUBS = {"subrip", "ass", "ssa", "mov_text", "webvtt", "text"}
PICTURE_SUBS = {"hdmv_pgs_subtitle", "dvd_subtitle", "dvb_subtitle"}
VIDEO_TAGS = {"h264": "avc1", "hevc": "hvc1"}


def probe(path):
    out = subprocess.run(
        ["ffprobe", "-v", "error", "-print_format", "json", "-show_streams", "-show_format", path],
        capture_output=True, text=True, check=True,
    ).stdout
    return json.loads(out)


def language(stream):
    return (stream.get("tags") or {}).get("language", "und").lower()


def plan(info):
    """Decide what to do with each stream. Returns (plan, reason-it-cannot-be-done)."""
    streams = info["streams"]
    videos = [s for s in streams if s["codec_type"] == "video"
              and not s.get("disposition", {}).get("attached_pic")]
    if not videos:
        return None, "no video stream"
    video = videos[0]
    if video["codec_name"] not in VIDEO_TAGS:
        return None, "video is %s, which needs re-encoding, not repackaging" % video["codec_name"]

    # Audio, grouped by language in order of first appearance, English first.
    by_language = {}
    for s in streams:
        if s["codec_type"] == "audio":
            by_language.setdefault(language(s), []).append(s)
    languages = sorted(by_language, key=lambda lang: (lang != "eng",))

    audio = []
    for lang in languages:
        tracks = by_language[lang]
        best = max(tracks, key=lambda s: (int(s.get("channels", 2)), s["codec_name"] in LOSSLESS))
        audio.append({"source": best["index"], "codec": "aac", "channels": 2, "language": lang,
                      "title": "Stereo"})
        if int(best.get("channels", 2)) > 2:
            surround = next((s for s in tracks if s["codec_name"] in PASSTHROUGH
                             and int(s.get("channels", 2)) > 2), None)
            if surround:
                audio.append({"source": surround["index"], "codec": "copy", "language": lang,
                              "title": "Surround"})
            else:
                audio.append({"source": best["index"], "codec": "ac3", "channels": 6,
                              "language": lang, "title": "Surround"})

    text_subs = [s for s in streams if s["codec_type"] == "subtitle" and s["codec_name"] in TEXT_SUBS]
    picture_subs = [s for s in streams if s["codec_type"] == "subtitle" and s["codec_name"] in PICTURE_SUBS]

    return {"video": video, "audio": audio, "text_subs": text_subs, "picture_subs": picture_subs}, None


def ffmpeg_args(src, dst, p, sidecars=()):
    """One ffmpeg run that reads the source once and writes the MP4 and any
    subtitle sidecars together. sidecars is (stream, path) pairs."""
    args = ["ffmpeg", "-nostdin", "-v", "error", "-y", "-i", src,
            "-map", "0:%d" % p["video"]["index"], "-c:v", "copy",
            "-tag:v", VIDEO_TAGS[p["video"]["codec_name"]]]

    for i, track in enumerate(p["audio"]):
        args += ["-map", "0:%d" % track["source"], "-c:a:%d" % i, track["codec"]]
        if track["codec"] == "aac":
            args += ["-ac:a:%d" % i, "2", "-b:a:%d" % i, "192k"]
        elif track["codec"] == "ac3":
            args += ["-ac:a:%d" % i, "6", "-b:a:%d" % i, "640k"]
        args += ["-metadata:s:a:%d" % i, "language=%s" % track["language"],
                 "-metadata:s:a:%d" % i, "title=%s" % track["title"],
                 "-disposition:a:%d" % i, "default" if i == 0 else "0"]

    for i, sub in enumerate(p["text_subs"]):
        # Not default: subtitles should be something the viewer turns on.
        args += ["-map", "0:%d" % sub["index"], "-c:s:%d" % i, "mov_text",
                 "-metadata:s:s:%d" % i, "language=%s" % language(sub),
                 "-disposition:s:%d" % i, "0"]

    # faststart puts the index at the front, so playback can begin before the
    # whole file has been fetched.
    args += ["-map_metadata", "0", "-map_chapters", "0", "-movflags", "+faststart", "-f", "mp4", dst]

    for sub, path in sidecars:
        args += ["-map", "0:%d" % sub["index"], "-c", "copy", "-f", "matroska", path]
    return args


def sidecar_paths(dst_stem, p):
    counts = {}
    paths = []
    for sub in p["picture_subs"]:
        lang = language(sub)
        counts[lang] = counts.get(lang, 0) + 1
        paths.append((sub, "%s.%s.%d.mks" % (dst_stem, lang, counts[lang])))
    return paths


def duration(info):
    try:
        return float(info["format"]["duration"])
    except (KeyError, ValueError):
        return 0.0


def verify(src_info, dst, p):
    """Returns why the output is not acceptable, or None when it is."""
    try:
        out = probe(dst)
    except (subprocess.CalledProcessError, ValueError):
        return "output could not be read back"
    kinds = [s["codec_type"] for s in out["streams"]]
    if kinds.count("video") != 1:
        return "output has no video"
    if kinds.count("audio") != len(p["audio"]):
        return "output has %d audio tracks, expected %d" % (kinds.count("audio"), len(p["audio"]))
    if abs(duration(out) - duration(src_info)) > 2.0:
        return "output runs %.0fs, source runs %.0fs" % (duration(out), duration(src_info))
    return None


def convert(src, dst):
    """Returns (status, detail): done, skipped, cannot, failed."""
    info = probe(src)
    p, reason = plan(info)
    if p is None:
        return "cannot", reason

    dst_stem = os.path.splitext(dst)[0]
    if os.path.exists(dst) and verify(info, dst, p) is None:
        return "skipped", "already converted"

    os.makedirs(os.path.dirname(dst), exist_ok=True)
    partial = dst + ".partial"
    sidecars = sidecar_paths(dst_stem, p)
    written = [partial] + [path + ".partial" for _, path in sidecars]

    def discard():
        for path in written:
            if os.path.exists(path):
                os.remove(path)

    try:
        subprocess.run(ffmpeg_args(src, partial, p, [(sub, path + ".partial") for sub, path in sidecars]),
                       check=True, capture_output=True, text=True)
    except subprocess.CalledProcessError as e:
        discard()
        return "failed", ((e.stderr or "").strip().splitlines() or ["ffmpeg failed"])[-1]

    problem = verify(info, partial, p)
    if problem:
        discard()
        return "failed", problem

    # The MP4 goes last: it is what marks the file as converted.
    for _, path in sidecars:
        os.replace(path + ".partial", path)
    os.replace(partial, dst)

    tracks = ", ".join("%s %s" % (t["language"], t["title"].lower()) for t in p["audio"])
    extras = "; %d picture subtitle track(s) saved beside it" % len(p["picture_subs"]) if p["picture_subs"] else ""
    return "done", "audio: %s%s" % (tracks, extras)


def describe(src):
    info = probe(src)
    p, reason = plan(info)
    if p is None:
        return "cannot: " + reason
    tracks = ", ".join("%s %s (%s)" % (t["language"], t["title"].lower(), t["codec"]) for t in p["audio"])
    return "video copy; audio: %s; %d text and %d picture subtitle track(s)" % (
        tracks, len(p["text_subs"]), len(p["picture_subs"]))


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("src_dir")
    parser.add_argument("out_dir")
    parser.add_argument("--match", help="only files whose name matches this regular expression")
    parser.add_argument("--limit", type=int, help="stop after converting this many files")
    parser.add_argument("--dry-run", action="store_true", help="say what would be done and change nothing")
    args = parser.parse_args()

    pattern = re.compile(args.match) if args.match else None
    sources = []
    for root, _, names in os.walk(args.src_dir):
        for name in names:
            if name.lower().endswith(".mkv") and not name.startswith("._") and (not pattern or pattern.search(name)):
                sources.append(os.path.join(root, name))
    sources.sort()

    counts = {"done": 0, "skipped": 0, "cannot": 0, "failed": 0}
    for src in sources:
        rel = os.path.relpath(src, args.src_dir)
        dst = os.path.join(args.out_dir, os.path.splitext(rel)[0] + ".mp4")

        if args.dry_run:
            print("%s -> %s" % (rel, describe(src)), flush=True)
            continue

        try:
            status, detail = convert(src, dst)
        except (subprocess.CalledProcessError, OSError, ValueError) as e:
            # One unreadable file should not end a long run.
            status, detail = "failed", "could not be read: %s" % e
        counts[status] += 1
        print("%-7s %s (%s)" % (status, rel, detail), flush=True)
        if args.limit and counts["done"] >= args.limit:
            break

    if not args.dry_run:
        print("%d converted, %d already done, %d cannot be repackaged, %d failed, of %d files" % (
            counts["done"], counts["skipped"], counts["cannot"], counts["failed"], len(sources)), flush=True)
    sys.exit(1 if counts["failed"] else 0)


if __name__ == "__main__":
    main()
