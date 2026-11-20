"""Unit tests for gRPC GraphService in search_engine_core."""
import concurrent.futures
import json
import os
import sqlite3
import sys
import tempfile
import unittest
from unittest.mock import MagicMock

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
import graph_pb2
import graph_pb2_grpc

from indexer.graph_db import GraphNode, write_graph_db
from indexer.spimi import build_index
from indexer.tokenizer import tokenize

from core.config import CoreConfig
from core.engine import Engine, ThreadStore
from core.graph_servicer import GraphServicer
from core.search_servicer import SearchServicer
from core.server import create_server


def create_graph_node(
    doc_id: int,
    title: str = "",
    snippet: str = "",
    tags=None,
    question_score: int = 10,
    best_answer_score: int = 5,
    best_answer_body: str = "<p>Answer body</p>",
    has_accepted_answer: bool = True,
    accepted_answer_id: int = 100,
    created_at: int = 1700000000,
) -> GraphNode:
    return GraphNode(
        doc_id=doc_id,
        title=title or f"Question title {doc_id}",
        snippet=snippet or f"Snippet for question {doc_id}",
        has_accepted_answer=has_accepted_answer,
        tags=tags if tags is not None else ["python", "graph"],
        question_score=question_score,
        best_answer_score=best_answer_score,
        best_answer_body=best_answer_body,
        accepted_answer_id=accepted_answer_id,
        created_at=created_at + doc_id,
    )


