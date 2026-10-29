"""Restore mob fields the TOML conversion dropped, from ROT .are #MOBILES.

Restores alignment, mob group (check_assist), start/default position, mana dice, and merges the ROT
act, affect, shield and off flags into the existing flag lists (ROT names
mapped to the names the Go loader reads). Re-runnable: managed keys are
replaced in place, flag lists are unions, mana dice are rewritten.

usage: python3 -I mobs.py <rot/area dir> <go/data/areas dir> [--write]
"""
import glob, os, re, sys, tomllib

# ROM flag letters: A-Z are bits 0-25, a-z bits 26-51 (merc.h aa, bb, ...).
ACT = ["npc", "sentinel", "scavenger", "key", "ranger", "aggressive", "stay_area", "wimpy", "pet",
       "train", "practice", "druid", "no_body", "nb_drop", "undead", "vampire", "cleric", "mage",
       "thief", "warrior", "no_align", "no_purge", "outdoors", "is_satan", "indoors", "is_priest",
       "healer", "gain", "update_always", "changer"]
AFF = ["blind", None, "detect_evil", "detect_invis", "detect_magic", "detect_hidden", "detect_good",
       None, "faerie_fire", "infrared", "curse", "farsight", "poison", None, None, "sneak", "hide",
       "sleep", "charm", "flying", "pass_door", "haste", "calm", "plague", "weaken", "dark_vision",
       "berserk", "swim", "regeneration", "slow"]
# The Go port keeps invisible, sanctuary and protect evil/good as affect
# bits; the elemental shields stay shield flags.
SHIELD_AFF = {1: "invisible", 7: "sanctuary", 13: "protect_evil", 14: "protect_good"}
SHIELD = {2: "ice", 3: "fire", 4: "shock"}
OFF = ["area_attack", "backstab", "bash", "berserk", "disarm", "dodge", "fade", "fast", "kick",
       "kick_dirt", "parry", "rescue", "tail", "trip", "crush", "assist_all", "assist_align",
       "assist_race", "assist_players", "assist_guard", "assist_vnum"]
POS = {"stand": "standing", "sit": "sitting", "rest": "resting", "sleep": "sleeping",
       "fight": "fighting", "8": "standing"}


def bits(tok):
    if re.fullmatch(r"-?\d+", tok):
        n = int(tok)
        return [i for i in range(64) if n >> i & 1]
    return [ord(c) - 65 if c <= "Z" else ord(c) - 97 + 26 for c in tok if c.isalpha()]


def names(tok, table):
    out = []
    for b in bits(tok):
        n = table.get(b) if isinstance(table, dict) else (table[b] if b < len(table) else None)
        if n:
            out.append(n)
    return out


def dice(s):
    m = re.fullmatch(r"(\d+)d(\d+)\+?(-?\d+)?", s)
    return (int(m[1]), int(m[2]), int(m[3] or 0)) if m else None


def parse_are(path):
    text = open(path, encoding="latin-1").read()
    m = re.search(r"^#MOBILES\s*$", text, re.M)
    if not m:
        return {}
    pos, mobs = m.end(), {}
    while True:
        m = re.compile(r"#(\d+)").search(text, pos)
        if not m or int(m.group(1)) == 0:
            break
        vnum, pos = int(m.group(1)), m.end()
        strs = []
        for _ in range(5):  # keywords, short, long, description, race
            end = text.index("~", pos)
            strs.append(text[pos:end].strip())
            pos = end + 1
        toks = []
        while len(toks) < 27:
            m2 = re.compile(r"\S+").search(text, pos)
            toks.append(m2.group(0))
            pos = m2.end()
        (act, aff, shd, align, group, _level, _hitroll, _hitd, manad, _damd, _damtype,
         *_ac, off, _imm, _res, _vuln, spos, dpos, _sex, _wealth, _form, _parts, _size, _mat) = toks
        mobs[vnum] = dict(
            short=strs[1], align=int(align), group=int(group), mana=dice(manad),
            act=names(act, ACT), aff=names(aff, AFF) + names(shd, SHIELD_AFF),
            shield=names(shd, SHIELD), off=names(off, OFF),
            start=POS.get(spos, "standing"), default=POS.get(dpos, "standing"),
        )
    return mobs


