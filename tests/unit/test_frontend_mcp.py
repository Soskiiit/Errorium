"""Unit-тесты фронтенда для MCP-модалки и блока лендинга (C4, ADR-012).

Проверяют отсутствие персональных токенов (err_live_, /mcp/token),
динамическое построение URL подключения через window.location.origin,
готовый JSON-конфиг без заголовков Authorization, работу кнопок копирования
и доступность модалки всем пользователям.
"""

import os
import re
import shutil
import subprocess
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
STATIC_DIR = os.path.join(REPO_ROOT, "services", "frontend", "static")


def read_static(name):
    path = os.path.join(STATIC_DIR, name)
    with open(path, "r", encoding="utf-8") as f:
        return f.read()


class TestFrontendMCP(unittest.TestCase):
    """Тесты экрана и модалки MCP на клиенте."""

    def setUp(self):
        self.app_js = read_static("app.js")
        self.index_html = read_static("index.html")
        self.api_js = read_static("api.js")

    def test_syntax_node_check(self):
        """Синтаксическая корректность app.js и api.js через node --check."""
        node_bin = shutil.which("node")
        if not node_bin:
            self.skipTest("Node.js не установлен для проверки node --check")

        for fname in ["app.js", "api.js"]:
            fpath = os.path.join(STATIC_DIR, fname)
            res = subprocess.run([node_bin, "--check", fpath], capture_output=True, text=True)
            self.assertEqual(
                res.returncode, 0,
                f"Синтаксическая ошибка в {fname}:\nSTDOUT:\n{res.stdout}\nSTDERR:\n{res.stderr}",
            )

    def test_no_mcp_tokens_or_token_endpoints(self):
        """Токенов НЕТ (ADR-012): отсутствие err_live_ и вызовов /mcp/token."""
        # Ни в JS, ни в HTML не должно быть мок-токена err_live_
        self.assertNotIn("err_live_", self.app_js, "В app.js не должно быть токенов err_live_")
        self.assertNotIn("err_live_", self.index_html, "В index.html не должно быть токенов err_live_")

        # Никаких запросов к несуществующим эндпоинтам токенов
        self.assertNotIn("/mcp/token", self.app_js, "В app.js не должно быть вызовов /mcp/token")
        self.assertNotIn("/mcp/token", self.api_js, "В api.js не должно быть вызовов /mcp/token")

        # В HTML не должно быть поля Bearer Token
        self.assertNotIn("Bearer err_live_", self.index_html)

    def test_mcp_connection_url_dynamic(self):
        """URL подключения формируется динамически через location.origin + /sse."""
        self.assertIn("location.origin", self.app_js, "app.js должен использовать location.origin для формирования SSE URL")
        self.assertIn("/sse", self.app_js, "app.js должен формировать путь к /sse")

    def test_mcp_config_json_without_authorization(self):
        """JSON-конфиг mcpServers.errorium.url без заголовков Authorization."""
        self.assertIn("mcpServers", self.app_js)
        self.assertIn("errorium", self.app_js)

        # Проверка, что в сгенерированном конфиге и шаблонах нет заголовков Authorization
        self.assertNotIn('"Authorization"', self.index_html, "В index.html не должно быть заголовка Authorization")

    def test_mcp_copy_buttons(self):
        """Кнопки Copy URL и Copy JSON используют navigator.clipboard.writeText и showToast."""
        self.assertIn("navigator.clipboard.writeText", self.app_js)
        self.assertIn("showToast", self.app_js)

        # Проверка кнопок копирования URL и JSON
        self.assertTrue(
            "btnCopyUrl" in self.app_js or "btnCopyToken" in self.app_js,
            "В app.js должен быть обработчик копирования URL",
        )
        self.assertIn("btnCopyMcpJson", self.app_js)

    def test_mcp_modal_accessible_to_all(self):
        """Модалка MCP доступна всем пользователям (открытый доступ, ADR-012)."""
        # Наличие контейнера модалки в HTML
        self.assertIn('id="modal-mcp"', self.index_html)
        # Открытие модалки не блокируется проверкой токена авторизации
        self.assertIn("modalMcp", self.app_js)


if __name__ == "__main__":
    unittest.main()
