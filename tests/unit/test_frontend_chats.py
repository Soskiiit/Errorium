import os
import re
import shutil
import subprocess
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
STATIC_DIR = os.path.join(REPO_ROOT, "services", "frontend", "static")


class TestFrontendChats(unittest.TestCase):
    """Тесты задачи C2: Чаты, сайдбар диалогов и RAG-карточки фронтенда."""

    def setUp(self):
        self.app_js_path = os.path.join(STATIC_DIR, "app.js")
        self.api_js_path = os.path.join(STATIC_DIR, "api.js")
        self.index_html_path = os.path.join(STATIC_DIR, "index.html")

        with open(self.app_js_path, "r", encoding="utf-8") as f:
            self.app_js = f.read()

    def test_syntax_node_check(self):
        """Проверка синтаксиса app.js и api.js через node --check."""
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

    def test_no_knowledge_base_or_mock_chats(self):
        """Проверка полного отсутствия KNOWLEDGE_BASE и статических моковых чатов в app.js."""
        self.assertNotIn(
            "KNOWLEDGE_BASE",
            self.app_js,
            "В app.js не должно остаться объекта KNOWLEDGE_BASE",
        )
        for mock_id in ["'chat-1'", '"chat-1"', "'chat-2'", '"chat-2"', "'chat-6'", '"chat-6"']:
            self.assertNotIn(
                mock_id,
                self.app_js,
                f"В app.js не должно остаться мокового идентификатора чата {mock_id}",
            )

    def test_chats_api_endpoints(self):
        """Проверка наличия вызовов API чатов: /chats (GET, POST, PATCH, DELETE), /messages, /search/rag."""
        # /chats endpoints
        self.assertRegex(
            self.app_js,
            r"""['"`]/chats\b""",
            "В app.js должен присутствовать вызов эндпоинта /chats",
        )
        self.assertIn(
            "DELETE",
            self.app_js,
            "В app.js должен присутствовать метод DELETE для удаления чата",
        )
        self.assertIn(
            "PATCH",
            self.app_js,
            "В app.js должен присутствовать метод PATCH для переименования/закрепления чата",
        )
        self.assertRegex(
            self.app_js,
            r"""is_pinned""",
            "В app.js должно присутствовать поле is_pinned для закрепления чатов",
        )

        # /messages endpoint
        self.assertRegex(
            self.app_js,
            r"""/messages\b""",
            "В app.js должен присутствовать вызов эндпоинта отправки сообщений /messages",
        )

        # /search/rag endpoint (гостевой режим)
        self.assertRegex(
            self.app_js,
            r"""['"`]/search/rag\b""",
            "В app.js должен присутствовать вызов эндпоинта /search/rag для гостевого режима",
        )

    def test_sse_streaming_with_errorium_api_stream(self):
        """Проверка использования ErroriumAPI.stream для SSE-стриминга сообщений и обработки событий."""
        self.assertIn(
            "ErroriumAPI.stream",
            self.app_js,
            "Для отправки сообщений со стримингом должен использоваться ErroriumAPI.stream",
        )
        # Обработка событий стриминга
        self.assertIn("user_message", self.app_js, "Должна быть обработка события user_message")
        self.assertIn("delta", self.app_js, "Должна быть обработка события delta")
        self.assertIn("done", self.app_js, "Должна быть обработка события done")

    def test_escape_html_and_xss_protection(self):
        """Проверка использования escapeHtml для вывода содержимого сообщений во избежание XSS."""
        self.assertIn(
            "function escapeHtml",
            self.app_js,
            "В app.js должна быть функция escapeHtml",
        )
        self.assertIn(
            "escapeHtml",
            self.app_js,
            "Функция escapeHtml должна применяться для экранирования контента",
        )

    def test_assistant_card_and_copy_snippet(self):
        """Проверка сохранения структуры .assistant-card, источников и функции copySnippet."""
        self.assertIn("assistant-card", self.app_js)
        self.assertIn("card-solution-text", self.app_js)
        self.assertIn("code-block-wrapper", self.app_js)
        self.assertIn("sources-list", self.app_js)
        self.assertIn("copySnippet", self.app_js)
        self.assertIn("openStarMapFromCard", self.app_js)

    def test_render_conversation_supports_history(self):
        """Проверка того, что renderConversation поддерживает рендеринг истории диалога."""
        self.assertIn("function renderConversation", self.app_js)


if __name__ == "__main__":
    unittest.main()
