"""Unit-тесты конфигурации, источников и конвейера индексатора (docs/Tech/03-data.md, п. 4.5, 4.9, 4.10).

Используются маленькие синтетические «дампы» с BOM, CRLF и служебными строками; архив 7z
подменяется скриптом-заглушкой `7zz`, чтобы проверить чтение потока без реального архива.
"""
import hashlib
import json
import os
import stat
import sys
import tempfile
import unittest
from unittest import mock

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
sys.path.insert(0, os.path.join(REPO_ROOT, "services", "search_engine_indexer", "src"))

from indexer.__main__ import main  # noqa: E402
from indexer.config import Config  # noqa: E402
from indexer.index_format import read_doc_lengths, read_lexicon, read_postings  # noqa: E402
from indexer.pipeline import run_corpus, run_sample  # noqa: E402
from indexer.sampler import in_sample  # noqa: E402
from indexer.sources import open_member  # noqa: E402

IN_S = [i for i in range(1, 5000) if in_sample(i, 100, 0)]
OUT_S = [i for i in range(1, 5000) if not in_sample(i, 100, 0)]


# ------------------------------------------------------------------ фикстуры

def row(**attrs):
    return "  <row " + " ".join(f'{k}="{v}"' for k, v in attrs.items()) + " />"


def question(qid, accepted=None, score=5, title="Title", body="Body", tags="&lt;python&gt;"):
    attrs = dict(Id=qid, PostTypeId=1, CreationDate="2008-08-07T00:45:51.980", Score=score)
    if accepted is not None:
        attrs["AcceptedAnswerId"] = accepted
    attrs.update(Body=body, Title=title, Tags=tags)
    return row(**attrs)


def answer(aid, parent, score=3, body="Answer"):
    return row(Id=aid, PostTypeId=2, ParentId=parent, CreationDate="2008-08-08T10:00:00.000",
               Score=score, Body=body)


def link(lid, post, related, kind):
    return row(Id=lid, CreationDate="2009-01-01T00:00:00.000", PostId=post, RelatedPostId=related,
               LinkTypeId=kind)


def xml_file(rows, root="posts", bom=True):
    head = ('<?xml version="1.0" encoding="utf-8"?>\r\n<!--\r\n  ContentLicense\r\n  -->\r\n'
            f"<{root}>\r\n")
    body = "".join(r + "\r\n" for r in rows)
    return ((("﻿" if bom else "") + head + body + f"</{root}>\r\n")).encode("utf-8")


FULL_POSTS = [
    question(100, accepted=101, score=5, title="NullPointerException in Java",
             body="&lt;p&gt;Why does &lt;code&gt;os.path.join&lt;/code&gt; fail with C++?&lt;/p&gt;"),
    answer(101, 100, score=7, body="&lt;p&gt;Use &lt;code&gt;pathlib&lt;/code&gt; instead&lt;/p&gt;"),
    question(200, accepted=201, score=3, title="Docker compose error", tags="&lt;docker&gt;&lt;docker-compose&gt;"),
    answer(201, 200, score=2, body="&lt;p&gt;restart the daemon&lt;/p&gt;"),
    answer(202, 200, score=50, body="&lt;p&gt;unrelated nonaccepted words&lt;/p&gt;"),
    question(300, score=9, title="No accepted answer here"),
    answer(301, 300),
    question(400, accepted=401, score=0, title="Zero score question"),
    answer(401, 400),
    row(Id=9000, PostTypeId=5, CreationDate="2008-01-01T00:00:00.000", Score=0, Body="wiki"),
]
FULL_LINKS = [
    link(1, 100, 200, 3),   # DUPLICATE_OF
    link(2, 200, 100, 1),   # LINKED_IN_ANSWER
    link(3, 100, 300, 1),   # 300 не валиден (нет принятого ответа)
    link(4, 100, 400, 3),   # 400 отброшен по Score
    link(5, 100, 999, 1),   # неизвестный тред
    link(6, 100, 200, 5),   # неизвестный тип связи
]


