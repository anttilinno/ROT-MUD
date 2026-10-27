"""Restore weapon_type / damage_type / flags in go/data/areas TOML from ROT .are files.

usage: python3 -I weapons.py <rot/area dir> <go/data/areas dir> [--write]
"""
import glob, os, re, shlex, sys

# ROM attack_table noun -> damage class (names parseDamageType accepts)
ATTACK = {
    "hit": None, "none": None,
    "slice": "slash", "stab": "pierce", "slash": "slash", "whip": "slash",
    "claw": "slash", "blast": "bash", "pound": "bash", "crush": "bash",
    "grep": "slash", "bite": "pierce", "pierce": "pierce", "suction": "bash",
    "beating": "bash", "digestion": "acid", "charge": "bash", "slap": "bash",
    "punch": "bash", "wrath": "energy", "magic": "energy", "divine": "holy",
    "cleave": "slash", "scratch": "pierce", "peck": "pierce", "peckb": "bash",
    "chop": "slash", "sting": "pierce", "smash": "bash", "shbite": "lightning",
    "flbite": "fire", "frbite": "cold", "acbite": "acid", "chomp": "pierce",
    "drain": "negative", "thrust": "pierce", "slime": "acid", "shock": "lightning",
    "thwack": "bash", "flame": "fire", "chill": "cold",
}
CLASSES = {"exotic", "sword", "dagger", "spear", "mace", "axe", "flail", "whip", "polearm"}
FLAGS = {"A": "flaming", "B": "frost", "C": "vampiric", "D": "sharp", "E": "vorpal",
         "F": "two_hands", "G": "shocking", "H": "poison"}


def read_tilde(lines, i):
    """Skip a ~-terminated string starting at line i; return next index."""
    while not lines[i].rstrip().endswith("~"):
        i += 1
    return i + 1


def parse_are(path):
    out = {}
    lines = open(path, encoding="latin-1").read().split("\n")
    try:
        i = lines.index("#OBJECTS") + 1
    except ValueError:
        return out
    while i < len(lines):
        m = re.match(r"^#(\d+)\s*$", lines[i])
        if not m:
            i += 1
            continue
        vnum = int(m.group(1))
        if vnum == 0:
            break
        i += 1
        for _ in range(4):  # name, short, long, material
            i = read_tilde(lines, i)
        itype = lines[i].split()[0]
        if itype == "weapon":
            out[vnum] = shlex.split(lines[i + 1].replace("'", '"'))
        i += 2
    return out


def main():
    are_dir, toml_dir, write = sys.argv[1], sys.argv[2], "--write" in sys.argv
    src = {}
    for f in glob.glob(os.path.join(are_dir, "*.are")):
        src.update(parse_are(f))

    fixed, missing, unknown = 0, [], set()
    for f in sorted(glob.glob(os.path.join(toml_dir, "*/objects/*.toml"))):
        lines = open(f).read().split("\n")
        out, vnum, changed = [], None, False
        for line in lines:
            m = re.match(r"^vnum = (\d+)$", line)
            if m:
                vnum = int(m.group(1))
            out.append(line)
            if line.strip() != "[objects.weapon]":
                continue
            vals = src.get(vnum)
            if vals is None:
                missing.append(vnum)
                continue
            wclass, noun, flags = vals[0].lower(), vals[3].lower(), vals[4]
            if wclass not in CLASSES:  # e.g. "staff": not in ROT's weapon_class, loads as exotic
                unknown.add(("class->exotic", wclass))
                wclass = "exotic"
            out.append(f'  weapon_type = "{wclass}"')
            if noun in ATTACK:
                if ATTACK[noun]:
                    out.append(f'  damage_type = "{ATTACK[noun]}"')
            else:
                unknown.add(("noun", noun))
            names = [FLAGS[c] for c in flags if c in FLAGS]
            if names:
                out.append("  flags = [" + ", ".join(f'"{n}"' for n in names) + "]")
            fixed += 1
            changed = True
        if changed and write:
            open(f, "w").write("\n".join(out))
    print(f"source weapons={len(src)} patched={fixed} not_in_source={missing} unknown={sorted(unknown)}")


main()
