"""Поисковый движок ядра: токенизация запроса, BM25, фильтры, Top-K
(docs/Tech/03-data.md, п. 6.3–6.4)."""
import math
import os
import tempfile
import unittest

from _core_fixture import build_index

from core.engine import SearchEngine
from core.reader import IndexNotReadyError, IndexReader

LESSON3 = [
    {"doc_id": 1, "text": "Возврат оплаты на карту"},
    {"doc_id": 2, "text": "Статус оплаты через приложение"},
    {"doc_id": 3, "text": "Возврат билета через кассу"},
]

TAGGED = [
    {"doc_id": 1, "text": "connection error in database driver", "tags": ["python", "postgresql"]},
    {"doc_id": 2, "text": "connection error in jdbc driver", "tags": ["java"]},
    {"doc_id": 3, "text": "connection error in docker network", "tags": ["docker"]},
    {"doc_id": 4, "text": "unrelated text about sorting", "tags": ["python"]},
]


class EngineTestBase(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self._tmp.cleanup)
        self.root = self._tmp.name

    def engine(self, docs, **kwargs):
        index_dir = build_index(self.root, docs)
        engine = SearchEngine(IndexReader(index_dir), **kwargs)
        self.addCleanup(engine.reader.close)
        return engine


class TestLesson3Example(EngineTestBase):
    def test_ranking_and_scores(self):
        engine = self.engine(LESSON3, k1=1.2, b=0.75)
        result = engine.search("Как происходит возврат оплаты?")
        idf = math.log(1.6)
        self.assertEqual([d for d, _ in result.hits], [1, 2, 3])
        self.assertAlmostEqual(result.hits[0][1], 2 * idf, places=9)
        self.assertAlmostEqual(result.hits[1][1], idf, places=9)
        self.assertAlmostEqual(result.hits[2][1], idf, places=9)
        self.assertEqual(result.total_hits, 3)
        self.assertGreaterEqual(result.latency_ms, 0.0)

    def test_tokenized_query_returned(self):
        engine = self.engine(LESSON3)
        result = engine.search("Как происходит возврат оплаты?")
        self.assertEqual(result.tokens, ["как", "происходит", "возврат", "оплаты"])

    def test_repeated_query_term_not_double_counted(self):
        engine = self.engine(LESSON3, k1=1.2, b=0.75)
        once = engine.search("возврат оплаты")
        twice = engine.search("возврат возврат оплаты оплаты")
        self.assertEqual(once.hits, twice.hits)

    def test_top_k_limits_hits_not_total(self):
        engine = self.engine(LESSON3)
        result = engine.search("возврат оплаты", top_k=2)
        self.assertEqual([d for d, _ in result.hits], [1, 2])
        self.assertEqual(result.total_hits, 3)

    def test_absent_term_gives_empty_result(self):
        engine = self.engine(LESSON3)
        result = engine.search("совершенно отсутствующий терм")
        self.assertEqual(result.hits, [])
        self.assertEqual(result.total_hits, 0)


class TestCodeIdentifiers(EngineTestBase):
    def test_whole_token_and_subtokens(self):
        docs = [
            {"doc_id": 1, "text": "java.lang.NullPointerException when calling method on null"},
            {"doc_id": 2, "text": "IndexOutOfBoundsException while reading the list"},
            {"doc_id": 3, "text": "docker-compose up fails with ECONNREFUSED on port 5432"},
        ]
        engine = self.engine(docs)
        self.assertEqual(engine.search("NullPointerException").hits[0][0], 1)
        self.assertEqual(engine.search("null pointer exception").hits[0][0], 1)
        self.assertEqual(engine.search("ECONNREFUSED").hits[0][0], 3)
        self.assertEqual(engine.search("docker-compose").hits[0][0], 3)

    def test_query_html_is_not_stripped(self):
        # List<String> не должен приниматься за HTML-тег (п. 5.1.3)
        docs = [{"doc_id": 1, "text": "generic List<String> type mismatch in java"}]
        engine = self.engine(docs)
        result = engine.search("List<String>")
        self.assertIn("list", result.tokens)
        self.assertIn("string", result.tokens)
        self.assertEqual(result.hits[0][0], 1)


class TestFilters(EngineTestBase):
    def test_tags_filter_is_conjunctive(self):
        engine = self.engine(TAGGED)
        result = engine.search("connection error", tags=["python"])
        self.assertEqual([d for d, _ in result.hits], [1])
        self.assertEqual(result.total_hits, 3)  # до фильтров
        result = engine.search("connection error", tags=["python", "postgresql"])
        self.assertEqual([d for d, _ in result.hits], [1])
        result = engine.search("connection error", tags=["python", "java"])
        self.assertEqual(result.hits, [])

    def test_tags_case_insensitive(self):
        engine = self.engine(TAGGED)
        result = engine.search("connection error", tags=["PYTHON"])
        self.assertEqual([d for d, _ in result.hits], [1])

    def test_language_acts_as_tag(self):
        engine = self.engine(TAGGED)
        result = engine.search("connection error", language="Java")
        self.assertEqual([d for d, _ in result.hits], [2])

    def test_no_filter_returns_all_candidates(self):
        engine = self.engine(TAGGED)
        result = engine.search("connection error")
        self.assertEqual([d for d, _ in result.hits], [1, 2, 3])


class TestTopKConfig(EngineTestBase):
    def test_default_top_k_is_10(self):
        docs = [{"doc_id": i, "text": "common term here"} for i in range(1, 16)]
        engine = self.engine(docs)
        self.assertEqual(len(engine.search("common").hits), 10)

    def test_top_k_capped_by_max(self):
        docs = [{"doc_id": i, "text": "common term here"} for i in range(1, 16)]
        engine = self.engine(docs, max_top_k=5)
        self.assertEqual(len(engine.search("common", top_k=1000).hits), 5)

    def test_nonpositive_top_k_falls_back_to_default(self):
        docs = [{"doc_id": i, "text": "common term here"} for i in range(1, 16)]
        engine = self.engine(docs)
        self.assertEqual(len(engine.search("common", top_k=0).hits), 10)


class TestReadiness(EngineTestBase):
    def test_not_ready_raises(self):
        engine = SearchEngine(IndexReader(os.path.join(self.root, "missing")))
        self.addCleanup(engine.reader.close)
        with self.assertRaises(IndexNotReadyError):
            engine.search("anything")

    def test_lazy_reload(self):
        engine = SearchEngine(IndexReader(os.path.join(self.root, "index")))
        self.addCleanup(engine.reader.close)
        with self.assertRaises(IndexNotReadyError):
            engine.search("возврат")
        build_index(self.root, LESSON3)
        self.assertEqual(engine.search("возврат").hits[0][0], 1)


if __name__ == "__main__":
    unittest.main()