"""Сквозная проверка ядра на индексе, построенном реальным индексатором:
баллы SearchEngine обязаны совпадать с эталонным BM25 из test_indexer_e2e.py
(независимый расчёт по файлам индекса). Только stdlib; выполняется на хосте
(`make test-integration`), в контейнере ядра пропускается (нет полного индексатора).
"""
import json
import math
import os
import sys
import tempfile
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
INDEXER_SRC = os.path.join(REPO_ROOT, "services", "search_engine_indexer", "src")
CORE_SRC = os.path.join(REPO_ROOT, "services", "search_engine_core", "src")
for _p in (CORE_SRC, INDEXER_SRC):
    if os.path.isdir(_p) and _p not in sys.path:
        sys.path.insert(0, _p)

INDEXER_AVAILABLE = os.path.isdir(INDEXER_SRC)
if INDEXER_AVAILABLE:
    from test_indexer_e2e import bm25_search, run_indexer
    from indexer.index_format import read_doc_lengths, read_lexicon, read_postings
    from indexer.tokenizer import tokenize
    from core.engine import SearchEngine
    from core.reader import IndexReader

LESSON3 = [
    {"doc_id": 1, "text": "Возврат оплаты на карту"},
    {"doc_id": 2, "text": "Статус оплаты через приложение"},
    {"doc_id": 3, "text": "Возврат билета через кассу"},
]

CODE_DOCS = [
    {"doc_id": 1, "title": "Crash", "text": "java.lang.NullPointerException when calling method on null"},
    {"doc_id": 2, "title": "Index", "text": "IndexOutOfBoundsException while reading the list"},
    {"doc_id": 3, "title": "Docker", "text": "docker-compose up fails with ECONNREFUSED on port 5432"},
    {"doc_id": 4, "title": "Plain", "text": "nothing relevant here at all"},
]


def reference_search(index_dir, query, k1, b, k=10):
    """Независимый эталон по файлам индекса с произвольными k1/b (формула методички)."""
    with open(os.path.join(index_dir, "metadata.json"), encoding="utf-8") as f:
        meta = json.load(f)
    lexicon = read_lexicon(os.path.join(index_dir, "lexicon.bin"))
    lengths = read_doc_lengths(os.path.join(index_dir, "doc_lengths.bin"))
    scores = {}
    for term in dict.fromkeys(tokenize(query)):
        entry = lexicon.get(term)
        if entry is None:
            continue
        idf = math.log(1 + (meta["num_documents"] - entry.df + 0.5) / (entry.df + 0.5))
        for doc_id, tf in read_postings(os.path.join(index_dir, "postings.bin"), entry):
            norm = k1 * (1 - b + b * lengths[doc_id] / meta["avg_doc_length"])
            scores[doc_id] = scores.get(doc_id, 0.0) + idf * tf * (k1 + 1) / (tf + norm)
    ranked = sorted(scores, key=lambda d: (-scores[d], d))[:k]
    return [(d, scores[d]) for d in ranked], len(scores)


@unittest.skipUnless(INDEXER_AVAILABLE, "полный индексатор недоступен (контейнер ядра)")
class TestCoreEngineParity(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self._tmp.cleanup)
        self.root = self._tmp.name

    def build(self, docs):
        corpus = os.path.join(self.root, "documents.jsonl")
        with open(corpus, "w", encoding="utf-8") as f:
            for d in docs:
                f.write(json.dumps(d, ensure_ascii=False) + "\n")
        out = os.path.join(self.root, "index")
        result = run_indexer("corpus", {"CORPUS_PATH": corpus, "INDEX_OUTPUT_PATH": out}, self.root)
        self.assertEqual(result.returncode, 0, result.stderr)
        return out

    def assert_parity(self, index_dir, queries, k1, b):
        engine = SearchEngine(IndexReader(index_dir), k1=k1, b=b)
        for query in queries:
            expected, total = reference_search(index_dir, query, k1, b)
            result = engine.search(query, top_k=10)
            self.assertEqual([d for d, _ in result.hits], [d for d, _ in expected], query)
            for (_, got), (_, want) in zip(result.hits, expected):
                self.assertAlmostEqual(got, want, places=9, msg=query)
            self.assertEqual(result.total_hits, total, query)

    def test_lesson3_matches_existing_reference(self):
        # сверка с эталоном занятия 3 из test_indexer_e2e (k1=1.2)
        index_dir = self.build(LESSON3)
        engine = SearchEngine(IndexReader(index_dir), k1=1.2, b=0.75)
        query = "Как происходит возврат оплаты?"
        expected = bm25_search(index_dir, query)
        result = engine.search(query)
        self.assertEqual([d for d, _ in result.hits], [d for d, _ in expected])
        for (_, got), (_, want) in zip(result.hits, expected):
            self.assertAlmostEqual(got, want, places=9)

    def test_lesson3_parity_default_and_lesson_params(self):
        index_dir = self.build(LESSON3)
        queries = ["Как происходит возврат оплаты?", "возврат", "оплаты возврат", "касса"]
        self.assert_parity(index_dir, queries, k1=1.2, b=0.75)
        self.assert_parity(index_dir, queries, k1=1.5, b=0.75)

    def test_code_identifiers_parity(self):
        index_dir = self.build(CODE_DOCS)
        queries = ["NullPointerException", "null pointer exception", "ECONNREFUSED",
                   "docker-compose", "index out of bounds", "полностью отсутствующий термин"]
        self.assert_parity(index_dir, queries, k1=1.5, b=0.75)


if __name__ == "__main__":
    unittest.main()