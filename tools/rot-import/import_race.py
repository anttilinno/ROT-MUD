"""Third pass: OR each mob's ROT race_table flags (off, imm, res, vuln and
affects) into its TOML, as ROT's db2.c does when it loads a mob. Idempotent.
Usage: python3 import_race.py <rot/src dir> <go/data/areas dir>"""
import glob, os, re, sys

src, dst = sys.argv[1], sys.argv[2]
text = open(os.path.join(src, "const.c"), encoding="latin-1").read()
start = text.index("race_table")
body = text[start:text.index("};", start)]
FIELDS = ("act", "affected_by", "off_flags", "imm_flags", "res_flags", "vuln_flags")
# Only flags the Go loader has (import_mobs.py's IMM and OFF tables).
IMM = set("summon charm magic weapon bash pierce slash fire cold lightning acid poison negative holy energy mental disease drowning light sound silver".split())
OFF = set("area_attack backstab bash berserk disarm dodge fade fast kick kick_dirt parry rescue tail trip crush".split())
KNOWN = {"off_flags": OFF, "imm_flags": IMM, "res_flags": IMM, "vuln_flags": IMM}
PREFIX = {"affected_by": "AFF_", "off_flags": "OFF_", "imm_flags": "IMM_", "res_flags": "RES_", "vuln_flags": "VULN_"}
races = {}
for m in re.finditer(r'\{\s*"([^"]+)",\s*(?:TRUE|FALSE),' + r"\s*([^,]*)," * 6, body):
    flags = {}
    for field, val in zip(FIELDS, m.groups()[1:]):
        if field == "act":
            continue
        names = [t.strip()[len(PREFIX[field]):].lower() for t in val.split("|") if t.strip().startswith(PREFIX[field])]
        names = [n for n in names if n in KNOWN.get(field, {n})]
        if names:
            flags[field] = names
    races[m.group(1)] = flags

changed = 0
for f in glob.glob(os.path.join(dst, "*", "mobs", "*.toml")):
    blocks = re.split(r"(?m)^(?=\[\[mobiles\]\]$)", open(f).read())
    dirty = False
    for i, blk in enumerate(blocks):
        m = re.search(r'(?m)^race = "([^"]+)"$', blk)
        if not m or not races.get(m.group(1)):
            continue
        for field, names in races[m.group(1)].items():
            fm = re.search(rf"(?m)^{field} = \[(.*)\]$", blk)
            have = re.findall(r'"([^"]+)"', fm.group(1)) if fm else []
            add = [n for n in names if n not in have]
            if not add:
                continue
            line = f"{field} = [" + ", ".join(f'"{n}"' for n in have + add) + "]"
            blk = blk[: fm.start()] + line + blk[fm.end():] if fm else blk.replace(m.group(0), m.group(0) + "\n" + line, 1)
            dirty = True
        if dirty and blocks[i] != blk:
            blocks[i], changed = blk, changed + 1
    if dirty:
        open(f, "w").write("".join(blocks))
print("mobs updated:", changed)
