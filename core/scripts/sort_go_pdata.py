#!/usr/bin/env python3
"""Make Go c-archive objects consumable by MSVC link.exe.

Go's Windows c-archive output currently contains a go.o member whose .pdata
RUNTIME_FUNCTION entries may not be sorted by function start address. MSVC's
link.exe rejects that with LNK1223. This script patches the archive in place by
sorting .pdata entries inside go.o and moving the corresponding relocation
records with their entries.
"""

from __future__ import annotations

import argparse
import struct
import sys
from dataclasses import dataclass
from pathlib import Path


ARCHIVE_MAGIC = b"!<arch>\n"
MEMBER_HEADER_SIZE = 60
COFF_HEADER_SIZE = 20
SECTION_HEADER_SIZE = 40
PDATA_ENTRY_SIZE = 12
RELOCATION_SIZE = 10


@dataclass
class ArchiveMember:
    name: str
    header_offset: int
    data_offset: int
    size: int


@dataclass
class Section:
    name: str
    raw_size: int
    raw_offset: int
    relocation_offset: int
    relocation_count: int


def parse_member_name(raw_name: bytes, longnames: bytes | None) -> str:
    name = raw_name.decode("utf-8", errors="replace").strip()
    if name == "/":
        return "/"
    if name == "//":
        return "//"
    if name.startswith("/") and name[1:].strip().isdigit() and longnames is not None:
        start = int(name[1:].strip())
        end = longnames.find(b"/\n", start)
        if end == -1:
            end = longnames.find(b"\x00", start)
        if end == -1:
            end = len(longnames)
        return longnames[start:end].decode("utf-8", errors="replace")
    return name.rstrip("/")


def list_members(data: bytes) -> list[ArchiveMember]:
    if not data.startswith(ARCHIVE_MAGIC):
        raise ValueError("not a COFF archive")

    members: list[ArchiveMember] = []
    longnames: bytes | None = None
    offset = len(ARCHIVE_MAGIC)

    while offset + MEMBER_HEADER_SIZE <= len(data):
        header = data[offset : offset + MEMBER_HEADER_SIZE]
        if header[58:60] != b"`\n":
            raise ValueError(f"invalid archive member header at offset {offset}")

        size_text = header[48:58].decode("ascii", errors="strict").strip()
        size = int(size_text) if size_text else 0
        data_offset = offset + MEMBER_HEADER_SIZE
        raw_name = header[:16]
        name = parse_member_name(raw_name, longnames)

        member = ArchiveMember(name, offset, data_offset, size)
        members.append(member)

        if name == "//":
            longnames = data[data_offset : data_offset + size]

        offset = data_offset + size
        if offset % 2:
            offset += 1

    return members


def section_name(raw: bytes, string_table: bytes) -> str:
    name = raw.rstrip(b"\x00")
    if name.startswith(b"/") and name[1:].isdigit():
        start = int(name[1:])
        end = string_table.find(b"\x00", start)
        if end == -1:
            end = len(string_table)
        return string_table[start:end].decode("utf-8", errors="replace")
    return name.decode("utf-8", errors="replace")


def parse_sections(obj: bytes) -> list[Section]:
    if len(obj) < COFF_HEADER_SIZE:
        raise ValueError("go.o is too small to be a COFF object")

    section_count = struct.unpack_from("<H", obj, 2)[0]
    symbol_table_offset = struct.unpack_from("<I", obj, 8)[0]
    symbol_count = struct.unpack_from("<I", obj, 12)[0]
    optional_header_size = struct.unpack_from("<H", obj, 16)[0]
    section_table_offset = COFF_HEADER_SIZE + optional_header_size
    string_table_offset = symbol_table_offset + symbol_count * 18
    string_table = obj[string_table_offset:] if string_table_offset < len(obj) else b""

    sections: list[Section] = []
    for index in range(section_count):
        offset = section_table_offset + index * SECTION_HEADER_SIZE
        if offset + SECTION_HEADER_SIZE > len(obj):
            raise ValueError("section table extends past end of go.o")

        raw_name = obj[offset : offset + 8]
        name = section_name(raw_name, string_table)
        raw_size = struct.unpack_from("<I", obj, offset + 16)[0]
        raw_offset = struct.unpack_from("<I", obj, offset + 20)[0]
        relocation_offset = struct.unpack_from("<I", obj, offset + 24)[0]
        relocation_count = struct.unpack_from("<H", obj, offset + 32)[0]
        sections.append(Section(name, raw_size, raw_offset, relocation_offset, relocation_count))

    return sections


