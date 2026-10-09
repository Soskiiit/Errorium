import os
import re
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
DEV_COMPOSE_PATH = os.path.join(REPO_ROOT, "docker-compose.dev.yml")
ENV_EXAMPLE_PATH = os.path.join(REPO_ROOT, ".env.example")


class TestDockerPortsConfiguration(unittest.TestCase):
    """Тестирование гибкой настройки портов для локальной разработки в Docker."""

    def test_dev_compose_exists(self):
        self.assertTrue(os.path.isfile(DEV_COMPOSE_PATH))

    def test_env_example_has_port_variables(self):
        """Проверка наличия всех переменных портов в .env.example."""
        with open(ENV_EXAMPLE_PATH, "r", encoding="utf-8") as f:
            content = f.read()

        expected_vars = [
            "POSTGRES_PORT=5432",
            "GARAGE_PORT=3900",
            "GARAGE_ADMIN_PORT=3903",
            "GATEWAY_PORT=8000",
            "FRONTEND_PORT=3000",
        ]
        for var in expected_vars:
            self.assertIn(
                var,
                content,
                f"Переменная {var} должна присутствовать в .env.example",
            )

    def test_dev_compose_port_parametrization(self):
        """Проверка параметризации портов с дефолтными значениями в docker-compose.dev.yml."""
        with open(DEV_COMPOSE_PATH, "r", encoding="utf-8") as f:
            compose_text = f.read()

        # Garage S3 API port (allows changing port 3900)
        self.assertRegex(
            compose_text,
            r'["\']127\.0\.0\.1:\$\{GARAGE_PORT:-3900\}:3900["\']',
            "Порт Garage S3 API должен поддерживать переопределение через GARAGE_PORT с дефолтом 3900",
        )

        # Garage Admin / Metrics port (allows changing port 3903)
        self.assertRegex(
            compose_text,
            r'["\']127\.0\.0\.1:\$\{GARAGE_ADMIN_PORT:-3903\}:3903["\']',
            "Порт Garage Admin API должен поддерживать переопределение через GARAGE_ADMIN_PORT с дефолтом 3903",
        )

        # Postgres port (allows changing port 5432)
        self.assertRegex(
            compose_text,
            r'["\']127\.0\.0\.1:\$\{POSTGRES_PORT:-5432\}:5432["\']',
            "Порт Postgres должен поддерживать переопределение через POSTGRES_PORT с дефолтом 5432",
        )

        # Gateway port (allows changing port 8000)
        self.assertRegex(
            compose_text,
            r'["\']127\.0\.0\.1:\$\{GATEWAY_PORT:-8000\}:8000["\']',
            "Порт Gateway должен поддерживать переопределение через GATEWAY_PORT с дефолтом 8000",
        )

        # Frontend port (allows changing port 3000)
        self.assertRegex(
            compose_text,
            r'["\']\$\{FRONTEND_PORT:-3000\}:80["\']',
            "Порт Frontend должен поддерживать переопределение через FRONTEND_PORT с дефолтом 3000",
        )


if __name__ == "__main__":
    unittest.main()
