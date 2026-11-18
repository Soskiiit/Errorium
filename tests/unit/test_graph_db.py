"""Unit-тесты записи graph.db (docs/Tech/03-data.md, раздел 2)."""
import json
import os
import sqlite3
import sys
import tempfile
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
sys.path.insert(0, os.path.join(REPO_ROOT, "services", "search_engine_indexer", "src"))

from indexer.graph_db import (  # noqa: E402
    SNIPPET_MAX_LEN,
    GraphNode,
    make_snippet,
    node_from_thread,
    write_graph_db,
)
from indexer.sampler import Thread  # noqa: E402


def thread(doc_id=10, **over):
    fields = dict(doc_id=doc_id, title="NPE in Java", question_body="<p>Why <code>x.y()</code> fails?</p>",
                  tags=["java", "c#"], question_score=9, accepted_answer_id=11,
                  best_answer_body="<p>Check <code>x</code> for null</p>", best_answer_score=4,
                  created_at=1218069951)
    fields.update(over)
    return Thread(**fields)


def node(doc_id, **over):
    fields = dict(doc_id=doc_id, title=f"t{doc_id}", snippet=f"s{doc_id}", has_accepted_answer=True,
                  tags=["python"], question_score=1, best_answer_score=2, best_answer_body="<p>a</p>",
                  accepted_answer_id=doc_id + 1000, created_at=100 + doc_id)
    fields.update(over)
    return GraphNode(**fields)


class TestSnippet(unittest.TestCase):
    def test_html_stripped_and_whitespace_collapsed(self):
        self.assertEqual(make_snippet("<p>Why   <code>x.y()</code>\n fails?</p>"), "Why x.y() fails?")

    def test_entities_unescaped(self):
        self.assertEqual(make_snippet("<p>List&lt;String&gt; &amp; more</p>"), "List<String> & more")

    def test_short_text_unchanged(self):
        self.assertEqual(make_snippet("<p>short</p>"), "short")

    def test_long_text_truncated_with_ellipsis(self):
        out = make_snippet("<p>" + "word " * 200 + "</p>")
        self.assertLessEqual(len(out), SNIPPET_MAX_LEN)
        self.assertTrue(out.endswith("…"))

    def test_exact_limit_not_truncated(self):
        text = "a" * SNIPPET_MAX_LEN
        self.assertEqual(make_snippet(text), text)

    def test_empty(self):
        self.assertEqual(make_snippet(""), "")


class TestNodeFromThread(unittest.TestCase):
    def test_fields(self):
        n = node_from_thread(thread())
        self.assertEqual(n.doc_id, 10)
        self.assertEqual(n.title, "NPE in Java")
        self.assertEqual(n.snippet, "Why x.y() fails?")
        self.assertTrue(n.has_accepted_answer)
        self.assertEqual(n.tags, ["java", "c#"])
        self.assertEqual(n.question_score, 9)
        self.assertEqual(n.best_answer_score, 4)
        self.assertEqual(n.accepted_answer_id, 11)
        self.assertEqual(n.created_at, 1218069951)

    def test_answer_body_kept_as_html(self):
        self.assertEqual(node_from_thread(thread()).best_answer_body, "<p>Check <code>x</code> for null</p>")