class TestCoreGraphGRPC(unittest.TestCase):
    def setUp(self):
        self.tmp_dir = tempfile.TemporaryDirectory()
        self.index_dir = self.tmp_dir.name

        # Create basic index files so Engine.load can succeed
        corpus = [(1, "python thread graph"), (2, "another thread topic")]
        tokenized_docs = [(doc_id, tokenize(text)) for doc_id, text in corpus]
        meta = build_index(tokenized_docs, self.index_dir)
        meta["build_timestamp"] = 1700000000
        with open(os.path.join(self.index_dir, "metadata.json"), "w", encoding="utf-8") as f:
            json.dump(meta, f, indent=2)

        # Base nodes
        # doc 1: primary query candidate
        # doc 2: other query candidate
        # doc 3: linked in answer neighbor from 1
        # doc 4: duplicate_of neighbor from 1 (outgoing)
        # doc 5: incoming duplicate_of neighbor to 1 (5 -> 1 DUPLICATE_OF)
        # doc 6: neighbor of 2
        # doc 7: unlinked node
        nodes = [
            create_graph_node(1, title="Primary Question 1", question_score=50, best_answer_score=40, tags=["python", "django"]),
            create_graph_node(2, title="Other Question 2", question_score=30, best_answer_score=20, tags=["python", "flask"]),
            create_graph_node(3, title="Linked Answer Question 3", question_score=15, best_answer_score=10),
            create_graph_node(4, title="Duplicate Question 4", question_score=25, best_answer_score=12),
            create_graph_node(5, title="Incoming Duplicate Question 5", question_score=18, best_answer_score=8),
            create_graph_node(6, title="Linked to Two 6", question_score=5, best_answer_score=2),
            create_graph_node(7, title="Unlinked Question 7", question_score=2, best_answer_score=1),
        ]

        # Edges (src, dst, link_type, weight)
        edges = [
            (1, 3, "LINKED_IN_ANSWER", 1.5),
            (1, 4, "DUPLICATE_OF", 2.5),
            (5, 1, "DUPLICATE_OF", 3.0),   # Incoming DUPLICATE_OF to 1
            (1, 2, "LINKED_IN_ANSWER", 0.5), # Edge between query nodes
            (2, 6, "LINKED_IN_ANSWER", 1.0),
        ]

        self.db_path = os.path.join(self.index_dir, "graph.db")
        write_graph_db(self.db_path, nodes, edges)

        self.config = CoreConfig(index_dir=self.index_dir, bm25_k1=1.5, bm25_b=0.75, grpc_port=50051)
        self.engine = Engine.load(self.index_dir, config=self.config)
        self.search_servicer = SearchServicer(engine=self.engine, index_dir=self.index_dir)
        self.graph_servicer = GraphServicer(search_servicer=self.search_servicer)

        self.server, _ = create_server(
            config=self.config,
            servicer=self.search_servicer,
            graph_servicer=self.graph_servicer,
        )
        port = self.server.add_insecure_port("127.0.0.1:0")
        self.server.start()

        self.channel = grpc.insecure_channel(f"127.0.0.1:{port}")
        self.stub = graph_pb2_grpc.GraphServiceStub(self.channel)
        self.server_port = port

    def tearDown(self):
        self.channel.close()
        self.server.stop(grace=None)
        self.tmp_dir.cleanup()

    def test_get_thread_graph_validation_empty_doc_ids(self):
        req = graph_pb2.ThreadGraphRequest(doc_ids=[])
        with self.assertRaises(grpc.RpcError) as ctx:
            self.stub.GetThreadGraph(req)
        self.assertEqual(ctx.exception.code(), grpc.StatusCode.INVALID_ARGUMENT)

    def test_get_thread_graph_validation_too_many_doc_ids(self):
        req = graph_pb2.ThreadGraphRequest(doc_ids=list(range(1, 22))) # 21 items
        with self.assertRaises(grpc.RpcError) as ctx:
            self.stub.GetThreadGraph(req)
        self.assertEqual(ctx.exception.code(), grpc.StatusCode.INVALID_ARGUMENT)

    def test_get_thread_graph_unready_index(self):
        empty_graph_servicer = GraphServicer(engine=None, search_servicer=None, thread_store=None)
        server = grpc.server(concurrent.futures.ThreadPoolExecutor(max_workers=2))
        graph_pb2_grpc.add_GraphServiceServicer_to_server(empty_graph_servicer, server)
        port = server.add_insecure_port("127.0.0.1:0")
        server.start()
        chan = grpc.insecure_channel(f"127.0.0.1:{port}")
        stub = graph_pb2_grpc.GraphServiceStub(chan)
        try:
            req = graph_pb2.ThreadGraphRequest(doc_ids=[1])
            with self.assertRaises(grpc.RpcError) as ctx:
                stub.GetThreadGraph(req)
            self.assertEqual(ctx.exception.code(), grpc.StatusCode.FAILED_PRECONDITION)
        finally:
            chan.close()
            server.stop(grace=None)

    def test_get_thread_graph_categories_and_incoming_duplicates(self):
        # Query with doc_ids=[1, 2]
        # Query nodes: 1 (PRIMARY_RESULT), 2 (OTHER_RESULT)
        # Neighbors of 1:
        #   3 (LINKED_IN_ANSWER from 1) -> LINKED
        #   4 (DUPLICATE_OF from 1) -> DUPLICATE
        #   5 (DUPLICATE_OF from 5 to 1) -> incoming duplicate -> DUPLICATE
        # Neighbors of 2:
        #   1 is already query node -> excluded from neighbors
        #   6 (LINKED_IN_ANSWER from 2) -> LINKED
        req = graph_pb2.ThreadGraphRequest(doc_ids=[1, 2])
        resp = self.stub.GetThreadGraph(req)

        nodes_by_id = {node.doc_id: node for node in resp.nodes}
        self.assertIn(1, nodes_by_id)
        self.assertIn(2, nodes_by_id)
        self.assertIn(3, nodes_by_id)
        self.assertIn(4, nodes_by_id)
        self.assertIn(5, nodes_by_id)
        self.assertIn(6, nodes_by_id)
        self.assertNotIn(7, nodes_by_id)

        # Check categories
        self.assertEqual(nodes_by_id[1].category, graph_pb2.NODE_CATEGORY_PRIMARY_RESULT)
        self.assertEqual(nodes_by_id[2].category, graph_pb2.NODE_CATEGORY_OTHER_RESULT)
        self.assertEqual(nodes_by_id[3].category, graph_pb2.NODE_CATEGORY_LINKED)
        self.assertEqual(nodes_by_id[4].category, graph_pb2.NODE_CATEGORY_DUPLICATE)
        self.assertEqual(nodes_by_id[5].category, graph_pb2.NODE_CATEGORY_DUPLICATE)
        self.assertEqual(nodes_by_id[6].category, graph_pb2.NODE_CATEGORY_LINKED)

        # Check fields of ThreadNode
        node1 = nodes_by_id[1]
        self.assertEqual(node1.score, 50.0) # question_score as double
        self.assertEqual(node1.title, "Primary Question 1")
        self.assertTrue(len(node1.snippet) > 0)
        self.assertEqual(list(node1.tags), ["python", "django"])
        self.assertTrue(node1.has_accepted_answer)
        self.assertEqual(node1.best_answer_score, 40)

        # Check edges
        edge_pairs = {(e.source_doc_id, e.target_doc_id, e.type) for e in resp.edges}
        self.assertIn((1, 3, graph_pb2.EDGE_TYPE_LINKED_IN_ANSWER), edge_pairs)
        self.assertIn((1, 4, graph_pb2.EDGE_TYPE_DUPLICATE_OF), edge_pairs)
        self.assertIn((5, 1, graph_pb2.EDGE_TYPE_DUPLICATE_OF), edge_pairs)
        self.assertIn((1, 2, graph_pb2.EDGE_TYPE_LINKED_IN_ANSWER), edge_pairs)
        self.assertIn((2, 6, graph_pb2.EDGE_TYPE_LINKED_IN_ANSWER), edge_pairs)

        # No self loops
        for e in resp.edges:
            self.assertNotEqual(e.source_doc_id, e.target_doc_id)

    def test_get_thread_graph_neighbor_limit_20_and_sorting(self):
        # Build a graph with 25 neighbors for query node 100
        tmp = tempfile.TemporaryDirectory()
        try:
            db_path = os.path.join(tmp.name, "graph.db")
            nodes = [create_graph_node(100, title="Root Query", question_score=100)]
            edges = []

            # Add 25 neighbors with doc_ids 201..225
            # Weights assigned so sorting by weight DESC, doc_id ASC can be verified
            # Give doc 201..210 weight 5.0, docs 211..225 weight 2.0
            for i in range(201, 226):
                nodes.append(create_graph_node(i, title=f"Neighbor {i}", question_score=i))
                weight = 5.0 if i <= 210 else 2.0
                edges.append((100, i, "LINKED_IN_ANSWER", weight))

            # Also add a self-loop on node 100 in sqlite directly to test "no self loops"
            write_graph_db(db_path, nodes, edges)
            with sqlite3.connect(db_path) as conn:
                conn.execute("INSERT INTO thread_links VALUES (100, 100, 'LINKED_IN_ANSWER', 10.0)")
                conn.commit()

            store = ThreadStore(db_path)
            servicer = GraphServicer(thread_store=store)

            req = graph_pb2.ThreadGraphRequest(doc_ids=[100])
            context = MagicMock()
            resp = servicer.GetThreadGraph(req, context)

            # Node 100 is query node (PRIMARY), neighbors must be exactly 20
            query_nodes = [n for n in resp.nodes if n.category == graph_pb2.NODE_CATEGORY_PRIMARY_RESULT]
            neighbor_nodes = [n for n in resp.nodes if n.category in (graph_pb2.NODE_CATEGORY_LINKED, graph_pb2.NODE_CATEGORY_DUPLICATE)]

            self.assertEqual(len(query_nodes), 1)
            self.assertEqual(len(neighbor_nodes), 20)
            self.assertEqual(len(resp.nodes), 21)

            # Verify sorting: weight DESC, doc_id ASC
            # Top 10 have weight 5.0 (doc_ids 201..210)
            # Next 10 have weight 2.0 (doc_ids 211..220)
            neighbor_ids = [n.doc_id for n in neighbor_nodes]
            expected_ids = list(range(201, 221)) # 201 to 220
            self.assertEqual(neighbor_ids, expected_ids)

            # Check edges: no self loops
            for e in resp.edges:
                self.assertNotEqual(e.source_doc_id, e.target_doc_id)
        finally:
            tmp.cleanup()

    def test_get_thread_graph_missing_nodes_skipped(self):
        # Query with [1, 9999] where 9999 does not exist in thread_nodes
        req = graph_pb2.ThreadGraphRequest(doc_ids=[1, 9999])
        resp = self.stub.GetThreadGraph(req)

        node_ids = {n.doc_id for n in resp.nodes}
        self.assertIn(1, node_ids)
        self.assertNotIn(9999, node_ids)

    def test_get_thread_details_success(self):
        req = graph_pb2.ThreadDetailsRequest(doc_id=1)
        resp = self.stub.GetThreadDetails(req)

        self.assertEqual(resp.doc_id, 1)
        self.assertEqual(resp.title, "Primary Question 1")
        self.assertEqual(resp.best_answer_body, "<p>Answer body</p>")
        self.assertEqual(resp.original_url, "https://stackoverflow.com/questions/1")

    def test_get_thread_details_not_found(self):
        req = graph_pb2.ThreadDetailsRequest(doc_id=99999)
        with self.assertRaises(grpc.RpcError) as ctx:
            self.stub.GetThreadDetails(req)
        self.assertEqual(ctx.exception.code(), grpc.StatusCode.NOT_FOUND)

    def test_get_thread_details_unready_index(self):
        empty_graph_servicer = GraphServicer(engine=None, search_servicer=None, thread_store=None)
        server = grpc.server(__import__('concurrent.futures').futures.ThreadPoolExecutor(max_workers=2))
        graph_pb2_grpc.add_GraphServiceServicer_to_server(empty_graph_servicer, server)
        port = server.add_insecure_port("127.0.0.1:0")
        server.start()
        chan = grpc.insecure_channel(f"127.0.0.1:{port}")
        stub = graph_pb2_grpc.GraphServiceStub(chan)
        try:
            req = graph_pb2.ThreadDetailsRequest(doc_id=1)
            with self.assertRaises(grpc.RpcError) as ctx:
                stub.GetThreadDetails(req)
            self.assertEqual(ctx.exception.code(), grpc.StatusCode.FAILED_PRECONDITION)
        finally:
            chan.close()
            server.stop(grace=None)


if __name__ == "__main__":
    unittest.main()
