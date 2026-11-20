"""End-to-end integration and static verification tests for Errorium Frontend & Stack.

Part of Task C5 (Iteration 2 / Milestone KT3).
Standard library only: unittest, urllib.request, urllib.error, json, os, subprocess, shutil, uuid, time.
"""

import json
import os
import re
import shutil
import subprocess
import time
import unittest
import urllib.error
import urllib.request
import uuid

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
STATIC_DIR = os.path.join(REPO_ROOT, "services", "frontend", "static")


def read_static(filename: str) -> str:
    filepath = os.path.join(STATIC_DIR, filename)
    with open(filepath, "r", encoding="utf-8") as f:
        return f.read()


class TestFrontendStatic(unittest.TestCase):
    """Статическая верификация фронтенда и контрактов SPA без необходимости запуска бэкенда."""

    @classmethod
    def setUpClass(cls):
        cls.app_js = read_static("app.js")
        cls.api_js = read_static("api.js")
        cls.index_html = read_static("index.html")
        cls.styles_css = read_static("styles.css")

    def test_node_syntax_check(self):
        """Проверка синтаксиса JS файлов через node --check."""
        node_bin = shutil.which("node")
        if not node_bin:
            self.skipTest("Node.js не установлен в окружении")

        for js_file in ["app.js", "api.js"]:
            full_path = os.path.join(STATIC_DIR, js_file)
            res = subprocess.run([node_bin, "--check", full_path], capture_output=True, text=True)
            self.assertEqual(
                res.returncode,
                0,
                f"Синтаксическая ошибка в {js_file}:\nSTDOUT:\n{res.stdout}\nSTDERR:\n{res.stderr}",
            )

    def test_spa_structure_and_assets(self):
        """Проверка структуры SPA: ключевые файлы, разметка, CSS-модули и ассеты."""
        required_files = [
            "index.html",
            "styles.css",
            "app.js",
            "api.js",
            "mascot.png",
            "blue-fire.png",
            "css/base.css",
            "css/layout.css",
            "css/sidebar.css",
            "css/hero.css",
            "css/landing.css",
            "css/chat.css",
            "css/prompt.css",
            "css/modals.css",
            "css/profile.css",
            "css/auth.css",
            "css/star-map.css",
        ]
        for rel_path in required_files:
            full_path = os.path.join(STATIC_DIR, rel_path)
            self.assertTrue(os.path.isfile(full_path), f"Файл {rel_path} не найден в {STATIC_DIR}")

        # Проверка подключения api.js и app.js в index.html
        self.assertIn("api.js", self.index_html)
        self.assertIn("app.js", self.index_html)
        self.assertIn("styles.css", self.index_html)

    def test_no_forbidden_libraries(self):
        """Проверка академических запретов: отсутствие сторонних UI/Graph фреймворков (D3, React Flow, Cytoscape, Vue, React)."""
        forbidden_patterns = [
            r"\bd3(?:\.v\d+)?(?:\.min)?\.js\b",
            r"\breact-flow\b",
            r"\bcytoscape(?:\.min)?\.js\b",
            r"\bvue(?:\.min)?\.js\b",
            r"\breact(?:\.production)?\.min\.js\b",
            r"\bangular(?:\.min)?\.js\b",
            r"\bbootstrap(?:\.min)?\.(?:js|css)\b",
            r"\bjquery(?:\.min)?\.js\b",
        ]

        combined_code = self.index_html + "\n" + self.app_js + "\n" + self.api_js
        for pattern in forbidden_patterns:
            matches = re.findall(pattern, combined_code, re.IGNORECASE)
            self.assertEqual(
                len(matches),
                0,
                f"Обнаружена запрещенная внешняя библиотека по шаблону '{pattern}': {matches}",
            )

    def test_routing_support(self):
        """Проверка клиентской маршрутизации SPA: пути / и /star-map, история браузера и возврат."""
        # Проверка обработки маршрутов в app.js
        self.assertIn("'/star-map'", self.app_js)
        self.assertIn("pushState", self.app_js)
        self.assertIn("popstate", self.app_js)
        self.assertIn("openStarMap", self.app_js)
        self.assertIn("closeStarMap", self.app_js)

        # Проверка разметки экрана Star Map в index.html
        self.assertIn('id="star-map-view"', self.index_html)
        self.assertIn('id="star-map-canvas"', self.index_html)
        self.assertIn('id="btn-star-map-back"', self.index_html)

    def test_star_map_real_data(self):
        """Проверка интеграции Star Map с реальным API (/graph/threads) и отсутствие устаревших мок-данных."""
        # Star Map должен запрашивать /graph/threads
        self.assertIn("/graph/threads", self.app_js)
        # Устаревший мок MOCK_RELATED_THREADS должен быть полностью удален (ADR-006 / C4)
        self.assertNotIn("MOCK_RELATED_THREADS", self.app_js)

        # Проверка поддержки категорий узлов графа
        for category in ["PRIMARY_RESULT", "OTHER_RESULT", "DUPLICATE", "LINKED"]:
            self.assertIn(category, self.app_js)

    def test_open_mcp_without_tokens(self):
        """Проверка открытого MCP без персональных токенов (ADR-012)."""
        # Отсутствие токенов err_live_ и эндпоинтов /mcp/token
        self.assertNotIn("err_live_", self.app_js)
        self.assertNotIn("err_live_", self.index_html)
        self.assertNotIn("/mcp/token", self.app_js)
        self.assertNotIn("/mcp/token", self.api_js)

        # Динамическое формирование URL к /sse через location.origin
        self.assertIn("location.origin", self.app_js)
        self.assertIn("/sse", self.app_js)
        self.assertIn("mcpServers", self.app_js)


