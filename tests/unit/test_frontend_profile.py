import os
import re
import shutil
import subprocess
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
STATIC_DIR = os.path.join(REPO_ROOT, "services", "frontend", "static")


class TestFrontendProfile(unittest.TestCase):
    """Тесты задачи C3: Профиль, аватар, теги, статус и SSO Яндекс ID."""

    def setUp(self):
        self.app_js_path = os.path.join(STATIC_DIR, "app.js")
        self.api_js_path = os.path.join(STATIC_DIR, "api.js")
        self.index_html_path = os.path.join(STATIC_DIR, "index.html")

        with open(self.app_js_path, "r", encoding="utf-8") as f:
            self.app_js = f.read()

        with open(self.api_js_path, "r", encoding="utf-8") as f:
            self.api_js = f.read()

        with open(self.index_html_path, "r", encoding="utf-8") as f:
            self.index_html = f.read()

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

    def test_profile_endpoints_get_and_patch(self):
        """Проверка вызовов GET /users/me и PATCH /users/me с обработкой ошибок 400 и 409."""
        # Наличие эндпоинта /users/me
        self.assertRegex(
            self.app_js,
            r"""['"`]/users/me['"`]""",
            "В app.js должен присутствовать вызов эндпоинта /users/me",
        )
        # Наличие методов GET и PATCH для /users/me
        self.assertIn("PATCH", self.app_js, "В app.js должен использоваться метод PATCH")
        # Проверка вызова GET при открытии профиля
        self.assertIn("loadUserProfile", self.app_js, "Должна быть функция loadUserProfile")
        self.assertIn("patchUserProfile", self.app_js, "Должна быть функция patchUserProfile")
        # Обработка конфликта (409) и невалидных данных (400)
        self.assertIn("409", self.app_js, "В app.js должна быть обработка ошибки 409 (email занят)")
        self.assertIn("400", self.app_js, "В app.js должна быть обработка ошибки 400")

    def test_avatar_upload_multipart(self):
        """Проверка multipart-загрузки аватара через POST /users/avatar."""
        # Эндпоинт загрузки аватара
        self.assertRegex(
            self.app_js,
            r"""['"`]/users/avatar['"`]""",
            "В app.js должен присутствовать вызов POST /users/avatar",
        )
        # Использование FormData для multipart/form-data
        self.assertIn("FormData", self.app_js, "Для загрузки аватара должен использоваться FormData")
        self.assertIn("'avatar'", self.app_js, "В FormData должно передаваться поле 'avatar'")
        # Обновление avatar_url и расчет glow
        self.assertIn("avatar_url", self.app_js, "Должно обрабатываться поле avatar_url из ответа")
        self.assertIn("extractDominantColor", self.app_js, "Должен вызываться расчет glow через extractDominantColor")
        # Обновление элементов аватара (в шапке, сайдбаре, профиле)
        self.assertIn("profileAvatarImg", self.app_js)
        self.assertIn("headerAvatarAuth", self.app_js)
        self.assertIn("sidebarAuthAvatar", self.app_js)
        # Обработка ошибок 502 и 400
        self.assertIn("502", self.app_js, "Должна быть обработка статуса 502 (S3 недоступен)")

    def test_tags_endpoint_and_dynamic_render(self):
        """Проверка запроса GET /tags и рендеринга в #tags-popover с data-tag."""
        self.assertRegex(
            self.app_js,
            r"""['"`]/tags['"`]""",
            "В app.js должен присутствовать вызов GET /tags",
        )
        self.assertIn("data-tag", self.app_js, "Динамический рендеринг тегов должен использовать атрибут data-tag")
        self.assertIn("tagsPopover", self.app_js, "Должно использоваться обращение к tagsPopover")
        # Функция рендеринга тегов
        self.assertIn("renderTagPills", self.app_js, "Должна присутствовать функция renderTagPills")
        self.assertIn("loadTags", self.app_js, "Должна присутствовать функция loadTags")
        # Статичные пилюли в index.html присутствуют как фолбэк
        self.assertIn("tag-pill", self.index_html, "В index.html должны присутствовать теги как фолбэк")

    def test_status_endpoint_and_settings_indicators(self):
        """Проверка запроса GET /status и индикаторов состояния ядра и RAG в модалке настроек."""
        self.assertRegex(
            self.app_js,
            r"""['"`]/status['"`]""",
            "В app.js должен присутствовать вызов GET /status",
        )
        # Проверка индикатора connected ядра
        self.assertIn("connected", self.app_js, "В app.js должна проверяться search_core.connected")
        # Текст состояния «Подключено» / «Недоступно»
        self.assertTrue(
            "Подключено" in self.app_js or "Connected" in self.app_js,
            "В app.js должен отображаться статус Подключено/Connected",
        )
        self.assertTrue(
            "Недоступно" in self.app_js or "Unavailable" in self.app_js,
            "В app.js должен отображаться статус Недоступно/Unavailable",
        )
        # Отображение RAG-провайдера (stub/openai/gigachat)
        self.assertIn("provider", self.app_js, "В app.js должно отображаться свойство provider из rag")
        self.assertIn("loadStatus", self.app_js, "Должна присутствовать функция loadStatus")
        self.assertIn("updateStatusUI", self.app_js, "Должна присутствовать функция updateStatusUI")

    def test_gender_and_language_enums(self):
        """Проверка строгого соответствия enum значениям openapi.yaml."""
        # Gender enum: Male, Female, Not specified
        self.assertIn("Male", self.app_js)
        self.assertIn("Female", self.app_js)
        self.assertIn("Not specified", self.app_js)

        # Language enum: English, Русский
        self.assertIn("English", self.app_js)
        self.assertIn("Русский", self.app_js)

        # Локальное переключение и сохранение языка
        self.assertIn("applyLanguage", self.app_js, "В app.js должна вызываться applyLanguage")

        # Profession
        self.assertIn("profession", self.app_js, "Профиль должен передавать поле profession в PATCH")

    def test_yandex_sso_and_callback(self):
        """Проверка рабочей кнопки Яндекс ID (редирект или ssoLogin) и обработки callback."""
        # Проверка метода ssoLogin в ErroriumAPI
        self.assertTrue(
            "ssoLogin" in self.api_js or "ssoLogin" in self.app_js,
            "Должен присутствовать вызов или реализация ssoLogin",
        )
        # Обработка Яндекс ID
        self.assertIn("yandex", self.app_js.lower(), "Должна быть обработка провайдера yandex")
        self.assertTrue(
            "/auth/sso/yandex/login" in self.app_js or "/auth/sso/login/yandex" in self.app_js or "ssoLogin('yandex')" in self.app_js,
            "Кнопка Яндекс ID должна инициировать редирект или вызывать ssoLogin('yandex')",
        )
        # Маршрут callback обрабатывает токены
        self.assertIn("/auth/callback", self.app_js, "Должна присутствовать обработка маршрута /auth/callback")
        self.assertIn("handleAuthCallback", self.app_js, "Должна быть функция handleAuthCallback")

        # Кнопки в разметке и коде
        self.assertIn("ref-btn-yandex", self.index_html)
        self.assertIn("ref-btn-register-yandex", self.index_html)
        self.assertIn("refBtnYandex", self.app_js)
        self.assertIn("refBtnRegisterYandex", self.app_js)


if __name__ == "__main__":
    unittest.main()
