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

Once those have been turned into text (scripts/pgs-to-srt.swift), put each
result beside its MP4 as "b.<lang>.srt" and run:

    remux-for-streaming.py --add-subtitles DIR

which adds every such .srt to its MP4 as a subtitle track the player can turn
on. An MP4 that already has subtitles is left alone.

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


def ffmpeg_args(src, dst, p, sidecars=(), srts=()):
    """One ffmpeg run that reads the source once and writes the MP4 and any
    subtitle sidecars together. sidecars is (stream, path) pairs; srts is
    (language, path) pairs of text subtitles to put in the MP4."""
    args = ["ffmpeg", "-nostdin", "-v", "error", "-y", "-i", src]
    for _, path in srts:
        args += ["-i", path]
    args += ["-map", "0:%d" % p["video"]["index"], "-c:v", "copy",
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

    for j, (lang, _) in enumerate(srts):
        i = len(p["text_subs"]) + j
        args += ["-map", "%d:0" % (j + 1), "-c:s:%d" % i, "mov_text",
                 "-metadata:s:s:%d" % i, "language=%s" % lang, "-disposition:s:%d" % i, "0"]

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
    except subprocess.CalledProcessError as e:
        why = ((e.stderr or "").strip().splitlines() or ["ffprobe exited %d" % e.returncode])[-1]
        return "output could not be read back: %s" % why
    except ValueError as e:
        return "output could not be read back: %s" % e
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
    # Text subtitles already waiting beside the destination go in during this
    # same pass, which saves rewriting the whole file to add them later.
    srts = srts_beside(dst)

    def discard():
        for path in written:
            if os.path.exists(path):
                os.remove(path)

    try:
        ran = subprocess.run(ffmpeg_args(src, partial, p, [(sub, path + ".partial") for sub, path in sidecars], srts),
                             check=True, capture_output=True, text=True)
    except subprocess.CalledProcessError as e:
        discard()
        return "failed", ((e.stderr or "").strip().splitlines() or ["ffmpeg failed"])[-1]

    problem = verify(info, partial, p)
    if problem:
        # ffmpeg can exit 0 having failed to finish the file. Whatever it
        # said on the way is the only account of why.
        said = " | ".join((ran.stderr or "").strip().splitlines()[-3:])
        if said:
            problem += " [ffmpeg: %s]" % said
        # REMUX_KEEP_FAILED=1 leaves the rejected output in place to look at.
        if not os.environ.get("REMUX_KEEP_FAILED"):
            discard()
        return "failed", problem

    # The MP4 goes last: it is what marks the file as converted.
    for _, path in sidecars:
        os.replace(path + ".partial", path)
    os.replace(partial, dst)

    tracks = ", ".join("%s %s" % (t["language"], t["title"].lower()) for t in p["audio"])
    extras = "; %d picture subtitle track(s) saved beside it" % len(p["picture_subs"]) if p["picture_subs"] else ""
    if srts:
        extras += "; subtitles: %s" % ", ".join(lang for lang, _ in srts)
    return "done", "audio: %s%s" % (tracks, extras)


def srts_beside(mp4):
    """The "<name>.<lang>.srt" files beside an MP4, as (language, path)."""
    stem = os.path.splitext(mp4)[0]
    folder = os.path.dirname(mp4) or "."
    prefix = os.path.basename(stem) + "."
    srts = []
    if not os.path.isdir(folder):
        return srts
    for name in sorted(os.listdir(folder)):
        # Nothing with a longer name that merely starts the same way.
        middle = name[len(prefix):-4] if name.startswith(prefix) and name.endswith(".srt") else ""
        if re.fullmatch(r"[a-z]{2,3}", middle):
            srts.append((middle, os.path.join(folder, name)))
    return srts


def add_subtitles(mp4):
    """Adds the .srt files beside an MP4 to it. Returns (status, detail)."""
    srts = srts_beside(mp4)
    if not srts:
        return "skipped", "no .srt beside it"

    info = probe(mp4)
    if any(s["codec_type"] == "subtitle" for s in info["streams"]):
        return "skipped", "already has subtitles"

    args = ["ffmpeg", "-nostdin", "-v", "error", "-y", "-i", mp4]
    for _, path in srts:
        args += ["-i", path]
    args += ["-map", "0:v", "-map", "0:a"]
    for i, (lang, _) in enumerate(srts):
        args += ["-map", "%d:0" % (i + 1), "-metadata:s:s:%d" % i, "language=%s" % lang,
                 "-disposition:s:%d" % i, "0"]
    partial = mp4 + ".partial"
    args += ["-c", "copy", "-c:s", "mov_text", "-map_metadata", "0", "-map_chapters", "0",
             "-movflags", "+faststart", "-f", "mp4", partial]

    try:
        subprocess.run(args, check=True, capture_output=True, text=True)
        out = probe(partial)
    except (subprocess.CalledProcessError, ValueError) as e:
        if os.path.exists(partial):
            os.remove(partial)
        return "failed", ((getattr(e, "stderr", "") or "").strip().splitlines() or ["could not add subtitles"])[-1]

    kinds = [s["codec_type"] for s in out["streams"]]
    before = [s["codec_type"] for s in info["streams"]]
    if (kinds.count("video"), kinds.count("audio")) != (before.count("video"), before.count("audio")) \
            or kinds.count("subtitle") != len(srts) or abs(duration(out) - duration(info)) > 2.0:
        os.remove(partial)
        return "failed", "the result did not match the original"

    os.replace(partial, mp4)
    return "done", "added %s" % ", ".join(lang for lang, _ in srts)


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
    parser.add_argument("src_dir", nargs="?")
    parser.add_argument("out_dir", nargs="?")
    parser.add_argument("--add-subtitles", metavar="DIR",
                        help="add each '<name>.<lang>.srt' in DIR to the MP4 it sits beside")
    parser.add_argument("--match", help="only files whose name matches this regular expression")
    parser.add_argument("--limit", type=int, help="stop after converting this many files")
    parser.add_argument("--dry-run", action="store_true", help="say what would be done and change nothing")
    args = parser.parse_args()

    pattern = re.compile(args.match) if args.match else None

    if args.add_subtitles:
        counts = {"done": 0, "skipped": 0, "failed": 0}
        for root, _, names in os.walk(args.add_subtitles):
            for name in sorted(names):
                if not name.lower().endswith(".mp4") or (pattern and not pattern.search(name)):
                    continue
                status, detail = add_subtitles(os.path.join(root, name))
                counts[status] += 1
                if status != "skipped":
                    print("%-7s %s (%s)" % (status, name, detail), flush=True)
        print("%d given subtitles, %d left alone, %d failed" % (counts["done"], counts["skipped"], counts["failed"]), flush=True)
        sys.exit(1 if counts["failed"] else 0)

    if not args.src_dir or not args.out_dir:
        parser.error("SRC_DIR and OUT_DIR are required unless --add-subtitles is given")
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