class FixtureCase(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self._tmp.cleanup)
        self.root = self._tmp.name

    def path(self, *parts):
        return os.path.join(self.root, *parts)

    def make_dump(self, name="dump", posts=None, links=None):
        d = self.path(name)
        os.makedirs(d, exist_ok=True)
        files = {"Posts.xml": xml_file(FULL_POSTS if posts is None else posts),
                 "PostLinks.xml": xml_file(FULL_LINKS if links is None else links, root="postlinks"),
                 "Tags.xml": xml_file([row(Id=1, TagName="python", Count=5)], root="tags")}
        for fname, data in files.items():
            with open(os.path.join(d, fname), "wb") as f:
                f.write(data)
        return d

    def config(self, raw, out=None, **env):
        base = {"RAW_DATA_PATH": raw, "INDEX_OUTPUT_PATH": out or self.path("index"),
                "SAMPLE_DENOM": "1", "SAMPLE_MIN_SCORE": "1"}
        base.update({k: str(v) for k, v in env.items()})
        return Config.from_env(base)

    @staticmethod
    def read(path):
        with open(path, "rb") as f:
            return f.read()


# ------------------------------------------------------------------ Config

class TestConfig(unittest.TestCase):
    def test_defaults(self):
        cfg = Config.from_env({})
        self.assertEqual(cfg.raw_data_path, "/app/data/raw")
        self.assertEqual(cfg.index_output_path, "/app/data/index")
        self.assertIsNone(cfg.so_dump_archive)
        self.assertEqual(cfg.corpus_path, "benchmark/corpus/documents.jsonl")
        self.assertEqual((cfg.sample_denom, cfg.sample_residue, cfg.sample_min_score), (100, 0, 1))
        self.assertEqual(cfg.graph_max_neighbors, 20)
        self.assertEqual(cfg.spimi_block_postings, 2_000_000)
        self.assertEqual(cfg.sevenzip_bin, "7zz")
        self.assertEqual(cfg.shard_count, 64)

    def test_overrides(self):
        cfg = Config.from_env({"SO_DUMP_ARCHIVE": "/dump/so.7z", "SAMPLE_DENOM": "10", "SAMPLE_RESIDUE": "3",
                               "SAMPLE_MIN_SCORE": "0", "GRAPH_MAX_NEIGHBORS": "5",
                               "SPIMI_BLOCK_POSTINGS": "1000", "SEVENZIP_BIN": "/opt/7zz"})
        self.assertEqual(cfg.so_dump_archive, "/dump/so.7z")
        self.assertEqual((cfg.sample_denom, cfg.sample_residue, cfg.sample_min_score), (10, 3, 0))
        self.assertEqual((cfg.graph_max_neighbors, cfg.spimi_block_postings), (5, 1000))
        self.assertEqual(cfg.sevenzip_bin, "/opt/7zz")

    def test_empty_archive_means_plain_files(self):
        self.assertIsNone(Config.from_env({"SO_DUMP_ARCHIVE": ""}).so_dump_archive)

    def test_invalid_values_rejected(self):
        for env in ({"SAMPLE_DENOM": "0"}, {"SAMPLE_DENOM": "abc"}, {"SAMPLE_DENOM": "10", "SAMPLE_RESIDUE": "10"},
                    {"SAMPLE_RESIDUE": "-1"}, {"GRAPH_MAX_NEIGHBORS": "-1"}, {"SPIMI_BLOCK_POSTINGS": "0"}):
            with self.subTest(env=env):
                with self.assertRaises(ValueError):
                    Config.from_env(env)


# ------------------------------------------------------------------ источники

class TestSources(FixtureCase):
    def fake_7zz(self):
        script = self.path("fake7zz")
        with open(script, "w") as f:
            f.write('#!/bin/sh\n'
                    'if [ $# -eq 0 ]; then echo "7-Zip (z) FAKE 0.0"; exit 0; fi\n'
                    '# args: x -so ARCHIVE MEMBER\n'
                    'if [ "$FAKE_7ZZ_FAIL" = "1" ]; then head -c 20 "$(dirname "$3")/$4"; echo boom >&2; exit 2; fi\n'
                    'cat "$(dirname "$3")/$4"\n')
        os.chmod(script, os.stat(script).st_mode | stat.S_IEXEC)
        return script

    def test_plain_file_member(self):
        raw = self.make_dump()
        with open_member(self.config(raw), "Posts.xml") as stream:
            data = b"".join(stream)
        self.assertEqual(data, xml_file(FULL_POSTS))

    def test_missing_plain_file_has_clear_error(self):
        with self.assertRaises(FileNotFoundError):
            with open_member(self.config(self.path("nothing")), "Posts.xml") as stream:
                list(stream)

    def test_archive_member_streamed_via_7zz(self):
        raw = self.make_dump()
        archive = os.path.join(raw, "so.7z")
        open(archive, "wb").close()
        cfg = self.config(raw, SO_DUMP_ARCHIVE=archive, SEVENZIP_BIN=self.fake_7zz())
        with open_member(cfg, "Posts.xml") as stream:
            data = b"".join(stream)
        self.assertEqual(data, xml_file(FULL_POSTS))

    def test_nonzero_exit_of_7zz_raises(self):
        raw = self.make_dump()
        archive = os.path.join(raw, "so.7z")
        open(archive, "wb").close()
        cfg = self.config(raw, SO_DUMP_ARCHIVE=archive, SEVENZIP_BIN=self.fake_7zz())
        with mock.patch.dict(os.environ, {"FAKE_7ZZ_FAIL": "1"}):
            with self.assertRaises(RuntimeError) as ctx:
                with open_member(cfg, "Posts.xml") as stream:
                    list(stream)
        self.assertIn("boom", str(ctx.exception))


