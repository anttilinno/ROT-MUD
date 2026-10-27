"""Restore object and room extra descriptions in go/data/areas TOML from ROT .are files.

A ROT extra description is an 'E' line, then 'keywords~', then text up to '~'.
usage: python3 -I extradescs.py <rot/area dir> <go/data/areas dir> [--write]
"""
import glob, json, os, re, sys

SECTION_END = re.compile(r"^#(OBJECTS|ROOMS|RESETS|SHOPS|SPECIALS|MOBPROGS|MOBILES|HELPS|\$)\b", re.M)


def section(text, name):
    if "#" + name not in text:
        return ""
    sec = text[text.index("#" + name) + len(name) + 1:]
    m = SECTION_END.search(sec)
    return sec[:m.start()] if m else sec


def parse_eds(sec):
    """vnum -> [(keywords, text)] for every 'E' block in a section."""
    out, vnum = {}, None
    lines = sec.split("\n")
    i = 0
    while i < len(lines):
        line = lines[i]
        m = re.match(r"^#(\d+)\s*$", line)
        if m:
            vnum = int(m.group(1))
        elif line.strip() == "E" and vnum:
            kw_line = lines[i + 1]
            if not kw_line.rstrip().endswith("~"):
                i += 1
                continue
            keywords = kw_line.rstrip()[:-1].split()
            body, j = [], i + 2
            while j < len(lines) and not lines[j].rstrip().endswith("~"):
                body.append(lines[j].rstrip())
                j += 1
            if j < len(lines):
                last = lines[j].rstrip()[:-1]
                if last:
                    body.append(last)
            text = "\n".join(body).strip("\n")
            if keywords and text:
                out.setdefault(vnum, []).append((keywords, text))
            i = j
        i += 1
    return out


def toml_block(table, keywords, text):
    kws = ", ".join(json.dumps(k) for k in keywords)
    body = text.replace("\\", "\\\\").replace('"""', '\\"\\"\\"')
    return [f"[[{table}.extra_descs]]", f"keywords = [{kws}]", f'description = """\n{body}"""', ""]


def patch(files, table, eds, write):
    done = 0
    for f in sorted(files):
        lines = open(f).read().split("\n")
        if any(l.strip() == f"[[{table}.extra_descs]]" for l in lines):
            continue  # already restored
        out, vnum, changed = [], None, False

        def flush():
            nonlocal done, changed
            if vnum in eds:
                while out and out[-1] == "":
                    out.pop()
                out.append("")
                for kws, text in eds[vnum]:
                    out.extend(toml_block(table, kws, text))
                    done += 1
                changed = True

        for line in lines:
            if line.strip() == f"[[{table}]]":
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
    return done


def main():
    are_dir, toml_dir, write = sys.argv[1], sys.argv[2], "--write" in sys.argv
    obj_eds, room_eds = {}, {}
    for f in glob.glob(os.path.join(are_dir, "*.are")):
        text = open(f, encoding="latin-1").read()
        obj_eds.update(parse_eds(section(text, "OBJECTS")))
        room_eds.update(parse_eds(section(text, "ROOMS")))
    no = patch(glob.glob(os.path.join(toml_dir, "*/objects/*.toml")), "objects", obj_eds, write)
    nr = patch(glob.glob(os.path.join(toml_dir, "*/rooms/*.toml")), "rooms", room_eds, write)
    print(f"source: objects {sum(map(len, obj_eds.values()))} rooms {sum(map(len, room_eds.values()))}; restored: objects {no} rooms {nr}")


main()
