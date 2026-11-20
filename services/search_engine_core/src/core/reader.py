"""Чтение индекса формата v1: mmap-постинги, словарь и длины в памяти,
валидация metadata.json (docs/Tech/03-data.md, п. 1.1, 6.2).
"""
import json
import mmap
import os
from typing import Dict, List, Optional, Tuple

from indexer import index_format as fmt

SUPPORTED_TOKENIZER_VERSION = "1"

_INDEX_FILES = ("lexicon.bin", "postings.bin", "doc_lengths.bin", "metadata.json", "graph.db")


class IndexNotReadyError(Exception):
    """Индекс не опубликован (нет маркера metadata.json)."""


class UnsupportedIndexError(Exception):
    """Индекс повреждён или имеет неподдерживаемую версию формата/токенизатора."""


class IndexReader:
    def __init__(self, index_dir: str):
        self.index_dir = index_dir
        self._meta: Optional[dict] = None
        self._lexicon: Dict[str, fmt.LexiconEntry] = {}
        self._lengths: Dict[int, int] = {}
        self._postings_file = None
        self._mm: Optional[mmap.mmap] = None

    @property
    def ready(self) -> bool:
        return self._meta is not None

    def try_load(self) -> bool:
        """False — индекс ещё не опубликован; UnsupportedIndexError — брак/чужая версия."""
        if self.ready:
            return True
        meta_path = os.path.join(self.index_dir, "metadata.json")
        if not os.path.isfile(meta_path):
            return False
        with open(meta_path, encoding="utf-8") as f:
            meta = json.load(f)
        if meta.get("format_version") != fmt.FORMAT_VERSION:
            raise UnsupportedIndexError(
                f"format_version={meta.get('format_version')} не поддерживается (ожидается {fmt.FORMAT_VERSION})")
        if meta.get("tokenizer_version") != SUPPORTED_TOKENIZER_VERSION:
            raise UnsupportedIndexError(
                f"tokenizer_version={meta.get('tokenizer_version')!r} не поддерживается "
                f"(ожидается {SUPPORTED_TOKENIZER_VERSION!r})")
        try:
            lexicon = fmt.read_lexicon(os.path.join(self.index_dir, "lexicon.bin"))
            lengths = fmt.read_doc_lengths(os.path.join(self.index_dir, "doc_lengths.bin"))
            postings_file = open(os.path.join(self.index_dir, "postings.bin"), "rb")
            mm = mmap.mmap(postings_file.fileno(), 0, access=mmap.ACCESS_READ)
            magic, version = fmt.POSTINGS_HEADER.unpack_from(mm, 0)
            if magic != fmt.POSTINGS_MAGIC or version != fmt.FORMAT_VERSION:
                raise ValueError(f"postings.bin: неверная сигнатура {magic!r} или версия {version}")
        except (ValueError, OSError) as exc:
            raise UnsupportedIndexError(f"индекс повреждён: {exc}") from exc
        self._meta, self._lexicon, self._lengths = meta, lexicon, lengths
        self._postings_file, self._mm = postings_file, mm
        return True

    def close(self) -> None:
        if self._mm is not None:
            self._mm.close()
            self._mm = None
        if self._postings_file is not None:
            self._postings_file.close()
            self._postings_file = None
        self._meta = None

    # ------------------------------------------------------------- статистика
    @property
    def n(self) -> int:
        return self._meta["num_documents"]

    @property
    def avgdl(self) -> float:
        return self._meta["avg_doc_length"]

    @property
    def vocabulary_size(self) -> int:
        return self._meta["vocabulary_size"]

    @property
    def build_timestamp(self) -> int:
        return self._meta["build_timestamp"]

    @property
    def index_size_bytes(self) -> int:
        total = 0
        for name in _INDEX_FILES:
            path = os.path.join(self.index_dir, name)
            if os.path.isfile(path):
                total += os.path.getsize(path)
        return total

    @property
    def lengths(self) -> Dict[int, int]:
        return self._lengths

    # ---------------------------------------------------------------- доступ
    def entry(self, term: str) -> Optional[fmt.LexiconEntry]:
        return self._lexicon.get(term)

    def postings(self, entry: fmt.LexiconEntry) -> List[Tuple[int, int]]:
        blob = self._mm[entry.offset:entry.offset + entry.length]
        if len(blob) != entry.length:
            raise UnsupportedIndexError("postings.bin: список выходит за пределы файла")
        return fmt.decode_postings(blob, entry.df)

    def doc_length(self, doc_id: int) -> int:
        return self._lengths[doc_id]