"""Интеграционные тесты gRPC-канала ядра: реальный сервер на случайном порту,
клиентские стабы SearchService и GraphService (docs/Tech/05-verification.md, п. 2).

На хосте без grpcio/pb2 тесты пропускаются; полный прогон — `make test-core`
(в контейнере ядра, где pb2 генерируются при сборке образа).
"""
import math
import os
import sys
import tempfile
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
UNIT_DIR = os.path.join(REPO_ROOT, "tests", "unit")
if UNIT_DIR not in sys.path:
    sys.path.insert(0, UNIT_DIR)

from _core_fixture import build_index, fmt  # noqa: E402
from indexer.tokenizer import tokenize  # noqa: E402

try:
    import grpc
    from core.server import build_server
    from core.engine import SearchEngine
    from core.reader import IndexReader
    import search_pb2
    import search_pb2_grpc
    import graph_pb2
    import graph_pb2_grpc
    GRPC_AVAILABLE = True
except ImportError:
    GRPC_AVAILABLE = False

DOCS = [
    {"doc_id": 1, "title": "Возврат оплаты", "text": "Возврат оплаты на карту",
     "tags": ["payments"], "question_score": 10, "best_answer_body": "<p>на карту</p>",
     "best_answer_score": 7, "accepted_answer_id": 101, "created_at": 1700000000},
    {"doc_id": 2, "title": "Статус оплаты", "text": "Статус оплаты через приложение",
     "tags": ["payments", "mobile"], "question_score": 5},
    {"doc_id": 3, "title": "Возврат билета", "text": "Возврат билета через кассу",
     "tags": ["tickets"], "question_score": 3},
]
EDGES = [(1, 3, "DUPLICATE_OF"), (2, 3, "LINKED_IN_ANSWER")]

QUERY = "Как происходит возврат оплаты?"


def reference_bm25(index_dir, query, k=10, k1=1.2, b=0.75):
    """Независимый эталон по файлам индекса (формула методички, занятие 3)."""
    import json
    lexicon = fmt.read_lexicon(os.path.join(index_dir, "lexicon.bin"))
    lengths = fmt.read_doc_lengths(os.path.join(index_dir, "doc_lengths.bin"))
    with open(os.path.join(index_dir, "metadata.json"), encoding="utf-8") as f:
        meta = json.load(f)
    n, avgdl = meta["num_documents"], meta["avg_doc_length"]
    scores = {}
    for term in dict.fromkeys(tokenize(query)):
        entry = lexicon.get(term)
        if entry is None:
            continue
        idf = math.log(1 + (n - entry.df + 0.5) / (entry.df + 0.5))
        for doc_id, tf in fmt.read_postings(os.path.join(index_dir, "postings.bin"), entry):
            norm = k1 * (1 - b + b * lengths[doc_id] / avgdl)
            scores[doc_id] = scores.get(doc_id, 0.0) + idf * tf * (k1 + 1) / (tf + norm)
    ranked = sorted(scores, key=lambda d: (-scores[d], d))[:k]
    return [(d, scores[d]) for d in ranked]


