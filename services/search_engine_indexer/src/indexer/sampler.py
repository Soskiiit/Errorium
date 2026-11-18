"""Потоковый сэмплер корпуса из дампа Stack Overflow (docs/Tech/03-data.md, раздел 4).

Файлы дампа читаются построчно (одна запись ``<row .../>`` на строку), без загрузки
в память и без распаковки на диск. Выборка тредов детерминирована: решение
принимается по хешу ``Id`` вопроса (для ответа — по хешу ``ParentId``), поэтому вопрос
и его ответы попадают в корпус вместе при любом порядке строк.

Только стандартная библиотека (ADR-008).
"""
import html
import re
from collections import defaultdict
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Callable, Dict, Iterable, Iterator, List, Optional, Set, Tuple

KNUTH_MULTIPLIER = 2654435761
_MASK32 = 0xFFFFFFFF

POST_QUESTION = 1
POST_ANSWER = 2

LINK_LINKED = 1
LINK_DUPLICATE = 3
_LINK_TYPES = {LINK_LINKED: "LINKED_IN_ANSWER", LINK_DUPLICATE: "DUPLICATE_OF"}

_ATTR = re.compile(r'(\w+)="([^"]*)"')
_HEAD = re.compile(r'\s*<row Id="(\d+)" PostTypeId="(\d+)"')
_PARENT = re.compile(r' ParentId="(\d+)"')
_HEAD_WINDOW = 400  # ParentId стоит в начале строки; длинное тело искать не нужно


# --------------------------------------------------------------------------- хеш

def thread_hash(thread_id: int) -> int:
    """Мультипликативный хеш Кнута: ``(id * 2654435761) mod 2^32``."""
    return (thread_id * KNUTH_MULTIPLIER) & _MASK32


def in_sample(thread_id: int, denom: int, residue: int = 0) -> bool:
    """Входит ли тред в выборку 1/``denom`` с остатком ``residue``."""
    return thread_hash(thread_id) % denom == residue


# ------------------------------------------------------------------------ чтение

def iter_lines(stream: Iterable[bytes]) -> Iterator[str]:
    """Двоичный поток (файл, stdout ``7zz``) -> строки без BOM, CR и LF.

    Отступ (индентация) строки сохраняется.
    """
    first = True
    for raw in stream:
        line = raw.decode("utf-8", errors="replace")
        if first:
            line = line.lstrip("﻿")
            first = False
        yield line.rstrip("\r\n")


def _normalize(line: str) -> str:
    return line.lstrip("﻿").rstrip("\r\n")


def _unescape(value: str) -> str:
    return html.unescape(value) if "&" in value else value


def _to_text(line) -> str:
    if isinstance(line, bytes):
        line = line.decode("utf-8", errors="replace")
    return _normalize(line)


def _attrs(line: str) -> Dict[str, str]:
    return {k: _unescape(v) for k, v in _ATTR.findall(line)}


def _opt_int(value: Optional[str]) -> Optional[int]:
    return int(value) if value not in (None, "") else None


def parse_tags(raw: Optional[str]) -> List[str]:
    """``<c++><c#><.net>`` (в том числе с экранированными скобками) -> список тегов."""
    if not raw:
        return []
    raw = html.unescape(raw)
    return [t for t in re.findall(r"<([^<>]+)>", raw) if t]


def parse_row(line) -> Optional[dict]:
    """Строка ``Posts.xml`` -> словарь полей поста или ``None`` для служебных строк.

    Атрибуты XML раскрываются один раз: ``Body`` остаётся HTML (с экранированными
    внутри него сущностями), ``Title`` — обычная строка.
    """
    text = _to_text(line)
    if not text.lstrip().startswith("<row "):
        return None
    a = _attrs(text)
    if "Id" not in a:
        return None
    return {
        "id": int(a["Id"]),
        "post_type": _opt_int(a.get("PostTypeId")),
        "parent_id": _opt_int(a.get("ParentId")),
        "accepted_answer_id": _opt_int(a.get("AcceptedAnswerId")),
        "score": int(a.get("Score", 0) or 0),
        "title": a.get("Title", ""),
        "body": a.get("Body", ""),
        "tags": parse_tags(_raw_attr(text, "Tags")),
        "creation_date": a.get("CreationDate", ""),
    }


def _raw_attr(line: str, name: str) -> Optional[str]:
    """Значение атрибута без раскрытия сущностей (для ``Tags``: ``&lt;c++&gt;``)."""
    m = re.search(r' %s="([^"]*)"' % re.escape(name), line)
    return m.group(1) if m else None


def parse_link_row(line) -> Optional[Tuple[int, int, int]]:
    """Строка ``PostLinks.xml`` -> ``(PostId, RelatedPostId, LinkTypeId)``."""
    text = _to_text(line)
    if not text.lstrip().startswith("<row "):
        return None
    a = _attrs(text)
    try:
        return int(a["PostId"]), int(a["RelatedPostId"]), int(a["LinkTypeId"])
    except (KeyError, ValueError):
        return None


