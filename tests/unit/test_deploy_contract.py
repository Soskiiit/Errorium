"""Статические проверки контракта автодеплоя (.github/workflows/deploy.yml <-> docker-compose.yml).

Прод-сервер получает только docker-compose.yml и deploy/ (без исходников),
поэтому каждый сервис прод-compose обязан иметь готовый image из GHCR,
а CI — собирать все сервисы с build:. Регрессия на падение деплоя
`resolve : lstat .../Errorium/services: no such file or directory`
(search_engine_core был добавлен в compose только с build:, без image и без CI-сборки).
"""
import base64
import os
import re
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))


def read(*parts):
    with open(os.path.join(REPO_ROOT, *parts), encoding="utf-8") as f:
        return f.read()


def service_blocks(compose_text):
    """Блоки сервисов внутри секции services: (якоря x-logging и volumes/networks не трогаем)."""
    section = re.search(r"(?ms)^services:\n(.*?)(?=^\S|\Z)", compose_text)
    if section is None:
        raise AssertionError("Секция services: не найдена в compose-файле")
    pattern = r"(?ms)^  ([a-z_]+):\n(.*?)(?=^  [a-z_]+:\n|\Z)"
    return {name: block for name, block in re.findall(pattern, section.group(1))}


class TestProdComposeImages(unittest.TestCase):
    def setUp(self):
        self.blocks = service_blocks(read("docker-compose.yml"))

    def test_expected_services_present(self):
        for name in ("postgres", "garage", "gateway", "search_engine_core", "frontend"):
            self.assertIn(name, self.blocks, f"Сервис {name} не найден в docker-compose.yml")

    def test_every_service_has_image(self):
        """На сервере нет исходников — compose не должен ничего собирать локально."""
        for name, block in self.blocks.items():
            self.assertRegex(block, r"(?m)^\s*image:\s*\S+", f"{name}: нет image: (сервер упадёт на build)")

    def test_built_images_follow_ghcr_naming(self):
        """Имена образов совпадают с шаблоном CI: errorium-<matrix.image>:develop."""
        workflow = read(".github", "workflows", "deploy.yml")
        for name, block in self.blocks.items():
            if "build:" not in block:
                continue
            match = re.search(r"image:\s*ghcr\.io/[^/]+/errorium-([a-z0-9-]+):develop", block)
            self.assertIsNotNone(match, f"{name}: image должен быть ghcr.io/<owner>/errorium-<name>:develop")
            suffix = match.group(1)
            self.assertIn(f"image: {suffix}", workflow, f"CI-матрица не собирает образ для {name} (image: {suffix})")


class TestWorkflowBuildMatrix(unittest.TestCase):
    def setUp(self):
        self.workflow = read(".github", "workflows", "deploy.yml")

    def test_core_dockerfile_in_matrix(self):
        self.assertIn("dockerfile: services/search_engine_core/Dockerfile", self.workflow)

    def test_build_step_uses_matrix_dockerfile(self):
        self.assertIn("file: ${{ matrix.service.dockerfile }}", self.workflow)

    def test_develop_tag_published(self):
        """Compose пинит тег :develop — CI обязан его публиковать."""
        self.assertIn("type=raw,value=develop", self.workflow)

    def test_scp_delivers_compose_and_deploy(self):
        self.assertRegex(self.workflow, r'source:\s*"docker-compose\.yml,deploy"')


class TestNucCertificate(unittest.TestCase):
    """Сертификат НУЦ Минцифры для GigaChat должен лежать в deploy/certs/ (едет на сервер через scp)."""

    CERT = os.path.join(REPO_ROOT, "deploy", "certs", "russian_trusted_root_ca.crt")

    def test_certificate_file_exists(self):
        self.assertTrue(os.path.isfile(self.CERT), "deploy/certs/russian_trusted_root_ca.crt отсутствует")

    def test_certificate_is_pem_root_ca(self):
        with open(self.CERT, encoding="utf-8") as f:
            pem = f.read()
        self.assertIn("-----BEGIN CERTIFICATE-----", pem)
        der = base64.b64decode("".join(
            line for line in pem.splitlines() if "CERTIFICATE" not in line
        ))
        self.assertIn(b"Russian Trusted Root CA", der)

    def test_env_example_points_to_shipped_file(self):
        text = read(".env.example")
        self.assertIn("GIGACHAT_CA_CERT=/app/certs/russian_trusted_root_ca.crt", text)


if __name__ == "__main__":
    unittest.main()