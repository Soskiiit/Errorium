"""Unit-тесты SPIMI-построителя индекса (docs/Tech/03-data.md, п. 1.1–1.2)."""
import json
import os
import random
import struct
import sys
import tempfile
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
sys.path.insert(0, os.path.join(REPO_ROOT, "services", "search_engine_indexer", "src"))

from indexer.index_format import (  # noqa: E402
    read_doc_lengths,
    read_lexicon,
    read_postings,
)
from indexer.spimi import build_index  # noqa: E402
from indexer.tokenizer import TOKENIZER_VERSION  # noqa: E402

DOCS = [
    (10, ["error", "null", "pointer", "error"]),
    (20, ["null", "exception"]),
    (30, ["error"]),
]
INDEX_FILES = ("lexicon.bin", "postings.bin", "doc_lengths.bin", "metadata.json")


def read_bytes(directory, name):
    with open(os.path.join(directory, name), "rb") as f:
        return f.read()


class IndexTestCase(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self._tmp.cleanup)
        self.root = self._tmp.name

    def out(self, name="index"):
        return os.path.join(self.root, name)


class TestBuildSmallIndex(IndexTestCase):
    def setUp(self):
        super().setUp()
        self.dir = self.out()
        self.meta = build_index(DOCS, self.dir, index_mode="sample", timestamp=1700000000,
                                source={"sample_denom": 100})

    def test_all_files_written_and_no_temp_left(self):
        self.assertEqual(sorted(os.listdir(self.dir)), sorted(INDEX_FILES))

    def test_metadata_file_and_return_value(self):
        with open(os.path.join(self.dir, "metadata.json"), encoding="utf-8") as f:
            meta = json.load(f)
        self.assertEqual(meta, self.meta)
        self.assertEqual(meta["format_version"], 1)
        self.assertEqual(meta["tokenizer_version"], TOKENIZER_VERSION)
        self.assertEqual(meta["index_mode"], "sample")
        self.assertEqual(meta["num_documents"], 3)
        self.assertEqual(meta["total_tokens"], 7)
        self.assertAlmostEqual(meta["avg_doc_length"], 7 / 3)
        self.assertEqual(meta["vocabulary_size"], 4)
        self.assertEqual(meta["build_timestamp"], 1700000000)
        self.assertEqual(meta["source"], {"sample_denom": 100})

    def test_doc_lengths(self):
        self.assertEqual(read_doc_lengths(os.path.join(self.dir, "doc_lengths.bin")),
                         {10: 4, 20: 2, 30: 1})

    def test_lexicon_sorted_with_df(self):
        lex = read_lexicon(os.path.join(self.dir, "lexicon.bin"))
        self.assertEqual(list(lex), ["error", "exception", "null", "pointer"])
        self.assertEqual({t: e.df for t, e in lex.items()},
                         {"error": 2, "exception": 1, "null": 2, "pointer": 1})

    def test_postings_lists(self):
        lex = read_lexicon(os.path.join(self.dir, "lexicon.bin"))
        path = os.path.join(self.dir, "postings.bin")
        self.assertEqual(read_postings(path, lex["error"]), [(10, 2), (30, 1)])
        self.assertEqual(read_postings(path, lex["null"]), [(10, 1), (20, 1)])
        self.assertEqual(read_postings(path, lex["pointer"]), [(10, 1)])
        self.assertEqual(read_postings(path, lex["exception"]), [(20, 1)])

    def test_df_equals_posting_count_and_offsets_are_contiguous(self):
        lex = read_lexicon(os.path.join(self.dir, "lexicon.bin"))
        header = 8
        expected_offset = header
        for entry in lex.values():
            self.assertEqual(entry.offset, expected_offset)
            expected_offset += entry.length
        self.assertEqual(expected_offset, len(read_bytes(self.dir, "postings.bin")))


