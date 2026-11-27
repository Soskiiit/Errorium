"""Запись graph.db — граф связанных тредов и метаданные узлов (docs/Tech/03-data.md, раздел 2).

Файл создаётся целиком (без инкрементальных обновлений): сначала во временный файл рядом,
затем ``os.replace``. Ядро открывает его только на чтение (``mode=ro``).
Только стандартная библиотека (ADR-008).
"""
import html
import json
import os
import re
import sqlite3
from dataclasses import dataclass
from typing import Dict, Iterable, List, Set, Tuple

from .sampler import Thread
from .tokenizer import clean_html

SNIPPET_MAX_LEN = 300
_BATCH = 5000
_WHITESPACE = re.compile(r"\s+")
_PARAGRAPH_BREAK = re.compile(r"\n\s*\n")

# Формат текста документов корпуса benchmark: «Question: …\n\nAnswer: …»
QUESTION_PREFIX = "Question: "
ANSWER_SEPARATOR = "\n\nAnswer: "

SCHEMA = """
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
"""
INDEXES = """
CREATE INDEX idx_thread_links_source ON thread_links(source_doc_id);
CREATE INDEX idx_thread_links_target ON thread_links(target_doc_id);
"""


@dataclass(frozen=True)
class GraphNode:
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


def make_snippet(text: str, is_html: bool = True) -> str:
    """Очищенный сниппет: без тегов, пробелы схлопнуты, длина ≤ ``SNIPPET_MAX_LEN``."""
    plain = clean_html(text) if is_html else text
    plain = _WHITESPACE.sub(" ", plain).strip()
    if len(plain) <= SNIPPET_MAX_LEN:
        return plain
    return plain[:SNIPPET_MAX_LEN - 1].rstrip() + "…"


def node_from_thread(thread: Thread) -> GraphNode:
    return GraphNode(
        doc_id=thread.doc_id,
        title=thread.title,
        snippet=make_snippet(thread.question_body),
        has_accepted_answer=True,
        tags=list(thread.tags),
        question_score=thread.question_score,
        best_answer_score=thread.best_answer_score,
        best_answer_body=thread.best_answer_body,  # HTML как в дампе
        accepted_answer_id=thread.accepted_answer_id,
        created_at=thread.created_at,
    )


def _answer_to_html(answer: str) -> str:
    """Обычный текст ответа -> HTML: абзацы ``<p>…</p>``, спецсимволы экранированы."""
    paragraphs = [p.strip("\n") for p in _PARAGRAPH_BREAK.split(answer.strip())]
    return "\n".join(f"<p>{html.escape(p, quote=False)}</p>" for p in paragraphs if p.strip())


def node_from_corpus_doc(doc_id: int, title: str, text: str, tags: Iterable[str] = ()) -> GraphNode:
    """Узел для режима ``corpus``: обычный текст.

    Если в тексте есть разделитель ``\\n\\nAnswer: ``, часть до него — вопрос (сниппет), часть
    после — ответ (``best_answer_body`` в HTML), и ``has_accepted_answer = True``; иначе ответа нет.
    """
    question, separator, answer = text.partition(ANSWER_SEPARATOR)
    has_answer = bool(separator) and bool(answer.strip())
    if not has_answer:
        question, answer = text, ""
    elif question.startswith(QUESTION_PREFIX):
        question = question[len(QUESTION_PREFIX):]
    return GraphNode(doc_id=doc_id, title=title, snippet=make_snippet(question, is_html=False),
                     has_accepted_answer=has_answer, tags=list(tags), question_score=0,
                     best_answer_score=0, best_answer_body=_answer_to_html(answer) if has_answer else "",
                     accepted_answer_id=0, created_at=0)


def _node_row(n: GraphNode) -> tuple:
    return (n.doc_id, n.title, n.snippet, 1 if n.has_accepted_answer else 0,
            json.dumps(list(n.tags), ensure_ascii=False), n.question_score, n.best_answer_score,
            n.best_answer_body, n.accepted_answer_id, n.created_at)


def write_graph_db(path: str, nodes: Iterable[GraphNode],
                   edges: Iterable[Tuple[int, int, str, float]]) -> Dict[str, int]:
    """Записывает ``graph.db`` атомарно; возвращает ``{"nodes": n, "edges": m}``.

    ``ValueError`` — повтор ``doc_id`` узла или ребро с неизвестным концом; при ошибке
    прежний файл (если был) остаётся нетронутым, временных файлов не остаётся.
    """
    tmp = path + ".tmp"
    if os.path.exists(tmp):
        os.remove(tmp)
    conn = sqlite3.connect(tmp)
    try:
        conn.execute("PRAGMA journal_mode = OFF")
        conn.execute("PRAGMA synchronous = OFF")
        conn.executescript(SCHEMA)
        seen: Set[int] = set()
        batch: List[tuple] = []
        for n in nodes:
            if n.doc_id in seen:
                raise ValueError(f"повторяющийся doc_id узла: {n.doc_id}")
            seen.add(n.doc_id)
            batch.append(_node_row(n))
            if len(batch) >= _BATCH:
                conn.executemany("INSERT INTO thread_nodes VALUES (?,?,?,?,?,?,?,?,?,?)", batch)
                batch = []
        if batch:
            conn.executemany("INSERT INTO thread_nodes VALUES (?,?,?,?,?,?,?,?,?,?)", batch)
        edge_count = 0
        batch = []
        for src, dst, kind, weight in edges:
            if src not in seen or dst not in seen:
                raise ValueError(f"ребро {src}->{dst} ссылается на неизвестный узел")
            batch.append((src, dst, kind, weight))
            edge_count += 1
            if len(batch) >= _BATCH:
                conn.executemany("INSERT INTO thread_links VALUES (?,?,?,?)", batch)
                batch = []
        if batch:
            conn.executemany("INSERT INTO thread_links VALUES (?,?,?,?)", batch)
        conn.executescript(INDEXES)
        conn.commit()
        conn.close()
        os.replace(tmp, path)
    except BaseException:
        conn.close()
        if os.path.exists(tmp):
            os.remove(tmp)
        raise
    return {"nodes": len(seen), "edges": edge_count}
