"""Search Engine Core: Index loading, BM25 ranking, and Graph Store."""
from dataclasses import dataclass
import json
import math
import os
import sqlite3
import time
from typing import Dict, Iterable, List, Optional, Sequence, Tuple

from indexer.index_format import (
    FORMAT_VERSION,
    LexiconEntry,
    read_doc_lengths,
    read_lexicon,
    read_postings,
)
from indexer.tokenizer import TOKENIZER_VERSION, tokenize

from .config import CoreConfig


@dataclass
class SearchHit:
    doc_id: int
    score: float
    title: str
    snippet: str
    tags: List[str]
    question_score: int
    accepted_answer_id: int
    best_answer_body: str
    best_answer_score: int
    created_at: int

    def __getitem__(self, item: str):
        return getattr(self, item)

    def get(self, item: str, default=None):
        return getattr(self, item, default)


@dataclass
class SearchResult:
    hits: List[SearchHit]
    total_hits: int
    latency_ms: float
    tokenized_query: List[str]

    def __getitem__(self, item: str):
        return getattr(self, item)

    def get(self, item: str, default=None):
        return getattr(self, item, default)


@dataclass(frozen=True)
class EngineStats:
    total_documents: int
    avg_doc_length: float
    vocabulary_size: int
    index_size_bytes: int
    is_ready: bool
    build_timestamp: int

    @property
    def N(self) -> int:
        return self.total_documents

    @property
    def avgdl(self) -> float:
        return self.avg_doc_length

    def __getitem__(self, item: str):
        return getattr(self, item)

    def get(self, item: str, default=None):
        return getattr(self, item, default)


@dataclass
class ThreadNode:
    doc_id: int
    title: str
    snippet: str
    has_accepted_answer: bool
    tags: List[str]
    question_score: int
    best_answer_score: int
    best_answer_body: str
    accepted_answer_id: int
    created_at: int

    def __getitem__(self, item: str):
        return getattr(self, item)

    def get(self, item: str, default=None):
        return getattr(self, item, default)


@dataclass
class ThreadEdge:
    source_doc_id: int
    target_doc_id: int
    link_type: str
    weight: float = 1.0

    def __getitem__(self, item: str):
        return getattr(self, item)

    def get(self, item: str, default=None):
        return getattr(self, item, default)


class Index:
    """In-memory representation of inverted index metadata, lexicon and document lengths."""

    def __init__(
        self,
        index_dir: str,
        lexicon: Dict[str, LexiconEntry],
        doc_lengths: Dict[int, int],
        num_documents: int,
        avg_doc_length: float,
        build_timestamp: int,
    ) -> None:
        self.index_dir = index_dir
        self.lexicon = lexicon
        self.doc_lengths = doc_lengths
        self.num_documents = num_documents
        self.avg_doc_length = avg_doc_length
        self.build_timestamp = build_timestamp
        self.postings_path = os.path.join(index_dir, "postings.bin")

    @property
    def N(self) -> int:
        return self.num_documents

    @property
    def avgdl(self) -> float:
        return self.avg_doc_length

    @property
    def vocabulary_size(self) -> int:
        return len(self.lexicon)

    def get_postings(self, entry: LexiconEntry) -> List[Tuple[int, int]]:
        return read_postings(self.postings_path, entry)

    @classmethod
    def load(cls, index_dir: str) -> "Index":
        metadata_path = os.path.join(index_dir, "metadata.json")
        if not os.path.isfile(metadata_path):
            raise FileNotFoundError(f"metadata.json not found in index directory: {index_dir}")

        with open(metadata_path, "r", encoding="utf-8") as f:
            metadata = json.load(f)

        fmt_ver = metadata.get("format_version")
        if fmt_ver != FORMAT_VERSION:
            raise ValueError(
                f"Unsupported index format_version: {fmt_ver!r} (expected {FORMAT_VERSION})"
            )

        tok_ver = str(metadata.get("tokenizer_version"))
        if tok_ver != TOKENIZER_VERSION:
            raise ValueError(
                f"Unsupported tokenizer_version: {tok_ver!r} (expected {TOKENIZER_VERSION})"
            )

        lexicon_path = os.path.join(index_dir, "lexicon.bin")
        doc_lengths_path = os.path.join(index_dir, "doc_lengths.bin")

        if not os.path.isfile(lexicon_path):
            raise FileNotFoundError(f"lexicon.bin not found in index directory: {index_dir}")
        if not os.path.isfile(doc_lengths_path):
            raise FileNotFoundError(f"doc_lengths.bin not found in index directory: {index_dir}")

        lexicon = read_lexicon(lexicon_path)
        doc_lengths = read_doc_lengths(doc_lengths_path)

        num_documents = int(metadata["num_documents"])
        avg_doc_length = float(metadata["avg_doc_length"])
        build_timestamp = int(metadata.get("build_timestamp", 0))

        return cls(
            index_dir=index_dir,
            lexicon=lexicon,
            doc_lengths=doc_lengths,
            num_documents=num_documents,
            avg_doc_length=avg_doc_length,
            build_timestamp=build_timestamp,
        )