# ------------------------------------------------------------------ sample

class TestRunSample(FixtureCase):
    def run_sample(self, raw=None, out=None, **env):
        raw = raw or self.make_dump()
        cfg = self.config(raw, out, **env)
        manifest = run_sample(cfg, log=lambda *_: None, timestamp=1700000000)
        return cfg, manifest

    def index_dir(self, cfg):
        return cfg.index_output_path

    def test_publishes_all_artifacts(self):
        cfg, _ = self.run_sample()
        self.assertEqual(sorted(os.listdir(self.index_dir(cfg))),
                         ["doc_lengths.bin", "graph.db", "lexicon.bin", "metadata.json",
                          "postings.bin", "sample_manifest.json"])

    def test_only_valid_threads_indexed(self):
        cfg, _ = self.run_sample()
        d = self.index_dir(cfg)
        self.assertEqual(sorted(read_doc_lengths(os.path.join(d, "doc_lengths.bin"))), [100, 200])
        with open(os.path.join(d, "metadata.json"), encoding="utf-8") as f:
            meta = json.load(f)
        self.assertEqual((meta["index_mode"], meta["num_documents"]), ("sample", 2))

    def test_text_of_question_and_accepted_answer_is_indexed_but_not_other_answers(self):
        cfg, _ = self.run_sample()
        d = self.index_dir(cfg)
        lex = read_lexicon(os.path.join(d, "lexicon.bin"))
        postings = os.path.join(d, "postings.bin")
        for term in ("nullpointerexception", "null", "os.path.join", "c++", "pathlib",   # тред 100
                     "compose", "daemon"):                                              # тред 200
            self.assertIn(term, lex, term)
        self.assertEqual([doc for doc, _ in read_postings(postings, lex["pathlib"])], [100])
        self.assertEqual([doc for doc, _ in read_postings(postings, lex["daemon"])], [200])
        self.assertNotIn("nonaccepted", lex)   # неприметый ответ не индексируется
        self.assertNotIn("zero", lex)           # тред с Score 0 отфильтрован

    def test_graph_db_nodes_and_edges(self):
        import sqlite3
        cfg, _ = self.run_sample()
        conn = sqlite3.connect(f"file:{os.path.join(self.index_dir(cfg), 'graph.db')}?mode=ro", uri=True)
        self.addCleanup(conn.close)
        self.assertEqual([r[0] for r in conn.execute("SELECT doc_id FROM thread_nodes ORDER BY doc_id")], [100, 200])
        self.assertEqual(conn.execute("SELECT source_doc_id, target_doc_id, link_type FROM thread_links "
                                      "ORDER BY 1, 2, 3").fetchall(),
                         [(100, 200, "DUPLICATE_OF"), (200, 100, "LINKED_IN_ANSWER")])
        title, accepted, body = conn.execute("SELECT title, accepted_answer_id, best_answer_body "
                                             "FROM thread_nodes WHERE doc_id = 100").fetchone()
        self.assertEqual((title, accepted), ("NullPointerException in Java", 101))
        self.assertEqual(body, "<p>Use <code>pathlib</code> instead</p>")

    def test_sample_file_is_normalized_and_filtered(self):
        raw = self.make_dump()
        cfg, _ = self.run_sample(raw=raw)
        data = self.read(os.path.join(raw, "sample", "Posts.sample.xml"))
        self.assertFalse(data.startswith(b"\xef\xbb\xbf"))
        self.assertNotIn(b"\r", data)
        self.assertNotIn(b'PostTypeId="5"', data)
        self.assertEqual(data.count(b"<row "), 9)   # 4 вопроса + 5 ответов; wiki отброшен

    def test_manifest_contents(self):
        raw = self.make_dump()
        cfg, manifest = self.run_sample(raw=raw)
        with open(os.path.join(self.index_dir(cfg), "sample_manifest.json"), encoding="utf-8") as f:
            on_disk = json.load(f)
        self.assertEqual(on_disk, manifest)
        sample_file = os.path.join(raw, "sample", "Posts.sample.xml")
        self.assertEqual(manifest["sample_denom"], 1)
        self.assertEqual(manifest["sample_min_score"], 1)
        self.assertEqual(manifest["questions_selected"], 4)
        self.assertEqual(manifest["answers_selected"], 5)
        self.assertEqual(manifest["valid_threads"], 2)
        self.assertEqual(manifest["edges"], 2)
        self.assertEqual(manifest["posts_sample_sha256"], hashlib.sha256(self.read(sample_file)).hexdigest())
        self.assertEqual(manifest["posts_sample_bytes"], os.path.getsize(sample_file))
        self.assertEqual(manifest["build_timestamp"], 1700000000)
        self.assertIsNone(manifest["archive"])

    def test_tags_copied(self):
        raw = self.make_dump()
        self.run_sample(raw=raw)
        self.assertEqual(self.read(os.path.join(raw, "sample", "Tags.xml")), xml_file(
            [row(Id=1, TagName="python", Count=5)], root="tags"))

    def test_deterministic(self):
        raw = self.make_dump()
        cfg1, m1 = self.run_sample(raw=raw, out=self.path("idx1"))
        sample1 = self.read(os.path.join(raw, "sample", "Posts.sample.xml"))
        cfg2, m2 = self.run_sample(raw=raw, out=self.path("idx2"))
        self.assertEqual(sample1, self.read(os.path.join(raw, "sample", "Posts.sample.xml")))
        self.assertEqual(m1, m2)
        for name in ("lexicon.bin", "postings.bin", "doc_lengths.bin", "metadata.json", "sample_manifest.json"):
            self.assertEqual(self.read(os.path.join(cfg1.index_output_path, name)),
                             self.read(os.path.join(cfg2.index_output_path, name)), name)

    def test_archive_mode_matches_plain_mode(self):
        raw = self.make_dump()
        archive = os.path.join(raw, "so.7z")
        open(archive, "wb").close()
        fake = TestSources.fake_7zz(self)
        plain_cfg, plain = self.run_sample(raw=raw, out=self.path("plain"))
        arch_cfg, arch = self.run_sample(raw=self.path("work"), out=self.path("arch"),
                                         SO_DUMP_ARCHIVE=archive, SEVENZIP_BIN=fake)
        self.assertEqual(arch["archive"]["name"], "so.7z")
        self.assertEqual(arch["archive"]["bytes"], 0)
        self.assertIn("FAKE", arch["archive"]["sevenzip_version"])
        for name in ("lexicon.bin", "postings.bin", "doc_lengths.bin"):
            self.assertEqual(self.read(os.path.join(plain_cfg.index_output_path, name)),
                             self.read(os.path.join(arch_cfg.index_output_path, name)), name)

    def test_failed_7zz_publishes_nothing(self):
        raw = self.make_dump()
        archive = os.path.join(raw, "so.7z")
        open(archive, "wb").close()
        out = self.path("index")
        with mock.patch.dict(os.environ, {"FAKE_7ZZ_FAIL": "1"}):
            with self.assertRaises(RuntimeError):
                self.run_sample(raw=self.path("work"), out=out, SO_DUMP_ARCHIVE=archive,
                                SEVENZIP_BIN=TestSources.fake_7zz(self))
        self.assertFalse(os.path.exists(os.path.join(out, "metadata.json")))


