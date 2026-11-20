"""Integration tests for Errorium Gateway REST API.

Standard library only (unittest, urllib.request, urllib.error, json, uuid, os, time).
"""

import json
import os
import time
import unittest
import urllib.error
import urllib.request
import uuid


class TestGatewayAPI(unittest.TestCase):
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

    def make_request(self, method, path, data=None, token=None):
        url = f"{self.base_url}{path}"
        headers = {}
        body = None
        if data is not None:
            headers["Content-Type"] = "application/json"
            body = json.dumps(data).encode("utf-8")
        if token:
            headers["Authorization"] = f"Bearer {token}"
        req = urllib.request.Request(url, data=body, headers=headers, method=method)
        try:
            with urllib.request.urlopen(req, timeout=5) as resp:
                status = resp.status
                headers_dict = dict(resp.headers)
                content = resp.read()
                return status, headers_dict, content
        except urllib.error.HTTPError as e:
            status = e.code
            headers_dict = dict(e.headers)
            content = e.read()
            return status, headers_dict, content

    def create_user_and_login(self):
        uid = uuid.uuid4().hex[:8]
        username = f"test_{uid}"
        email = f"test_{uid}@example.com"
        password = "Password123!"

        status, _, content = self.make_request(
            "POST",
            "/api/v1/auth/register",
            {"username": username, "email": email, "password": password},
        )
        self.assertEqual(status, 201, f"Register failed: {content.decode('utf-8')}")

        data = json.loads(content.decode("utf-8"))
        access_token = data["access_token"]
        refresh_token = data["refresh_token"]
        return access_token, refresh_token, data["user"], email, password

    def test_health(self):
        status, _, content = self.make_request("GET", "/health")
        self.assertEqual(status, 200)
        data = json.loads(content.decode("utf-8"))
        self.assertEqual(data.get("status"), "ok")

    def test_openapi_and_docs(self):
        # 1. GET /openapi.yaml
        status, headers, content = self.make_request("GET", "/openapi.yaml")
        self.assertEqual(status, 200)
        ct = headers.get("Content-Type", "")
        self.assertTrue("yaml" in ct, f"Expected yaml in Content-Type, got {ct}")
        yaml_text = content.decode("utf-8")
        self.assertIn("openapi: 3.0.3", yaml_text)

        # 2. GET /docs
        status, headers, content = self.make_request("GET", "/docs")
        self.assertEqual(status, 200)
        ct = headers.get("Content-Type", "")
        self.assertTrue("text/html" in ct, f"Expected text/html in Content-Type, got {ct}")
        html_text = content.decode("utf-8")
        self.assertIn("swagger-ui-dist@5.35.0", html_text)
        self.assertIn("/openapi.yaml", html_text)

    def test_auth_lifecycle(self):
        uid = uuid.uuid4().hex[:8]
        username = f"auth_{uid}"
        email = f"auth_{uid}@example.com"
        password = "Password123!"

        # Register
        status, _, content = self.make_request(
            "POST",
            "/api/v1/auth/register",
            {"username": username, "email": email, "password": password},
        )
        self.assertEqual(status, 201)
        reg_data = json.loads(content.decode("utf-8"))
        self.assertIn("access_token", reg_data)
        self.assertIn("refresh_token", reg_data)
        self.assertEqual(reg_data["user"]["email"], email.lower())

        # Login
        status, _, content = self.make_request(
            "POST",
            "/api/v1/auth/login",
            {"email": email, "password": password},
        )
        self.assertEqual(status, 200)
        login_data = json.loads(content.decode("utf-8"))
        access_token = login_data["access_token"]
        self.assertTrue(bool(access_token))

        # Get Me
        status, _, content = self.make_request("GET", "/api/v1/auth/me", token=access_token)
        self.assertEqual(status, 200)
        me_data = json.loads(content.decode("utf-8"))
        self.assertEqual(me_data["email"], email.lower())
        self.assertEqual(me_data["username"], username)

    def test_chats_crud_and_messages(self):
        access_token, _, _, _, _ = self.create_user_and_login()

        # Create chat
        status, _, content = self.make_request(
            "POST",
            "/api/v1/chats",
            {"title": "Новый чат"},
            token=access_token,
        )
        self.assertEqual(status, 201)
        chat = json.loads(content.decode("utf-8"))
        chat_id = chat["id"]
        self.assertTrue(bool(chat_id))

        # Send message with [Python] tag
        query = "[Python] TypeError: unsupported operand"
        status, _, content = self.make_request(
            "POST",
            f"/api/v1/chats/{chat_id}/messages",
            {"query": query},
            token=access_token,
        )
        self.assertEqual(status, 201)
        msg_resp = json.loads(content.decode("utf-8"))
        user_msg = msg_resp.get("user_message")
        asst_msg = msg_resp.get("assistant_message")
        self.assertIsNotNone(user_msg)
        self.assertIsNotNone(asst_msg)
        self.assertEqual(user_msg.get("sender_role"), "user")
        self.assertEqual(user_msg.get("content"), query)

        # Verify extracted bracketed tag
        tag_names = [t["name"] for t in user_msg.get("tags", [])]
        self.assertIn("python", tag_names)

        self.assertEqual(asst_msg.get("sender_role"), "assistant")
        self.assertTrue(bool(asst_msg.get("content")))
        self.assertIn("code", asst_msg)
        self.assertIn("rag_sources", asst_msg)

        # Get chat: verify auto-generated title and messages
        status, _, content = self.make_request(
            "GET",
            f"/api/v1/chats/{chat_id}",
            token=access_token,
        )
        self.assertEqual(status, 200)
        chat_details = json.loads(content.decode("utf-8"))
        self.assertEqual(chat_details["chat"]["title"], "TypeError: unsupported operand")
        self.assertGreaterEqual(len(chat_details["messages"]), 2)

        # Patch chat (pin + rename)
        status, _, content = self.make_request(
            "PATCH",
            f"/api/v1/chats/{chat_id}",
            {"title": "Renamed Test Chat", "is_pinned": True},
            token=access_token,
        )
        self.assertEqual(status, 200)
        patched_chat = json.loads(content.decode("utf-8"))
        self.assertEqual(patched_chat["title"], "Renamed Test Chat")
        self.assertTrue(patched_chat["is_pinned"])

        # Delete chat
        status, _, _ = self.make_request(
            "DELETE",
            f"/api/v1/chats/{chat_id}",
            token=access_token,
        )
        self.assertEqual(status, 204)

        # Subsequent GET -> 404
        status, _, _ = self.make_request(
            "GET",
            f"/api/v1/chats/{chat_id}",
            token=access_token,
        )
        self.assertEqual(status, 404)

    def test_search_determinism(self):
        query_payload = {"query": "SyntaxError invalid syntax in Python"}
        status1, _, content1 = self.make_request("POST", "/api/v1/search", query_payload)
        status2, _, content2 = self.make_request("POST", "/api/v1/search", query_payload)

        self.assertEqual(status1, 200)
        self.assertEqual(status2, 200)

        data1 = json.loads(content1.decode("utf-8"))
        data2 = json.loads(content2.decode("utf-8"))

        self.assertEqual(data1["total_hits"], data2["total_hits"])
        self.assertEqual(len(data1["results"]), len(data2["results"]))

        for item1, item2 in zip(data1["results"], data2["results"]):
            self.assertEqual(item1["doc_id"], item2["doc_id"])
            self.assertEqual(item1["title"], item2["title"])
            self.assertEqual(item1["url"], item2["url"])
            self.assertEqual(item1["snippet"], item2["snippet"])
            self.assertAlmostEqual(item1["bm25_score"], item2["bm25_score"], places=4)

    def test_graph_threads(self):
        status, _, content = self.make_request(
            "GET",
            "/api/v1/graph/threads?doc_ids=27539308,15478423",
        )
        self.assertEqual(status, 200)
        data = json.loads(content.decode("utf-8"))
        self.assertIn("nodes", data)
        self.assertIn("edges", data)
        if not data["nodes"]:
            raise unittest.SkipTest(
                "В текущем индексе нет SO-фикстур 27539308/15478423 "
                "(нужен выборка-индекс из дампа: make index-sample)"
            )
        self.assertGreater(len(data["nodes"]), 0)
        self.assertGreater(len(data["edges"]), 0)

        categories = {n["category"] for n in data["nodes"]}
        self.assertIn("PRIMARY_RESULT", categories)

        for edge in data["edges"]:
            self.assertIn("source_doc_id", edge)
            self.assertIn("target_doc_id", edge)
            self.assertIn("type", edge)

    def test_graph_thread_details(self):
        # 1. Existing fixture
        status, _, content = self.make_request("GET", "/api/v1/graph/threads/27539308")
        if status == 404:
            raise unittest.SkipTest(
                "В текущем индексе нет SO-фикстуры 27539308 "
                "(нужен выборка-индекс из дампа: make index-sample)"
            )
        self.assertEqual(status, 200)
        data = json.loads(content.decode("utf-8"))
        self.assertEqual(data["doc_id"], 27539308)
        self.assertTrue(bool(data.get("title")))
        self.assertTrue(bool(data.get("best_answer_body")))

        # 2. Non-existent fixture -> 404
        status, _, _ = self.make_request("GET", "/api/v1/graph/threads/999999999")
        self.assertEqual(status, 404)

    def test_tags_catalog(self):
        status, _, content = self.make_request("GET", "/api/v1/tags")
        self.assertEqual(status, 200)
        data = json.loads(content.decode("utf-8"))
        self.assertIn("items", data)

        names = [item["name"] for item in data["items"]]
        expected_tags = [
            "Go", "Python", "Java", "C#", "C++", "JS", "SQL", "Rust",
            "TypeScript", "Docker", "Linux", "Kubernetes", "PostgreSQL",
            "Redis", "GraphQL", "React", "Vue", "Swift", "Kotlin",
            "PHP", "Git", "Nginx", "gRPC",
        ]
        self.assertEqual(names, expected_tags)

    def test_system_status(self):
        status, _, content = self.make_request("GET", "/api/v1/status")
        self.assertEqual(status, 200)
        data = json.loads(content.decode("utf-8"))

        self.assertIn("gateway", data)
        self.assertIn("search_core", data)
        self.assertIn("rag", data)

        self.assertEqual(data["gateway"]["status"], "ok")
        self.assertEqual(data["gateway"]["version"], "1.0.0")
        self.assertIn("connected", data["search_core"])
        self.assertIn("address", data["search_core"])
        self.assertIn("enabled", data["rag"])
        self.assertEqual(data["rag"]["provider"], "stub")

    def test_password_forgot(self):
        status, _, content = self.make_request(
            "POST",
            "/api/v1/auth/password/forgot",
            {"email": "forgot_e2e@example.com"},
        )
        self.assertEqual(status, 202)

    def test_mcp_sse_anonymous_flow(self):
        """MCP SSE transport: сессия живёт, пока открыт GET /sse; POST /mcp/messages
        отвечает 202, а JSON-RPC результат приходит событием message в SSE-поток
        (см. openapi.yaml /mcp/messages)."""
        import queue
        import threading

        events_q = queue.Queue()
        resp_holder = {}
        stop = threading.Event()

        def sse_reader():
            req = urllib.request.Request(
                f"{self.base_url}/sse", headers={"Accept": "text/event-stream"}
            )
            try:
                with urllib.request.urlopen(req, timeout=30) as resp:
                    resp_holder["resp"] = resp
                    events_q.put(("status", resp.status))
                    events_q.put(("content_type", resp.headers.get("Content-Type", "")))
                    current_event = None
                    while not stop.is_set():
                        raw = resp.readline()
                        if not raw:
                            break
                        line = raw.decode("utf-8").rstrip("\r\n")
                        if line.startswith("event:"):
                            current_event = line.split(":", 1)[1].strip()
                        elif line.startswith("data:"):
                            data = line.split(":", 1)[1].strip()
                            if current_event == "endpoint" and "sessionId=" in data:
                                events_q.put(("session", data.split("sessionId=")[1].strip()))
                            elif current_event == "message":
                                events_q.put(("message", data))
            except Exception as e:
                if not stop.is_set():
                    events_q.put(("error", str(e)))

        reader = threading.Thread(target=sse_reader, daemon=True)
        reader.start()
        try:
            session_id = None
            deadline = time.time() + 10
            while time.time() < deadline and session_id is None:
                try:
                    kind, val = events_q.get(timeout=0.5)
                except queue.Empty:
                    continue
                if kind == "status":
                    self.assertEqual(val, 200)
                elif kind == "content_type":
                    self.assertTrue("text/event-stream" in val)
                elif kind == "session":
                    session_id = val
                elif kind == "error":
                    self.fail(f"Failed to read from /sse stream: {val}")

            self.assertTrue(bool(session_id), "Expected sessionId in /sse endpoint event")

            init_payload = {
                "jsonrpc": "2.0",
                "id": 1,
                "method": "initialize",
                "params": {
                    "protocolVersion": "2024-11-05",
                    "capabilities": {},
                    "clientInfo": {"name": "test-e2e", "version": "1.0"},
                },
            }
            status, _, _ = self.make_request(
                "POST",
                f"/mcp/messages?sessionId={session_id}",
                init_payload,
            )
            self.assertEqual(status, 202)

            resp_data = None
            deadline = time.time() + 15
            while time.time() < deadline and resp_data is None:
                try:
                    kind, val = events_q.get(timeout=0.5)
                except queue.Empty:
                    continue
                if kind == "message":
                    data = json.loads(val)
                    if data.get("id") == 1:
                        resp_data = data
                elif kind == "error":
                    self.fail(f"SSE stream error: {val}")

            self.assertIsNotNone(resp_data, "Expected JSON-RPC response in SSE stream")
            self.assertEqual(resp_data.get("jsonrpc"), "2.0")
            self.assertIn("result", resp_data)
        finally:
            stop.set()
            if "resp" in resp_holder:
                resp_holder["resp"].close()

    def test_admin_flow(self):
        admin_email = os.environ.get("FIRST_ADMIN_EMAIL", "admin@errorium.ru")
        admin_password = os.environ.get("FIRST_ADMIN_PASSWORD", "AdminSecurePassword2026!")

        status, _, content = self.make_request(
            "POST",
            "/api/v1/auth/login",
            {"email": admin_email, "password": admin_password},
        )
        if status != 200:
            raise unittest.SkipTest(f"Admin login failed (status {status}), bootstrap admin may not be configured in current environment")

        login_data = json.loads(content.decode("utf-8"))
        admin_token = login_data["access_token"]
        if login_data["user"].get("role") != "admin":
            raise unittest.SkipTest("Logged in user is not an admin")

        # 1. GET /api/v1/admin/stats
        status_stats, _, content_stats = self.make_request(
            "GET",
            "/api/v1/admin/stats",
            token=admin_token,
        )
        if status_stats == 200:
            stats_data = json.loads(content_stats.decode("utf-8"))
            self.assertIn("total_documents", stats_data)
            self.assertIn("avg_doc_length", stats_data)
            self.assertIn("is_ready", stats_data)
            self.assertIn("latency_ms", stats_data)

        # 2. GET /api/v1/admin/users
        status_users, _, content_users = self.make_request(
            "GET",
            "/api/v1/admin/users",
            token=admin_token,
        )
        self.assertEqual(status_users, 200)
        users_data = json.loads(content_users.decode("utf-8"))
        self.assertIn("items", users_data)
        self.assertIn("total", users_data)

        # 3. GET /api/v1/admin/rag/config
        status_rag, _, content_rag = self.make_request(
            "GET",
            "/api/v1/admin/rag/config",
            token=admin_token,
        )
        self.assertEqual(status_rag, 200)
        rag_data = json.loads(content_rag.decode("utf-8"))
        self.assertIn("provider", rag_data)

        # 4. PUT /api/v1/admin/rag/config
        status_put, _, _ = self.make_request(
            "PUT",
            "/api/v1/admin/rag/config",
            {
                "provider": rag_data.get("provider", "stub"),
                "model": rag_data.get("model", "stub-model"),
                "system_prompt": rag_data.get("system_prompt", "Errorium prompt"),
                "temperature": 0.3,
                "max_tokens": 1200,
            },
            token=admin_token,
        )
        self.assertEqual(status_put, 200)

    def test_search_rag_sse(self):
        url = f"{self.base_url}/api/v1/search/rag"
        body = json.dumps({"query": "SyntaxError in Python"}).encode("utf-8")
        req = urllib.request.Request(
            url,
            data=body,
            headers={
                "Content-Type": "application/json",
                "Accept": "text/event-stream",
            },
            method="POST",
        )
        try:
            with urllib.request.urlopen(req, timeout=10) as resp:
                self.assertEqual(resp.status, 200)
                ct = resp.headers.get("Content-Type", "")
                self.assertTrue("text/event-stream" in ct)
                content = resp.read().decode("utf-8")
                self.assertIn("event: delta\n", content)
                self.assertIn("event: done\n", content)
        except Exception as e:
            self.fail(f"SSE /search/rag request failed: {e}")


if __name__ == "__main__":
    unittest.main()
