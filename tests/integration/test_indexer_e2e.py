"""Интеграционные тесты индексатора: запуск как отдельного процесса (`python -m indexer ...`) и
проверка артефактов «глазами читателя» — эталонным расчётом BM25 по формуле методички
(docs/Tech/03-data.md; AGENTS.md, п. 2.1). Расчёт живёт только в тесте и не использует поисковых
библиотек; он нужен, чтобы убедиться: N, avgdl, df, tf и длины в индексе согласованы между собой.
"""
import json
import math
import os
import subprocess
import sys
import tempfile
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
SRC = os.path.join(REPO_ROOT, "services", "search_engine_indexer", "src")
sys.path.insert(0, SRC)

from indexer.index_format import read_doc_lengths, read_lexicon, read_postings  # noqa: E402
from indexer.tokenizer import tokenize  # noqa: E402

K1, B = 1.2, 0.75


def run_indexer(command, env_extra, cwd):
    env = {k: v for k, v in os.environ.items() if k in ("PATH", "HOME", "LANG", "LC_ALL", "SYSTEMROOT")}
    env["PYTHONPATH"] = SRC
    env.update(env_extra)
    return subprocess.run([sys.executable, "-m", "indexer", command], env=env, cwd=cwd,
                          capture_output=True, text=True, timeout=120)


def bm25_search(index_dir, query, k=10):
    """Эталонный BM25 по файлам индекса (IDF = ln(1 + (N-df+.5)/(df+.5)))."""
    lexicon = read_lexicon(os.path.join(index_dir, "lexicon.bin"))
    lengths = read_doc_lengths(os.path.join(index_dir, "doc_lengths.bin"))
    with open(os.path.join(index_dir, "metadata.json"), encoding="utf-8") as f:
        meta = json.load(f)
    n, avgdl = meta["num_documents"], meta["avg_doc_length"]
    scores = {}
    for term in dict.fromkeys(tokenize(query)):
        entry = lexicon.get(term)
        if entry is None:
            continue
        idf = math.log(1 + (n - entry.df + 0.5) / (entry.df + 0.5))
        for doc_id, tf in read_postings(os.path.join(index_dir, "postings.bin"), entry):
            norm = K1 * (1 - B + B * lengths[doc_id] / avgdl)
            scores[doc_id] = scores.get(doc_id, 0.0) + idf * tf * (K1 + 1) / (tf + norm)
    ranked = sorted(scores, key=lambda d: (-scores[d], d))[:k]
    return [(d, scores[d]) for d in ranked]


class TestCorpusEndToEnd(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self._tmp.cleanup)
        self.root = self._tmp.name

    def write_corpus(self, docs):
        path = os.path.join(self.root, "documents.jsonl")
        with open(path, "w", encoding="utf-8") as f:
            for d in docs:
                f.write(json.dumps(d, ensure_ascii=False) + "\n")
        return path

    def build(self, docs):
        out = os.path.join(self.root, "index")
        result = run_indexer("corpus", {"CORPUS_PATH": self.write_corpus(docs), "INDEX_OUTPUT_PATH": out}, self.root)
        self.assertEqual(result.returncode, 0, result.stderr)
        return out

    def test_lesson3_example_reproduced_from_index_files(self):
        """Пример занятия 3: возврат оплаты. N=3, avgdl=4, IDF(возврат)=ln(1.6)≈0.470, A=0.940."""
        out = self.build([
            {"doc_id": 1, "text": "Возврат оплаты на карту"},
            {"doc_id": 2, "text": "Статус оплаты через приложение"},
            {"doc_id": 3, "text": "Возврат билета через кассу"},
        ])
        hits = bm25_search(out, "Как происходит возврат оплаты?")
        self.assertEqual([d for d, _ in hits], [1, 2, 3])
        idf = math.log(1.6)
        self.assertAlmostEqual(idf, 0.470, places=3)
        self.assertAlmostEqual(hits[0][1], 2 * idf, places=9)   # два совпадения, tf=1, L=avgdl
        self.assertAlmostEqual(hits[1][1], idf, places=9)
        self.assertAlmostEqual(hits[2][1], idf, places=9)       # равные баллы — по возрастанию doc_id

    def test_error_identifiers_found_by_whole_token_and_by_parts(self):
        out = self.build([
            {"doc_id": 1, "title": "Crash", "text": "java.lang.NullPointerException when calling method on null"},
            {"doc_id": 2, "title": "Index", "text": "IndexOutOfBoundsException while reading the list"},
            {"doc_id": 3, "title": "Docker", "text": "docker-compose up fails with ECONNREFUSED on port 5432"},
            {"doc_id": 4, "title": "Plain", "text": "nothing relevant here at all"},
        ])
        self.assertEqual(bm25_search(out, "NullPointerException")[0][0], 1)
        self.assertEqual(bm25_search(out, "null pointer exception")[0][0], 1)   # по подтокенам
        self.assertEqual(bm25_search(out, "ECONNREFUSED")[0][0], 3)
        self.assertEqual(bm25_search(out, "docker-compose")[0][0], 3)
        self.assertEqual(bm25_search(out, "index out of bounds")[0][0], 2)
        self.assertEqual(bm25_search(out, "полностью отсутствующий термин"), [])

    def test_missing_corpus_gives_nonzero_exit_and_message(self):
        out = os.path.join(self.root, "index")
        result = run_indexer("corpus", {"CORPUS_PATH": os.path.join(self.root, "none.jsonl"),
                                        "INDEX_OUTPUT_PATH": out}, self.root)
        self.assertEqual(result.returncode, 1)
        self.assertIn("ошибка", result.stderr)
        self.assertFalse(os.path.exists(os.path.join(out, "metadata.json")))


