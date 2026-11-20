import json
import math
import os
import sys
import tempfile
import unittest

# Ensure search_engine_core and search_engine_indexer packages are in sys.path
PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
CORE_SRC = os.path.join(PROJECT_ROOT, "services/search_engine_core/src")
INDEXER_SRC = os.path.join(PROJECT_ROOT, "services/search_engine_indexer/src")

if CORE_SRC not in sys.path:
    sys.path.insert(0, CORE_SRC)
if INDEXER_SRC not in sys.path:
    sys.path.insert(0, INDEXER_SRC)

from indexer.graph_db import GraphNode, write_graph_db
from indexer.spimi import build_index
from indexer.tokenizer import tokenize

from core.config import CoreConfig
from core.engine import Engine, Index, ThreadStore


def manual_bm25_score(query_tokens, doc_tokens, all_docs_tokens, N, avgdl, k1=1.5, b=0.75):
    """Independent ground truth implementation of BM25 from the Bauman IU5 specification."""
    unique_terms = set(query_tokens)
    doc_len = len(doc_tokens)
    score = 0.0

    for term in unique_terms:
        # Calculate DF
        df = sum(1 for d in all_docs_tokens if term in d)
        if df == 0:
            continue
        # Term frequency in doc
        tf = doc_tokens.count(term)
        if tf == 0:
            continue

        idf = math.log(1.0 + (N - df + 0.5) / (df + 0.5))
        tf_component = (tf * (k1 + 1.0)) / (tf + k1 * (1.0 - b + b * (doc_len / avgdl)))
        score += idf * tf_component

    return score


