"""Restore [mobiles.shop] in go/data/areas TOML from ROT #SHOPS sections.

ROT line: keeper type1..type5 profit_buy profit_sell open_hour close_hour.
ROT's close hour is inclusive; the Go Shop.IsOpen treats it as exclusive,
so close+1 is written (0..23 becomes 0..24, "always open").
usage: python3 -I shops.py <rot/area dir> <go/data/areas dir> [--write]
"""
import glob, os, re, sys

ITEM = {1: "light", 2: "scroll", 3: "wand", 4: "staff", 5: "weapon", 8: "treasure",
        9: "armor", 10: "potion", 11: "clothing", 12: "furniture", 13: "trash",
        15: "container", 17: "drink", 18: "key", 19: "food", 20: "money", 22: "boat",
        25: "fountain", 26: "pill", 28: "map", 29: "portal", 32: "gem", 33: "jewelry"}


def main():
    are_dir, toml_dir, write = sys.argv[1], sys.argv[2], "--write" in sys.argv
    shops, unknown = {}, set()
    for f in glob.glob(os.path.join(are_dir, "*.are")):
        text = open(f, encoding="latin-1").read()
        if "#SHOPS" not in text:
            continue
        for line in text[text.index("#SHOPS") + 6:].split("\n"):
            tok = line.split()
            if not tok:
                continue
            if tok[0] == "0":
                break
            if len(tok) < 10 or not all(t.lstrip("-").isdigit() for t in tok[:10]):
                continue
            n = list(map(int, tok[:10]))
            types = []
            for t in n[1:6]:
                if t in ITEM:
                    types.append(ITEM[t])
                elif t:
                    unknown.add(t)
            shops[n[0]] = (types, n[6], n[7], n[8], n[9] + 1)

    done = 0
    for f in sorted(glob.glob(os.path.join(toml_dir, "*/mobs/*.toml"))):
        lines = open(f).read().split("\n")
        if any(l.strip() == "[mobiles.shop]" for l in lines):
            continue
        out, vnum, changed = [], None, False

        def flush():
            nonlocal done, changed
            if vnum in shops:
                types, pb, ps, oh, ch = shops[vnum]
                while out and out[-1] == "":
                    out.pop()
                bt = ", ".join(f'"{t}"' for t in types)
                out.extend(["", "  [mobiles.shop]", f"  buy_types = [{bt}]", f"  profit_buy = {pb}",
                            f"  profit_sell = {ps}", f"  open_hour = {oh}", f"  close_hour = {ch}", ""])
                done += 1
                changed = True

        for line in lines:
            if line.strip() == "[[mobiles]]":
                if vnum is not None:
                    flush()
                vnum = None
            m = re.match(r"^vnum = (\d+)$", line)
            if m and vnum is None:
                vnum = int(m.group(1))
            out.append(line)
        if vnum is not None:
            flush()
        if changed and write:
            open(f, "w").write("\n".join(out).rstrip("\n") + "\n")
    print(f"source shops={len(shops)} restored={done} unknown item types={sorted(unknown)}")


main()
