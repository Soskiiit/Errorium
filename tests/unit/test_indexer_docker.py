"""Статические проверки запуска индексатора в контейнере (docs/Tech/03-data.md, п. 4.10; AGENTS.md, п. 5).

Индексатор — офлайн-задача, поэтому у него отдельный compose-файл: основной
`docker-compose.yml` (автодеплой на `main`) не меняется.
"""
import os
import re
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))


def read(*parts):
    with open(os.path.join(REPO_ROOT, *parts), encoding="utf-8") as f:
        return f.read()


class TestIndexerDockerfile(unittest.TestCase):
    def setUp(self):
        self.text = read("services", "search_engine_indexer", "Dockerfile")

    def test_python_base_and_7zip(self):
        self.assertRegex(self.text, r"(?m)^FROM python:3\.")
        self.assertRegex(self.text, r"apk add[^\n]*\b7zip\b")

    def test_entrypoint_runs_indexer_module(self):
        self.assertIn("PYTHONPATH=/app/src", self.text)
        self.assertRegex(self.text, r'ENTRYPOINT \["python", "-m", "indexer"\]')

    def test_copies_source(self):
        self.assertRegex(self.text, r"(?m)^COPY src/ /app/src/")


class TestIndexerCompose(unittest.TestCase):
    def setUp(self):
        self.text = read("docker-compose.indexer.yml")

    def test_service_defined_and_built_from_indexer_dir(self):
        self.assertRegex(self.text, r"(?m)^  indexer:")
        self.assertIn("context: ./services/search_engine_indexer", self.text)

    def test_archive_mounted_read_only(self):
        self.assertRegex(self.text, r"\$\{SO_DUMP_DIR:-\./data/dump\}:/dump:ro")
        self.assertIn("SO_DUMP_ARCHIVE=/dump/${SO_DUMP_FILE:-stackoverflow.com.7z}", self.text)

    def test_data_and_benchmark_mounted(self):
        self.assertIn("./data:/app/data", self.text)
        self.assertRegex(self.text, r"\./benchmark:/app/benchmark(:ro)?")

    def test_memory_limit_matches_spec(self):
        self.assertRegex(self.text, r"mem_limit:\s*4g")

    def test_main_compose_untouched_by_indexer(self):
        self.assertNotIn("indexer", read("docker-compose.yml"))


class TestMakefileTargets(unittest.TestCase):
    def setUp(self):
        self.text = read("Makefile")

    def test_targets_declared_phony(self):
        phony = re.search(r"(?m)^\.PHONY:(.*)$", self.text).group(1)
        for target in ("index-sample", "index-corpus"):
            self.assertIn(target, phony.split())

    def test_runner_uses_separate_compose_file(self):
        self.assertRegex(self.text, r"(?m)^INDEXER_RUN = docker compose -f docker-compose\.indexer\.yml run --rm$")

    def test_sample_target_runs_in_container(self):
        self.assertRegex(self.text, r"(?ms)^index-sample:.*?\$\(INDEXER_RUN\) indexer sample")

    def test_corpus_target_uses_separate_index_dir(self):
        block = re.search(r"(?ms)^index-corpus:(.*?)(?=^\S)", self.text).group(1)
        self.assertIn("$(INDEXER_RUN)", block)
        self.assertIn("-e INDEX_OUTPUT_PATH=/app/data/index-benchmark", block)
        self.assertIn("indexer corpus", block)

    def test_benchmark_builds_corpus_index_first(self):
        self.assertRegex(self.text, r"(?m)^benchmark:\s*index-corpus\s*$")

    def test_help_mentions_targets(self):
        self.assertIn("make index-sample", self.text)
        self.assertIn("make index-corpus", self.text)


class TestEnvExample(unittest.TestCase):
    def test_indexer_variables_documented(self):
        text = read(".env.example")
        for var in ("SO_DUMP_DIR", "SO_DUMP_FILE", "SAMPLE_DENOM", "SAMPLE_MIN_SCORE",
                    "GRAPH_MAX_NEIGHBORS", "SPIMI_BLOCK_POSTINGS"):
            self.assertRegex(text, rf"(?m)^#?\s*{var}=", var)


if __name__ == "__main__":
    unittest.main()