class TestFrontendE2E(unittest.TestCase):
    """Сквозной интеграционный сценарий жизненного цикла приложения.

    Выполняет реальные HTTP/SSE запросы к Gateway API при наличии запущенного сервиса.
    Автоматически переходит в SkipTest, если сервис Gateway недоступен.
    """

    @classmethod
    def setUpClass(cls):
        cls.base_url = os.environ.get("GATEWAY_URL", "http://localhost:8000").rstrip("/")
        try:
            req = urllib.request.Request(f"{cls.base_url}/health")
            with urllib.request.urlopen(req, timeout=3) as resp:
                if resp.status != 200:
                    raise unittest.SkipTest(
                        f"Gateway service unhealthy at {cls.base_url} (status {resp.status})"
                    )
        except unittest.SkipTest:
            raise
        except Exception as e:
            raise unittest.SkipTest(f"Gateway service unreachable at {cls.base_url}: {e}")

    def make_request(self, method, path, data=None, token=None, headers=None, is_json=True):
        url = f"{self.base_url}{path}"
        req_headers = dict(headers or {})
        body = None

        if data is not None:
            if is_json:
                req_headers["Content-Type"] = "application/json"
                body = json.dumps(data).encode("utf-8")
            else:
                body = data

        if token:
            req_headers["Authorization"] = f"Bearer {token}"

        req = urllib.request.Request(url, data=body, headers=req_headers, method=method)
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

    def create_user_and_login(self):
        uid = uuid.uuid4().hex[:8]
        username = f"e2e_user_{uid}"
        email = f"e2e_{uid}@example.com"
        password = "Password123!"

        status, _, content = self.make_request(
            "POST",
            "/api/v1/auth/register",
            {"username": username, "email": email, "password": password},
        )
        self.assertEqual(status, 201, f"Регистрация не удалась: {content.decode('utf-8')}")

        data = json.loads(content.decode("utf-8"))
        access_token = data["access_token"]
        refresh_token = data["refresh_token"]
        user_info = data["user"]
        return access_token, refresh_token, user_info, email, password

    def test_01_user_profile_and_avatar_lifecycle(self):
        """Сценарий 1: Регистрация -> вход -> профиль GET/PATCH -> загрузка аватара."""
        access_token, _, user_info, _, _ = self.create_user_and_login()
        user_id = user_info["id"]

        # GET /api/v1/users/me
        status, _, content = self.make_request("GET", "/api/v1/users/me", token=access_token)
        self.assertEqual(status, 200)
        profile = json.loads(content.decode("utf-8"))
        self.assertEqual(profile["id"], user_id)

        # PATCH /api/v1/users/me: profession, gender, language
        patch_payload = {
            "profession": "Senior Python Developer",
            "gender": "Male",
            "language": "Русский",
        }
        status, _, content = self.make_request(
            "PATCH",
            "/api/v1/users/me",
            data=patch_payload,
            token=access_token,
        )
        self.assertEqual(status, 200)
        updated = json.loads(content.decode("utf-8"))
        self.assertEqual(updated.get("profession"), "Senior Python Developer")
        self.assertEqual(updated.get("gender"), "Male")
        self.assertEqual(updated.get("language"), "Русский")

        # POST /api/v1/users/avatar (multipart/form-data)
        minimal_png = (
            b"\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89"
            b"\x00\x00\x00\nIDATx\x9cc\x00\x01\x00\x00\x05\x00\x01\r\n-\xb4\x00\x00\x00\x00IEND\xaeB`\x82"
        )
        boundary = f"----WebKitFormBoundary{uuid.uuid4().hex}"
        part_header = (
            f"--{boundary}\r\n"
            f'Content-Disposition: form-data; name="avatar"; filename="avatar.png"\r\n'
            f"Content-Type: image/png\r\n\r\n"
        ).encode("utf-8")
        part_footer = f"\r\n--{boundary}--\r\n".encode("utf-8")
        multipart_body = part_header + minimal_png + part_footer

        status, _, avatar_content = self.make_request(
            "POST",
            "/api/v1/users/avatar",
            data=multipart_body,
            token=access_token,
            headers={"Content-Type": f"multipart/form-data; boundary={boundary}"},
            is_json=False,
        )
        # В зависимости от наличия работающего S3-хранилища (Garage) в тестовом окружении
        if status == 200:
            avatar_res = json.loads(avatar_content.decode("utf-8"))
            self.assertIn("avatar_url", avatar_res)
            self.assertTrue(bool(avatar_res["avatar_url"]))
        else:
            # Если хранилище недоступно в изолированном окружении, статус должен быть 502
            self.assertIn(status, [200, 502])

    def test_02_search_solution_and_star_map(self):
        """Сценарий 2: Поиск ошибки '[Python] IndexError: list index out of range' -> выдача -> Star Map."""
        query = "[Python] IndexError: list index out of range"
        status, _, content = self.make_request("POST", "/api/v1/search", {"query": query})
        self.assertEqual(status, 200)

        search_data = json.loads(content.decode("utf-8"))
        self.assertIn("results", search_data)
        results = search_data["results"]

        doc_ids_list = [str(r["doc_id"]) for r in results[:5]] if results else ["27539308", "15478423"]
        doc_ids_param = ",".join(doc_ids_list)

        # GET /api/v1/graph/threads?doc_ids=...
        status_graph, _, graph_content = self.make_request(
            "GET",
            f"/api/v1/graph/threads?doc_ids={doc_ids_param}",
        )
        self.assertEqual(status_graph, 200)
        graph_data = json.loads(graph_content.decode("utf-8"))
        self.assertIn("nodes", graph_data)
        self.assertIn("edges", graph_data)
        self.assertGreater(len(graph_data["nodes"]), 0)

        # Проверка категорий узлов
        categories = {n.get("category") for n in graph_data["nodes"]}
        self.assertIn("PRIMARY_RESULT", categories)

    def test_03_sse_streaming_rag(self):
        """Сценарий 3: SSE-стриминг RAG (/api/v1/search/rag) с Accept: text/event-stream."""
        url = f"{self.base_url}/api/v1/search/rag"
        body = json.dumps({"query": "[Python] IndexError: list index out of range"}).encode("utf-8")
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
                self.assertIn("text/event-stream", ct)

                stream_content = resp.read().decode("utf-8")
                self.assertIn("event: delta\n", stream_content)
                self.assertIn("event: done\n", stream_content)
        except Exception as e:
            self.fail(f"Ошибка чтения SSE-потока: {e}")

    def test_04_chats_full_crud(self):
        """Сценарий 4: Чаты: создание, добавление сообщения, переименование, закрепление, удаление."""
        access_token, _, _, _, _ = self.create_user_and_login()

        # Создание диалога
        status, _, content = self.make_request(
            "POST",
            "/api/v1/chats",
            {"title": "Диалог E2E"},
            token=access_token,
        )
        self.assertEqual(status, 201)
        chat = json.loads(content.decode("utf-8"))
        chat_id = chat["id"]

        # Добавление сообщения
        query = "[Python] IndexError: list index out of range"
        status_msg, _, content_msg = self.make_request(
            "POST",
            f"/api/v1/chats/{chat_id}/messages",
            {"query": query},
            token=access_token,
        )
        self.assertEqual(status_msg, 201)
        msg_resp = json.loads(content_msg.decode("utf-8"))
        self.assertIn("user_message", msg_resp)
        self.assertIn("assistant_message", msg_resp)

        # Переименование и закрепление
        status_patch, _, content_patch = self.make_request(
            "PATCH",
            f"/api/v1/chats/{chat_id}",
            {"title": "Закрепленный диалог IndexError", "is_pinned": True},
            token=access_token,
        )
        self.assertEqual(status_patch, 200)
        patched = json.loads(content_patch.decode("utf-8"))
        self.assertEqual(patched["title"], "Закрепленный диалог IndexError")
        self.assertTrue(patched["is_pinned"])

        # Удаление
        status_del, _, _ = self.make_request(
            "DELETE",
            f"/api/v1/chats/{chat_id}",
            token=access_token,
        )
        self.assertEqual(status_del, 204)

        # Проверка удаления -> 404
        status_get, _, _ = self.make_request(
            "GET",
            f"/api/v1/chats/{chat_id}",
            token=access_token,
        )
        self.assertEqual(status_get, 404)

    def test_05_guest_search_without_token(self):
        """Сценарий 5: Гостевой поиск без токена авторизации."""
        status, _, content = self.make_request(
            "POST",
            "/api/v1/search",
            {"query": "ZeroDivisionError integer division or modulo by zero"},
        )
        self.assertEqual(status, 200)
        data = json.loads(content.decode("utf-8"))
        self.assertIn("results", data)
        self.assertIn("total_hits", data)

    def test_06_open_mcp_sse_and_tools_call(self):
        """Сценарий 6: Open MCP: GET /sse удерживается открытым, JSON-RPC initialize,
        tools/list и tools/call errorium_search_solution отправляются POST /mcp/messages
        (202), результаты приходят событием message в SSE-поток (openapi.yaml)."""
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
                with urllib.request.urlopen(req, timeout=60) as resp:
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
                    self.assertIn("text/event-stream", val)
                elif kind == "session":
                    session_id = val
                elif kind == "error":
                    self.fail(f"Не удалось подключиться к /sse: {val}")

            self.assertTrue(bool(session_id), "sessionId не найден в /sse ответе")

            def rpc_call(req_id, method, params=None):
                payload = {"jsonrpc": "2.0", "id": req_id, "method": method}
                if params is not None:
                    payload["params"] = params
                status, _, _ = self.make_request(
                    "POST",
                    f"/mcp/messages?sessionId={session_id}",
                    payload,
                )
                self.assertEqual(status, 202)
                deadline = time.time() + 30
                while time.time() < deadline:
                    try:
                        kind, val = events_q.get(timeout=0.5)
                    except queue.Empty:
                        continue
                    if kind == "message":
                        data = json.loads(val)
                        if data.get("id") == req_id:
                            return data
                    elif kind == "error":
                        self.fail(f"Ошибка SSE-потока: {val}")
                self.fail(f"JSON-RPC ответ id={req_id} не пришёл в SSE-поток")

            # 1. initialize
            init_resp = rpc_call(1, "initialize", {
                "protocolVersion": "2024-11-05",
                "capabilities": {},
                "clientInfo": {"name": "test-e2e", "version": "1.0"},
            })
            self.assertEqual(init_resp.get("jsonrpc"), "2.0")
            self.assertIn("result", init_resp)

            # 2. tools/list
            tools_resp = rpc_call(2, "tools/list")
            tool_names = [t["name"] for t in tools_resp.get("result", {}).get("tools", [])]
            self.assertIn("errorium_search_solution", tool_names)

            # 3. tools/call errorium_search_solution
            call_resp = rpc_call(3, "tools/call", {
                "name": "errorium_search_solution",
                "arguments": {"query": "IndexError: list index out of range"},
            })
            self.assertIn("result", call_resp)
            self.assertIn("content", call_resp["result"])
        finally:
            stop.set()
            if "resp" in resp_holder:
                resp_holder["resp"].close()

    def test_07_admin_api_lifecycle(self):
        """Сценарий 7: Admin API: stats, index reload, rag config, блокировка пользователя."""
        admin_email = os.environ.get("FIRST_ADMIN_EMAIL", "admin@errorium.ru")
        admin_password = os.environ.get("FIRST_ADMIN_PASSWORD", "AdminSecurePassword2026!")

        status, _, content = self.make_request(
            "POST",
            "/api/v1/auth/login",
            {"email": admin_email, "password": admin_password},
        )
        if status != 200:
            raise unittest.SkipTest("Bootstrap учетная запись администратора недоступна")

        admin_data = json.loads(content.decode("utf-8"))
        admin_token = admin_data["access_token"]
        if admin_data["user"].get("role") != "admin":
            raise unittest.SkipTest("Пользователь не обладает ролью admin")

        # 1. GET /api/v1/admin/stats
        status_stats, _, content_stats = self.make_request(
            "GET",
            "/api/v1/admin/stats",
            token=admin_token,
        )
        self.assertEqual(status_stats, 200)
        stats = json.loads(content_stats.decode("utf-8"))
        self.assertIn("total_documents", stats)
        self.assertIn("avg_doc_length", stats)

        # 2. POST /api/v1/admin/index/reload
        status_reload, _, _ = self.make_request(
            "POST",
            "/api/v1/admin/index/reload",
            token=admin_token,
        )
        self.assertIn(status_reload, [200, 502])

        # 3. GET и PUT /api/v1/admin/rag/config
        status_rag, _, content_rag = self.make_request(
            "GET",
            "/api/v1/admin/rag/config",
            token=admin_token,
        )
        self.assertEqual(status_rag, 200)
        rag_conf = json.loads(content_rag.decode("utf-8"))

        status_put, _, _ = self.make_request(
            "PUT",
            "/api/v1/admin/rag/config",
            data={
                "provider": rag_conf.get("provider", "stub"),
                "model": rag_conf.get("model", "gpt-4o-mini"),
                "system_prompt": "Updated E2E RAG prompt",
                "temperature": 0.2,
                "max_tokens": 1000,
            },
            token=admin_token,
        )
        self.assertEqual(status_put, 200)

        # 4. Блокировка пользователя
        _, _, new_user, victim_email, victim_password = self.create_user_and_login()
        victim_id = new_user["id"]

        status_block, _, content_block = self.make_request(
            "PATCH",
            f"/api/v1/admin/users/{victim_id}",
            data={"is_active": False},
            token=admin_token,
        )
        self.assertEqual(status_block, 200)
        blocked_user = json.loads(content_block.decode("utf-8"))
        self.assertFalse(blocked_user["is_active"])

        # Попытка входа заблокированного пользователя -> 403 (openapi.yaml: is_active=false)
        status_login, _, _ = self.make_request(
            "POST",
            "/api/v1/auth/login",
            {"email": victim_email, "password": victim_password},
        )
        self.assertEqual(status_login, 403)


if __name__ == "__main__":
    unittest.main()
