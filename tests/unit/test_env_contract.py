"""Статические проверки контракта переменных окружения (.env.example <-> docker-compose).

Имена переменных в .env.example должны совпадать с тем, что читают compose-файлы
и gateway (services/gateway/pkg/config/config.go). Регрессия на рассинхрон
DB_* / POSTGRES_* и проброс ключей SSO/LLM/SMTP в контейнер gateway.
"""
import os
import re
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))


def read(*parts):
    with open(os.path.join(REPO_ROOT, *parts), encoding="utf-8") as f:
        return f.read()


def gateway_block(compose_text):
    match = re.search(r"(?ms)^  gateway:\n(.*?)(?=^  [a-z_]+:\n|\Z)", compose_text)
    if match is None:
        raise AssertionError("Сервис gateway не найден в compose-файле")
    return match.group(1)


class TestEnvExampleNaming(unittest.TestCase):
    def setUp(self):
        self.text = read(".env.example")

    def test_postgres_credentials_documented(self):
        """Compose создаёт postgres из POSTGRES_*, а не из DB_*."""
        for var in ("POSTGRES_DB", "POSTGRES_USER", "POSTGRES_PASSWORD"):
            self.assertRegex(self.text, rf"(?m)^{var}=", f"{var} должна быть в .env.example")

    def test_sso_redirect_base_documented(self):
        self.assertRegex(self.text, r"(?m)^SSO_REDIRECT_BASE=")

    def test_frontend_urls_documented(self):
        self.assertRegex(self.text, r"(?m)^FRONTEND_URL=")
        self.assertRegex(self.text, r"(?m)^FRONTEND_HTTP_PORT=")
        self.assertRegex(self.text, r"(?m)^FRONTEND_HTTPS_PORT=")

    def test_gateway_key_variables_documented(self):
        """Переменные, которые gateway читает через os.Getenv, должны быть в примере."""
        for var in (
            "JWT_SECRET_KEY", "ACCESS_TOKEN_EXPIRE_MINUTES", "REFRESH_TOKEN_EXPIRE_DAYS",
            "FIRST_ADMIN_EMAIL", "FIRST_ADMIN_PASSWORD",
            "SMTP_HOST", "SMTP_PORT", "SMTP_USER", "SMTP_PASSWORD", "SMTP_FROM",
            "LLM_PROVIDER", "LLM_TEMPERATURE", "LLM_MAX_TOKENS", "LLM_MAX_CONCURRENT",
            "RAG_SYSTEM_PROMPT", "OPENAI_BASE_URL", "OPENAI_API_KEY", "OPENAI_MODEL_NAME",
            "GIGACHAT_CLIENT_ID", "GIGACHAT_CLIENT_SECRET", "GIGACHAT_SCOPE",
            "GIGACHAT_OAUTH_URL", "GIGACHAT_BASE_URL", "GIGACHAT_MODEL", "GIGACHAT_CA_CERT",
            "MCP_MAX_SESSIONS_PER_IP", "MCP_MAX_SESSIONS", "MCP_CALLS_PER_MIN",
            "YANDEX_CLIENT_ID", "GOOGLE_CLIENT_ID", "GITHUB_CLIENT_ID",
        ):
            self.assertRegex(self.text, rf"(?m)^{var}=", f"{var} должна быть в .env.example")

    def test_gigachat_ca_cert_is_container_path(self):
        """Gateway читает файл внутри контейнера: путь должен вести в /app/certs."""
        self.assertRegex(self.text, r"(?m)^GIGACHAT_CA_CERT=/app/certs/")

    def test_no_duplicate_active_assignments(self):
        for var in ("SEARCH_GRPC_PORT", "POSTGRES_PASSWORD", "SSO_REDIRECT_BASE"):
            found = re.findall(rf"(?m)^{var}=", self.text)
            self.assertEqual(len(found), 1, f"{var} присваивается {len(found)} раз в .env.example")

    def test_dev_gateway_api_url_is_localhost(self):
        """`make dev` запускается на хосте, имя gateway там не резолвится."""
        self.assertRegex(self.text, r"(?m)^GATEWAY_API_URL=http://localhost:8000")


class TestComposeEnvPassthrough(unittest.TestCase):
    """Ключи из .env должны доходить до контейнера gateway (env_file)."""

    def test_gateway_env_file_in_both_composes(self):
        for name in ("docker-compose.yml", "docker-compose.dev.yml"):
            block = gateway_block(read(name))
            self.assertRegex(block, r"(?m)^\s*env_file:\s*\.env\s*$", f"{name}: gateway без env_file: .env")

    def test_gateway_certs_mounted_in_both_composes(self):
        for name in ("docker-compose.yml", "docker-compose.dev.yml"):
            block = gateway_block(read(name))
            self.assertIn("./deploy/certs:/app/certs:ro", block, f"{name}: gateway без монтирования deploy/certs")

    def test_postgres_uses_postgres_variables(self):
        for name in ("docker-compose.yml", "docker-compose.dev.yml"):
            text = read(name)
            self.assertIn("${POSTGRES_PASSWORD", text, f"{name}: postgres должен читать POSTGRES_PASSWORD")


if __name__ == "__main__":
    unittest.main()