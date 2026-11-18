"""Бинарный формат индекса, версия 1 (docs/Tech/03-data.md, п. 1.1).

Контракт между индексатором (запись) и поисковым ядром (чтение). Все целые —
little-endian; ``varint`` — LEB128 (7 бит данных на байт, младшие группы первыми,
старший бит = «есть следующий байт»). Только стандартная библиотека (ADR-008).
"""
import struct
from typing import Dict, List, NamedTuple, Sequence, Tuple

FORMAT_VERSION = 1

LEXICON_MAGIC = b"ELX1"
POSTINGS_MAGIC = b"EPS1"
DOC_LENGTHS_MAGIC = b"EDL1"

LEXICON_HEADER = struct.Struct("<4sII")   # magic, format_version, vocab_size
POSTINGS_HEADER = struct.Struct("<4sI")   # magic, format_version
DOC_LENGTHS_HEADER = struct.Struct("<4sII")  # magic, format_version, N

_TERM_LEN = struct.Struct("<H")
_TERM_TAIL = struct.Struct("<IQI")        # df, offset, length
_DOC_RECORD = struct.Struct("<II")        # doc_id, length

MAX_TERM_BYTES = 0xFFFF


class LexiconEntry(NamedTuple):
    df: int
    offset: int
    length: int


# ---------------------------------------------------------------------- varint

def encode_varint(value: int) -> bytes:
    if value < 0:
        raise ValueError("varint не кодирует отрицательные числа")
    out = bytearray()
    while value >= 0x80:
        out.append((value & 0x7F) | 0x80)
        value >>= 7
    out.append(value)
    return bytes(out)


def decode_varint(data: bytes, pos: int) -> Tuple[int, int]:
    """Возвращает ``(значение, позиция_после)``."""
    result = 0
    shift = 0
    n = len(data)
    while True:
        if pos >= n:
            raise ValueError("усечённый varint")
        byte = data[pos]
        pos += 1
        result |= (byte & 0x7F) << shift
        if not byte & 0x80:
            return result, pos
        shift += 7


# -------------------------------------------------------------------- постинги

def encode_postings(pairs: Sequence[Tuple[int, int]]) -> bytes:
    """``[(doc_id, tf), ...]`` (doc_id строго по возрастанию) -> ``varint(Δdoc_id) varint(tf)``."""
    out = bytearray()
    prev = 0
    for doc_id, tf in pairs:
        if doc_id <= prev:
            raise ValueError(f"doc_id должен строго возрастать и быть > 0: {doc_id} после {prev}")
        if tf < 1:
            raise ValueError(f"tf должен быть >= 1: {tf}")
        out += encode_varint(doc_id - prev)
        out += encode_varint(tf)
        prev = doc_id
    return bytes(out)


def decode_postings(data: bytes, df: int) -> List[Tuple[int, int]]:
    pairs: List[Tuple[int, int]] = []
    pos = 0
    doc_id = 0
    for _ in range(df):
        delta, pos = decode_varint(data, pos)
        tf, pos = decode_varint(data, pos)
        doc_id += delta
        pairs.append((doc_id, tf))
    if pos != len(data):
        raise ValueError("лишние байты после списка постингов")
    return pairs


# ----------------------------------------------------------------------- чтение

def _check_header(magic: bytes, version: int, expected_magic: bytes, what: str) -> None:
    if magic != expected_magic:
        raise ValueError(f"{what}: неверная сигнатура {magic!r}")
    if version != FORMAT_VERSION:
        raise ValueError(f"{what}: неподдерживаемая версия формата {version}")


def read_lexicon(path: str) -> Dict[str, LexiconEntry]:
    """Читает ``lexicon.bin`` целиком; порядок ключей = порядок в файле (по байтам UTF-8)."""
    with open(path, "rb") as f:
        data = f.read()
    magic, version, vocab = LEXICON_HEADER.unpack_from(data, 0)
    _check_header(magic, version, LEXICON_MAGIC, "lexicon.bin")
    pos = LEXICON_HEADER.size
    lexicon: Dict[str, LexiconEntry] = {}
    for _ in range(vocab):
        (tlen,) = _TERM_LEN.unpack_from(data, pos)
        pos += _TERM_LEN.size
        term = data[pos:pos + tlen].decode("utf-8")
        pos += tlen
        df, offset, length = _TERM_TAIL.unpack_from(data, pos)
        pos += _TERM_TAIL.size
        lexicon[term] = LexiconEntry(df, offset, length)
    if pos != len(data):
        raise ValueError("lexicon.bin: лишние байты в конце файла")
    return lexicon


def read_doc_lengths(path: str) -> Dict[int, int]:
    with open(path, "rb") as f:
        data = f.read()
    magic, version, n = DOC_LENGTHS_HEADER.unpack_from(data, 0)
    _check_header(magic, version, DOC_LENGTHS_MAGIC, "doc_lengths.bin")
    if len(data) != DOC_LENGTHS_HEADER.size + n * _DOC_RECORD.size:
        raise ValueError("doc_lengths.bin: размер файла не соответствует N")
    return dict(_DOC_RECORD.iter_unpack(data[DOC_LENGTHS_HEADER.size:]))


def read_postings(path: str, entry: LexiconEntry) -> List[Tuple[int, int]]:
    """Читает список постингов терма по записи словаря."""
    with open(path, "rb") as f:
        head = f.read(POSTINGS_HEADER.size)
        magic, version = POSTINGS_HEADER.unpack(head)
        _check_header(magic, version, POSTINGS_MAGIC, "postings.bin")
        f.seek(entry.offset)
        blob = f.read(entry.length)
    if len(blob) != entry.length:
        raise ValueError("postings.bin: список выходит за пределы файла")
    return decode_postings(blob, entry.df)
