"""Restore object values in go/data/areas objects TOML from ROT .are files.

Writes [objects.container|light|magic|drink|food|money|portal] sections from
the ROT value line (v0..v4), skipping objects that already have that section.
usage: python3 -I objvalues.py <rot/area dir> <go/data/areas dir> [--write]
"""
import glob, json, os, re, sys
from collections import Counter

CONT = {"A": "closeable", "B": "pickproof", "C": "closed", "D": "locked"}


def tok(line):
    return [x[1:-1] if x.startswith("'") else x for x in re.findall(r"'[^']*'|\S+", line)]


def num(s):
    return int(s) if s.lstrip("-").isdigit() else 0


def letters(s):
    if s.lstrip("-").isdigit():
        n = int(s)
        return {chr(65 + i) for i in range(26) if n >> i & 1}
    return set(s)


def spell(s):
    return "" if s in ("", "-1", "0") or s.lstrip("-").isdigit() else s


def section(typ, v):
    v = (v + ["0"] * 5)[:5]
    if typ == "container":
        flags = sorted(CONT[c] for c in letters(v[1]) if c in CONT)
        return "container", {"capacity": num(v[0]), "flags": flags, "key": max(0, num(v[2])), "max_weight": num(v[3])}
    if typ == "light":
        return "light", {"duration": num(v[2])}
    if typ in ("potion", "scroll", "pill"):
        sp = [spell(x) for x in v[1:5] if spell(x)]
        return ("magic", {"level": num(v[0]), "spells": sp}) if sp else (None, None)
    if typ in ("wand", "staff"):
        sp = [spell(v[3])] if spell(v[3]) else []
        return "magic", {"level": num(v[0]), "max_charges": num(v[1]), "charges": num(v[2]), "spells": sp}
    if typ in ("drink", "fountain"):
        return "drink", {"capacity": num(v[0]), "current": num(v[1]), "liquid": v[2], "poisoned": num(v[3]) != 0 or v[3].isalpha()}
    if typ == "food":
        return "food", {"full": num(v[0]), "hunger": num(v[1]), "poisoned": num(v[3]) != 0 or v[3].isalpha()}
    if typ == "money":
        return "money", {"amount": num(v[0]) + num(v[1]) * 100}
    if typ == "portal":
        return "portal", {"charges": num(v[0]), "dest_vnum": num(v[3])}
    return None, None


def toml_value(x):
    if isinstance(x, bool):
        return "true" if x else "false"
    if isinstance(x, list):
        return "[" + ", ".join(json.dumps(i) for i in x) + "]"
    if isinstance(x, str):
        return json.dumps(x)
    return str(x)


def parse(path):
    text = open(path, encoding="latin-1").read()
    if "#OBJECTS" not in text:
        return {}
    L = text[text.index("#OBJECTS") + 9:].split("\n")
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
        for _ in range(4):  # name, short, long, material
            while not L[i].rstrip().endswith("~"):
                i += 1
            i += 1
        typ = L[i].split()[0]
        out[v] = (typ, tok(L[i + 1]))
        i += 2
    return out


def main():
    are_dir, toml_dir, write = sys.argv[1], sys.argv[2], "--write" in sys.argv
    src = {}
    for f in glob.glob(os.path.join(are_dir, "*.are")):
        src.update(parse(f))
    done = Counter()
    for f in sorted(glob.glob(os.path.join(toml_dir, "*/objects/*.toml"))):
        lines = open(f).read().split("\n")
        out, vnum, block, changed = [], None, [], False

        def flush():
            nonlocal changed
            if vnum is None or vnum not in src:
                out.extend(block)
                return
            typ, vals = src[vnum]
            name, fields = section(typ, vals)
            if not name or any(l.strip() == f"[objects.{name}]" for l in block):
                out.extend(block)
                return
            body = block[:]
            while body and body[-1] == "":
                body.pop()
            body += ["", f"  [objects.{name}]"] + [f"  {k} = {toml_value(x)}" for k, x in fields.items()] + [""]
            out.extend(body)
            done[name] += 1
            changed = True

        for line in lines:
            if line.strip() == "[[objects]]":
                flush()
                block, vnum = [], None
            m = re.match(r"^vnum = (\d+)$", line)
            if m and vnum is None:
                vnum = int(m.group(1))
            block.append(line)
        flush()
        if changed and write:
            open(f, "w").write("\n".join(out).rstrip("\n") + "\n")
    print("restored sections:", dict(done))


main()
