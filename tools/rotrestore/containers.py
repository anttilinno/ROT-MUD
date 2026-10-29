"""Restore container contents (ROM 'P' resets) into go/data/areas room TOML.

A 'P' reset fills the container most recently loaded by an 'O' reset (in a
room) or a 'G'/'E' reset (carried by the last 'M' mob). Its contents become
a `contents` list on that obj reset or mob equip. Re-runnable: existing
contents lists are removed first.

usage: python3 -I containers.py <rot/area dir> <go/data/areas dir> [--write]
"""
import glob, os, re, sys, tomllib

OBJ_HDR, EQ_HDR = "[[rooms.obj_resets.contents]]", "[[rooms.mob_resets.equips.contents]]"


def parse_resets(path):
    """[(target, (vnum, max, count))]; target is ('room', room, container)
    or ('mob', room, mob, container)."""
    text = open(path, encoding="latin-1").read()
    m = re.search(r"^#RESETS\s*$", text, re.M)
    if not m:
        return []
    out, where, mob = [], {}, None
    for line in text[m.end():].split("\n"):
        t = line.split()
        if not t or t[0] == "*":
            continue
        if t[0] == "S":
            break
        if t[0] == "M":
            mob = (int(t[4]), int(t[2]))
        elif t[0] == "O":
            where[int(t[2])] = ("room", int(t[4]), int(t[2]))
        elif t[0] in "GE" and mob:
            where[int(t[2])] = ("mob", mob[0], mob[1], int(t[2]))
        elif t[0] == "P":
            target = where.get(int(t[4]))
            limit = int(t[3])
            limit = 6 if limit > 50 else 0 if limit == -1 else limit
            count = max(int(t[5]) if len(t) > 5 and t[5].lstrip("-").isdigit() else 1, 1)
            if target:
                out.append((target, (int(t[2]), limit, count)))
            else:
                print(f"  no container for {os.path.basename(path)}: {line.strip()}", file=sys.stderr)
    return out


def strip_contents(lines):
    out, skip = [], False
    for line in lines:
        s = line.strip()
        if s in (OBJ_HDR, EQ_HDR):
            skip = True
            while out and not out[-1].strip():
                out.pop()
            continue
        if skip and (not s or s.startswith("[")):
            skip = False
        if not skip:
            out.append(line)
    return out


def header_at(lines, i):
    s = lines[i]
    return len(s) - len(s.lstrip()) if s.strip().startswith("[[") else None


def block_end(lines, start):
    """Index after the last field of the block whose header is at start,
    past any deeper-nested blocks."""
    indent = header_at(lines, start)
    end = start + 1
    while end < len(lines):
        h = header_at(lines, end)
        if h is not None and h <= indent:
            break
        end += 1
    while not lines[end - 1].strip():
        end -= 1
    return end


def find_block(lines, header, vnum, start=0, stop=None):
    stop = len(lines) if stop is None else stop
    for i in range(start, stop - 1):
        if lines[i].strip() == header and lines[i + 1].strip() == f"vnum = {vnum}":
            return i
    return None


def main():
    src, dst, write = sys.argv[1], sys.argv[2], "--write" in sys.argv
    lst = open(os.path.join(src, "area.lst")).read().split()
    resets = []
    for name in lst:
        if name.endswith(".are") and os.path.exists(os.path.join(src, name)):
            resets += parse_resets(os.path.join(src, name))
    by_room = {}
    for target, item in resets:
        by_room.setdefault(target[1], []).append((target, item))

    placed, missed = 0, []
    for f in sorted(glob.glob(os.path.join(dst, "*", "rooms", "*.toml"))):
        text = open(f).read()
        lines = strip_contents(text.split("\n"))
        rooms = [i for i, l in enumerate(lines) if l == "[[rooms]]"]
        for k in reversed(range(len(rooms))):  # bottom up keeps indexes valid
            lo, hi = rooms[k], rooms[k + 1] if k + 1 < len(rooms) else len(lines)
            vnum = int(lines[lo + 1].split("=")[1])
            for target, (ovnum, limit, count) in by_room.get(vnum, []):
                if target[0] == "room":
                    at, pad, hdr = find_block(lines, "[[rooms.obj_resets]]", target[2], lo, hi), "    ", OBJ_HDR
                else:
                    m = find_block(lines, "[[rooms.mob_resets]]", target[2], lo, hi)
                    at = m is not None and find_block(lines, "[[rooms.mob_resets.equips]]", target[3], m,
                                                      block_end(lines, m))
                    pad, hdr = "      ", EQ_HDR
                if at is None or at is False:
                    missed.append((vnum, target, ovnum))
                    continue
                end = block_end(lines, at)
                block = ["", pad + hdr, f"{pad}vnum = {ovnum}"]
                block += [f"{pad}max = {limit}"] if limit else []
                block += [f"{pad}count = {count}"] if count > 1 else []
                lines[end:end] = block
                hi += len(block)
                placed += 1
        new = "\n".join(lines)
        tomllib.loads(new)  # must stay valid
        if write and new != text:
            open(f, "w").write(new)
    print(f"P resets {len(resets)}; placed {placed}; missed {len(missed)} {missed[:10]}")


main()
