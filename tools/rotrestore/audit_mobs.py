"""Audit: ROT .are #MOBILES (+ #SPECIALS) vs go/data/areas mobs TOML, per field.

usage: python3 -I audit_mobs.py <rot dir (contains area/ and src/)> <go/data/areas>
Prints, per field: how many mobs match, differ, or lack the field in TOML.
"""
import glob, os, re, sys, tomllib
from collections import Counter

ROT, AREAS = sys.argv[1], sys.argv[2]
SRC = os.path.join(ROT, "src")


def defines():
    out = {}
    for line in open(os.path.join(SRC, "merc.h"), encoding="latin-1"):
        m = re.match(r"#define\s+(\w+)\s+\(?([A-Za-z]{1,2}|\d+)\)?", line)
        if m:
            out[m.group(1)] = m.group(2)
    return out


DEF = defines()


def table(name):
    """letter -> flag name for a tables.c flag_type array."""
    src = open(os.path.join(SRC, "tables.c"), encoding="latin-1").read()
    m = re.search(r"flag_type\s+" + name + r"\[\]\s*=\s*\{(.*?)\n\};", src, re.S)
    out = {}
    for n, d in re.findall(r'\{\s*"([^"]+)",\s*(\w+)', m.group(1)):
        key = d if re.fullmatch(r"[A-Z]|[a-z]{1,2}", d) else DEF.get(d, d)
        out[key] = n
    return out


def letters(s, tbl):
    if s == "0":
        return set()
    out, i = set(), 0
    while i < len(s):
        # ROM: aa, bb .. are two-letter bits (a-z after Z)
        if s[i].islower() and i + 1 < len(s) and s[i + 1] == s[i]:
            key = s[i:i + 2]
            i += 2
        else:
            key = s[i]
            i += 1
        out.add(tbl.get(key, "?" + key))
    return out


ACT, AFF, OFF, IMM = table("act_flags"), table("affect_flags"), table("off_flags"), table("imm_flags")
RES, VULN, SHIELD = table("res_flags"), table("vuln_flags"), table("shield_flags")


def read_str(lines, i):
    buf = []
    while True:
        line = lines[i]
        i += 1
        if line.rstrip().endswith("~"):
            buf.append(line.rstrip()[:-1])
            return "\n".join(buf), i
        buf.append(line)


def parse_mobs(path):
    text = open(path, encoding="latin-1").read()
    if "#MOBILES" not in text:
        return {}
    lines = text[text.index("#MOBILES") + 9:].split("\n")
    out, i = {}, 0
    while i < len(lines):
        m = re.match(r"^#(\d+)\s*$", lines[i])
        if not m:
            i += 1
            continue
        vnum = int(m.group(1))
        if vnum == 0:
            break
        i += 1
        try:
            _, i = read_str(lines, i)  # name
            _, i = read_str(lines, i)  # short
            _, i = read_str(lines, i)  # long
            _, i = read_str(lines, i)  # desc
            race, i = read_str(lines, i)
            a = lines[i].split(); i += 1
            b = lines[i].split(); i += 1
            ac = lines[i].split(); i += 1
            f = lines[i].split(); i += 1
            p = lines[i].split(); i += 1
            out[vnum] = dict(
                race=race.strip(), act=letters(a[0], ACT), aff=letters(a[1], AFF),
                shield=letters(a[2], SHIELD) if len(a) >= 5 else set(),
                align=int(a[-2]), level=int(b[0]), hitroll=int(b[1]),
                hit=b[2], mana=b[3], dam=b[4], damtype=b[5] if len(b) > 5 else "",
                ac=[int(x) * 10 for x in ac[:4]],
                off=letters(f[0], OFF), imm=letters(f[1], IMM), res=letters(f[2], RES), vuln=letters(f[3], VULN),
                start=p[0], default=p[1], sex=p[2],
            )
        except (IndexError, ValueError):
            pass
    return out