# ------------------------------------------------------------------ отбор строк

def select_rows(lines: Iterable[str], in_corpus: Callable[[int], bool]) -> Iterator[str]:
    """Проход 1 по ``Posts.xml``: оставляет вопросы с ``Id`` из корпуса и ответы с
    ``ParentId`` из корпуса. Строки возвращаются нормализованными (без BOM/CR/LF).

    Разбирается только начало строки — полный разбор 60 млн записей на Python был бы
    слишком дорогим.
    """
    for line in lines:
        text = _normalize(line)
        m = _HEAD.match(text)
        if not m:
            continue
        post_id, post_type = int(m.group(1)), int(m.group(2))
        if post_type == POST_QUESTION:
            if in_corpus(post_id):
                yield text
        elif post_type == POST_ANSWER:
            p = _PARENT.search(text, 0, _HEAD_WINDOW)
            if p and in_corpus(int(p.group(1))):
                yield text


def build_neighbor_set(links: Iterable[Tuple[int, ...]], denom: int, residue: int,
                       max_neighbors: int) -> Set[int]:
    """Проход 0 по ``PostLinks.xml``: соседи выборки по связям.

    Для связи, у которой ровно один конец входит в выборку, второй конец добавляется
    в множество соседей, но не более ``max_neighbors`` на узел выборки (хабы с тысячами
    дубликатов не раздувают корпус). Порядок связей определяет, какие именно соседи
    попадут под лимит, поэтому результат детерминирован при фиксированном порядке файла.
    """
    per_node: Dict[int, Set[int]] = defaultdict(set)
    neighbors: Set[int] = set()
    for link in links:
        a, b = link[0], link[1]
        a_in, b_in = in_sample(a, denom, residue), in_sample(b, denom, residue)
        if a_in == b_in:
            continue  # оба в выборке или оба вне её
        node, other = (a, b) if a_in else (b, a)
        bucket = per_node[node]
        if other in bucket or len(bucket) >= max_neighbors:
            continue
        bucket.add(other)
        neighbors.add(other)
    return neighbors


# ------------------------------------------------------------------ сборка тредов

@dataclass(frozen=True)
class Thread:
    """Тред SO: вопрос + принятый ответ (документ индекса и узел графа)."""
    doc_id: int
    title: str
    question_body: str
    tags: List[str]
    question_score: int
    accepted_answer_id: int
    best_answer_body: str
    best_answer_score: int
    created_at: int


def _unix(date: str) -> int:
    if not date:
        return 0
    try:
        dt = datetime.strptime(date[:19], "%Y-%m-%dT%H:%M:%S")
    except ValueError:
        return 0
    return int(dt.replace(tzinfo=timezone.utc).timestamp())


def build_threads(rows: Iterable[dict], min_score: int = 1) -> List[Thread]:
    """Соединяет вопросы с принятыми ответами (порядок строк не важен).

    Тред валиден, если у вопроса есть ``AcceptedAnswerId``, ``Score >= min_score``, а
    строка принятого ответа присутствует и принадлежит этому вопросу. Результат
    отсортирован по ``doc_id``.
    """
    questions: Dict[int, dict] = {}
    answers: Dict[int, dict] = {}
    for r in rows:
        if r["post_type"] == POST_QUESTION and r["accepted_answer_id"]:
            questions[r["id"]] = r
        elif r["post_type"] == POST_ANSWER:
            answers[r["id"]] = r
    threads: List[Thread] = []
    for qid in sorted(questions):
        q = questions[qid]
        if q["score"] < min_score:
            continue
        a = answers.get(q["accepted_answer_id"])
        if a is None or a["parent_id"] != qid:
            continue
        threads.append(Thread(
            doc_id=qid,
            title=q["title"],
            question_body=q["body"],
            tags=q["tags"],
            question_score=q["score"],
            accepted_answer_id=a["id"],
            best_answer_body=a["body"],
            best_answer_score=a["score"],
            created_at=_unix(q["creation_date"]),
        ))
    return threads


def build_edges(links: Iterable[Tuple[int, ...]],
                valid_ids: Set[int]) -> List[Tuple[int, int, str, float]]:
    """Рёбра графа ``thread_links``: оба конца — валидные треды корпуса.

    ``LinkTypeId`` 1 -> ``LINKED_IN_ANSWER``, 3 -> ``DUPLICATE_OF``; остальные типы,
    петли и дубликаты отбрасываются; результат отсортирован.
    """
    edges: Set[Tuple[int, int, str, float]] = set()
    for src, dst, link_type in links:
        kind = _LINK_TYPES.get(link_type)
        if kind is None or src == dst or src not in valid_ids or dst not in valid_ids:
            continue
        edges.add((src, dst, kind, 1.0))
    return sorted(edges)
