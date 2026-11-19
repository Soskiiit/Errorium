import concurrent.futures
import json
import os
import shutil
import sys
import tempfile
import unittest

PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
VENV_PACKAGES = os.path.join(PROJECT_ROOT, ".venv/lib/python3.14/site-packages")
if os.path.isdir(VENV_PACKAGES) and VENV_PACKAGES not in sys.path:
    sys.path.insert(0, VENV_PACKAGES)

CORE_SRC = os.path.join(PROJECT_ROOT, "services/search_engine_core/src")
INDEXER_SRC = os.path.join(PROJECT_ROOT, "services/search_engine_indexer/src")
if CORE_SRC not in sys.path:
    sys.path.insert(0, CORE_SRC)
if INDEXER_SRC not in sys.path:
    sys.path.insert(0, INDEXER_SRC)

import grpc
import search_pb2
import search_pb2_grpc

from indexer.graph_db import GraphNode, write_graph_db
from indexer.spimi import build_index
from indexer.tokenizer import tokenize

from core.config import CoreConfig
from core.engine import Engine
from core.search_servicer import SearchServicer
from core.server import create_server
from core.smoke import run_smoke


def build_fixture_index(index_dir: str, corpus, nodes=None, build_ts=1700000000):
    tokenized_docs = [(doc_id, tokenize(text)) for doc_id, text in corpus]
    meta = build_index(tokenized_docs, index_dir)
    meta["build_timestamp"] = build_ts
    with open(os.path.join(index_dir, "metadata.json"), "w", encoding="utf-8") as f:
        json.dump(meta, f, indent=2)

    if nodes is None:
        nodes = [
            GraphNode(
                doc_id=doc_id,
                title=f"Question {doc_id}",
                snippet=f"Snippet for doc {doc_id} text",
                has_accepted_answer=True,
                tags=["python", "indexerror"],
                question_score=15,
                best_answer_score=20,
                best_answer_body="<p>Try checking your index bounds</p>",
                accepted_answer_id=doc_id * 10,
                created_at=build_ts + doc_id,
            )
            for doc_id, _ in corpus
        ]

    write_graph_db(os.path.join(index_dir, "graph.db"), nodes, [])
    return meta


