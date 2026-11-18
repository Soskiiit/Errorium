"""Unit-тесты бинарного формата индекса (docs/Tech/03-data.md, п. 1.1)."""
import os
import sys
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
sys.path.insert(0, os.path.join(REPO_ROOT, "services", "search_engine_indexer", "src"))

from indexer.index_format import (  # noqa: E402
    decode_postings,
    decode_varint,
    encode_postings,
    encode_varint,
)


class TestVarint(unittest.TestCase):
    def test_known_encodings(self):
        self.assertEqual(encode_varint(0), b"\x00")
        self.assertEqual(encode_varint(1), b"\x01")
        self.assertEqual(encode_varint(127), b"\x7f")
        self.assertEqual(encode_varint(128), b"\x80\x01")
        self.assertEqual(encode_varint(300), b"\xac\x02")
        self.assertEqual(encode_varint(16384), b"\x80\x80\x01")

    def test_roundtrip(self):
        for value in (0, 1, 127, 128, 255, 300, 16383, 16384, 2**21, 2**32 - 1, 2**32, 2**63):
            with self.subTest(value=value):
                data = encode_varint(value)
                decoded, pos = decode_varint(data, 0)
                self.assertEqual(decoded, value)
                self.assertEqual(pos, len(data))

    def test_decode_from_offset(self):
        data = b"\xff" + encode_varint(300) + b"\x05"
        value, pos = decode_varint(data, 1)
        self.assertEqual((value, pos), (300, 3))

    def test_negative_rejected(self):
        with self.assertRaises(ValueError):
            encode_varint(-1)

    def test_truncated_rejected(self):
        with self.assertRaises(ValueError):
            decode_varint(b"\x80", 0)
        with self.assertRaises(ValueError):
            decode_varint(b"", 0)


class TestPostingsCodec(unittest.TestCase):
    def test_known_layout_delta_then_tf(self):
        data = encode_postings([(5, 2), (9, 1), (1000000, 300)])
        # (5, 2) | (9-5=4, 1) | (1000000-9=999991, 300)
        self.assertEqual(data[:4], b"\x05\x02\x04\x01")
        self.assertEqual(decode_postings(data, 3), [(5, 2), (9, 1), (1000000, 300)])

    def test_first_delta_is_relative_to_zero(self):
        self.assertEqual(encode_postings([(1, 1)]), b"\x01\x01")

    def test_roundtrip_large_tf_beyond_16_bit(self):
        pairs = [(1, 70000), (2, 1), (4_000_000_000, 65536)]
        self.assertEqual(decode_postings(encode_postings(pairs), 3), pairs)

    def test_empty_list(self):
        self.assertEqual(encode_postings([]), b"")
        self.assertEqual(decode_postings(b"", 0), [])

    def test_doc_ids_must_strictly_increase(self):
        with self.assertRaises(ValueError):
            encode_postings([(5, 1), (5, 1)])
        with self.assertRaises(ValueError):
            encode_postings([(9, 1), (5, 1)])

    def test_tf_must_be_positive(self):
        with self.assertRaises(ValueError):
            encode_postings([(5, 0)])

    def test_doc_id_must_be_positive(self):
        with self.assertRaises(ValueError):
            encode_postings([(0, 1)])

    def test_decode_checks_count(self):
        data = encode_postings([(5, 2), (9, 1)])
        with self.assertRaises(ValueError):
            decode_postings(data, 3)           # данных меньше, чем df
        with self.assertRaises(ValueError):
            decode_postings(data + b"\x01", 2)  # лишние байты


if __name__ == "__main__":
    unittest.main()