class TestByteLayout(IndexTestCase):
    """Независимая проверка байтов по спецификации (без модуля чтения)."""

    def setUp(self):
        super().setUp()
        self.dir = self.out()
        build_index(DOCS, self.dir, timestamp=0)

    def test_lexicon_layout(self):
        data = read_bytes(self.dir, "lexicon.bin")
        self.assertEqual(data[:4], b"ELX1")
        version, vocab = struct.unpack_from("<II", data, 4)
        self.assertEqual((version, vocab), (1, 4))
        pos = 12
        terms = []
        for _ in range(vocab):
            (tlen,) = struct.unpack_from("<H", data, pos)
            pos += 2
            term = data[pos:pos + tlen].decode("utf-8")
            pos += tlen
            df, offset, length = struct.unpack_from("<IQI", data, pos)
            pos += 16
            terms.append((term, df))
        self.assertEqual(pos, len(data))
        self.assertEqual(terms, [("error", 2), ("exception", 1), ("null", 2), ("pointer", 1)])

    def test_postings_header(self):
        data = read_bytes(self.dir, "postings.bin")
        self.assertEqual(data[:4], b"EPS1")
        self.assertEqual(struct.unpack_from("<I", data, 4)[0], 1)

    def test_doc_lengths_layout(self):
        data = read_bytes(self.dir, "doc_lengths.bin")
        self.assertEqual(data[:4], b"EDL1")
        version, n = struct.unpack_from("<II", data, 4)
        self.assertEqual((version, n), (1, 3))
        records = [struct.unpack_from("<II", data, 12 + 8 * i) for i in range(n)]
        self.assertEqual(records, [(10, 4), (20, 2), (30, 1)])
        self.assertEqual(len(data), 12 + 8 * n)


class TestBlocksAndOrder(IndexTestCase):
    @staticmethod
    def corpus(seed=7, docs=200, vocab=40):
        rnd = random.Random(seed)
        words = [f"w{i}" for i in range(vocab)] + ["ошибка", "c++", "os.path.join"]
        out = []
        for doc_id in range(1, docs + 1):
            out.append((doc_id * 3, [rnd.choice(words) for _ in range(rnd.randint(1, 30))]))
        return out

    def assert_same_index(self, a, b):
        for name in INDEX_FILES:
            self.assertEqual(read_bytes(a, name), read_bytes(b, name), name)

    def test_many_small_blocks_equal_single_block(self):
        docs = self.corpus()
        one, many = self.out("one"), self.out("many")
        build_index(docs, one, block_postings=10**9, timestamp=0)
        build_index(docs, many, block_postings=7, timestamp=0)
        self.assert_same_index(one, many)

    def test_blocks_actually_spilled(self):
        docs = self.corpus(docs=50)
        stats = {}
        build_index(docs, self.out(), block_postings=5, timestamp=0, stats=stats)
        self.assertGreater(stats["blocks"], 1)
        single = {}
        build_index(docs, self.out("single"), block_postings=10**9, timestamp=0, stats=single)
        self.assertEqual(single["blocks"], 1)

    def test_arbitrary_document_order_gives_same_index(self):
        docs = self.corpus()
        shuffled = docs[:]
        random.Random(1).shuffle(shuffled)
        a, b = self.out("sorted"), self.out("shuffled")
        build_index(docs, a, block_postings=11, timestamp=0)
        build_index(shuffled, b, block_postings=11, timestamp=0)
        self.assert_same_index(a, b)

    def test_postings_merged_by_doc_id_across_blocks(self):
        docs = [(5, ["x"]), (1, ["x"]), (9, ["x"]), (3, ["x"])]
        d = self.out()
        build_index(docs, d, block_postings=1, timestamp=0)
        lex = read_lexicon(os.path.join(d, "lexicon.bin"))
        self.assertEqual(read_postings(os.path.join(d, "postings.bin"), lex["x"]),
                         [(1, 1), (3, 1), (5, 1), (9, 1)])

    def test_unicode_terms_sorted_by_utf8_bytes(self):
        d = self.out()
        build_index([(1, ["ошибка", "zeta", "error", "c++"])], d, timestamp=0)
        lex = read_lexicon(os.path.join(d, "lexicon.bin"))
        keys = list(lex)
        self.assertEqual(keys, sorted(keys, key=lambda t: t.encode("utf-8")))
        self.assertEqual(set(keys), {"ошибка", "zeta", "error", "c++"})