class TestCoreGRPC(unittest.TestCase):
    def setUp(self):
        self.tmp_dir = tempfile.TemporaryDirectory()
        self.index_dir = self.tmp_dir.name

        self.corpus = [
            (1, "python IndexError list index out of range"),
            (2, "python KeyError dictionary key not found"),
            (3, "python IndexError tuple index out of range"),
        ]
        build_fixture_index(self.index_dir, self.corpus)

        self.config = CoreConfig(index_dir=self.index_dir, bm25_k1=1.5, bm25_b=0.75, grpc_port=50051)
        self.engine = Engine.load(self.index_dir, config=self.config)
        self.servicer = SearchServicer(engine=self.engine, index_dir=self.index_dir)

        self.server, _ = create_server(config=self.config, servicer=self.servicer)
        port = self.server.add_insecure_port("127.0.0.1:0")
        self.server.start()

        self.channel = grpc.insecure_channel(f"127.0.0.1:{port}")
        self.stub = search_pb2_grpc.SearchServiceStub(self.channel)
        self.server_port = port

    def tearDown(self):
        self.channel.close()
        self.server.stop(grace=None)
        self.tmp_dir.cleanup()

    def test_health_check_serving(self):
        resp = self.stub.HealthCheck(search_pb2.HealthCheckRequest())
        self.assertEqual(resp.status, search_pb2.HealthCheckResponse.ServingStatus.SERVING)
        self.assertEqual(resp.version, "1.0.0")

    def test_health_check_not_serving_when_unready(self):
        empty_servicer = SearchServicer(engine=None, index_dir=self.index_dir)
        server, _ = create_server(servicer=empty_servicer)
        port = server.add_insecure_port("127.0.0.1:0")
        server.start()
        chan = grpc.insecure_channel(f"127.0.0.1:{port}")
        stub = search_pb2_grpc.SearchServiceStub(chan)
        try:
            resp = stub.HealthCheck(search_pb2.HealthCheckRequest())
            self.assertEqual(resp.status, search_pb2.HealthCheckResponse.ServingStatus.NOT_SERVING)
            self.assertEqual(resp.version, "1.0.0")
        finally:
            chan.close()
            server.stop(grace=None)

    def test_search_success(self):
        req = search_pb2.SearchRequest(query="python IndexError", top_k=10, include_raw_body=False)
        resp = self.stub.Search(req)

        self.assertEqual(resp.total_hits, 3)
        self.assertEqual(len(resp.hits), 3)
        self.assertGreater(resp.latency_ms, 0.0)
        self.assertIn("python", resp.tokenized_query)
        self.assertIn("indexerror", resp.tokenized_query)

        hit = resp.hits[0]
        self.assertIn(hit.doc_id, (1, 3))
        self.assertGreater(hit.score, 0.0)
        self.assertTrue(hit.title.startswith("Question"))
        self.assertIn("Snippet for doc", hit.snippet)
        self.assertEqual(hit.raw_body, "")
        self.assertEqual(hit.tags, ["python", "indexerror"])
        self.assertEqual(hit.question_score, 15)
        self.assertEqual(hit.best_answer_score, 20)
        self.assertEqual(hit.best_answer_body, "<p>Try checking your index bounds</p>")

    def test_search_include_raw_body(self):
        req = search_pb2.SearchRequest(query="python IndexError", top_k=1, include_raw_body=True)
        resp = self.stub.Search(req)
        self.assertGreater(len(resp.hits), 0)
        hit = resp.hits[0]
        self.assertEqual(hit.raw_body, hit.snippet)

    def test_search_empty_query_raises_invalid_argument(self):
        req = search_pb2.SearchRequest(query="", top_k=10)
        with self.assertRaises(grpc.RpcError) as ctx:
            self.stub.Search(req)
        self.assertEqual(ctx.exception.code(), grpc.StatusCode.INVALID_ARGUMENT)

        req_spaces = search_pb2.SearchRequest(query="   ", top_k=10)
        with self.assertRaises(grpc.RpcError) as ctx:
            self.stub.Search(req_spaces)
        self.assertEqual(ctx.exception.code(), grpc.StatusCode.INVALID_ARGUMENT)

    def test_search_unready_index_raises_failed_precondition(self):
        empty_servicer = SearchServicer(engine=None, index_dir=self.index_dir)
        server, _ = create_server(servicer=empty_servicer)
        port = server.add_insecure_port("127.0.0.1:0")
        server.start()
        chan = grpc.insecure_channel(f"127.0.0.1:{port}")
        stub = search_pb2_grpc.SearchServiceStub(chan)
        try:
            req = search_pb2.SearchRequest(query="python", top_k=10)
            with self.assertRaises(grpc.RpcError) as ctx:
                stub.Search(req)
            self.assertEqual(ctx.exception.code(), grpc.StatusCode.FAILED_PRECONDITION)
            self.assertIn("index not ready", ctx.exception.details())
        finally:
            chan.close()
            server.stop(grace=None)

    def test_get_document_success(self):
        resp = self.stub.GetDocument(search_pb2.GetDocumentRequest(doc_id=1))
        doc = resp.document
        self.assertEqual(doc.doc_id, 1)
        self.assertEqual(doc.score, 0.0)
        self.assertEqual(doc.title, "Question 1")
        self.assertEqual(doc.question_score, 15)

    def test_get_document_not_found(self):
        with self.assertRaises(grpc.RpcError) as ctx:
            self.stub.GetDocument(search_pb2.GetDocumentRequest(doc_id=99999))
        self.assertEqual(ctx.exception.code(), grpc.StatusCode.NOT_FOUND)

    def test_batch_get_documents(self):
        resp = self.stub.BatchGetDocuments(search_pb2.BatchGetDocumentsRequest(doc_ids=[1, 99999, 2]))
        docs = resp.documents
        self.assertEqual(len(docs), 2)
        doc_ids = {d.doc_id for d in docs}
        self.assertEqual(doc_ids, {1, 2})
        for d in docs:
            self.assertEqual(d.score, 0.0)

    def test_get_stats(self):
        resp = self.stub.GetStats(search_pb2.GetStatsRequest())
        self.assertEqual(resp.total_documents, 3)
        self.assertGreater(resp.avg_doc_length, 0.0)
        self.assertGreater(resp.vocabulary_size, 0)
        self.assertGreater(resp.index_size_bytes, 0)
        self.assertTrue(resp.is_ready)
        self.assertEqual(resp.build_timestamp, 1700000000)

    def test_reload_index_success(self):
        new_tmp = tempfile.TemporaryDirectory()
        try:
            new_corpus = [
                (10, "golang runtime error nil pointer dereference"),
                (20, "golang panic index out of range"),
            ]
            new_meta = build_fixture_index(new_tmp.name, new_corpus, build_ts=1750000000)

            # Copy new index files into servicer's index_dir
            for f in os.listdir(new_tmp.name):
                src = os.path.join(new_tmp.name, f)
                dst = os.path.join(self.index_dir, f)
                if os.path.isfile(src):
                    shutil.copy2(src, dst)

            resp = self.stub.ReloadIndex(search_pb2.ReloadIndexRequest())
            self.assertTrue(resp.ok)
            self.assertEqual(resp.message, "")
            self.assertEqual(resp.build_timestamp, 1750000000)

            # Search with new index
            s_resp = self.stub.Search(search_pb2.SearchRequest(query="golang pointer", top_k=10))
            self.assertEqual(s_resp.total_hits, 2)
            self.assertEqual(s_resp.hits[0].doc_id, 10)
        finally:
            new_tmp.cleanup()

    def test_reload_index_failure_keeps_old_engine(self):
        # Corrupt metadata.json
        meta_path = os.path.join(self.index_dir, "metadata.json")
        with open(meta_path, "w", encoding="utf-8") as f:
            f.write("corrupted json content")

        resp = self.stub.ReloadIndex(search_pb2.ReloadIndexRequest())
        self.assertFalse(resp.ok)
        self.assertTrue(len(resp.message) > 0)

        # Old engine still serves requests!
        s_resp = self.stub.Search(search_pb2.SearchRequest(query="python IndexError", top_k=10))
        self.assertEqual(s_resp.total_hits, 3)

    def test_smoke_script(self):
        addr = f"127.0.0.1:{self.server_port}"
        ok = run_smoke(addr)
        self.assertTrue(ok)


if __name__ == "__main__":
    unittest.main()
