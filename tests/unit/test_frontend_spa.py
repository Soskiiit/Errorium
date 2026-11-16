import os
import re
import shutil
import subprocess
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
STATIC_DIR = os.path.join(REPO_ROOT, "services", "frontend", "static")

class TestFrontendSPA(unittest.TestCase):

    def _load_css(self):
        css_path = os.path.join(STATIC_DIR, "styles.css")
        with open(css_path, "r", encoding="utf-8") as f:
            content = f.read()
        def repl(m):
            rel = m.group(1).lstrip('./')
            p = os.path.join(STATIC_DIR, rel)
            if os.path.isfile(p):
                with open(p, "r", encoding="utf-8") as cf:
                    return cf.read()
            return m.group(0)
        return re.sub(r"""@import\s+url\(['"]([^'"]+\.css)['"]\);?""", repl, content)

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
            "css/base.css",
            "css/layout.css",
            "css/sidebar.css",
            "css/hero.css",
            "css/landing.css",
            "css/chat.css",
            "css/prompt.css",
            "css/modals.css",
            "css/responsive.css",
            "css/profile.css",
            "css/auth.css",
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
        css = self._load_css()

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
        css = self._load_css()

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
        css = self._load_css()

        # Поиск блока .sidebar-bottom
        sidebar_bottom_match = re.search(r"\.sidebar-bottom\s*\{([^}]+)\}", css)
        self.assertIsNotNone(sidebar_bottom_match, "Селектор .sidebar-bottom должен присутствовать в styles.css")
        bottom_css = sidebar_bottom_match.group(1)

        # Проверка прижатия к низу через margin-top: auto
        self.assertIn("margin-top: auto", bottom_css, ".sidebar-bottom должен иметь margin-top: auto для прижатия к низу")

    def test_desktop_sidebar_and_header_rules(self):
        """Проверка правил для широких экранов: постоянный сайдбар, скрытие крестика и скрытие центрального логотипа."""
        css = self._load_css()

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
        css = self._load_css()
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

    def test_reference_modals_structure(self):
        """Проверка разметки и элементов модальных окон из референсов:
        Auth-screen.png, Change-password-email-screen.png, Change-password-screen.png, Reset-password-alert.png."""
        html_path = os.path.join(STATIC_DIR, "index.html")
        with open(html_path, "r", encoding="utf-8") as f:
            html = f.read()

        # 1. Auth Reference Modal (Auth-screen.png)
        self.assertIn("modal-auth-reference", html)
        self.assertIn("ref-auth-email", html)
        self.assertIn("ref-auth-password", html)
        self.assertIn("Remember me", html)
        self.assertIn("Forgot password?", html)
        self.assertIn("btn-green-pill", html)
        self.assertIn("ref-btn-login", html)
        self.assertIn("Continue with Yandex ID", html)
        self.assertIn("yandex-red-badge", html)
        self.assertIn("New here?", html)
        self.assertIn("Sign up", html)

        # 2. Change Password Email Modal (Change-password-email-screen.png)
        self.assertIn("modal-change-password-email", html)
        self.assertIn("Enter your account email", html)
        self.assertIn("ref-reset-email", html)
        self.assertIn("ref-btn-send-reset", html)
        self.assertIn("We've sent you an email.", html)
        self.assertIn("Don't forget to check your spam folder", html)

        # 3. Change Password Modal (Change-password-screen.png)
        self.assertIn("modal-change-password", html)
        self.assertIn("Change password", html)
        self.assertIn("Enter a new password", html)
        self.assertIn("Enter it again", html)
        self.assertIn("ref-new-password", html)
        self.assertIn("ref-confirm-password", html)
        self.assertIn("ref-btn-approve-password", html)
        self.assertIn("Approve", html)

        # 4. Reset Password Alert Modal (Reset-password-alert.png)
        self.assertIn("modal-reset-password-alert", html)
        self.assertIn("alert-warning-icon", html)
        self.assertIn("Keep in mind that changing your password will sign you out of all other devices", html)
        self.assertIn("btn-amber-pill", html)
        self.assertIn("ref-btn-alert-send-reset", html)

        # 5. Registration Modal (Registration dialog matching Auth modal style)
        self.assertIn("modal-register-reference", html)
        self.assertIn("ref-register-username", html)
        self.assertIn("ref-register-email", html)
        self.assertIn("ref-register-password", html)
        self.assertIn("ref-register-confirm", html)
        self.assertIn("ref-register-terms", html)
        self.assertIn("ref-btn-register", html)
        self.assertIn("Create account", html)
        self.assertIn("Sign up with Yandex ID", html)
        self.assertIn("Already have an account?", html)
        self.assertIn("ref-link-signin", html)

    def test_profile_screen_structure(self):
        """Проверка разметки экрана профиля пользователя (Profile-screen.png)."""
        html_path = os.path.join(STATIC_DIR, "index.html")
        with open(html_path, "r", encoding="utf-8") as f:
            html = f.read()

        self.assertIn("profile-view", html)
        self.assertIn("profile-avatar-large", html)
        self.assertIn("profile-avatar-edit-badge", html)
        self.assertIn("profile-username", html)
        self.assertIn("Test_user", html)

        # Секции экрана профиля
        self.assertIn("Personal information", html)
        self.assertIn("Test_user@mail.ru", html)
        self.assertIn("Backend developer", html)
        self.assertIn("profile-info-gender-row", html)
        self.assertIn("Male", html)

        self.assertIn("Security", html)
        self.assertIn("profile-row-security", html)

        self.assertIn("Language", html)
        self.assertIn("English", html)
        self.assertIn("profile-info-lang", html)

    def test_tags_popover_structure(self):
        """Проверка поповера тегов технологий (Chat-screen-with-tags.png)."""
        html_path = os.path.join(STATIC_DIR, "index.html")
        with open(html_path, "r", encoding="utf-8") as f:
            html = f.read()

        self.assertIn("tags-popover", html)
        self.assertIn("tags-grid", html)

        # 8 референсных тегов
        tags = ["Go", "Python", "Java", "C#", "C++", "JS", "SQL", "Rust"]
        for tag in tags:
            self.assertIn(f'>{tag}<', html, f"Тег {tag} должен присутствовать в tags-popover")

        tag_classes = ["tag-go", "tag-python", "tag-java", "tag-csharp", "tag-cpp", "tag-js", "tag-sql", "tag-rust"]
        for cls in tag_classes:
            self.assertIn(cls, html, f"CSS класс {cls} должен присутствовать в разметке тегов")

    def test_authorized_sidebar_and_header_state(self):
        """Проверка переключения между гостевым и авторизованным состоянием сайдбара и шапки (Chat-screen-menu-authorized.png)."""
        html_path = os.path.join(STATIC_DIR, "index.html")
        js_path = os.path.join(STATIC_DIR, "app.js")

        with open(html_path, "r", encoding="utf-8") as f:
            html = f.read()
        with open(js_path, "r", encoding="utf-8") as f:
            js = f.read()

        # Разметка авторизованного футера
        self.assertIn("sidebar-user-guest", html)
        self.assertIn("sidebar-user-auth", html)
        self.assertIn("sidebar-auth-avatar", html)
        self.assertIn("sidebar-auth-name", html)
        self.assertIn("sidebar-auth-role", html)
        self.assertIn("header-avatar-guest", html)
        self.assertIn("header-avatar-auth", html)

        # Логика в app.js
        self.assertIn("errorium_auth_user", js)
        self.assertIn("updateAuthUI", js)
        self.assertIn("switchToProfileView", js)
        self.assertIn("sidebarUserAuth?.addEventListener('click'", js)
        self.assertIn("elements.promptFooter.classList.add('hidden')", js)

    def test_chat_message_appearance_and_layout(self):
        """Проверка восстановления внешнего вида и структуры сообщений чата (R1, R2):
        - User сообщения рендерятся в .user-bubble со стилями (#1a233b, моноширинный шрифт) без иконки галочки.
        - Ответ ассистента рендерит .assistant-card напрямую без приветственного пузыря и без .dialog-assistant-wrapper.
        - .messages-container содержит строки .message-row.user и .message-row.assistant."""
        css_path = os.path.join(STATIC_DIR, "styles.css")
        js_path = os.path.join(STATIC_DIR, "app.js")

        css = self._load_css()
        with open(js_path, "r", encoding="utf-8") as f:
            js = f.read()

        # 1. Проверка стилей .user-bubble в styles.css
        self.assertIn(".user-bubble", css)
        self.assertIn("#1a233b", css)
        self.assertIn("rgba(255, 255, 255, 0.12)", css)
        self.assertIn("18px 18px 4px 18px", css)
        self.assertIn("12px 18px", css)
        self.assertIn("#ffffff", css)
        self.assertIn("var(--font-mono)", css)

        # 2. Проверка стилей .message-row и .assistant-card в styles.css
        self.assertIn(".message-row.user", css)
        self.assertIn(".message-row.assistant", css)
        self.assertIn(".assistant-card", css)

        # 3. Проверка разметки и рендеринга сообщений в app.js
        self.assertIn('<div class="message-row user">', js)
        self.assertIn('<div class="user-bubble">${escapeHtml(userText)}</div>', js)
        self.assertIn('<div class="message-row assistant">', js)
        self.assertIn('<div class="assistant-card">', js)
        self.assertIn("Errorium RAG Synthesis (BM25 Core)", js)

        # 4. Проверка отсутствия старых dialog bubble оберток, аватарок и галочек
        self.assertNotIn("dialog-user-bubble", js)
        self.assertNotIn("dialog-assistant-wrapper", js)
        self.assertNotIn("dialog-assistant-bubble", js)
        self.assertNotIn("dialog-check-icon", js)
        self.assertNotIn("SyntaxError fixing", js)

        self.assertNotIn("dialog-user-bubble", css)
        self.assertNotIn("dialog-assistant-wrapper", css)
        self.assertNotIn("dialog-check-icon", css)

    def test_sources_title_truncation(self):
        """Проверка усечения длинных заголовков источников Stack Overflow с многоточием (text-overflow: ellipsis)."""
        css = self._load_css()

        self.assertIn(".source-title", css)
        self.assertIn("text-overflow: ellipsis", css)
        self.assertIn("white-space: nowrap", css)
        self.assertIn("overflow: hidden", css)

    def test_spa_client_routing(self):
        """Проверка реализации клиентского SPA-роутинга через History API (pushState/popstate)."""
        js_path = os.path.join(STATIC_DIR, "app.js")
        with open(js_path, "r", encoding="utf-8") as f:
            js = f.read()

        self.assertIn("renderCurrentRoute", js)
        self.assertIn("window.history.pushState", js)
        self.assertIn("popstate", js)
        self.assertIn("/profile", js)
        self.assertIn("/chat", js)
        self.assertIn("/login", js)
        self.assertIn("/register", js)

    # Обратная совместимость для селекторов тестов
    test_conversational_dialog_bubbles = test_chat_message_appearance_and_layout

    def test_landing_page_structure_and_anchors(self):
        """Проверка наличия лендинга Errorium, якорных секций #landing-mcp и #landing-about, а также стилей."""
        html_path = os.path.join(STATIC_DIR, "index.html")
        css_path = os.path.join(STATIC_DIR, "styles.css")
        js_path = os.path.join(STATIC_DIR, "app.js")

        with open(html_path, "r", encoding="utf-8") as f:
            html = f.read()
        css = self._load_css()
        with open(js_path, "r", encoding="utf-8") as f:
            js = f.read()

        # 1. Секции лендинга и якоря в HTML
        self.assertIn('id="landing-intro"', html)
        self.assertIn('id="mcp"', html)
        self.assertIn('id="about"', html)
        self.assertIn('href="#mcp"', html)
        self.assertIn('href="#about"', html)
        self.assertIn("Hello, this is", html)
        self.assertIn("Errorium", html)
        self.assertIn("Set up MCP", html)
        self.assertIn("About Errorium", html)
        self.assertIn("landing-mcp-token", html)
        self.assertIn("landing-mcp-json-snippet", html)
        self.assertIn("landing-standalone-page", html)
        self.assertIn("landing-nav-anchors", html)
        self.assertIn("btn-landing-hero-start", html)

        # 2. Стили лендинга в styles.css
        self.assertIn(".landing-content-wrap", css)
        self.assertIn(".landing-section", css)
        self.assertIn(".landing-standalone-page", css)
        self.assertIn(".landing-nav-anchors", css)

        # 3. Интеграция с роутингом и плавным скроллом в app.js
        self.assertIn("switchToLandingView", js)
        self.assertIn("landing-active", js)
        self.assertIn("btnLandingCopyToken", js)
        self.assertIn("btnLandingCopyJson", js)
        self.assertIn("'/landing'", js)
        self.assertIn("'/overview'", js)

    def test_app_js_syntax(self):
        """Проверка синтаксической валидности services/frontend/static/app.js через node --check."""
        js_path = os.path.join(STATIC_DIR, "app.js")
        self.assertTrue(os.path.isfile(js_path), f"Файл {js_path} должен существовать")

        node_bin = shutil.which("node")
        if node_bin:
            res = subprocess.run(
                [node_bin, "--check", js_path],
                capture_output=True,
                text=True,
            )
            self.assertEqual(
                res.returncode,
                0,
                f"Синтаксическая ошибка в {js_path}:\nSTDOUT:\n{res.stdout}\nSTDERR:\n{res.stderr}",
            )
        else:
            self.skipTest("Node.js не установлен в текущем окружении для запуска node --check")

if __name__ == "__main__":
    unittest.main()