def norm(s):
    return re.sub(r"\s+", " ", s).strip().lower()


def toml_list(xs):
    return "[" + ", ".join(f'"{x}"' for x in xs) + "]"


def merge(have, add):
    return list(have or []) + [x for x in add if x not in (have or [])]


def restore_block(lines, t, d):
    """Rewrite one [[mobiles]] block's lines with the ROT fields in d."""
    want = {
        "alignment": str(d["align"]),
        "group": str(d["group"]) if d["group"] else None,
        "act_flags": toml_list(merge(t.get("act_flags"), d["act"])),
        "affected_by": toml_list(merge(t.get("affected_by"), d["aff"])) if d["aff"] or t.get("affected_by") else None,
        "shield_flags": toml_list(d["shield"]) if d["shield"] else None,
        "off_flags": toml_list(merge(t.get("off_flags"), d["off"])) if d["off"] or t.get("off_flags") else None,
        "start_pos": f'"{d["start"]}"' if d["start"] != "standing" else None,
        "default_pos": f'"{d["default"]}"' if d["default"] != "standing" else None,
    }
    out, done, in_mana = [], set(), False
    for line in lines:
        if line.strip() == "[mobiles.mana_dice]":
            in_mana = True
            if out and not out[-1].strip():
                out.pop()
            continue
        if in_mana:
            if line.strip().startswith("[") or not line.strip():
                in_mana = False
            else:
                continue
        key = line.split(" = ", 1)[0] if re.match(r"^[a-z_]+ = ", line) else None
        if key in want:
            done.add(key)
            if want[key] is not None:
                out.append(f"{key} = {want[key]}")
            continue
        out.append(line)
    at = next(i for i, l in enumerate(out) if l.startswith("level = ")) + 1
    for key, val in want.items():
        if key not in done and val is not None:
            out.insert(at, f"{key} = {val}")
            at += 1
    tail = []
    while out and not out[-1].strip():
        tail.append(out.pop())
    if d["mana"] and d["mana"] != (0, 0, 0):
        n, s, b = d["mana"]
        out += ["", "  [mobiles.mana_dice]", f"  number = {n}", f"  size = {s}", f"  bonus = {b}"]
    return out + tail


def main():
    src, dst, write = sys.argv[1], sys.argv[2], "--write" in sys.argv
    orig = {}
    for f in glob.glob(os.path.join(src, "*.are")):
        try:
            orig.update(parse_are(f))
        except (ValueError, IndexError) as e:  # files ROT itself can't load (not in area.lst)
            print(f"  skip {os.path.basename(f)}: {e}", file=sys.stderr)
    restored, skipped = 0, []
    for f in sorted(glob.glob(os.path.join(dst, "*", "mobs", "*.toml"))):
        text = open(f).read()
        parsed = {m["vnum"]: m for m in tomllib.loads(text).get("mobiles", [])}
        parts = re.split(r"(?m)^(?=\[\[mobiles\]\]$)", text)
        out = []
        for part in parts:
            m = re.search(r"(?m)^vnum = (\d+)$", part)
            if not m:
                out.append(part)
                continue
            vnum = int(m.group(1))
            t, d = parsed[vnum], orig.get(vnum)
            if d is None or norm(d["short"]) != norm(t.get("short_desc", "")):
                skipped.append(vnum)
                out.append(part)
                continue
            restored += 1
            out.append("\n".join(restore_block(part.split("\n"), t, d)))
        new = "".join(out)
        tomllib.loads(new)  # must stay valid
        if write and new != text:
            open(f, "w").write(new)
    print(f"source {len(orig)}; restored {restored}; skipped {len(skipped)} {skipped[:15]}")


main()
