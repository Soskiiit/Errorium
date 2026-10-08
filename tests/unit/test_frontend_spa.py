import os
import re
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
STATIC_DIR = os.path.join(REPO_ROOT, "services", "frontend", "static")

class TestFrontendSPA(unittest.TestCase):
    def test_static_files_exist(self):
        """Проверка наличия всех ключевых файлов SPA и ассетов."""
        required_files = [
            "index.html",
            "styles.css",
            "app.js",
            "mascot.png",
            "blue-fire.png",
            "robots.txt",
            "sitemap.xml",
        ]
        for filename in required_files:
            file_path = os.path.join(STATIC_DIR, filename)
            self.assertTrue(
                os.path.isfile(file_path),
                f"Файл {filename} должен существовать в {STATIC_DIR}"
            )

    def test_no_frameworks_in_static(self):
        """Проверка жесткого требования: чистый SPA без фреймворков (No React/Vue/Angular/Bootstrap)."""
        html_path = os.path.join(STATIC_DIR, "index.html")
        js_path = os.path.join(STATIC_DIR, "app.js")

        with open(html_path, "r", encoding="utf-8") as f:
            html = f.read()

        forbidden_cdn_substrings = [
            "react.production",
            "vue.global",
            "angular",
            "bootstrap.min",
            "tailwind",
        ]
        for sub in forbidden_cdn_substrings:
            self.assertNotIn(
                sub, html.lower(),
                f"В index.html обнаружен запрещенный фреймворк или CDN: {sub}"
            )

        with open(js_path, "r", encoding="utf-8") as f:
            js = f.read()

        forbidden_js_imports = [
            "import React",
            "from 'react'",
            "from 'vue'",
            "import Vue",
        ]
        for sub in forbidden_js_imports:
            self.assertNotIn(
                sub, js,
                f"В app.js обнаружен импорт фреймворка: {sub}"
            )

    def test_html_structure_matches_references(self):
        """Проверка соответствия разметки макетам Chat-screen.png и Chat-screen-menu-not-authorized.png."""
        html_path = os.path.join(STATIC_DIR, "index.html")
        with open(html_path, "r", encoding="utf-8") as f:
            html = f.read()

        # 1. Адаптивный viewport
        self.assertIn('name="viewport"', html)
        self.assertIn('content="width=device-width', html)

        # 2. Шапка
        self.assertIn("btn-menu", html)
        self.assertIn("Errorium", html)
        self.assertIn("btn-profile", html)

        # 3. Hero-экран и маскот с фоном blue-fire.png
        self.assertIn("mascot.png", html)
        self.assertIn("blue-fire.png", html)
        self.assertIn("Element", html)
        self.assertIn("404.", html)
        self.assertIn("Reacts with developers", html)
        self.assertIn("Stack overflow. Issue overknown.", html)

        # 4. Нижний ввод (Prompt bar)
        self.assertIn("What went wrong?", html)
        self.assertIn("btn-send", html)
        self.assertIn("AI can make mistakes. Check important information.", html)

        # 5. Боковое меню (Drawer)
        self.assertIn("sidebar", html)
        self.assertIn("Recent chats", html)
        self.assertIn("New chat", html)
        self.assertIn("MCP", html)
        self.assertIn("About us", html)
        self.assertIn("Log in", html)
        self.assertIn("Meet new features!", html)

        # 6. Референсные чаты из Chat-screen-menu-not-authorized.png
        reference_errors = [
            "SyntaxError: invalid syntax",
            "IndentationError: unexpected indent",
            "TypeError: ‘list’ object cannot be interpre...",
            "NameError: name ‘foobar’ is not defined",
            "UnboundLocalError: local variable ‘foobar’...",
            "TypeError: ‘str’ object does not support it...",
            "KeyError: ‘spam’",
        ]
        for err in reference_errors:
            self.assertIn(err, html, f"В боковом меню не найден референсный чат: {err}")

        # 7. Окрашивание интерфейса браузера в цвет фона (#070a13)
        self.assertIn('name="theme-color"', html)
        self.assertIn('#070a13', html)
        self.assertIn('apple-mobile-web-app-status-bar-style', html)

    def test_css_responsive_and_spotlight(self):
        """Проверка стилей: наличие адаптивных медиа-запросов, запрет горизонтального скролла и фона маскота."""
        css_path = os.path.join(STATIC_DIR, "styles.css")
        with open(css_path, "r", encoding="utf-8") as f:
            css = f.read()

        # Адаптивность через медиа-запросы
        self.assertIn("@media", css)

        # Предотвращение горизонтального скролла на мобильных устройствах
        self.assertIn("overflow-x: hidden", css)

        # Стилизация фона маскота blue-fire
        self.assertIn("hero-glow-aura", css)
        self.assertIn("blue-fire-bg", css)

        # Темная тема
        self.assertTrue("#0" in css or "rgb" in css or "hsl" in css)

    def test_dev_environment_files_exist(self):
        """Проверка наличия файлов окружения для локальной разработки."""
        dev_server_path = os.path.join(REPO_ROOT, "services", "frontend", "dev_server.py")
        nginx_dev_path = os.path.join(REPO_ROOT, "services", "frontend", "nginx.dev.conf")
        compose_dev_path = os.path.join(REPO_ROOT, "docker-compose.dev.yml")

        self.assertTrue(os.path.isfile(dev_server_path), "dev_server.py должен существовать")
        self.assertTrue(os.path.isfile(nginx_dev_path), "nginx.dev.conf должен существовать")
        self.assertTrue(os.path.isfile(compose_dev_path), "docker-compose.dev.yml должен существовать")

    def test_dev_server_logic(self):
        """Проверка логики dev-сервера (SPA fallback и MIME-типы)."""
        import sys
        sys.path.insert(0, os.path.join(REPO_ROOT, "services", "frontend"))
        import dev_server
        self.assertEqual(dev_server.PORT, 3000)
        self.assertTrue(os.path.isdir(dev_server.STATIC_DIR))

    def test_dev_server_http_serving(self):
        """Тест реального ответа HTTP сервера на ephemeral порту."""
        import http.server
        import socket
        import threading
        import urllib.request
        import sys

        sys.path.insert(0, os.path.join(REPO_ROOT, "services", "frontend"))
        import dev_server

        # Найти свободный порт
        with socket.socket() as s:
            s.bind(('', 0))
            free_port = s.getsockname()[1]

        server = http.server.ThreadingHTTPServer(('', free_port), dev_server.ErroriumDevHandler)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()

        try:
            # 1. GET /
            req = urllib.request.Request(f"http://127.0.0.1:{free_port}/")
            with urllib.request.urlopen(req, timeout=2) as resp:
                self.assertEqual(resp.status, 200)
                body = resp.read().decode("utf-8")
                self.assertIn("Errorium", body)

            # 2. HEAD /styles.css
            req_head = urllib.request.Request(f"http://127.0.0.1:{free_port}/styles.css", method="HEAD")
            with urllib.request.urlopen(req_head, timeout=2) as resp:
                self.assertEqual(resp.status, 200)
                self.assertIn("text/css", resp.headers.get("Content-Type", ""))

            # 3. SPA Fallback: GET /some-random-route -> serves index.html
            req_spa = urllib.request.Request(f"http://127.0.0.1:{free_port}/unknown-spa-route")
            with urllib.request.urlopen(req_spa, timeout=2) as resp:
                self.assertEqual(resp.status, 200)
                body = resp.read().decode("utf-8")
                self.assertIn("Errorium", body)
        finally:
            server.shutdown()
            server.server_close()

if __name__ == "__main__":
    unittest.main()


