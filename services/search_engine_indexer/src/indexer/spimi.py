"""SPIMI-построитель инвертированного индекса (docs/Tech/03-data.md, п. 1.2).

Single-Pass In-Memory Indexing: токены накапливаются в блоке в памяти, при превышении
порога блок сбрасывается на диск отсортированным по терму, затем блоки сливаются
k-way слиянием. Постинги одного терма из разных блоков склеиваются по возрастанию
``doc_id`` — документы могут поступать в произвольном порядке.

Только стандартная библиотека (ADR-008); готовые поисковые библиотеки запрещены (ADR-001).
"""
import heapq
import json
import os
import shutil
import struct
import time
from collections import Counter
from typing import Dict, Iterable, Iterator, List, Optional, Sequence, Tuple

from .index_format import (
    DOC_LENGTHS_HEADER,
    DOC_LENGTHS_MAGIC,
    FORMAT_VERSION,
    LEXICON_HEADER,
    LEXICON_MAGIC,
    MAX_TERM_BYTES,
    POSTINGS_HEADER,
    POSTINGS_MAGIC,
    decode_postings,
    encode_postings,
)
from .tokenizer import TOKENIZER_VERSION

DEFAULT_BLOCK_POSTINGS = 2_000_000
MAX_DOC_ID = 0xFFFFFFFF
BUILDING_DIR = ".building"
INDEX_FILES = ("lexicon.bin", "postings.bin", "doc_lengths.bin")
METADATA_FILE = "metadata.json"

_RUN_HEAD = struct.Struct("<H")
_RUN_TAIL = struct.Struct("<II")  # df, длина закодированного списка в байтах
_LEXICON_TAIL = struct.Struct("<IQI")  # df, offset, length


def _write_run(path: str, block: Dict[bytes, List[Tuple[int, int]]]) -> None:
    """Сбрасывает блок на диск (внутренний формат индексатора, не контракт)."""
    with open(path, "wb", buffering=1 << 20) as f:
        for term in sorted(block):
            pairs = sorted(block[term])
            data = encode_postings(pairs)
            f.write(_RUN_HEAD.pack(len(term)))
            f.write(term)
            f.write(_RUN_TAIL.pack(len(pairs), len(data)))
            f.write(data)


def _read_run(path: str) -> Iterator[Tuple[bytes, List[Tuple[int, int]]]]:
    with open(path, "rb", buffering=1 << 20) as f:
        while True:
            head = f.read(_RUN_HEAD.size)
            if not head:
                return
            (term_len,) = _RUN_HEAD.unpack(head)
            term = f.read(term_len)
            df, data_len = _RUN_TAIL.unpack(f.read(_RUN_TAIL.size))
            yield term, decode_postings(f.read(data_len), df)


def _merged_terms(runs: Sequence[str]) -> Iterator[Tuple[bytes, List[Tuple[int, int]]]]:
    """k-way слияние блоков: термы по возрастанию байтов, постинги по возрастанию doc_id."""
    streams = [_read_run(p) for p in runs]
    current: Optional[bytes] = None
    parts: List[List[Tuple[int, int]]] = []
    for term, pairs in heapq.merge(*streams, key=lambda item: item[0]):
        if term != current:
            if current is not None:
                yield current, list(heapq.merge(*parts))
            current, parts = term, []
        parts.append(pairs)
    if current is not None:
        yield current, list(heapq.merge(*parts))


def build_index(docs: Iterable[Tuple[int, Sequence[str]]], out_dir: str, *,
                index_mode: str = "sample",
                block_postings: int = DEFAULT_BLOCK_POSTINGS,
                timestamp: Optional[int] = None,
                source: Optional[dict] = None,
                stats: Optional[dict] = None,
                extra_files: Optional[Dict[str, str]] = None) -> dict:
    """Строит индекс из потока ``(doc_id, токены)`` и записывает его в ``out_dir``.

    Документы могут идти в любом порядке. Каталог ``out_dir`` может быть точкой
    монтирования, поэтому сборка идёт в ``out_dir/.building``, файлы переносятся через
    ``os.replace``, а ``metadata.json`` записывается последним — он служит маркером
    готовности индекса. При ошибке прежний индекс (если был) остаётся нетронутым.

    ``extra_files`` — ``{имя: путь}`` дополнительных артефактов (``graph.db``,
    ``sample_manifest.json``): они переносятся в ``out_dir`` тем же шагом, до маркера, и
    не затрагиваются, если сборка упала. Возвращает содержимое ``metadata.json``;
    ``stats`` (если передан) получает ``{"blocks": число сброшенных блоков}``.
    """
    extra_files = dict(extra_files or {})
    for name in extra_files:
        if (os.path.basename(name) != name or name in INDEX_FILES or name == METADATA_FILE
                or name.startswith(".")):
            raise ValueError(f"недопустимое имя дополнительного файла: {name!r}")
    building = os.path.join(out_dir, BUILDING_DIR)
    if os.path.isdir(building):
        shutil.rmtree(building)
    os.makedirs(building)
    try:
        meta = _build(docs, building, index_mode, block_postings, timestamp, source, stats)
        _publish(building, out_dir, meta, extra_files)
        return meta
    finally:
        shutil.rmtree(building, ignore_errors=True)


