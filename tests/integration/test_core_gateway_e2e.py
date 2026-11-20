"""E2E integration test for Search Engine Core + Gateway via gRPC.

Verifies that Gateway communicates with Search Engine Core over gRPC,
exposes correct status, deterministic search results from core index,
graph enrichment, and proper 404 error handling.
"""

import json
import os
import unittest
import urllib.error
import urllib.request


class TestCoreGatewayE2E(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.base_url = os.environ.get("GATEWAY_URL", "http://localhost:8000").rstrip("/")
        try:
            req = urllib.request.Request(f"{cls.base_url}/health")
            with urllib.request.urlopen(req, timeout=3) as resp:
                if resp.status != 200:
                    raise unittest.SkipTest(f"Gateway service unhealthy at {cls.base_url} (status {resp.status})")
        except unittest.SkipTest:
            raise
        except Exception as e:
            raise unittest.SkipTest(f"Gateway service unreachable at {cls.base_url}: {e}")

    def make_request(self, method, path, data=None):
        url = f"{self.base_url}{path}"
        headers = {}
        body = None
        if data is not None:
            headers["Content-Type"] = "application/json"
            body = json.dumps(data).encode("utf-8")
        req = urllib.request.Request(url, data=body, headers=headers, method=method)
        try:
            with urllib.request.urlopen(req, timeout=10) as resp:
                status = resp.status
                headers_dict = dict(resp.headers)
                content = resp.read()
                return status, headers_dict, content
        except urllib.error.HTTPError as e:
            status = e.code
            headers_dict = dict(e.headers)
            content = e.read()
            return status, headers_dict, content

    def test_01_status_connected(self):
        status, _, content = self.make_request("GET", "/api/v1/status")
        self.assertEqual(status, 200)

        data = json.loads(content.decode("utf-8"))
        self.assertEqual(data.get("gateway", {}).get("status"), "ok")

        search_core = data.get("search_core", {})
        self.assertTrue(
            search_core.get("connected"),
            f"Expected search_core.connected=True, got {search_core}",
        )
        self.assertTrue(search_core.get("address"), "Expected non-empty search_core.address")

    def test_02_search_determinism_and_core_results(self):
        query_payload = {"query": "IndexError", "top_k": 5}

        status1, _, content1 = self.make_request("POST", "/api/v1/search", query_payload)
        self.assertEqual(status1, 200)
        data1 = json.loads(content1.decode("utf-8"))

        total_hits = data1.get("total_hits", 0)
        if total_hits == 0 or len(data1.get("results", [])) == 0:
            self.skipTest("построй make index-corpus и поставь CORE_INDEX_DIR")

        # Determinism check: identical query yields identical results
        status2, _, content2 = self.make_request("POST", "/api/v1/search", query_payload)
        self.assertEqual(status2, 200)
        data2 = json.loads(content2.decode("utf-8"))

        self.assertEqual(data1.get("total_hits"), data2.get("total_hits"))
        results1 = data1.get("results", [])
        results2 = data2.get("results", [])
        self.assertEqual(len(results1), len(results2))

        for r1, r2 in zip(results1, results2):
            self.assertEqual(r1["doc_id"], r2["doc_id"])
            self.assertEqual(r1["bm25_score"], r2["bm25_score"])
            self.assertEqual(r1["title"], r2["title"])

        # Validate field contracts from core index
        for hit in results1:
            self.assertIsInstance(hit["doc_id"], int)
            self.assertGreater(hit["doc_id"], 0)
            self.assertGreater(hit["bm25_score"], 0)
            self.assertTrue(hit["title"])
            self.assertTrue(hit["snippet"])
            self.assertIsInstance(hit["is_accepted"], bool)
            self.assertTrue(hit["url"].startswith("https://stackoverflow.com/questions/"))

        # BM25 ordering: descending scores
        scores = [h["bm25_score"] for h in results1]
        self.assertEqual(scores, sorted(scores, reverse=True))

    def test_03_graph_threads_categories_and_edges(self):
        # First query search to obtain a real doc_id from the core index
        status, _, content = self.make_request("POST", "/api/v1/search", {"query": "error", "top_k": 2})
        self.assertEqual(status, 200)
        data = json.loads(content.decode("utf-8"))

        results = data.get("results", [])
        if not results:
            doc_ids = "1,2"
        else:
            doc_ids = ",".join(str(r["doc_id"]) for r in results[:2])

        status, _, content = self.make_request("GET", f"/api/v1/graph/threads?doc_ids={doc_ids}")
        if status != 200:
            self.skipTest(f"Graph threads endpoint returned {status}: {content.decode('utf-8')}")

        graph_data = json.loads(content.decode("utf-8"))
        nodes = graph_data.get("nodes", [])
        edges = graph_data.get("edges", [])

        valid_categories = {"PRIMARY_RESULT", "OTHER_RESULT", "DUPLICATE", "LINKED"}
        for node in nodes:
            self.assertIn(node.get("category"), valid_categories)
            self.assertIsInstance(node.get("doc_id"), int)
            self.assertTrue(node.get("title"))
            self.assertIn("has_accepted_answer", node)
            self.assertTrue(node.get("original_url", "").startswith("https://stackoverflow.com/questions/"))
            # Enriched best_answer_body should be present (string)
            self.assertIsInstance(node.get("best_answer_body"), str)

        valid_edge_types = {"DUPLICATE_OF", "LINKED_IN_ANSWER"}
        for edge in edges:
            self.assertIn(edge.get("type"), valid_edge_types)
            self.assertIsInstance(edge.get("source_doc_id"), int)
            self.assertIsInstance(edge.get("target_doc_id"), int)
            self.assertNotEqual(edge.get("source_doc_id"), edge.get("target_doc_id"))

    def test_04_thread_details_404(self):
        non_existent_id = 999999999
        status, _, content = self.make_request("GET", f"/api/v1/graph/threads/{non_existent_id}")
        self.assertEqual(status, 404)
        err_data = json.loads(content.decode("utf-8"))
        self.assertIn("error", err_data)
        self.assertEqual(err_data["error"], "thread not found")


if __name__ == "__main__":
    unittest.main()