class TestNeighborExpansion(FixtureCase):
    """Выборка 1/100: тред вне выборки попадает в корпус как сосед по связи."""

    def setUp(self):
        super().setUp()
        self.s, self.o, self.x = IN_S[0], OUT_S[0], OUT_S[1]
        posts = []
        for qid, aid in ((self.s, 7_000_001), (self.o, 7_000_002), (self.x, 7_000_003)):
            posts += [question(qid, accepted=aid, score=4, title=f"thread {qid}"), answer(aid, qid)]
        self.posts = posts
        self.links = [link(1, self.s, self.o, 3)]

    def docs(self, **env):
        raw = self.make_dump(posts=self.posts, links=self.links)
        cfg = self.config(raw, SAMPLE_DENOM=100, SAMPLE_RESIDUE=0, **env)
        run_sample(cfg, log=lambda *_: None, timestamp=0)
        return sorted(read_doc_lengths(os.path.join(cfg.index_output_path, "doc_lengths.bin"))), cfg

    def test_neighbor_added_and_unlinked_thread_excluded(self):
        docs, cfg = self.docs()
        self.assertEqual(docs, sorted([self.s, self.o]))
        import sqlite3
        conn = sqlite3.connect(f"file:{os.path.join(cfg.index_output_path, 'graph.db')}?mode=ro", uri=True)
        self.addCleanup(conn.close)
        self.assertEqual(conn.execute("SELECT source_doc_id, target_doc_id, link_type FROM thread_links").fetchall(),
                         [(self.s, self.o, "DUPLICATE_OF")])

    def test_zero_neighbor_cap_disables_expansion(self):
        docs, _ = self.docs(GRAPH_MAX_NEIGHBORS=0)
        self.assertEqual(docs, [self.s])


