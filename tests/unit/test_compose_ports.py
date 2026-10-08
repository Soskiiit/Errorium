from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parents[2]


class ComposePortContractTest(unittest.TestCase):
    def test_production_host_ports_come_from_environment(self):
        compose = (ROOT / "docker-compose.yml").read_text(encoding="utf-8")
        self.assertIn('${GATEWAY_PORT:-8000}:8000', compose)
        self.assertIn('${FRONTEND_HTTP_PORT:-80}:80', compose)
        self.assertIn('${FRONTEND_HTTPS_PORT:-443}:443', compose)

    def test_frontend_dev_command_and_port_exist(self):
        makefile = (ROOT / "Makefile").read_text(encoding="utf-8")
        dev_compose = (ROOT / "docker-compose.dev.yml").read_text(encoding="utf-8")
        self.assertIn("frontend-dev-server:", makefile)
        self.assertIn('${FRONTEND_DEV_PORT:-3000}:8080', dev_compose)
        self.assertIn("./services/frontend/static:/app:ro", dev_compose)
        self.assertIn("name: errorium-frontend-dev", dev_compose)
        self.assertNotIn("container_name:", dev_compose)
        dockerfile = (ROOT / "services" / "frontend" / "Dockerfile.dev").read_text(encoding="utf-8")
        self.assertIn("browser-sync@3.0.4", dockerfile)


if __name__ == "__main__":
    unittest.main()
