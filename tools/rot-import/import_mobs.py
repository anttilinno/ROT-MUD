"""One-off: restore mob fields the TOML conversion dropped (hitroll, AC,
damage type, off/imm/res/vuln flags, size, race) from the original ROT 1.4
.are files. Usage: python3 import_mobs.py <rot/area dir> <go/data/areas dir> [--write]"""
import glob, os, re, sys

ATTACK = {  # ROT const.c attack_table: noun -> damage class
    "none": None, "slice": "slash", "stab": "pierce", "slash": "slash", "whip": "slash",
    "claw": "slash", "blast": "bash", "pound": "bash", "crush": "bash", "grep": "slash",
    "bite": "pierce", "pierce": "pierce", "suction": "bash", "beating": "bash",
    "digestion": "acid", "charge": "bash", "slap": "bash", "punch": "bash", "wrath": "energy",
    "magic": "energy", "divine": "holy", "cleave": "slash", "scratch": "pierce", "peck": "pierce",
    "peckb": "bash", "chop": "slash", "sting": "pierce", "smash": "bash", "shbite": "lightning",
    "flbite": "fire", "frbite": "cold", "acbite": "acid", "chomp": "pierce", "drain": "negative",
    "thrust": "pierce", "slime": "acid", "shock": "lightning", "thwack": "bash", "flame": "fire",
    "chill": "cold", "typo": "slash",
}
IMM = "summon charm magic weapon bash pierce slash fire cold lightning acid poison negative holy energy mental disease drowning light sound".split()
IMM_EXTRA = {24: "silver"}  # ROM letter Y; U-X (and Z iron) have no Go flag
OFF = "area_attack backstab bash berserk disarm dodge fade fast kick kick_dirt parry rescue tail trip crush".split()
SIZES = {"tiny", "small", "medium", "large", "huge", "giant"}


def bits(tok):
    if re.fullmatch(r"-?\d+", tok):
        n = int(tok)
        return [i for i in range(64) if n >> i & 1]
    out = []
    for c in tok:
        if "A" <= c <= "Z":
            out.append(ord(c) - 65)
        elif "a" <= c <= "z":
            out.append(ord(c) - 97 + 26)
    return out


def names(tok, table, extra=None):
    out = []
    for b in bits(tok):
        if b < len(table):
            out.append(table[b])
        elif extra and b in extra:
            out.append(extra[b])
    return out


def read_string(text, pos):
    end = text.index("~", pos)
    return text[pos:end].strip(), end + 1


def parse_are(path):
    text = open(path, encoding="latin-1").read()
    m = re.search(r"^#MOBILES\s*$", text, re.M)
    if not m:
        return {}
    pos, mobs = m.end(), {}
    while True:
        m = re.compile(r"#(\d+)").search(text, pos)
        if not m:
            break
        vnum = int(m.group(1))
        if vnum == 0:
            break
        pos = m.end()
        strs = []
        for _ in range(5):  # keywords, short, long, description, race
            s, pos = read_string(text, pos)
            strs.append(s)
        toks, p = [], pos
        while len(toks) < 27:
            m2 = re.compile(r"\S+").search(text, p)
            toks.append(m2.group(0))
            p = m2.end()
        pos = p
        (act, aff, shd, align, group, level, hitroll, hitd, manad, damd, damtype,
         ac1, ac2, ac3, ac4, off, imm, res, vuln, spos, dpos, sex, wealth,
         form, parts, size, material) = toks[:27]
        mobs[vnum] = dict(
            short=strs[1], race=strs[4], hitroll=int(hitroll),
            ac=[int(a) * 10 for a in (ac1, ac2, ac3, ac4)],
            damage_type=ATTACK.get(damtype.lower()),
            off=names(off, OFF), imm=names(imm, IMM, IMM_EXTRA),
            res=names(res, IMM, IMM_EXTRA), vuln=names(vuln, IMM, IMM_EXTRA),
            size=size.lower() if size.lower() in SIZES else None,
        )
    return mobs


def norm(s):
    return re.sub(r"\s+", " ", s).strip().lower()


def main():
    src, dst, write = sys.argv[1], sys.argv[2], "--write" in sys.argv
    orig = {}
    for f in glob.glob(os.path.join(src, "*.are")):
        try:
            orig.update(parse_are(f))
        except Exception as e:
            print(f"  skip {os.path.basename(f)}: {e}", file=sys.stderr)
    matched = mismatched = missing = 0
    for f in sorted(glob.glob(os.path.join(dst, "*", "mobs", "*.toml"))):
        lines = open(f).read().split("\n")
        out, vnum, short = [], None, None
        for line in lines:
            if line.startswith("vnum = "):
                vnum = int(line.split("=")[1])
            elif line.startswith("short_desc = "):
                short = line.split("=", 1)[1].strip().strip('"')
            out.append(line)
            if line.startswith("level = ") and vnum is not None:
                d = orig.get(vnum)
                if d is None:
                    missing += 1
                    continue
                if norm(d["short"]) != norm(short or ""):
                    mismatched += 1
                    print(f"  mismatch {vnum}: {short!r} vs {d['short']!r}", file=sys.stderr)
                    continue
                matched += 1
                out.append(f"hitroll = {d['hitroll']}")
                out.append("ac = [" + ", ".join(map(str, d["ac"])) + "]")
                if d["damage_type"]:
                    out.append(f'damage_type = "{d["damage_type"]}"')
                for key in ("off", "imm", "res", "vuln"):
                    if d[key]:
                        out.append(f"{key}_flags = [" + ", ".join(f'"{n}"' for n in d[key]) + "]")
                if d["size"] and d["size"] != "medium":
                    out.append(f'size = "{d["size"]}"')
                if d["race"]:
                    out.append(f'race = "{d["race"]}"')
        if write:
            open(f, "w").write("\n".join(out))
    print(f"original mobs {len(orig)}; matched {matched}, short mismatch {mismatched}, not in originals {missing}")


main()