class ThreadStore:
    """Read-only access to graph.db storing thread nodes and relations."""

    def __init__(self, db_path: str) -> None:
        self.db_path = db_path
        if not os.path.isfile(db_path):
            raise FileNotFoundError(f"graph.db not found at {db_path}")

    def _get_connection(self) -> sqlite3.Connection:
        abs_path = os.path.abspath(self.db_path)
        uri = f"file:{abs_path}?mode=ro"
        conn = sqlite3.connect(uri, uri=True)
        conn.row_factory = sqlite3.Row
        return conn

    def nodes(self, doc_ids: Iterable[int]) -> Dict[int, ThreadNode]:
        ids = list(doc_ids)
        if not ids:
            return {}

        result: Dict[int, ThreadNode] = {}
        batch_size = 500

        conn = self._get_connection()
        try:
            for i in range(0, len(ids), batch_size):
                chunk = ids[i : i + batch_size]
                placeholders = ",".join("?" for _ in chunk)
                query = f"""
                    SELECT doc_id, title, snippet, has_accepted_answer, tags,
                           question_score, best_answer_score, best_answer_body,
                           accepted_answer_id, created_at
                    FROM thread_nodes
                    WHERE doc_id IN ({placeholders})
                """
                cursor = conn.execute(query, chunk)
                for row in cursor:
                    tags_raw = row["tags"]
                    tags = json.loads(tags_raw) if tags_raw else []
                    doc_id = row["doc_id"]
                    result[doc_id] = ThreadNode(
                        doc_id=doc_id,
                        title=row["title"],
                        snippet=row["snippet"],
                        has_accepted_answer=bool(row["has_accepted_answer"]),
                        tags=tags,
                        question_score=row["question_score"],
                        best_answer_score=row["best_answer_score"],
                        best_answer_body=row["best_answer_body"],
                        accepted_answer_id=row["accepted_answer_id"],
                        created_at=row["created_at"],
                    )
        finally:
            conn.close()
        return result

    def details(self, doc_id: int) -> Optional[ThreadNode]:
        nodes_map = self.nodes([doc_id])
        return nodes_map.get(doc_id)

    def neighbors(self, doc_ids: Iterable[int]) -> List[ThreadEdge]:
        ids = list(doc_ids)
        if not ids:
            return []

        edges: List[ThreadEdge] = []
        batch_size = 500

        conn = self._get_connection()
        try:
            for i in range(0, len(ids), batch_size):
                chunk = ids[i : i + batch_size]
                placeholders = ",".join("?" for _ in chunk)
                # Outgoing edges from doc_ids UNION incoming DUPLICATE_OF edges to doc_ids
                query = f"""
                    SELECT source_doc_id, target_doc_id, link_type, weight
                    FROM thread_links
                    WHERE source_doc_id IN ({placeholders})
                    UNION
                    SELECT source_doc_id, target_doc_id, link_type, weight
                    FROM thread_links
                    WHERE target_doc_id IN ({placeholders}) AND link_type = 'DUPLICATE_OF'
                """
                params = chunk + chunk
                cursor = conn.execute(query, params)
                for row in cursor:
                    edges.append(
                        ThreadEdge(
                            source_doc_id=row["source_doc_id"],
                            target_doc_id=row["target_doc_id"],
                            link_type=row["link_type"],
                            weight=float(row["weight"]),
                        )
                    )
        finally:
            conn.close()
        return edges