# ------------------------------------------------------------------ corpus

class TestRunCorpus(FixtureCase):
    DOCS = [
        {"doc_id": 1, "title": "NullPointerException", "text": "Calling List<String> methods on null.", "tags": ["java"]},
        {"doc_id": 2, "text": "Use os.path.join to build paths in C++ or Python."},
        {"doc_id": 3, "title": "Docker", "text": "docker-compose up fails"},
    ]

    def write_corpus(self, docs=None, name="documents.jsonl", extra_lines=()):
        path = self.path(name)
        with open(path, "w", encoding="utf-8") as f:
            for d in (TestRunCorpus.DOCS if docs is None else docs):
                f.write(json.dumps(d, ensure_ascii=False) + "\n")
            for line in extra_lines:
                f.write(line + "\n")
        return path

    def cfg(self, corpus, out=None):
        return Config.from_env({"CORPUS_PATH": corpus, "INDEX_OUTPUT_PATH": out or self.path("bench-index")})

    def test_index_built_from_jsonl(self):
        cfg = self.cfg(self.write_corpus())
        meta = run_corpus(cfg, log=lambda *_: None, timestamp=5)
        d = cfg.index_output_path
        self.assertEqual((meta["index_mode"], meta["num_documents"], meta["build_timestamp"]), ("corpus", 3, 5))
        self.assertEqual(sorted(read_doc_lengths(os.path.join(d, "doc_lengths.bin"))), [1, 2, 3])
        lex = read_lexicon(os.path.join(d, "lexicon.bin"))
        for term in ("nullpointerexception", "null", "list", "string", "os.path.join", "c++", "docker-compose"):
            self.assertIn(term, lex, term)
        self.assertEqual(sorted(os.listdir(d)),
                         ["doc_lengths.bin", "graph.db", "lexicon.bin", "metadata.json", "postings.bin"])

    def test_text_is_plain_not_html_cleaned(self):
        cfg = self.cfg(self.write_corpus([{"doc_id": 1, "text": "List<String> x"}]))
        run_corpus(cfg, log=lambda *_: None)
        lex = read_lexicon(os.path.join(cfg.index_output_path, "lexicon.bin"))
        self.assertIn("list", lex)
        self.assertIn("string", lex)   # `<String>` не принят за HTML-тег

    def test_graph_db_has_nodes_without_edges(self):
        import sqlite3
        cfg = self.cfg(self.write_corpus())
        run_corpus(cfg, log=lambda *_: None)
        conn = sqlite3.connect(f"file:{os.path.join(cfg.index_output_path, 'graph.db')}?mode=ro", uri=True)
        self.addCleanup(conn.close)
        self.assertEqual(conn.execute("SELECT COUNT(*) FROM thread_links").fetchone(), (0,))
        rows = conn.execute("SELECT doc_id, has_accepted_answer, tags FROM thread_nodes ORDER BY doc_id").fetchall()
        self.assertEqual(rows, [(1, 0, '["java"]'), (2, 0, "[]"), (3, 0, "[]")])

    def test_source_recorded_and_deterministic(self):
        corpus = self.write_corpus()
        cfg1, cfg2 = self.cfg(corpus, self.path("i1")), self.cfg(corpus, self.path("i2"))
        m1 = run_corpus(cfg1, log=lambda *_: None, timestamp=0)
        m2 = run_corpus(cfg2, log=lambda *_: None, timestamp=0)
        self.assertEqual(m1, m2)
        self.assertEqual(m1["source"]["corpus_file"], "documents.jsonl")
        self.assertEqual(m1["source"]["corpus_sha256"], hashlib.sha256(self.read(corpus)).hexdigest())
        for name in ("lexicon.bin", "postings.bin", "doc_lengths.bin", "metadata.json"):
            self.assertEqual(self.read(os.path.join(cfg1.index_output_path, name)),
                             self.read(os.path.join(cfg2.index_output_path, name)), name)

    def test_blank_lines_ignored(self):
        cfg = self.cfg(self.write_corpus(extra_lines=["", "   "]))
        self.assertEqual(run_corpus(cfg, log=lambda *_: None)["num_documents"], 3)

    def test_invalid_input_rejected_with_line_number(self):
        cases = {
            "not json": ["{broken"],
            "no text": [json.dumps({"doc_id": 9})],
            "no doc_id": [json.dumps({"text": "x"})],
            "string doc_id": [json.dumps({"doc_id": "D01", "text": "x"})],
            "zero doc_id": [json.dumps({"doc_id": 0, "text": "x"})],
            "bool doc_id": [json.dumps({"doc_id": True, "text": "x"})],
            "tags not list": [json.dumps({"doc_id": 9, "text": "x", "tags": "java"})],
        }
        for label, lines in cases.items():
            with self.subTest(label):
                corpus = self.write_corpus(extra_lines=lines, name=f"{label.replace(' ', '_')}.jsonl")
                with self.assertRaises(ValueError) as ctx:
                    run_corpus(self.cfg(corpus, self.path("bad-" + label.replace(" ", "_"))), log=lambda *_: None)
                self.assertIn(":4:", str(ctx.exception))

    def test_duplicate_doc_id_rejected(self):
        corpus = self.write_corpus([{"doc_id": 1, "text": "a b"}, {"doc_id": 1, "text": "c d"}])
        with self.assertRaises(ValueError):
            run_corpus(self.cfg(corpus), log=lambda *_: None)

    def test_missing_corpus_file(self):
        with self.assertRaises(FileNotFoundError):
            run_corpus(self.cfg(self.path("missing.jsonl")), log=lambda *_: None)


