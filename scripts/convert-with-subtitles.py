#!/usr/bin/env python3
"""
Repackage a folder of MKVs on the server as MP4 with readable subtitles,
driven from the Mac.

    convert-with-subtitles.py --host user@server --src DIR --out DIR [--match REGEX] [--limit N]

The work is split because each machine can do only half of it. The server has
the files; the Mac has the text recognition that turns Blu-ray picture
subtitles into text. So for each file this:

  1. has the server stream the picture-subtitle track here (a few MB),
  2. reads the text off it with scripts/pgs-to-srt.swift,
  3. sends the resulting .srt back, beside where the MP4 will go,
  4. has the server repackage the file with scripts/remux-for-streaming.py,
     which picks the .srt up and puts it in the MP4 in the same pass.

Files already converted are skipped, and ones converted earlier without
subtitles get them added, so the script can be stopped and run again. Nothing
in --src is changed.

The server needs ffmpeg, python3 and a copy of remux-for-streaming.py (see
--remote-script). This Mac needs ssh access to it without a password.
"""

import argparse
import os
import re
import shlex
import subprocess
import sys
import tempfile
import time

HERE = os.path.dirname(os.path.abspath(__file__))


def remote(host, command, **kwargs):
    """Runs a command on the server. command is a list; it is quoted here, so
    names with spaces, quotes and exclamation marks survive the trip."""
    line = " ".join(shlex.quote(part) for part in command)
    return subprocess.run(["ssh", "-o", "BatchMode=yes", host, line], **kwargs)


def build_recogniser(workdir):
    binary = os.path.join(workdir, "pgs-to-srt")
    subprocess.run(["swiftc", "-O", "-o", binary, os.path.join(HERE, "pgs-to-srt.swift")],
                   check=True, capture_output=True)
    return binary


def make_subtitles(host, src_path, srt_remote, recogniser, workdir):
    """Returns a short description of what happened."""
    sup = os.path.join(workdir, "track.sup")
    srt = os.path.join(workdir, "track.srt")
    with open(sup, "wb") as out:
        pulled = remote(host, ["ffmpeg", "-nostdin", "-v", "error", "-i", src_path,
                               "-map", "0:s:0", "-c", "copy", "-f", "sup", "pipe:1"],
                        stdout=out, stderr=subprocess.PIPE)
    if pulled.returncode != 0 or os.path.getsize(sup) == 0:
        return "no picture subtitles to read"

    read = subprocess.run([recogniser, sup, srt], capture_output=True, text=True)
    if read.returncode != 0 or not os.path.exists(srt) or os.path.getsize(srt) == 0:
        return "subtitles could not be read"

    with open(srt, "rb") as data:
        sent = remote(host, ["sh", "-c", 'mkdir -p "$(dirname "$1")" && cat > "$1"', "sh", srt_remote],
                      stdin=data, capture_output=True)
    if sent.returncode != 0:
        return "subtitles could not be sent back"
    return read.stdout.strip()


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--host", required=True)
    parser.add_argument("--src", required=True, help="folder of MKVs on the server")
    parser.add_argument("--out", required=True, help="folder on the server for the MP4s")
    parser.add_argument("--match", help="only files whose name matches this regular expression")
    parser.add_argument("--limit", type=int, help="stop after this many files")
    parser.add_argument("--language", default="eng", help="language of the subtitle track (default eng)")
    parser.add_argument("--remote-script", default="remux-for-streaming.py",
                        help="path to remux-for-streaming.py on the server (default: in the home folder)")
    args = parser.parse_args()

    listing = remote(args.host, ["find", args.src, "-maxdepth", "1", "-name", "*.mkv", "-printf", "%f\\n"],
                     capture_output=True, text=True)
    if listing.returncode != 0:
        sys.exit("could not list %s on %s: %s" % (args.src, args.host, listing.stderr.strip()))
    pattern = re.compile(args.match) if args.match else None
    names = sorted(n for n in listing.stdout.splitlines()
                   if n and not n.startswith("._") and (not pattern or pattern.search(n)))

    failed = 0
    with tempfile.TemporaryDirectory() as workdir:
        recogniser = build_recogniser(workdir)

        for count, name in enumerate(names, 1):
            if args.limit and count > args.limit:
                break
            started = time.time()
            stem = os.path.splitext(name)[0]
            srt_remote = "%s/%s.%s.srt" % (args.out, stem, args.language)

            have_srt = remote(args.host, ["test", "-s", srt_remote]).returncode == 0
            subtitles = "subtitles already read" if have_srt else make_subtitles(
                args.host, "%s/%s" % (args.src, name), srt_remote, recogniser, workdir)

            converted = remote(args.host, ["python3", args.remote_script, args.src, args.out,
                                           "--match", "^%s$" % re.escape(name)],
                               capture_output=True, text=True)
            # For a file converted on an earlier run, before its subtitles
            # existed. A no-op for one that just got them in the pass above.
            added = remote(args.host, ["python3", args.remote_script, "--add-subtitles", args.out,
                                       "--match", "^%s$" % re.escape(stem + ".mp4")],
                           capture_output=True, text=True)

            status = (converted.stdout.strip().splitlines() or ["no output"])[0].split(" ", 1)[0]
            if converted.returncode != 0 or added.returncode != 0:
                failed += 1
                status = "FAILED"
            print("%s  %3d/%d  %-7s %s  [%s; %.0fs]" % (
                time.strftime("%H:%M"), count, len(names), status, stem[:60], subtitles, time.time() - started),
                flush=True)
            if status == "FAILED":
                print("        " + (converted.stdout + converted.stderr + added.stdout + added.stderr).strip()[-300:],
                      flush=True)

    print("finished: %d files, %d failed" % (min(len(names), args.limit or len(names)), failed), flush=True)
    sys.exit(1 if failed else 0)


if __name__ == "__main__":
    main()
