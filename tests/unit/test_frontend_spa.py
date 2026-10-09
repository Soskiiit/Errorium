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

    def test_short_viewport_height_scaling(self):
        """Проверка адаптивности фона маскота blue-fire на экранах с небольшой высотой (Galaxy Fold, landscape)."""
        css_path = os.path.join(STATIC_DIR, "styles.css")
        with open(css_path, "r", encoding="utf-8") as f:
            css = f.read()

        # Проверка наличия ограничения по высоте вьюпорта (max-height)
        self.assertIn("max-height", css)
        self.assertIn("@media (max-height:", css)

        # Проверка, что на малых по высоте экранах огонек уменьшается
        fold_match = re.search(r"@media\s*\(\s*max-height\s*:\s*(?:720px|700px|680px|650px|600px)\s*\)\s*\{([^}]+(?:\{[^}]*\}[^}]*)*)\}", css)
        self.assertIsNotNone(fold_match, "Медиа-запрос для экранов с малой высотой должен присутствовать в styles.css")
        fold_css = fold_match.group(1)
        self.assertIn(".hero-glow-aura", fold_css)

    def test_sidebar_bottom_pinned_to_bottom(self):
        """Проверка фиксации нижней панели сайдбара (MCP, About us, Login) внизу при сворачивании недавних чатов."""
        css_path = os.path.join(STATIC_DIR, "styles.css")
        with open(css_path, "r", encoding="utf-8") as f:
            css = f.read()

        # Поиск блока .sidebar-bottom
        sidebar_bottom_match = re.search(r"\.sidebar-bottom\s*\{([^}]+)\}", css)
        self.assertIsNotNone(sidebar_bottom_match, "Селектор .sidebar-bottom должен присутствовать в styles.css")
        bottom_css = sidebar_bottom_match.group(1)

        # Проверка прижатия к низу через margin-top: auto
        self.assertIn("margin-top: auto", bottom_css, ".sidebar-bottom должен иметь margin-top: auto для прижатия к низу")

    def test_desktop_sidebar_and_header_rules(self):
        """Проверка правил для широких экранов: постоянный сайдбар, скрытие крестика и скрытие центрального логотипа."""
        css_path = os.path.join(STATIC_DIR, "styles.css")
        with open(css_path, "r", encoding="utf-8") as f:
            css = f.read()

        # Извлекаем весь блок адаптивного сайдбара с учетом вложенных фигурных скобок
        start_idx = css.find("@media (min-width: 930px)")
        if start_idx == -1:
            start_idx = css.find("@media (min-width: 1024px)")
        self.assertNotEqual(start_idx, -1, "Блок адаптивного десктопного медиа-запроса должен присутствовать в styles.css")

        open_braces = 0
        found_first = False
        end_idx = -1
        for i in range(start_idx, len(css)):
            if css[i] == '{':
                open_braces += 1
                found_first = True
            elif css[i] == '}':
                open_braces -= 1
                if found_first and open_braces == 0:
                    end_idx = i
                    break

        self.assertNotEqual(end_idx, -1, "Конец блока @media (min-width: 1024px) должен быть найден")
        desktop_css = css[start_idx:end_idx]

        # 1. Сайдбар не сворачивается: позиционирование и видимость
        self.assertIn(".sidebar-drawer", desktop_css)
        self.assertIn("visibility: visible", desktop_css)

        # 2. Бэкдроп скрыт на широких экранах
        self.assertIn(".drawer-backdrop", desktop_css)
        self.assertIn("display: none", desktop_css)

        # 3. Крестик скрывается т.к. меню не закрывается
        self.assertTrue(".close-btn" in desktop_css or "#btn-close-sidebar" in desktop_css)

        # 4. Логотип Errorium посередине шапки скрывается во избежание дублирования
        self.assertTrue(".header-logo" in desktop_css or "#header-logo" in desktop_css)

        # 5. Кнопка меню (гамбургер) скрывается на широких экранах
        self.assertTrue(".header-menu-btn" in desktop_css or "#btn-menu" in desktop_css)

    def test_collapsible_sidebar_desktop_feature(self):
        """Проверка возможности сворачивания бокового меню на широких устройствах и кнопки с заданным SVG."""
        html_path = os.path.join(STATIC_DIR, "index.html")
        css_path = os.path.join(STATIC_DIR, "styles.css")
        js_path = os.path.join(STATIC_DIR, "app.js")

        with open(html_path, "r", encoding="utf-8") as f:
            html = f.read()
        with open(css_path, "r", encoding="utf-8") as f:
            css = f.read()
        with open(js_path, "r", encoding="utf-8") as f:
            js = f.read()

        # 1. Наличие кнопки с заданным SVG в разметке
        self.assertIn("btn-toggle-sidebar", html)
        self.assertIn("M6.5 17.5L6.5 6.5", html)
        self.assertIn("19.362 3.32698", html)

        # 2. Наличие стилей сворачивания сайдбара для широких экранов
        self.assertIn("sidebar-toggle-btn", css)
        self.assertIn("sidebar-collapsed", css)

        # 3. Наличие логики переключения сворачивания в app.js и сохранения в localStorage
        self.assertIn("toggleDesktopSidebar", js)
        self.assertIn("errorium_sidebar_state", js)
        self.assertIn("localStorage.getItem", js)
        self.assertIn("localStorage.setItem", js)

        # 4. Защита от FOUC/передергивания при перезагрузке (preload и ранний скрипт в head)
        self.assertIn("preload", html)
        self.assertIn("preload", css)
        self.assertIn("html.sidebar-collapsed", css)
        self.assertIn("document.documentElement.classList.add('sidebar-collapsed')", html)

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


