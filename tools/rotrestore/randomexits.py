"""Restore maze exit shuffling (ROM 'R' resets) as `random_exits` on rooms.

usage: python3 -I randomexits.py <rot/area dir> <go/data/areas dir> [--write]
Re-runnable: existing random_exits lines are replaced.
"""
import glob, os, re, sys, tomllib


def main():
    src, dst, write = sys.argv[1], sys.argv[2], "--write" in sys.argv
    want = {}
    for name in open(os.path.join(src, "area.lst")).read().split():
        f = os.path.join(src, name)
        if not name.endswith(".are") or not os.path.exists(f):
            continue
        text = open(f, encoding="latin-1").read()
        m = re.search(r"^#RESETS\s*$", text, re.M)
        if m:
            for room, n in re.findall(r"^R\s+\d+\s+(\d+)\s+(\d+)", text[m.end():], re.M):
                want[int(room)] = int(n)
    done = 0
    for f in sorted(glob.glob(os.path.join(dst, "*", "rooms", "*.toml"))):
        text = open(f).read()
        out, vnum = [], None
        for line in text.split("\n"):
            if line.startswith("random_exits = "):
                continue
            out.append(line)
            m = re.match(r"^vnum = (\d+)$", line)
            if m:
                vnum = int(m.group(1))
            elif line.startswith("name = ") and vnum in want:
                out.append(f"random_exits = {want[vnum]}")
                done += 1
        new = "\n".join(out)
        tomllib.loads(new)
        if write and new != text:
            open(f, "w").write(new)
    print(f"R resets {len(want)}; restored {done}")


main()
