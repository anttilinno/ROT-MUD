"""Second pass: restore ROT act class flags the conversion dropped
(E ranger, L druid, P vampire) onto act_flags of matching TOML mobs."""
import glob, os, re, sys
ns = {}
exec(open(os.path.join(os.path.dirname(__file__), "import_mobs.py")).read().replace("\nmain()\n", "\n"), ns)
EXTRA = {"E": "ranger", "L": "druid", "P": "vampire"}
src, dst = sys.argv[1], sys.argv[2]
acts = {}
for f in glob.glob(os.path.join(src, "*.are")):
    text = open(f, encoding="latin-1").read()
    m = re.search(r"^#MOBILES\s*$", text, re.M)
    if not m:
        continue
    pos = m.end()
    try:
        while True:
            mm = re.compile(r"#(\d+)").search(text, pos)
            if not mm or mm.group(1) == "0":
                break
            pos = mm.end()
            for _ in range(5):
                _, pos = ns["read_string"](text, pos)
            tok = re.compile(r"\S+").search(text, pos).group(0)
            acts[int(mm.group(1))] = [n for k, n in EXTRA.items() if k in tok and not tok.isdigit()]
    except Exception:
        pass
changed = 0
for f in glob.glob(os.path.join(dst, "*", "mobs", "*.toml")):
    lines, vnum, dirty = open(f).read().split("\n"), None, False
    for i, line in enumerate(lines):
        if line.startswith("vnum = "):
            vnum = int(line.split("=")[1])
        elif line.startswith("act_flags = [") and acts.get(vnum):
            add = [n for n in acts[vnum] if f'"{n}"' not in line]
            if add:
                lines[i] = line[:-1] + "".join(f', "{n}"' for n in add) + "]"
                dirty, changed = True, changed + 1
    if dirty:
        open(f, "w").write("\n".join(lines))
print("mobs updated:", changed)
