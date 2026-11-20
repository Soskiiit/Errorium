"""Синтетический индекс формата v1 для unit-тестов ядра (без запуска индексатора).

Пишет те же артефакты, что и индексатор (docs/Tech/03-data.md, п. 1.1, раздел 2),
кодируя их общим модулем index_format; graph.db создаётся напрямую sqlite3.
Работает и на хосте (полный пакет indexer), и в контейнере ядра (вендоренные
tokenizer + index_format).
"""
import json
import os
import sqlite3
import struct
import sys
import time
from collections import Counter, defaultdict

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
CORE_SRC = os.path.join(REPO_ROOT, "services", "search_engine_core", "src")
INDEXER_SRC = os.path.join(REPO_ROOT, "services", "search_engine_indexer", "src")
for _p in (CORE_SRC, INDEXER_SRC):
    if os.path.isdir(_p) and _p not in sys.path:
        sys.path.insert(0, _p)

from indexer import index_format as fmt  # noqa: E402
from indexer.tokenizer import tokenize  # noqa: E402

GRAPH_SCHEMA = """
CREATE TABLE thread_nodes (
    doc_id              INTEGER PRIMARY KEY,
    title               TEXT NOT NULL,
    snippet             TEXT NOT NULL,
    has_accepted_answer BOOLEAN DEFAULT FALSE,
    tags                TEXT,
    question_score      INTEGER DEFAULT 0,
    best_answer_score   INTEGER DEFAULT 0,
    best_answer_body    TEXT,
    accepted_answer_id  INTEGER DEFAULT 0,
    created_at          INTEGER DEFAULT 0
);
CREATE TABLE thread_links (
    source_doc_id INTEGER NOT NULL,
    target_doc_id INTEGER NOT NULL,
    link_type     TEXT NOT NULL,
    weight        REAL DEFAULT 1.0,
    PRIMARY KEY (source_doc_id, target_doc_id, link_type)
);
CREATE INDEX idx_thread_links_source ON thread_links(source_doc_id);
CREATE INDEX idx_thread_links_target ON thread_links(target_doc_id);
"""


def build_index(root, docs, edges=(), metadata=None, write_metadata=True):
    """docs: [{doc_id, text, title?, tags?, question_score?, best_answer_body?, ...}].

    edges: [(source, target, link_type)]. metadata — переопределения полей
    metadata.json (для негативных тестов). Возвращает каталог индекса.
    """
    index_dir = os.path.join(root, "index")
    os.makedirs(index_dir, exist_ok=True)

    lengths = {}
    postings = defaultdict(list)
    total_tokens = 0
    for doc in sorted(docs, key=lambda d: d["doc_id"]):
        doc_id = doc["doc_id"]
        tokens = tokenize((doc.get("title", "") + " " + doc["text"]).strip())
        lengths[doc_id] = len(tokens)
        total_tokens += len(tokens)
        for term, tf in Counter(tokens).items():
            postings[term].append((doc_id, tf))

    postings_blob = bytearray(fmt.POSTINGS_HEADER.pack(fmt.POSTINGS_MAGIC, fmt.FORMAT_VERSION))
    lexicon = {}
    for term in sorted(postings, key=lambda t: t.encode("utf-8")):
        pairs = postings[term]
        blob = fmt.encode_postings(pairs)
        lexicon[term] = fmt.LexiconEntry(len(pairs), len(postings_blob), len(blob))
        postings_blob += blob
    with open(os.path.join(index_dir, "postings.bin"), "wb") as f:
        f.write(postings_blob)

    lexicon_blob = bytearray(fmt.LEXICON_HEADER.pack(fmt.LEXICON_MAGIC, fmt.FORMAT_VERSION, len(lexicon)))
    for term, entry in lexicon.items():
        raw = term.encode("utf-8")
        lexicon_blob += struct_pack_term(raw, entry)
    with open(os.path.join(index_dir, "lexicon.bin"), "wb") as f:
        f.write(lexicon_blob)

    n = len(lengths)
    dl_blob = bytearray(fmt.DOC_LENGTHS_HEADER.pack(fmt.DOC_LENGTHS_MAGIC, fmt.FORMAT_VERSION, n))
    for doc_id in sorted(lengths):
        dl_blob += fmt._DOC_RECORD.pack(doc_id, lengths[doc_id])
    with open(os.path.join(index_dir, "doc_lengths.bin"), "wb") as f:
        f.write(dl_blob)

    conn = sqlite3.connect(os.path.join(index_dir, "graph.db"))
    conn.executescript(GRAPH_SCHEMA)
    for doc in docs:
        conn.execute(
            "INSERT INTO thread_nodes VALUES (?,?,?,?,?,?,?,?,?,?)",
            (doc["doc_id"], doc.get("title", ""), doc.get("snippet", doc["text"][:300]),
             1 if doc.get("best_answer_body") else 0,
             json.dumps(doc.get("tags", []), ensure_ascii=False),
             doc.get("question_score", 0), doc.get("best_answer_score", 0),
             doc.get("best_answer_body", ""), doc.get("accepted_answer_id", 0),
             doc.get("created_at", 0)))
    conn.executemany("INSERT INTO thread_links VALUES (?,?,?,1.0)", edges)
    conn.commit()
    conn.close()

    if write_metadata:
        meta = {
            "format_version": fmt.FORMAT_VERSION,
            "tokenizer_version": "1",
            "index_mode": "corpus",
            "num_documents": n,
            "avg_doc_length": (total_tokens / n) if n else 0.0,
            "total_tokens": total_tokens,
            "vocabulary_size": len(lexicon),
            "build_timestamp": int(time.time()),
            "source": {},
        }
        meta.update(metadata or {})
        with open(os.path.join(index_dir, "metadata.json"), "w", encoding="utf-8") as f:
            json.dump(meta, f, ensure_ascii=False)
    return index_dir


def struct_pack_term(raw, entry):
    return struct.pack("<H", len(raw)) + raw + struct.pack("<IQI", entry.df, entry.offset, entry.length)