class TestCoreEngine(unittest.TestCase):
    def setUp(self):
        self.tmp_dir = tempfile.TemporaryDirectory()
        self.index_dir = self.tmp_dir.name

    def tearDown(self):
        self.tmp_dir.cleanup()

    def _build_test_index(self, corpus, nodes=None, links=None, metadata_override=None):
        # 1. Build binary index
        tokenized_docs = [(doc_id, tokenize(text)) for doc_id, text in corpus]
        meta = build_index(tokenized_docs, self.index_dir)

        if metadata_override:
            meta.update(metadata_override)
            with open(os.path.join(self.index_dir, "metadata.json"), "w", encoding="utf-8") as f:
                json.dump(meta, f, indent=2)

        # 2. Build graph.db
        if nodes is None:
            nodes = [
                GraphNode(
                    doc_id=doc_id,
                    title=f"Title {doc_id}",
                    snippet=f"Snippet for {doc_id}",
                    has_accepted_answer=True,
                    tags=["python", "error"],
                    question_score=10,
                    best_answer_score=5,
                    best_answer_body="<p>Answer</p>",
                    accepted_answer_id=doc_id * 10,
                    created_at=1700000000 + doc_id,
                )
                for doc_id, _ in corpus
            ]
        if links is None:
            links = []

        write_graph_db(os.path.join(self.index_dir, "graph.db"), nodes, links)
        return meta

    def test_config_validation(self):
        # Default config
        cfg = CoreConfig()
        self.assertEqual(cfg.index_dir, "/app/data/index")
        self.assertAlmostEqual(cfg.bm25_k1, 1.5)
        self.assertAlmostEqual(cfg.bm25_b, 0.75)
        self.assertEqual(cfg.grpc_port, 50051)

        # Valid custom k1 in [1.2, 2.0]
        cfg_custom = CoreConfig(bm25_k1=1.2)
        self.assertAlmostEqual(cfg_custom.bm25_k1, 1.2)
        cfg_custom2 = CoreConfig(bm25_k1=2.0)
        self.assertAlmostEqual(cfg_custom2.bm25_k1, 2.0)

        # Invalid k1 outside [1.2, 2.0]
        with self.assertRaises(ValueError):
            CoreConfig(bm25_k1=1.19)
        with self.assertRaises(ValueError):
            CoreConfig(bm25_k1=2.01)

    def test_bm25_scoring_matches_reference(self):
        # 3 documents corpus
        corpus = [
            (1, "python error exception"),
            (2, "python code bug python"),
            (3, "java exception error failure bug"),
        ]
        meta = self._build_test_index(corpus)
        N = meta["num_documents"]
        avgdl = meta["avg_doc_length"]
        all_tokens = [tokenize(text) for _, text in corpus]

        cfg = CoreConfig(index_dir=self.index_dir, bm25_k1=1.5, bm25_b=0.75)
        engine = Engine.load(self.index_dir, config=cfg)

        query = "python error"
        query_tokens = tokenize(query)
        result = engine.search(query, top_k=10)

        # Total docs matching score > 0: doc 1 and doc 2 (has python), doc 3 (has error)
        self.assertEqual(result.total_hits, 3)
        self.assertEqual(len(result.hits), 3)

        # Verify exact BM25 scores against ground truth calculation
        for hit in result.hits:
            doc_idx = hit.doc_id - 1
            expected_score = manual_bm25_score(
                query_tokens, all_tokens[doc_idx], all_tokens, N, avgdl, k1=1.5, b=0.75
            )
            self.assertAlmostEqual(hit.score, expected_score, places=5)

        # Doc 1 has both 'python' and 'error', should be ranked #1
        self.assertEqual(result.hits[0].doc_id, 1)

    def test_equal_scores_sort_by_doc_id_asc(self):
        # Doc 20 and Doc 10 have identical content
        corpus = [
            (20, "nullpointerexception java error"),
            (10, "nullpointerexception java error"),
        ]
        self._build_test_index(corpus)
        engine = Engine.load(self.index_dir)

        result = engine.search("nullpointerexception", top_k=10)
        self.assertEqual(len(result.hits), 2)
        self.assertAlmostEqual(result.hits[0].score, result.hits[1].score, places=6)
        self.assertEqual(result.hits[0].doc_id, 10)
        self.assertEqual(result.hits[1].doc_id, 20)

    def test_top_k_clamping(self):
        corpus = [(i, f"keyword token_{i}") for i in range(1, 60)]
        self._build_test_index(corpus)
        engine = Engine.load(self.index_dir)

        # Clamped to 1 when <= 0
        res_zero = engine.search("keyword", top_k=0)
        self.assertEqual(len(res_zero.hits), 1)

        res_neg = engine.search("keyword", top_k=-10)
        self.assertEqual(len(res_neg.hits), 1)

        # Clamped to 50 when > 50
        res_large = engine.search("keyword", top_k=100)
        self.assertEqual(len(res_large.hits), 50)
        self.assertEqual(res_large.total_hits, 59)

        # In-between
        res_norm = engine.search("keyword", top_k=5)
        self.assertEqual(len(res_norm.hits), 5)

    def test_unknown_term_and_empty_query(self):
        corpus = [
            (1, "python flask web"),
        ]
        self._build_test_index(corpus)
        engine = Engine.load(self.index_dir)

        # Empty query
        res_empty = engine.search("")
        self.assertEqual(res_empty.total_hits, 0)
        self.assertEqual(len(res_empty.hits), 0)
        self.assertEqual(res_empty.tokenized_query, [])

        res_spaces = engine.search("   ")
        self.assertEqual(res_spaces.total_hits, 0)
        self.assertEqual(len(res_spaces.hits), 0)

        # Term outside vocabulary -> 0 hits
        res_unknown = engine.search("nonexistentterm123")
        self.assertEqual(res_unknown.total_hits, 0)
        self.assertEqual(len(res_unknown.hits), 0)

        # Mixed query: one known term, one unknown term
        res_mixed = engine.search("python nonexistentterm123")
        self.assertEqual(res_mixed.total_hits, 1)
        self.assertEqual(res_mixed.hits[0].doc_id, 1)

    def test_tag_and_language_filter(self):
        corpus = [
            (1, "connection timeout database"),
            (2, "connection timeout database"),
            (3, "connection timeout database"),
        ]
        nodes = [
            GraphNode(1, "T1", "S1", True, ["python", "django"], 5, 5, "", 0, 0),
            GraphNode(2, "T2", "S2", True, ["python", "fastapi"], 5, 5, "", 0, 0),
            GraphNode(3, "T3", "S3", True, ["java", "spring-boot"], 5, 5, "", 0, 0),
        ]
        self._build_test_index(corpus, nodes=nodes)
        engine = Engine.load(self.index_dir)

        # Filter tags (case-insensitive, any match)
        res_django = engine.search("timeout", tags=["DJANGO"])
        self.assertEqual(res_django.total_hits, 1)
        self.assertEqual(res_django.hits[0].doc_id, 1)

        res_multi_tags = engine.search("timeout", tags=["fastapi", "spring-boot"])
        self.assertEqual(res_multi_tags.total_hits, 2)
        doc_ids = {h.doc_id for h in res_multi_tags.hits}
        self.assertEqual(doc_ids, {2, 3})

        # Language filter (must contain tag == language.lower())
        res_java = engine.search("timeout", language="Java")
        self.assertEqual(res_java.total_hits, 1)
        self.assertEqual(res_java.hits[0].doc_id, 3)

        # Combining tags and language
        res_comb = engine.search("timeout", tags=["django"], language="python")
        self.assertEqual(res_comb.total_hits, 1)
        self.assertEqual(res_comb.hits[0].doc_id, 1)

        res_comb_none = engine.search("timeout", tags=["django"], language="java")
        self.assertEqual(res_comb_none.total_hits, 0)

    def test_document_missing_in_graph_db_skipped(self):
        corpus = [
            (1, "memory leak exception"),
            (2, "memory leak exception"),
        ]
        # Only node 1 is in graph.db
        nodes = [
            GraphNode(1, "Title 1", "Snippet 1", True, ["python"], 5, 5, "", 0, 0),
        ]
        self._build_test_index(corpus, nodes=nodes)
        engine = Engine.load(self.index_dir)

        res = engine.search("memory leak")
        self.assertEqual(res.total_hits, 1)
        self.assertEqual(len(res.hits), 1)
        self.assertEqual(res.hits[0].doc_id, 1)

    def test_load_validation_and_failures(self):
        corpus = [(1, "sample text")]
        self._build_test_index(corpus)

        # Missing metadata.json
        meta_file = os.path.join(self.index_dir, "metadata.json")
        os.remove(meta_file)
        with self.assertRaises((FileNotFoundError, ValueError)):
            Index.load(self.index_dir)

        # Invalid format_version
        with open(meta_file, "w", encoding="utf-8") as f:
            json.dump({"format_version": 2, "tokenizer_version": "1", "num_documents": 1, "avg_doc_length": 2.0}, f)
        with self.assertRaises(ValueError):
            Index.load(self.index_dir)

        # Invalid tokenizer_version
        with open(meta_file, "w", encoding="utf-8") as f:
            json.dump({"format_version": 1, "tokenizer_version": "2", "num_documents": 1, "avg_doc_length": 2.0}, f)
        with self.assertRaises(ValueError):
            Index.load(self.index_dir)

    def test_n_and_avgdl_from_metadata(self):
        corpus = [(1, "short text"), (2, "another text")]
        # Override metadata with dummy N and avgdl
        self._build_test_index(corpus, metadata_override={"num_documents": 1000, "avg_doc_length": 99.5})
        index = Index.load(self.index_dir)

        self.assertEqual(index.N, 1000)
        self.assertAlmostEqual(index.avgdl, 99.5)

    def test_tokenized_query_preserves_subtokens(self):
        corpus = [(1, "NullPointerException in java thread")]
        self._build_test_index(corpus)
        engine = Engine.load(self.index_dir)

        res = engine.search("NullPointerException")
        expected_subtokens = ["nullpointerexception", "null", "pointer", "exception"]
        for st in expected_subtokens:
            self.assertIn(st, res.tokenized_query)

    def test_thread_store_neighbors(self):
        nodes = [
            GraphNode(1, "N1", "S1", True, ["go"], 0, 0, "", 0, 0),
            GraphNode(2, "N2", "S2", True, ["go"], 0, 0, "", 0, 0),
            GraphNode(3, "N3", "S3", True, ["go"], 0, 0, "", 0, 0),
            GraphNode(4, "N4", "S4", True, ["go"], 0, 0, "", 0, 0),
        ]
        # Links:
        # 1 -> 2 (LINKED_IN_ANSWER) (outgoing from 1)
        # 1 -> 3 (DUPLICATE_OF) (outgoing from 1)
        # 4 -> 1 (DUPLICATE_OF) (incoming duplicate to 1)
        # 4 -> 2 (LINKED_IN_ANSWER) (unrelated to 1)
        links = [
            (1, 2, "LINKED_IN_ANSWER", 1.0),
            (1, 3, "DUPLICATE_OF", 1.0),
            (4, 1, "DUPLICATE_OF", 1.0),
            (4, 2, "LINKED_IN_ANSWER", 1.0),
        ]
        db_path = os.path.join(self.index_dir, "graph.db")
        write_graph_db(db_path, nodes, links)

        store = ThreadStore(db_path)
        neighbors = store.neighbors([1])

        # Neighbors should contain outgoing 1->2, 1->3 and incoming DUPLICATE_OF 4->1
        # It must NOT contain 4->2
        edge_tuples = [(e["source_doc_id"], e["target_doc_id"], e["link_type"]) for e in neighbors]
        self.assertIn((1, 2, "LINKED_IN_ANSWER"), edge_tuples)
        self.assertIn((1, 3, "DUPLICATE_OF"), edge_tuples)
        self.assertIn((4, 1, "DUPLICATE_OF"), edge_tuples)
        self.assertNotIn((4, 2, "LINKED_IN_ANSWER"), edge_tuples)
        self.assertEqual(len(neighbors), 3)

        # Test details
        detail1 = store.details(1)
        self.assertIsNotNone(detail1)
        self.assertEqual(detail1["title"], "N1")
        self.assertEqual(detail1["tags"], ["go"])

        detail_missing = store.details(999)
        self.assertIsNone(detail_missing)

    def test_engine_stats(self):
        corpus = [
            (1, "python asyncio coroutine"),
            (2, "python threading worker"),
        ]
        self._build_test_index(corpus)
        engine = Engine.load(self.index_dir)

        stats = engine.stats()
        self.assertEqual(stats.N, 2)
        self.assertEqual(stats.total_documents, 2)
        self.assertGreater(stats.avgdl, 0.0)
        self.assertGreater(stats.vocabulary_size, 0)
        self.assertGreater(stats.index_size_bytes, 0)
        self.assertTrue(stats.is_ready)
        self.assertGreater(stats.build_timestamp, 0)


if __name__ == "__main__":
    unittest.main()
