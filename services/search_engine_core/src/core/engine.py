"""Оркестрация поиска: токенизация запроса → BM25 → фильтры → Top-K
(docs/Tech/03-data.md, п. 6.3–6.4).
"""
import os
import time
from dataclasses import dataclass
from typing import List, Optional, Sequence, Tuple

from indexer.tokenizer import tokenize

from . import bm25
from .graph import GraphStore
from .reader import IndexNotReadyError, IndexReader

DEFAULT_TOP_K = 10


@dataclass(frozen=True)
class SearchResult:
    hits: List[Tuple[int, float]]    # (doc_id, score), убывание балла, tie-break по doc_id
    total_hits: int                  # документы с >= 1 термом запроса, до фильтров
    tokens: List[str]                # токенизированный запрос в порядке эмиссии
    latency_ms: float


class SearchEngine:
    def __init__(self, reader: IndexReader, graph: Optional[GraphStore] = None,
                 k1: float = 1.5, b: float = 0.75, max_top_k: int = 100):
        self.reader = reader
        self.graph = graph if graph is not None else GraphStore(
            os.path.join(reader.index_dir, "graph.db"))
        self.k1 = k1
        self.b = b
        self.max_top_k = max_top_k

    def ensure_ready(self) -> None:
        if not self.reader.ready:
            self.reader.try_load()
        if not self.reader.ready:
            raise IndexNotReadyError(
                f"индекс не опубликован в {self.reader.index_dir} (нет metadata.json); "
                "постройте индекс: make index-sample / make index-corpus")

    def search(self, query: str, top_k: int = DEFAULT_TOP_K,
               tags: Sequence[str] = (), language: str = "") -> SearchResult:
        start = time.perf_counter()
        self.ensure_ready()
        tokens = tokenize(query)
        k = min(top_k if top_k > 0 else DEFAULT_TOP_K, self.max_top_k)

        term_postings = []
        for term in dict.fromkeys(tokens):
            entry = self.reader.entry(term)
            if entry is not None:
                term_postings.append((entry.df, self.reader.postings(entry)))
        scores = bm25.score_documents(term_postings, self.reader.lengths,
                                      self.reader.n, self.reader.avgdl, self.k1, self.b)
        total_hits = len(scores)

        wanted = {t.casefold() for t in tags}
        if language:
            wanted.add(language.casefold())
        if wanted and scores:
            meta = self.graph.nodes(list(scores))
            scores = {d: s for d, s in scores.items()
                      if d in meta and wanted <= {t.casefold() for t in meta[d].tags}}

        hits = bm25.top_k(scores, k)
        return SearchResult(hits=hits, total_hits=total_hits, tokens=tokens,
                            latency_ms=(time.perf_counter() - start) * 1000.0)