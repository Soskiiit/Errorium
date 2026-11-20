import os
import re
import shutil
import subprocess
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
STATIC_DIR = os.path.join(REPO_ROOT, "services", "frontend", "static")


class TestFrontendAPI(unittest.TestCase):

    def setUp(self):
        self.api_js_path = os.path.join(STATIC_DIR, "api.js")
        self.app_js_path = os.path.join(STATIC_DIR, "app.js")
        self.index_html_path = os.path.join(STATIC_DIR, "index.html")

    def test_api_js_exists(self):
        """Проверка существования services/frontend/static/api.js."""
        self.assertTrue(
            os.path.isfile(self.api_js_path),
            f"Файл {self.api_js_path} должен существовать"
        )

    def test_syntax_api_and_app_js(self):
        """Проверка синтаксиса api.js и app.js через node --check."""
        node_bin = shutil.which("node")
        if not node_bin:
            self.skipTest("Node.js не установлен для проверки node --check")

        for file_path in [self.api_js_path, self.app_js_path]:
            res = subprocess.run(
                [node_bin, "--check", file_path],
                capture_output=True,
                text=True,
            )
            self.assertEqual(
                res.returncode,
                0,
                f"Синтаксическая ошибка в {file_path}:\nSTDOUT:\n{res.stdout}\nSTDERR:\n{res.stderr}",
            )

    def test_api_js_structure_and_methods(self):
        """Проверка архитектуры api.js: IIFE, экспорт ErroriumAPI, методы request/stream/tokens."""
        with open(self.api_js_path, "r", encoding="utf-8") as f:
            code = f.read()

        # IIFE и экспорт в window.ErroriumAPI
        self.assertIn("window.ErroriumAPI", code, "api.js должен экспортировать window.ErroriumAPI")

        # Базовый префикс API
        self.assertIn("/api/v1", code, "api.js должен использовать базовый префикс /api/v1")

        # Ключ хранилища токенов
        self.assertIn("errorium_tokens", code, "api.js должен использовать ключ localStorage errorium_tokens")

        # Ключевые методы
        required_methods = ["request", "stream", "getTokens", "setTokens", "clearTokens"]
        for method in required_methods:
            self.assertRegex(
                code,
                rf"\b{method}\b",
                f"В ErroriumAPI должен присутствовать метод {method}"
            )

        # Логика refresh и errorium:logout
        self.assertIn("/auth/refresh", code, "api.js должен выполнять ротацию через /auth/refresh")
        self.assertIn("errorium:logout", code, "api.js должен диспатчить событие errorium:logout при провале refresh")
        self.assertIn("Authorization", code, "api.js должен передавать заголовок Authorization: Bearer")

        # Потоковый SSE-стриминг
        self.assertIn("text/event-stream", code, "stream() должен отправлять заголовок Accept: text/event-stream")
        self.assertIn("getReader", code, "stream() должен использовать ReadableStream (getReader)")

    def test_index_html_includes_api_before_app(self):
        """Проверка подключения api.js в index.html строго ДО app.js."""
        with open(self.index_html_path, "r", encoding="utf-8") as f:
            html = f.read()

        # Ищем теги скриптов api.js и app.js
        api_match = re.search(r'<script[^>]+src=["\']([^"\']*api\.js)["\']', html)
        app_match = re.search(r'<script[^>]+src=["\']([^"\']*app\.js)["\']', html)

        self.assertIsNotNone(api_match, "В index.html должен быть тег подключения api.js")
        self.assertIsNotNone(app_match, "В index.html должен быть тег подключения app.js")

        self.assertLess(
            api_match.start(),
            app_match.start(),
            "Тег подключения api.js должен располагаться строго ДО тега app.js"
        )

        # Проверка конкретного пути подключения (/static/api.js или /api.js)
        self.assertRegex(api_match.group(1), r"^/(static/)?api\.js$")

    def test_no_mock_test_user_in_form_submits(self):
        """Проверка отсутствия мок-логики Test_user в обработчиках сабмита форм app.js."""
        with open(self.app_js_path, "r", encoding="utf-8") as f:
            code = f.read()

        # Проверяем, что в сабмитах форм логина и регистрации нет статического хардкода Test_user
        # Ищем сабмит refAuthForm
        auth_form_idx = code.find("refAuthForm")
        self.assertNotEqual(auth_form_idx, -1, "refAuthForm должен присутствовать в app.js")

        # В обработчиках форм не должно быть мокового создания Test_user
        self.assertNotIn(
            "username: 'Test_user'",
            code,
            "В app.js не должно оставаться хардкода { username: 'Test_user' }"
        )

    def test_app_js_uses_real_auth_endpoints(self):
        """Проверка вызова реальных эндпоинтов авторизации из app.js."""
        with open(self.app_js_path, "r", encoding="utf-8") as f:
            code = f.read()

        # Вызовы API авторизации
        self.assertIn("/auth/login", code, "app.js должен вызывать /auth/login")
        self.assertIn("/auth/register", code, "app.js должен вызывать /auth/register")
        self.assertIn("/auth/logout", code, "app.js должен вызывать /auth/logout")
        self.assertIn("/auth/me", code, "app.js должен вызывать /auth/me")
        self.assertIn("/auth/sso/login/", code, "app.js должен инициировать вход через SSO /auth/sso/login/{provider}")

        # Проверка обработки маршрута SSO callback
        self.assertIn("/auth/callback", code, "app.js должен обрабатывать маршрут /auth/callback")
        self.assertIn("location.hash", code, "app.js должен извлекать токены из location.hash при /auth/callback")

        # Проверка обработки события errorium:logout
        self.assertIn("errorium:logout", code, "app.js должен слушать событие errorium:logout")

    def test_dev_server_serves_api_js(self):
        """Проверка отдачи api.js через dev_server по /static/api.js и /api.js."""
        import http.server
        import socket
        import sys
        import threading
        import urllib.request

        sys.path.insert(0, os.path.join(REPO_ROOT, "services", "frontend"))
        import dev_server

        with socket.socket() as s:
            s.bind(("", 0))
            free_port = s.getsockname()[1]

        server = http.server.ThreadingHTTPServer(("", free_port), dev_server.ErroriumDevHandler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()

        try:
            for path in ["/static/api.js", "/api.js", "/static/app.js", "/app.js"]:
                req = urllib.request.Request(f"http://127.0.0.1:{free_port}{path}")
                with urllib.request.urlopen(req, timeout=2) as resp:
                    self.assertEqual(resp.status, 200)
                    self.assertIn("text/javascript", resp.headers.get("Content-Type", ""))
                    content = resp.read().decode("utf-8")
                    self.assertTrue(len(content) > 0)
        finally:
            server.shutdown()
            server.server_close()


if __name__ == "__main__":
    unittest.main()