class TestSampleEndToEnd(unittest.TestCase):
    def test_sample_from_plain_files_as_subprocess(self):
        with tempfile.TemporaryDirectory() as root:
            raw = os.path.join(root, "raw")
            os.makedirs(raw)
            posts = [
                '  <row Id="10" PostTypeId="1" CreationDate="2010-01-01T00:00:00.000" Score="5" '
                'AcceptedAnswerId="11" Body="&lt;p&gt;How to join paths with os.path.join?&lt;/p&gt;" '
                'Title="Joining paths in Python" Tags="&lt;python&gt;" />',
                '  <row Id="11" PostTypeId="2" ParentId="10" CreationDate="2010-01-02T00:00:00.000" Score="9" '
                'Body="&lt;p&gt;Use &lt;code&gt;pathlib&lt;/code&gt; Path objects&lt;/p&gt;" />',
                '  <row Id="20" PostTypeId="1" CreationDate="2010-02-01T00:00:00.000" Score="3" '
                'AcceptedAnswerId="21" Body="&lt;p&gt;docker-compose fails with ECONNREFUSED&lt;/p&gt;" '
                'Title="Docker network error" Tags="&lt;docker&gt;" />',
                '  <row Id="21" PostTypeId="2" ParentId="20" CreationDate="2010-02-02T00:00:00.000" Score="4" '
                'Body="&lt;p&gt;Start the daemon first&lt;/p&gt;" />',
            ]
            links = ['  <row Id="1" CreationDate="2010-03-01T00:00:00.000" PostId="10" RelatedPostId="20" LinkTypeId="1" />']
            for name, rows, tag in (("Posts.xml", posts, "posts"), ("PostLinks.xml", links, "postlinks"),
                                    ("Tags.xml", [], "tags")):
                with open(os.path.join(raw, name), "w", encoding="utf-8", newline="") as f:
                    f.write("﻿<?xml version=\"1.0\" encoding=\"utf-8\"?>\r\n" f"<{tag}>\r\n"
                            + "".join(r + "\r\n" for r in rows) + f"</{tag}>\r\n")
            out = os.path.join(root, "index")
            result = run_indexer("sample", {"RAW_DATA_PATH": raw, "INDEX_OUTPUT_PATH": out, "SAMPLE_DENOM": "1"}, root)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("валидных тредов: 2", result.stdout)
            self.assertEqual(bm25_search(out, "pathlib objects")[0][0], 10)
            self.assertEqual(bm25_search(out, "ECONNREFUSED daemon")[0][0], 20)
            with open(os.path.join(out, "sample_manifest.json"), encoding="utf-8") as f:
                manifest = json.load(f)
            self.assertEqual((manifest["valid_threads"], manifest["edges"]), (2, 1))


if __name__ == "__main__":
    unittest.main()
