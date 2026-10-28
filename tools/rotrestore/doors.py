"""Restore exit door flags in go/data/areas TOML from ROT .are files.

ROT exit `locks`: 1 door, 2 door+pickproof, 3 door+nopass, 4 door+nopass+pickproof.
`D 0 room dir state` resets: 1 closed, 2 closed+locked.

usage: python3 -I doors.py <rot/area dir> <go/data/areas dir> [--write]
"""
import glob, os, re, sys

DIRS = ["north", "east", "south", "west", "up", "down"]
LOCKS = {0: [], 1: ["door"], 2: ["door", "pickproof"], 3: ["door", "nopass"],
         4: ["door", "nopass", "pickproof"]}
STATE = {0: [], 1: ["closed"], 2: ["closed", "locked"]}


def parse_are(path, locks, state):
    text = open(path, encoding="latin-1").read()
    for m in re.finditer(r"^D\s+\d+\s+(\d+)\s+(\d+)\s+(\d+)", text, re.M):
        state[(int(m[1]), int(m[2]))] = int(m[3])
    if "#ROOMS" not in text:
        return
    lines = text[text.index("#ROOMS"):].split("\n")
    vnum = None
    for i, line in enumerate(lines):
        if re.match(r"^#\d+\s*$", line):
            vnum = int(line[1:])
            if vnum == 0:
                break
        m = re.match(r"^D([0-5])\s*$", line)
        if m and vnum:
            # description~ keyword~ then "locks key to_vnum"; a ~ may sit mid-line
            rest = "\n".join(lines[i + 1:i + 40])
            after = rest.split("~", 2)[2]
            tok = after.split()[0]
            if tok.lstrip("-").isdigit():  # malformed exits (e.g. stonebow 13124) are skipped
                locks[(vnum, int(m[1]))] = int(tok)


def main():
    are_dir, toml_dir, write = sys.argv[1], sys.argv[2], "--write" in sys.argv
    locks, state = {}, {}
    for f in glob.glob(os.path.join(are_dir, "*.are")):
        parse_are(f, locks, state)

    changed_exits = 0
    for f in sorted(glob.glob(os.path.join(toml_dir, "*/rooms/*.toml"))):
        lines = open(f).read().split("\n")
        out, room, i = [], None, 0
        changed = False
        while i < len(lines):
            line = lines[i]
            m = re.match(r"^vnum = (\d+)$", line)
            if m:
                room = int(m.group(1))
            out.append(line)
            m = re.match(r'^(\s*)direction = "(\w+)"$', line)
            if m and room is not None and m.group(2) in DIRS and (room, DIRS.index(m.group(2))) in locks:
                indent, d = m.group(1), DIRS.index(m.group(2))
                # drop this exit's existing flags line (if any) within the table
                j = i + 1
                while j < len(lines) and lines[j].strip() and not lines[j].lstrip().startswith("["):
                    if lines[j].lstrip().startswith("flags = "):
                        old = lines.pop(j)
                        continue
                    j += 1
                flags = LOCKS.get(locks.get((room, d), 0), []) + STATE.get(state.get((room, d), 0), [])
                if flags:
                    out.append(indent + "flags = [" + ", ".join(f'"{x}"' for x in flags) + "]")
                changed_exits += 1
                changed = True
            i += 1
        if changed and write:
            open(f, "w").write("\n".join(out))
    print(f"locks={len(locks)} resets={len(state)} exits rewritten={changed_exits}")


main()