# ------------------------------------------------------------------ CLI

class TestCli(FixtureCase):
    def test_corpus_command(self):
        corpus = TestRunCorpus.write_corpus(self)
        out = self.path("cli-index")
        env = {"CORPUS_PATH": corpus, "INDEX_OUTPUT_PATH": out}
        self.assertEqual(main(["corpus"], env=env, log=lambda *_: None), 0)
        self.assertTrue(os.path.exists(os.path.join(out, "metadata.json")))

    def test_sample_command(self):
        raw = self.make_dump()
        out = self.path("cli-sample")
        env = {"RAW_DATA_PATH": raw, "INDEX_OUTPUT_PATH": out, "SAMPLE_DENOM": "1"}
        self.assertEqual(main(["sample"], env=env, log=lambda *_: None), 0)
        self.assertTrue(os.path.exists(os.path.join(out, "sample_manifest.json")))

    def test_runtime_error_returns_1_and_publishes_nothing(self):
        out = self.path("cli-bad")
        env = {"CORPUS_PATH": self.path("missing.jsonl"), "INDEX_OUTPUT_PATH": out}
        self.assertEqual(main(["corpus"], env=env, log=lambda *_: None), 1)
        self.assertFalse(os.path.exists(os.path.join(out, "metadata.json")))

    def test_invalid_config_returns_1(self):
        self.assertEqual(main(["sample"], env={"SAMPLE_DENOM": "0"}, log=lambda *_: None), 1)

    def test_usage_errors_exit_2(self):
        for argv in ([], ["bogus"]):
            with self.subTest(argv=argv):
                with mock.patch("sys.stderr"):
                    with self.assertRaises(SystemExit) as ctx:
                        main(argv, env={}, log=lambda *_: None)
                self.assertEqual(ctx.exception.code, 2)


if __name__ == "__main__":
    unittest.main()
