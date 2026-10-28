"""Restore room_flags (and heal/mana rates) in go/data/areas rooms TOML from ROT .are files.

ROT room header: 'area room_flags sector'; flags are letters (ABC) or a decimal
bitmask. 'H n' / 'M n' lines set heal and mana rates.
usage: python3 -I roomflags.py <rot/area dir> <go/data/areas dir> [--write]
"""
import glob, os, re, sys

FLAGS = {"A": "dark", "C": "no_mob", "D": "indoors", "J": "private", "K": "safe", "L": "solitary",
         "M": "pet_shop", "N": "no_recall", "O": "imp_only", "P": "gods_only", "Q": "heroes_only",
         "R": "newbies_only", "S": "law", "T": "nowhere"}


def to_letters(s):
    if s.lstrip("-").isdigit():
        n = int(s)
        return {chr(65 + i) for i in range(26) if n >> i & 1}
    return set(s)


def parse(path):
    text = open(path, encoding="latin-1").read()
    if "#ROOMS" not in text:
        return {}
    L = text[text.index("#ROOMS") + 7:].split("\n")
    out, i = {}, 0
    while i < len(L):
        m = re.match(r"^#(\d+)\s*$", L[i])
        if not m:
            i += 1
            continue
        v = int(m.group(1))
        if v == 0:
            break
        i += 1
        for _ in range(2):  # name~, description~
            while not L[i].rstrip().endswith("~"):
                i += 1
            i += 1
        tok = L[i].split()
        i += 1
        room = {"flags": sorted(FLAGS[c] for c in to_letters(tok[1]) if c in FLAGS) if len(tok) > 1 else []}
        while i < len(L) and L[i].strip() != "S" and not re.match(r"^#\d+\s*$", L[i]):
            s = L[i].split()
            for k in range(0, len(s) - 1):
                if s[k] in ("H", "M") and s[k + 1].isdigit() and L[i][0] in "HM":
                    room["heal" if s[k] == "H" else "mana"] = int(s[k + 1])
            i += 1
        out[v] = room
    return out


def main():
    are_dir, toml_dir, write = sys.argv[1], sys.argv[2], "--write" in sys.argv
    src = {}
    for f in glob.glob(os.path.join(are_dir, "*.are")):
        src.update(parse(f))
    nflags = nrates = 0
    for f in sorted(glob.glob(os.path.join(toml_dir, "*/rooms/*.toml"))):
        lines = open(f).read().split("\n")
        out, vnum, changed = [], None, False
        for line in lines:
            if line.strip() == "[[rooms]]":
                vnum = None
            m = re.match(r"^vnum = (\d+)$", line)
            if m and vnum is None:
                vnum = int(m.group(1))
            if line.startswith(("room_flags = ", "mana_rate = ", "heal_rate = ")) and vnum in src:
                continue  # rewritten below
            out.append(line)
            if line.startswith("sector = ") and vnum in src:
                r = src[vnum]
                if r["flags"]:
                    out.append("room_flags = [" + ", ".join(f'"{x}"' for x in r["flags"]) + "]")
                    nflags += 1
                for key, tk in (("heal", "heal_rate"), ("mana", "mana_rate")):
                    if key in r:
                        out.append(f"{tk} = {r[key]}")
                        nrates += 1
                changed = True
        if changed and write:
            open(f, "w").write("\n".join(out))
    print(f"source rooms={len(src)} rooms with flags restored={nflags} rates={nrates}")


main()