class Engine:
    """Search Engine Core coordinating index search and graph metadata enrichment."""

    def __init__(
        self,
        index: Index,
        thread_store: ThreadStore,
        config: Optional[CoreConfig] = None,
    ) -> None:
        self.index = index
        self.thread_store = thread_store
        self.config = config or CoreConfig()

    @classmethod
    def load(
        cls,
        index_dir: Optional[str] = None,
        config: Optional[CoreConfig] = None,
    ) -> "Engine":
        cfg = config or CoreConfig.from_env()
        target_dir = index_dir or cfg.index_dir

        index = Index.load(target_dir)
        graph_db_path = os.path.join(target_dir, "graph.db")
        thread_store = ThreadStore(graph_db_path)
        return cls(index, thread_store, cfg)

    def stats(self) -> EngineStats:
        index_dir = self.index.index_dir
        total_size = 0
        if os.path.isdir(index_dir):
            for entry in os.scandir(index_dir):
                if entry.is_file():
                    total_size += entry.stat().st_size

        return EngineStats(
            total_documents=self.index.num_documents,
            avg_doc_length=self.index.avg_doc_length,
            vocabulary_size=self.index.vocabulary_size,
            index_size_bytes=total_size,
            is_ready=True,
            build_timestamp=self.index.build_timestamp,
        )

    def search(
        self,
        query: str,
        top_k: int = 10,
        tags: Optional[Sequence[str]] = None,
        language: Optional[str] = None,
    ) -> SearchResult:
        t0 = time.perf_counter()

        # Tokenize query without clean_html
        tokenized_query = tokenize(query)
        if not tokenized_query:
            latency_ms = (time.perf_counter() - t0) * 1000.0
            return SearchResult(
                hits=[],
                total_hits=0,
                latency_ms=latency_ms,
                tokenized_query=[],
            )

        k1 = self.config.bm25_k1
        b = self.config.bm25_b
        N = self.index.N
        avgdl = self.index.avgdl

        # Score documents across unique query terms
        unique_terms = set(tokenized_query)
        doc_scores: Dict[int, float] = {}

        for term in unique_terms:
            entry = self.index.lexicon.get(term)
            if entry is None:
                # Term outside lexicon has contribution 0
                continue

            df = entry.df
            # BM25 IDF: ln(1 + (N - DF + 0.5) / (DF + 0.5))
            idf = math.log(1.0 + (N - df + 0.5) / (df + 0.5))

            postings = self.index.get_postings(entry)
            for doc_id, tf in postings:
                doc_len = self.index.doc_lengths.get(doc_id, 0)
                len_norm = (doc_len / avgdl) if avgdl > 0 else 0.0
                denominator = tf + k1 * (1.0 - b + b * len_norm)
                tf_part = (tf * (k1 + 1.0)) / denominator
                doc_scores[doc_id] = doc_scores.get(doc_id, 0.0) + idf * tf_part

        if not doc_scores:
            latency_ms = (time.perf_counter() - t0) * 1000.0
            return SearchResult(
                hits=[],
                total_hits=0,
                latency_ms=latency_ms,
                tokenized_query=tokenized_query,
            )

        # Candidate documents with score > 0
        candidate_ids = [d for d, s in doc_scores.items() if s > 0]
        if not candidate_ids:
            latency_ms = (time.perf_counter() - t0) * 1000.0
            return SearchResult(
                hits=[],
                total_hits=0,
                latency_ms=latency_ms,
                tokenized_query=tokenized_query,
            )

        # Enrich and filter using thread_nodes from graph.db
        nodes_map = self.thread_store.nodes(candidate_ids)

        # Prepare normalized filter criteria
        filter_tags = {t.lower() for t in tags} if tags else None
        filter_lang = language.lower() if language else None

        valid_candidates: List[Tuple[float, int, ThreadNode]] = []

        for doc_id in candidate_ids:
            node = nodes_map.get(doc_id)
            if node is None:
                # Documents missing in graph.db are skipped
                continue

            doc_tags = [t.lower() for t in node.tags]

            # Filter tags: doc must have at least one common tag
            if filter_tags is not None:
                if not any(t in filter_tags for t in doc_tags):
                    continue

            # Filter language: tag language.lower() must be present
            if filter_lang is not None:
                if filter_lang not in doc_tags:
                    continue

            score = doc_scores[doc_id]
            valid_candidates.append((score, doc_id, node))

        total_hits = len(valid_candidates)

        # Sort: score DESC, ties broken by doc_id ASC
        valid_candidates.sort(key=lambda item: (-item[0], item[1]))

        # Clamp top_k in [1, 50]
        clamped_top_k = max(1, min(50, top_k))
        top_candidates = valid_candidates[:clamped_top_k]

        hits: List[SearchHit] = []
        for score, doc_id, node in top_candidates:
            hits.append(
                SearchHit(
                    doc_id=doc_id,
                    score=score,
                    title=node.title,
                    snippet=node.snippet,
                    tags=node.tags,
                    question_score=node.question_score,
                    accepted_answer_id=node.accepted_answer_id,
                    best_answer_body=node.best_answer_body,
                    best_answer_score=node.best_answer_score,
                    created_at=node.created_at,
                )
            )

        latency_ms = (time.perf_counter() - t0) * 1000.0
        return SearchResult(
            hits=hits,
            total_hits=total_hits,
            latency_ms=latency_ms,
            tokenized_query=tokenized_query,
        )