def sort_pdata_section(obj: bytearray, section: Section) -> bool:
    if section.raw_size == 0 or section.raw_size % PDATA_ENTRY_SIZE != 0:
        return False

    raw_start = section.raw_offset
    raw_end = raw_start + section.raw_size
    if raw_end > len(obj):
        raise ValueError(f"{section.name} raw data extends past end of go.o")

    entries = [
        bytes(obj[offset : offset + PDATA_ENTRY_SIZE])
        for offset in range(raw_start, raw_end, PDATA_ENTRY_SIZE)
    ]
    keyed_entries = [
        (struct.unpack_from("<III", entry, 0), index, entry)
        for index, entry in enumerate(entries)
    ]
    sorted_entries = sorted(keyed_entries, key=lambda item: item[0])
    if [item[1] for item in sorted_entries] == list(range(len(entries))):
        return False

    for new_index, (_, _old_index, entry) in enumerate(sorted_entries):
        start = raw_start + new_index * PDATA_ENTRY_SIZE
        obj[start : start + PDATA_ENTRY_SIZE] = entry

    relocation_start = section.relocation_offset
    relocation_end = relocation_start + section.relocation_count * RELOCATION_SIZE
    if relocation_end > len(obj):
        raise ValueError(f"{section.name} relocations extend past end of go.o")

    relocations_by_entry: dict[int, list[bytes]] = {}
    trailing_relocations: list[bytes] = []
    for offset in range(relocation_start, relocation_end, RELOCATION_SIZE):
        relocation = bytes(obj[offset : offset + RELOCATION_SIZE])
        virtual_address = struct.unpack_from("<I", relocation, 0)[0]
        old_index = virtual_address // PDATA_ENTRY_SIZE
        if old_index < len(entries):
            relocations_by_entry.setdefault(old_index, []).append(relocation)
        else:
            trailing_relocations.append(relocation)

    sorted_relocations: list[bytes] = []
    for new_index, (_, old_index, _entry) in enumerate(sorted_entries):
        for relocation in relocations_by_entry.get(old_index, []):
            old_virtual_address = struct.unpack_from("<I", relocation, 0)[0]
            field_offset = old_virtual_address % PDATA_ENTRY_SIZE
            patched = bytearray(relocation)
            struct.pack_into("<I", patched, 0, new_index * PDATA_ENTRY_SIZE + field_offset)
            sorted_relocations.append(bytes(patched))

    sorted_relocations.extend(trailing_relocations)
    if len(sorted_relocations) != section.relocation_count:
        raise ValueError(f"{section.name} relocation regrouping lost records")

    for index, relocation in enumerate(sorted_relocations):
        start = relocation_start + index * RELOCATION_SIZE
        obj[start : start + RELOCATION_SIZE] = relocation

    return True


def patch_go_object(obj: bytes) -> tuple[bytes, int]:
    patched = bytearray(obj)
    changed = 0
    for section in parse_sections(obj):
        if section.name == ".pdata" or section.name.startswith(".pdata$"):
            if sort_pdata_section(patched, section):
                changed += 1
    return bytes(patched), changed


def patch_archive(path: Path) -> int:
    data = bytearray(path.read_bytes())
    members = list_members(data)
    go_members = [member for member in members if Path(member.name).name == "go.o"]
    if not go_members:
        raise ValueError("go.o was not found in archive")
    if len(go_members) > 1:
        raise ValueError("multiple go.o members found in archive")

    member = go_members[0]
    original = bytes(data[member.data_offset : member.data_offset + member.size])
    patched, changed_sections = patch_go_object(original)
    if len(patched) != len(original):
        raise AssertionError("patched go.o size changed")

    if changed_sections:
        data[member.data_offset : member.data_offset + member.size] = patched
        path.write_bytes(data)

    return changed_sections


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("archive", type=Path)
    args = parser.parse_args()

    try:
        changed_sections = patch_archive(args.archive)
    except Exception as exc:
        print(f"failed to patch {args.archive}: {exc}", file=sys.stderr)
        return 1

    if changed_sections:
        print(f"patched {args.archive}: sorted {changed_sections} .pdata section(s) in go.o")
    else:
        print(f"checked {args.archive}: go.o .pdata section(s) already sorted")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