@unittest.skipUnless(GRPC_AVAILABLE, "grpcio/pb2 недоступны на хосте (запустите make test-core)")
class TestGrpcSearch(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls._tmp = tempfile.TemporaryDirectory()
        cls.index_dir = build_index(cls._tmp.name, DOCS, edges=EDGES)
        cls.engine = SearchEngine(IndexReader(cls.index_dir), k1=1.2, b=0.75)
        cls.server, port = build_server(cls.engine, 0)
        cls.channel = grpc.insecure_channel(f"localhost:{port}")
        cls.search = search_pb2_grpc.SearchServiceStub(cls.channel)
        cls.graph = graph_pb2_grpc.GraphServiceStub(cls.channel)

    @classmethod
    def tearDownClass(cls):
        cls.channel.close()
        cls.server.stop(grace=None).wait()
        cls.engine.reader.close()
        cls._tmp.cleanup()

    def test_search_matches_reference_bm25(self):
        response = self.search.Search(search_pb2.SearchRequest(query=QUERY, top_k=10))
        expected = reference_bm25(self.index_dir, QUERY)
        self.assertEqual([h.doc_id for h in response.hits], [d for d, _ in expected])
        for hit, (_, score) in zip(response.hits, expected):
            self.assertAlmostEqual(hit.score, score, places=9)
        self.assertEqual(response.total_hits, 3)
        self.assertEqual(list(response.tokenized_query), tokenize(QUERY))
        self.assertGreaterEqual(response.latency_ms, 0.0)

    def test_search_hit_fields_from_graph_db(self):
        response = self.search.Search(search_pb2.SearchRequest(query=QUERY, top_k=1))
        hit = response.hits[0]
        self.assertEqual(hit.doc_id, 1)
        self.assertEqual(hit.title, "Возврат оплаты")
        self.assertEqual(list(hit.tags), ["payments"])
        self.assertEqual(hit.question_score, 10)
        self.assertEqual(hit.accepted_answer_id, 101)
        self.assertEqual(hit.best_answer_body, "<p>на карту</p>")
        self.assertEqual(hit.best_answer_score, 7)
        self.assertEqual(hit.created_at, 1700000000)
        self.assertEqual(hit.raw_body, "")  # include_raw_body не задан

    def test_include_raw_body(self):
        response = self.search.Search(
            search_pb2.SearchRequest(query=QUERY, top_k=1, include_raw_body=True))
        self.assertTrue(response.hits[0].raw_body)

    def test_search_top_k_and_tags_filter(self):
        response = self.search.Search(search_pb2.SearchRequest(query=QUERY, top_k=2))
        self.assertEqual(len(response.hits), 2)
        self.assertEqual(response.total_hits, 3)
        response = self.search.Search(
            search_pb2.SearchRequest(query="возврат", tags=["tickets"]))
        self.assertEqual([h.doc_id for h in response.hits], [3])

    def test_search_absent_term(self):
        response = self.search.Search(search_pb2.SearchRequest(query="нет такого терма"))
        self.assertEqual(len(response.hits), 0)
        self.assertEqual(response.total_hits, 0)

    def test_get_document_and_not_found(self):
        response = self.search.GetDocument(search_pb2.GetDocumentRequest(doc_id=2))
        self.assertEqual(response.document.doc_id, 2)
        self.assertEqual(response.document.title, "Статус оплаты")
        self.assertEqual(response.document.score, 0.0)
        with self.assertRaises(grpc.RpcError) as ctx:
            self.search.GetDocument(search_pb2.GetDocumentRequest(doc_id=999))
        self.assertEqual(ctx.exception.code(), grpc.StatusCode.NOT_FOUND)

    def test_batch_get_documents_keeps_order_and_skips_missing(self):
        response = self.search.BatchGetDocuments(
            search_pb2.BatchGetDocumentsRequest(doc_ids=[3, 999, 1]))
        self.assertEqual([d.doc_id for d in response.documents], [3, 1])

    def test_get_stats(self):
        response = self.search.GetStats(search_pb2.GetStatsRequest())
        self.assertTrue(response.is_ready)
        self.assertEqual(response.total_documents, 3)
        self.assertGreater(response.avg_doc_length, 0.0)
        self.assertGreater(response.vocabulary_size, 0)
        self.assertGreater(response.index_size_bytes, 0)
        self.assertGreater(response.build_timestamp, 0)

    def test_health_check_serving(self):
        response = self.search.HealthCheck(search_pb2.HealthCheckRequest())
        self.assertEqual(response.status, search_pb2.HealthCheckResponse.SERVING)
        self.assertTrue(response.version)

    def test_thread_graph(self):
        response = self.graph.GetThreadGraph(graph_pb2.ThreadGraphRequest(doc_ids=[1, 2]))
        by_id = {n.doc_id: n for n in response.nodes}
        self.assertEqual(by_id[1].category, graph_pb2.NODE_CATEGORY_PRIMARY_RESULT)
        self.assertEqual(by_id[2].category, graph_pb2.NODE_CATEGORY_OTHER_RESULT)
        self.assertEqual(by_id[3].category, graph_pb2.NODE_CATEGORY_DUPLICATE)
        self.assertEqual(by_id[1].score, 10.0)
        edges = {(e.source_doc_id, e.target_doc_id, e.type) for e in response.edges}
        self.assertEqual(edges, {(1, 3, graph_pb2.EDGE_TYPE_DUPLICATE_OF),
                                 (2, 3, graph_pb2.EDGE_TYPE_LINKED_IN_ANSWER)})

    def test_thread_details_and_not_found(self):
        response = self.graph.GetThreadDetails(graph_pb2.ThreadDetailsRequest(doc_id=1))
        self.assertEqual(response.title, "Возврат оплаты")
        self.assertEqual(response.best_answer_body, "<p>на карту</p>")
        self.assertEqual(response.original_url, "https://stackoverflow.com/questions/1")
        with self.assertRaises(grpc.RpcError) as ctx:
            self.graph.GetThreadDetails(graph_pb2.ThreadDetailsRequest(doc_id=999))
        self.assertEqual(ctx.exception.code(), grpc.StatusCode.NOT_FOUND)


@unittest.skipUnless(GRPC_AVAILABLE, "grpcio/pb2 недоступны на хосте (запустите make test-core)")
class TestGrpcNotReady(unittest.TestCase):
    def test_search_failed_precondition_and_health_not_serving(self):
        with tempfile.TemporaryDirectory() as root:
            engine = SearchEngine(IndexReader(os.path.join(root, "missing")))
            server, port = build_server(engine, 0)
            try:
                with grpc.insecure_channel(f"localhost:{port}") as channel:
                    search = search_pb2_grpc.SearchServiceStub(channel)
                    with self.assertRaises(grpc.RpcError) as ctx:
                        search.Search(search_pb2.SearchRequest(query="anything"))
                    self.assertEqual(ctx.exception.code(), grpc.StatusCode.FAILED_PRECONDITION)
                    health = search.HealthCheck(search_pb2.HealthCheckRequest())
                    self.assertEqual(health.status, search_pb2.HealthCheckResponse.NOT_SERVING)
                    stats = search.GetStats(search_pb2.GetStatsRequest())
                    self.assertFalse(stats.is_ready)
            finally:
                server.stop(grace=None).wait()
                engine.reader.close()

    def test_rebuilding_index_status(self):
        with tempfile.TemporaryDirectory() as root:
            index_dir = os.path.join(root, "index")
            os.makedirs(os.path.join(index_dir, ".building"))
            engine = SearchEngine(IndexReader(index_dir))
            server, port = build_server(engine, 0)
            try:
                with grpc.insecure_channel(f"localhost:{port}") as channel:
                    search = search_pb2_grpc.SearchServiceStub(channel)
                    health = search.HealthCheck(search_pb2.HealthCheckRequest())
                    self.assertEqual(health.status, search_pb2.HealthCheckResponse.REBUILDING_INDEX)
            finally:
                server.stop(grace=None).wait()
                engine.reader.close()

    def test_lazy_reload_after_index_built(self):
        with tempfile.TemporaryDirectory() as root:
            engine = SearchEngine(IndexReader(os.path.join(root, "index")), k1=1.2, b=0.75)
            server, port = build_server(engine, 0)
            try:
                with grpc.insecure_channel(f"localhost:{port}") as channel:
                    search = search_pb2_grpc.SearchServiceStub(channel)
                    with self.assertRaises(grpc.RpcError):
                        search.Search(search_pb2.SearchRequest(query=QUERY))
                    build_index(root, DOCS)
                    response = search.Search(search_pb2.SearchRequest(query=QUERY))
                    self.assertEqual(response.hits[0].doc_id, 1)
                    health = search.HealthCheck(search_pb2.HealthCheckRequest())
                    self.assertEqual(health.status, search_pb2.HealthCheckResponse.SERVING)
            finally:
                server.stop(grace=None).wait()
                engine.reader.close()


if __name__ == "__main__":
    unittest.main()