"""Интеграционный тест benchmark: сборка индекса корпуса «вживую» и контроль разметки
(docs/Tech/05-verification.md, раздел 3; docs/Tech/03-data.md, п. 4.9).

Эталонный BM25 (из test_indexer_e2e) нужен только для проверки согласованности разметки с индексом;
официальные метрики считает benchmark/evaluate.py.
"""
import json
import os
import sys
import tempfile
import time
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.abspath(os.path.join(HERE, "../.."))
sys.path.insert(0, HERE)

from test_indexer_e2e import bm25_search, run_indexer  # noqa: E402

CORPUS = os.path.join(REPO_ROOT, "benchmark", "corpus", "documents.jsonl")
QUESTIONS = os.path.join(REPO_ROOT, "benchmark", "questions.json")


class TestBenchmarkEndToEnd(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls._tmp = tempfile.TemporaryDirectory()
        cls.out = os.path.join(cls._tmp.name, "index-benchmark")
        started = time.time()
        result = run_indexer("corpus", {"CORPUS_PATH": CORPUS, "INDEX_OUTPUT_PATH": cls.out}, cls._tmp.name)
        cls.build_seconds = time.time() - started
        assert result.returncode == 0, result.stderr
        with open(QUESTIONS, encoding="utf-8") as f:
            cls.questions = json.load(f)["questions"]

    @classmethod
    def tearDownClass(cls):
        cls._tmp.cleanup()

    def labels(self, question, k=10):
        return [f"D{doc_id:02d}" for doc_id, _ in bm25_search(self.out, question, k)]

    def test_build_within_five_seconds(self):
        self.assertLess(self.build_seconds, 5.0)

    def test_index_describes_thirty_documents(self):
        with open(os.path.join(self.out, "metadata.json"), encoding="utf-8") as f:
            meta = json.load(f)
        self.assertEqual((meta["index_mode"], meta["num_documents"]), ("corpus", 30))

    def test_each_answerable_question_finds_a_relevant_document_in_top_10(self):
        for q in self.questions:
            if q["type"] != "answerable":
                continue
            with self.subTest(question=q["id"]):
                self.assertTrue(set(q["relevant_docs"]) & set(self.labels(q["question"])))

    def test_reference_recall_and_mrr_are_sane(self):
        answerable = [q for q in self.questions if q["type"] == "answerable"]
        recall10 = 0.0
        rr = 0.0
        for q in answerable:
            hits = self.labels(q["question"])
            rel = set(q["relevant_docs"])
            recall10 += len(rel & set(hits)) / len(rel)
            first = next((i + 1 for i, h in enumerate(hits) if h in rel), None)
            rr += 1 / first if first else 0.0
        self.assertGreater(recall10 / len(answerable), 0.5)
        self.assertGreater(rr / len(answerable), 0.5)

    def test_graph_db_gives_rag_the_answer_of_every_document(self):
        import sqlite3
        conn = sqlite3.connect(f"file:{os.path.join(self.out, 'graph.db')}?mode=ro", uri=True)
        self.addCleanup(conn.close)
        rows = conn.execute("SELECT doc_id, has_accepted_answer, snippet, best_answer_body FROM thread_nodes").fetchall()
        self.assertEqual(len(rows), 30)
        for doc_id, accepted, snippet, answer in rows:
            with self.subTest(doc_id=doc_id):
                self.assertEqual(accepted, 1)
                self.assertTrue(snippet and not snippet.startswith("Question:"))
                self.assertTrue(answer.startswith("<p>") and len(answer) > 50)

    def test_build_is_deterministic(self):
        second = os.path.join(self._tmp.name, "second")
        result = run_indexer("corpus", {"CORPUS_PATH": CORPUS, "INDEX_OUTPUT_PATH": second}, self._tmp.name)
        self.assertEqual(result.returncode, 0, result.stderr)
        for name in ("lexicon.bin", "postings.bin", "doc_lengths.bin"):
            with open(os.path.join(self.out, name), "rb") as a, open(os.path.join(second, name), "rb") as b:
                self.assertEqual(a.read(), b.read(), name)


if __name__ == "__main__":
    unittest.main()
