"""Restore mob wealth (`gold`, copper) in go/data/areas TOML from ROT .are files.

ROT mob line 'start_pos default_pos sex wealth' carries the wealth.
usage: python3 -I wealth.py <rot/area dir> <go/data/areas dir> [--write]
"""
import glob, os, re, sys

POS = {"stand", "sleep", "rest", "sit", "fight", "dead", "mortal", "incap", "stun"}


def parse_are(path, out):
    text = open(path, encoding="latin-1").read()
    if "#MOBILES" not in text:
        return
    vnum = None
    for line in text[text.index("#MOBILES"):].split("\n"):
        if line.startswith("#OBJECTS"):
            break
        m = re.match(r"^#(\d+)\s*$", line)
        if m:
            vnum = int(m.group(1))
            continue
        tok = line.split()
        if vnum and len(tok) == 4 and tok[0] in POS and tok[1] in POS and tok[3].isdigit():
            out[vnum] = int(tok[3])
            vnum = None


def main():
    are_dir, toml_dir, write = sys.argv[1], sys.argv[2], "--write" in sys.argv
    wealth = {}
    for f in glob.glob(os.path.join(are_dir, "*.are")):
        parse_are(f, wealth)

    done, missing = 0, []
    for f in sorted(glob.glob(os.path.join(toml_dir, "*/mobs/*.toml"))):
        out, vnum, changed = [], None, False
        for line in open(f).read().split("\n"):
            m = re.match(r"^vnum = (\d+)$", line)
            if m:
                vnum = int(m.group(1))
            if line.startswith("gold = "):
                continue  # replaced below
            out.append(line)
            if re.match(r"^level = -?\d+$", line) and vnum is not None:
                if vnum in wealth:
                    out.append(f"gold = {wealth[vnum]}")
                    done += 1
                    changed = True
                else:
                    missing.append(vnum)
        if changed and write:
            open(f, "w").write("\n".join(out))
    print(f"source={len(wealth)} restored={done} not_in_source={missing[:20]} ({len(missing)})")


main()