def specials(path):
    text = open(path, encoding="latin-1").read()
    if "#SPECIALS" not in text:
        return {}
    sec = text[text.index("#SPECIALS"):]
    return {int(v): s for v, s in re.findall(r"^M\s+(\d+)\s+(\w+)", sec, re.M)}


def dice(d):
    m = re.match(r"(\d+)d(\d+)\+?(-?\d+)?", d)
    return (int(m[1]), int(m[2]), int(m[3] or 0)) if m else None


src, spec = {}, {}
for f in glob.glob(os.path.join(ROT, "area", "*.are")):
    src.update(parse_mobs(f))
    spec.update(specials(f))

toml = {}
for f in glob.glob(os.path.join(AREAS, "*/mobs/*.toml")):
    for m in tomllib.load(open(f, "rb")).get("mobiles", []):
        toml[m["vnum"]] = m

stats = {k: Counter() for k in ["race", "act", "aff", "shield", "align", "level", "hitroll", "hit_dice", "mana_dice",
                                "dam_dice", "dam_type", "ac", "off", "imm", "res", "vuln", "start_pos", "sex", "special"]}
examples = {k: [] for k in stats}


def note(k, ok, ex):
    stats[k]["match" if ok else "differ"] += 1
    if not ok and len(examples[k]) < 3:
        examples[k].append(ex)


def flagset(lst):
    return {x.lower().replace(" ", "_") for x in (lst or [])}


for v, s in src.items():
    t = toml.get(v)
    if not t:
        continue
    note("race", (t.get("race") or "").lower() == s["race"].lower(), (v, s["race"], t.get("race")))
    note("act", {x.lower() for x in s["act"]} - {"npc", "is_npc"} <= flagset(t.get("act_flags")) | {"npc"}, (v, sorted(s["act"]), t.get("act_flags")))
    note("aff", {x.lower() for x in s["aff"]} <= flagset(t.get("affected_by")), (v, sorted(s["aff"]), t.get("affected_by")))
    note("shield", not s["shield"] or {x.lower() for x in s["shield"]} <= flagset(t.get("affected_by")) | flagset(t.get("shield_flags")), (v, sorted(s["shield"]), t.get("affected_by")))
    note("align", t.get("alignment", 0) == s["align"], (v, s["align"], t.get("alignment")))
    note("level", t.get("level") == s["level"], (v, s["level"], t.get("level")))
    note("hitroll", t.get("hitroll", 0) == s["hitroll"], (v, s["hitroll"], t.get("hitroll")))
    for k, sk, tk in [("hit_dice", "hit", "hit_dice"), ("mana_dice", "mana", "mana_dice"), ("dam_dice", "dam", "damage_dice")]:
        td = t.get(tk) or {}
        note(k, dice(s[sk]) == (td.get("number", 0), td.get("size", 0), td.get("bonus", 0)), (v, s[sk], td))
    note("dam_type", (t.get("damage_type") or "") == s["damtype"], (v, s["damtype"], t.get("damage_type")))
    note("ac", list(t.get("ac", [0, 0, 0, 0])) == s["ac"], (v, s["ac"], t.get("ac")))
    for k, tk in [("off", "off_flags"), ("imm", "imm_flags"), ("res", "res_flags"), ("vuln", "vuln_flags")]:
        note(k, {x.lower() for x in s[k]} <= flagset(t.get(tk)), (v, sorted(s[k]), t.get(tk)))
    note("start_pos", (t.get("start_pos") or "stand") == s["start"], (v, s["start"], t.get("start_pos")))
    note("sex", (t.get("sex") or "") == s["sex"], (v, s["sex"], t.get("sex")))
    note("special", t.get("special", "") == spec.get(v, ""), (v, spec.get(v), t.get("special")))

print(f"mobs: source {len(src)}, in toml {sum(1 for v in src if v in toml)}")
for k, c in stats.items():
    print(f"  {k:10} match {c['match']:5}  differ {c['differ']:5}   e.g. {examples[k][:2]}")