class TestWriteGraphDb(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self._tmp.cleanup)
        self.path = os.path.join(self._tmp.name, "graph.db")

    def connect(self):
        conn = sqlite3.connect(f"file:{self.path}?mode=ro", uri=True)
        self.addCleanup(conn.close)
        return conn

    def write(self, nodes=None, edges=None):
        nodes = [node(1), node(2), node(3)] if nodes is None else nodes
        edges = [(1, 2, "DUPLICATE_OF", 1.0), (2, 3, "LINKED_IN_ANSWER", 1.0)] if edges is None else edges
        return write_graph_db(self.path, nodes, edges)

    def test_counts_returned(self):
        self.assertEqual(self.write(), {"nodes": 3, "edges": 2})

    def test_schema_matches_spec(self):
        self.write()
        conn = self.connect()
        cols = [r[1] for r in conn.execute("PRAGMA table_info(thread_nodes)")]
        self.assertEqual(cols, ["doc_id", "title", "snippet", "has_accepted_answer", "tags", "question_score",
                                "best_answer_score", "best_answer_body", "accepted_answer_id", "created_at"])
        cols = [r[1] for r in conn.execute("PRAGMA table_info(thread_links)")]
        self.assertEqual(cols, ["source_doc_id", "target_doc_id", "link_type", "weight"])
        indexes = {r[1] for r in conn.execute("PRAGMA index_list(thread_links)")}
        self.assertIn("idx_thread_links_source", indexes)
        self.assertIn("idx_thread_links_target", indexes)

    def test_node_roundtrip(self):
        self.write(nodes=[node(7, tags=["c++", ".net"], has_accepted_answer=False, best_answer_body="")], edges=[])
        row = self.connect().execute(
            "SELECT doc_id, title, snippet, has_accepted_answer, tags, question_score, best_answer_score, "
            "best_answer_body, accepted_answer_id, created_at FROM thread_nodes").fetchone()
        self.assertEqual(row[:4], (7, "t7", "s7", 0))
        self.assertEqual(json.loads(row[4]), ["c++", ".net"])
        self.assertEqual(row[5:], (1, 2, "", 1007, 107))

    def test_spec_queries_work_read_only(self):
        self.write()
        conn = self.connect()
        adj = conn.execute("SELECT target_doc_id, link_type, weight FROM thread_links "
                           "WHERE source_doc_id IN (?, ?)", (1, 2)).fetchall()
        self.assertEqual(sorted(adj), [(2, "DUPLICATE_OF", 1.0), (3, "LINKED_IN_ANSWER", 1.0)])
        meta = conn.execute("SELECT doc_id, title, snippet, has_accepted_answer, tags, question_score, "
                            "best_answer_score FROM thread_nodes WHERE doc_id IN (?, ?)", (1, 3)).fetchall()
        self.assertEqual([m[0] for m in meta], [1, 3])
        details = conn.execute("SELECT title, best_answer_body FROM thread_nodes WHERE doc_id = ?", (2,)).fetchone()
        self.assertEqual(details, ("t2", "<p>a</p>"))

    def test_empty_graph(self):
        self.assertEqual(self.write(nodes=[], edges=[]), {"nodes": 0, "edges": 0})
        self.assertEqual(self.connect().execute("SELECT COUNT(*) FROM thread_nodes").fetchone(), (0,))

    def test_edge_with_unknown_endpoint_rejected(self):
        with self.assertRaises(ValueError):
            self.write(edges=[(1, 99, "DUPLICATE_OF", 1.0)])
        self.assertFalse(os.path.exists(self.path))

    def test_duplicate_node_rejected_and_no_partial_file(self):
        with self.assertRaises(ValueError):
            self.write(nodes=[node(1), node(1)], edges=[])
        self.assertEqual(os.listdir(os.path.dirname(self.path)), [])

    def test_failed_rewrite_keeps_previous_file(self):
        self.write()
        with open(self.path, "rb") as f:
            before = f.read()
        with self.assertRaises(ValueError):
            self.write(nodes=[node(1), node(1)], edges=[])
        with open(self.path, "rb") as f:
            self.assertEqual(f.read(), before)
        self.assertEqual(os.listdir(os.path.dirname(self.path)), ["graph.db"])

    def test_rewrite_replaces_content(self):
        self.write()
        self.write(nodes=[node(5)], edges=[])
        self.assertEqual(self.connect().execute("SELECT doc_id FROM thread_nodes").fetchall(), [(5,)])

    def test_accepts_generators(self):
        counts = write_graph_db(self.path, (node(i) for i in (1, 2)), iter([(1, 2, "DUPLICATE_OF", 1.0)]))
        self.assertEqual(counts, {"nodes": 2, "edges": 1})


if __name__ == "__main__":
    unittest.main()