class TestEdgeCases(IndexTestCase):
    def test_tf_above_16_bit(self):
        d = self.out()
        build_index([(1, ["a"] * 70000)], d, timestamp=0)
        lex = read_lexicon(os.path.join(d, "lexicon.bin"))
        self.assertEqual(read_postings(os.path.join(d, "postings.bin"), lex["a"]), [(1, 70000)])
        self.assertEqual(read_doc_lengths(os.path.join(d, "doc_lengths.bin")), {1: 70000})

    def test_empty_document_counts_in_n(self):
        d = self.out()
        meta = build_index([(1, ["a"]), (2, [])], d, timestamp=0)
        self.assertEqual(meta["num_documents"], 2)
        self.assertAlmostEqual(meta["avg_doc_length"], 0.5)
        self.assertEqual(read_doc_lengths(os.path.join(d, "doc_lengths.bin")), {1: 1, 2: 0})

    def test_empty_corpus(self):
        d = self.out()
        meta = build_index([], d, timestamp=0)
        self.assertEqual((meta["num_documents"], meta["vocabulary_size"], meta["avg_doc_length"]),
                         (0, 0, 0.0))
        self.assertEqual(read_lexicon(os.path.join(d, "lexicon.bin")), {})
        self.assertEqual(read_doc_lengths(os.path.join(d, "doc_lengths.bin")), {})

    def test_duplicate_doc_id_rejected(self):
        with self.assertRaises(ValueError):
            build_index([(1, ["a"]), (1, ["b"])], self.out(), timestamp=0)

    def test_non_positive_doc_id_rejected(self):
        with self.assertRaises(ValueError):
            build_index([(0, ["a"])], self.out(), timestamp=0)


class TestExtraFiles(IndexTestCase):
    """Дополнительные артефакты (graph.db, манифест) публикуются вместе с индексом, до маркера."""

    def make_extra(self, name="graph.src", content=b"graph-bytes"):
        path = os.path.join(self.root, name)
        with open(path, "wb") as f:
            f.write(content)
        return path

    def test_extra_file_moved_into_index_dir(self):
        src = self.make_extra()
        d = self.out()
        build_index(DOCS, d, timestamp=0, extra_files={"graph.db": src})
        self.assertEqual(read_bytes(d, "graph.db"), b"graph-bytes")
        self.assertFalse(os.path.exists(src))
        self.assertIn("metadata.json", os.listdir(d))

    def test_failed_build_does_not_touch_extra_or_index_dir(self):
        src = self.make_extra()
        d = self.out()

        def docs():
            yield 1, ["a"]
            raise RuntimeError("boom")

        with self.assertRaises(RuntimeError):
            build_index(docs(), d, timestamp=0, extra_files={"graph.db": src})
        self.assertTrue(os.path.exists(src))
        self.assertFalse(os.path.exists(os.path.join(d, "graph.db")))

    def test_extra_file_name_must_be_plain(self):
        src = self.make_extra()
        for bad in ("../evil.bin", "sub/dir.db", "metadata.json", "lexicon.bin"):
            with self.subTest(name=bad):
                with self.assertRaises(ValueError):
                    build_index(DOCS, self.out(), timestamp=0, extra_files={bad: src})
        self.assertTrue(os.path.exists(src))

    def test_old_extra_file_replaced(self):
        d = self.out()
        build_index(DOCS, d, timestamp=0, extra_files={"graph.db": self.make_extra(content=b"old")})
        build_index(DOCS, d, timestamp=0, extra_files={"graph.db": self.make_extra("g2", b"new")})
        self.assertEqual(read_bytes(d, "graph.db"), b"new")


class TestAtomicity(IndexTestCase):
    def test_failed_build_leaves_no_marker_and_no_temp(self):
        d = self.out()

        def docs():
            yield 1, ["a", "b"]
            raise RuntimeError("source broke")

        with self.assertRaises(RuntimeError):
            build_index(docs(), d, block_postings=1, timestamp=0)
        self.assertFalse(os.path.exists(os.path.join(d, "metadata.json")))
        self.assertEqual([x for x in os.listdir(d)] if os.path.isdir(d) else [], [])

    def test_failed_rebuild_keeps_previous_index(self):
        d = self.out()
        build_index(DOCS, d, timestamp=1)
        before = {name: read_bytes(d, name) for name in INDEX_FILES}

        def docs():
            yield 1, ["z"]
            raise RuntimeError("source broke")

        with self.assertRaises(RuntimeError):
            build_index(docs(), d, block_postings=1, timestamp=2)
        after = {name: read_bytes(d, name) for name in INDEX_FILES}
        self.assertEqual(before, after)
        self.assertNotIn(".building", os.listdir(d))

    def test_rebuild_replaces_previous_index(self):
        d = self.out()
        build_index(DOCS, d, timestamp=1)
        meta = build_index([(7, ["only"])], d, timestamp=2)
        self.assertEqual(meta["num_documents"], 1)
        lex = read_lexicon(os.path.join(d, "lexicon.bin"))
        self.assertEqual(list(lex), ["only"])
        self.assertEqual(sorted(os.listdir(d)), sorted(INDEX_FILES))


if __name__ == "__main__":
    unittest.main()
