"""Граф связанных тредов: категории узлов, рёбра, детали (docs/Tech/03-data.md, раздел 2, п. 6.5)."""
import os
import tempfile
import unittest

from _core_fixture import build_index

from core.graph import GraphStore

DOCS = [
    {"doc_id": 1, "title": "Primary thread", "text": "question one", "tags": ["python"],
     "question_score": 100, "best_answer_body": "<p>answer one</p>", "best_answer_score": 50,
     "accepted_answer_id": 11, "created_at": 1700000000},
    {"doc_id": 2, "title": "Second result", "text": "question two", "tags": ["java"],
     "question_score": 20},
    {"doc_id": 3, "title": "Duplicate thread", "text": "question three", "question_score": 5},
    {"doc_id": 4, "title": "Linked thread", "text": "question four", "question_score": 7},
]
EDGES = [
    (1, 3, "DUPLICATE_OF"),
    (1, 4, "LINKED_IN_ANSWER"),
    (2, 4, "LINKED_IN_ANSWER"),
]


class TestThreadGraph(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self._tmp.cleanup)
        self.index_dir = build_index(self._tmp.name, DOCS, edges=EDGES)
        self.store = GraphStore(os.path.join(self.index_dir, "graph.db"))

    def test_categories_primary_first_then_other(self):
        view = self.store.thread_graph([2, 1, 999])
        by_id = {n.doc_id: n for n in view.nodes}
        self.assertEqual(by_id[2].category, "PRIMARY_RESULT")   # первый существующий
        self.assertEqual(by_id[1].category, "OTHER_RESULT")
        self.assertNotIn(999, by_id)                            # неизвестный пропущен

    def test_neighbors_categories_and_scores(self):
        view = self.store.thread_graph([1])
        by_id = {n.doc_id: n for n in view.nodes}
        self.assertEqual(by_id[1].category, "PRIMARY_RESULT")
        self.assertEqual(by_id[3].category, "DUPLICATE")
        self.assertEqual(by_id[4].category, "LINKED")
        # score узла = question_score (голоса SO)
        self.assertEqual(by_id[1].score, 100.0)
        self.assertEqual(by_id[3].score, 5.0)
        self.assertEqual(by_id[1].tags, ["python"])
        self.assertTrue(by_id[1].has_accepted_answer)
        self.assertFalse(by_id[3].has_accepted_answer)

    def test_result_node_keeps_category_when_also_neighbor(self):
        view = self.store.thread_graph([1, 3])
        by_id = {n.doc_id: n for n in view.nodes}
        self.assertEqual(by_id[3].category, "OTHER_RESULT")

    def test_edges_from_all_requested_nodes(self):
        view = self.store.thread_graph([1, 2])
        edges = {(e.source, e.target, e.type) for e in view.edges}
        self.assertEqual(edges, {(1, 3, "DUPLICATE_OF"), (1, 4, "LINKED_IN_ANSWER"),
                                 (2, 4, "LINKED_IN_ANSWER")})

    def test_edges_deduplicated(self):
        view = self.store.thread_graph([1, 1, 2])
        self.assertEqual(len(view.edges), 3)
        self.assertEqual(len({n.doc_id for n in view.nodes}), len(view.nodes))

    def test_empty_request(self):
        view = self.store.thread_graph([])
        self.assertEqual(view.nodes, [])
        self.assertEqual(view.edges, [])

    def test_details(self):
        details = self.store.details(1)
        self.assertEqual(details.title, "Primary thread")
        self.assertEqual(details.best_answer_body, "<p>answer one</p>")
        self.assertEqual(details.original_url, "https://stackoverflow.com/questions/1")

    def test_details_missing_doc(self):
        self.assertIsNone(self.store.details(999))

    def test_nodes_metadata(self):
        nodes = self.store.nodes([1, 999])
        self.assertEqual(set(nodes), {1})
        self.assertEqual(nodes[1].question_score, 100)
        self.assertEqual(nodes[1].accepted_answer_id, 11)
        self.assertEqual(nodes[1].created_at, 1700000000)
        self.assertEqual(nodes[1].best_answer_score, 50)

    def test_missing_db_file_is_tolerated(self):
        store = GraphStore(os.path.join(self._tmp.name, "absent.db"))
        self.assertEqual(store.nodes([1]), {})
        self.assertEqual(store.thread_graph([1]).nodes, [])
        self.assertIsNone(store.details(1))

    def test_read_only_no_side_files(self):
        self.store.thread_graph([1, 2])
        self.store.details(1)
        for suffix in ("-wal", "-shm", "-journal"):
            self.assertFalse(os.path.exists(os.path.join(self.index_dir, "graph.db") + suffix))


if __name__ == "__main__":
    unittest.main()