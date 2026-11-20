"""Read-only доступ к graph.db: узлы, окрестность, детали треда
(docs/Tech/03-data.md, раздел 2 и п. 6.5).
"""
import json
import os
import sqlite3
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, List, Optional, Sequence

CATEGORY_BY_LINK = {"DUPLICATE_OF": "DUPLICATE", "LINKED_IN_ANSWER": "LINKED"}

_NODE_COLUMNS = ("doc_id, title, snippet, has_accepted_answer, tags, question_score, "
                 "best_answer_score, best_answer_body, accepted_answer_id, created_at")
_CHUNK = 500


@dataclass(frozen=True)
class Node:
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


@dataclass(frozen=True)
class GraphNode:
    doc_id: int
    category: str            # PRIMARY_RESULT | OTHER_RESULT | DUPLICATE | LINKED
    score: float             # голоса вопроса SO (семантика Star Map)
    title: str
    snippet: str
    has_accepted_answer: bool
    tags: List[str]
    best_answer_score: int


@dataclass(frozen=True)
class GraphEdge:
    source: int
    target: int
    type: str                # DUPLICATE_OF | LINKED_IN_ANSWER


@dataclass(frozen=True)
class GraphView:
    nodes: List[GraphNode]
    edges: List[GraphEdge]


@dataclass(frozen=True)
class ThreadDetails:
    doc_id: int
    title: str
    best_answer_body: str
    original_url: str


def _node_from_row(row) -> Node:
    return Node(doc_id=row[0], title=row[1], snippet=row[2], has_accepted_answer=bool(row[3]),
                tags=json.loads(row[4]) if row[4] else [], question_score=row[5],
                best_answer_score=row[6], best_answer_body=row[7] or "",
                accepted_answer_id=row[8], created_at=row[9])


def _chunks(ids: Sequence[int]):
    for i in range(0, len(ids), _CHUNK):
        yield ids[i:i + _CHUNK]


class GraphStore:
    """Соединение открывается на запрос в режиме ro; отсутствующий файл — пустой граф."""

    def __init__(self, db_path: str):
        self.db_path = db_path

    def _connect(self) -> Optional[sqlite3.Connection]:
        if not os.path.isfile(self.db_path):
            return None
        uri = Path(self.db_path).resolve().as_uri() + "?mode=ro"
        return sqlite3.connect(uri, uri=True)

    def nodes(self, doc_ids: Sequence[int]) -> Dict[int, Node]:
        ids = list(dict.fromkeys(doc_ids))
        conn = self._connect()
        if conn is None or not ids:
            return {}
        result: Dict[int, Node] = {}
        try:
            for chunk in _chunks(ids):
                placeholders = ",".join("?" * len(chunk))
                rows = conn.execute(
                    f"SELECT {_NODE_COLUMNS} FROM thread_nodes WHERE doc_id IN ({placeholders})", chunk)
                for row in rows:
                    result[row[0]] = _node_from_row(row)
        finally:
            conn.close()
        return result

    def thread_graph(self, doc_ids: Sequence[int]) -> GraphView:
        requested = list(dict.fromkeys(doc_ids))
        nodes_map = self.nodes(requested)
        ordered = [d for d in requested if d in nodes_map]
        if not ordered:
            return GraphView([], [])

        edges: List[GraphEdge] = []
        seen_edges = set()
        neighbor_ids: List[int] = []
        conn = self._connect()
        try:
            for chunk in _chunks(ordered):
                placeholders = ",".join("?" * len(chunk))
                rows = conn.execute(
                    "SELECT source_doc_id, target_doc_id, link_type FROM thread_links "
                    f"WHERE source_doc_id IN ({placeholders})", chunk)
                for source, target, link_type in rows:
                    key = (source, target, link_type)
                    if key in seen_edges:
                        continue
                    seen_edges.add(key)
                    edges.append(GraphEdge(source, target, link_type))
                    if target not in nodes_map and target not in neighbor_ids:
                        neighbor_ids.append(target)
        finally:
            conn.close()
        nodes_map.update(self.nodes(neighbor_ids))

        def to_graph_node(node: Node, category: str) -> GraphNode:
            return GraphNode(doc_id=node.doc_id, category=category,
                             score=float(node.question_score), title=node.title,
                             snippet=node.snippet, has_accepted_answer=node.has_accepted_answer,
                             tags=node.tags, best_answer_score=node.best_answer_score)

        result_nodes = [to_graph_node(nodes_map[d], "PRIMARY_RESULT" if i == 0 else "OTHER_RESULT")
                        for i, d in enumerate(ordered)]
        result_ids = set(ordered)
        neighbor_categories: Dict[int, str] = {}
        for edge in edges:
            if edge.target not in result_ids:
                category = CATEGORY_BY_LINK.get(edge.type, "LINKED")
                prev = neighbor_categories.get(edge.target)
                # дубликат «сильнее» ссылки, если узел достижим обоими типами рёбер
                if prev is None or (prev == "LINKED" and category == "DUPLICATE"):
                    neighbor_categories[edge.target] = category
        for doc_id in sorted(neighbor_categories):
            if doc_id in nodes_map:
                result_nodes.append(to_graph_node(nodes_map[doc_id], neighbor_categories[doc_id]))
        return GraphView(result_nodes, edges)

    def details(self, doc_id: int) -> Optional[ThreadDetails]:
        conn = self._connect()
        if conn is None:
            return None
        try:
            row = conn.execute(
                "SELECT title, best_answer_body FROM thread_nodes WHERE doc_id = ?", (doc_id,)).fetchone()
        finally:
            conn.close()
        if row is None:
            return None
        return ThreadDetails(doc_id=doc_id, title=row[0], best_answer_body=row[1] or "",
                             original_url=f"https://stackoverflow.com/questions/{doc_id}")