def _build(docs, building, index_mode, block_postings, timestamp, source, stats) -> dict:
    lengths: Dict[int, int] = {}
    block: Dict[bytes, List[Tuple[int, int]]] = {}
    block_size = 0
    runs: List[str] = []

    def flush() -> None:
        nonlocal block, block_size
        if not block:
            return
        path = os.path.join(building, f"block_{len(runs):06d}.run")
        _write_run(path, block)
        runs.append(path)
        block, block_size = {}, 0

    for doc_id, tokens in docs:
        if not 0 < doc_id <= MAX_DOC_ID:
            raise ValueError(f"doc_id вне диапазона uint32: {doc_id}")
        if doc_id in lengths:
            raise ValueError(f"повторяющийся doc_id: {doc_id}")
        lengths[doc_id] = len(tokens)
        for term, tf in Counter(tokens).items():
            raw = term.encode("utf-8")
            if len(raw) > MAX_TERM_BYTES:
                raise ValueError("терм длиннее 65535 байт")
            block.setdefault(raw, []).append((doc_id, tf))
            block_size += 1
        if block_size >= block_postings:
            flush()
    flush()
    if stats is not None:
        stats["blocks"] = len(runs)

    vocabulary = _write_postings_and_lexicon(runs, building)
    _write_doc_lengths(lengths, building)

    total_tokens = sum(lengths.values())
    n = len(lengths)
    return {
        "format_version": FORMAT_VERSION,
        "tokenizer_version": TOKENIZER_VERSION,
        "index_mode": index_mode,
        "num_documents": n,
        "avg_doc_length": (total_tokens / n) if n else 0.0,
        "total_tokens": total_tokens,
        "vocabulary_size": vocabulary,
        "build_timestamp": int(time.time()) if timestamp is None else int(timestamp),
        "source": dict(source or {}),
    }


def _write_postings_and_lexicon(runs: Sequence[str], building: str) -> int:
    vocab = 0
    with open(os.path.join(building, "postings.bin"), "wb", buffering=1 << 20) as pf, \
            open(os.path.join(building, "lexicon.bin"), "wb", buffering=1 << 20) as lf:
        pf.write(POSTINGS_HEADER.pack(POSTINGS_MAGIC, FORMAT_VERSION))
        lf.write(LEXICON_HEADER.pack(LEXICON_MAGIC, FORMAT_VERSION, 0))  # vocab_size допишем
        for term, pairs in _merged_terms(runs):
            data = encode_postings(pairs)
            offset = pf.tell()
            pf.write(data)
            lf.write(_RUN_HEAD.pack(len(term)))
            lf.write(term)
            lf.write(_LEXICON_TAIL.pack(len(pairs), offset, len(data)))
            vocab += 1
        lf.seek(0)
        lf.write(LEXICON_HEADER.pack(LEXICON_MAGIC, FORMAT_VERSION, vocab))
    return vocab


def _write_doc_lengths(lengths: Dict[int, int], building: str) -> None:
    with open(os.path.join(building, "doc_lengths.bin"), "wb", buffering=1 << 20) as f:
        f.write(DOC_LENGTHS_HEADER.pack(DOC_LENGTHS_MAGIC, FORMAT_VERSION, len(lengths)))
        for doc_id in sorted(lengths):
            f.write(struct.pack("<II", doc_id, lengths[doc_id]))


def _publish(building: str, out_dir: str, meta: dict, extra_files: Dict[str, str]) -> None:
    """Переносит файлы на место; ``metadata.json`` — последним (маркер готовности)."""
    meta_tmp = os.path.join(building, METADATA_FILE)
    with open(meta_tmp, "w", encoding="utf-8") as f:
        f.write(json.dumps(meta, ensure_ascii=False, sort_keys=True, indent=2) + "\n")
    old_marker = os.path.join(out_dir, METADATA_FILE)
    if os.path.exists(old_marker):
        os.remove(old_marker)  # пока идёт замена файлов, индекс считается неполным
    for name in INDEX_FILES:
        os.replace(os.path.join(building, name), os.path.join(out_dir, name))
    for name, src in extra_files.items():
        shutil.move(src, os.path.join(out_dir, name))
    os.replace(meta_tmp, old_marker)
