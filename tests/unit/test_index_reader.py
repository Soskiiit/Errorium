"""Чтение индекса формата v1 ядром: mmap-постинги, валидация metadata.json
(docs/Tech/03-data.md, п. 1.1, 6.2)."""
import os
import tempfile
import unittest

from _core_fixture import build_index, fmt

from core.reader import IndexReader, UnsupportedIndexError

DOCS = [
    {"doc_id": 1, "title": "NPE", "text": "java.lang.NullPointerException when calling method"},
    {"doc_id": 2, "title": "Docker", "text": "docker-compose up fails with ECONNREFUSED"},
    {"doc_id": 3, "title": "Paths", "text": "os.path.join or pathlib for paths"},
]


class ReaderTestBase(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self._tmp.cleanup)
        self.root = self._tmp.name

    def build(self, **kwargs):
        return build_index(self.root, DOCS, **kwargs)

    def make_reader(self, index_dir):
        reader = IndexReader(index_dir)
        self.addCleanup(reader.close)
        return reader


class TestIndexReader(ReaderTestBase):
    def test_load_and_stats_from_metadata(self):
        # avgdl берётся из metadata.json, а не пересчитывается
        index_dir = self.build(metadata={"avg_doc_length": 42.5, "build_timestamp": 1700000000})
        reader = self.make_reader(index_dir)
        self.assertTrue(reader.try_load())
        self.assertTrue(reader.ready)
        self.assertEqual(reader.n, 3)
        self.assertEqual(reader.avgdl, 42.5)
        self.assertEqual(reader.build_timestamp, 1700000000)
        self.assertGreater(reader.vocabulary_size, 0)
        self.assertGreater(reader.index_size_bytes, 0)

    def test_mmap_postings_match_reference_decoder(self):
        index_dir = self.build()
        reader = self.make_reader(index_dir)
        reader.try_load()
        reference_lexicon = fmt.read_lexicon(os.path.join(index_dir, "lexicon.bin"))
        self.assertGreater(len(reference_lexicon), 5)
        for term, entry in reference_lexicon.items():
            got = reader.postings(reader.entry(term))
            want = fmt.read_postings(os.path.join(index_dir, "postings.bin"), entry)
            self.assertEqual(got, want, term)

    def test_unknown_term_has_no_entry(self):
        reader = self.make_reader(self.build())
        reader.try_load()
        self.assertIsNone(reader.entry("совершенноотсутствующийтерм"))

    def test_doc_lengths(self):
        reader = self.make_reader(self.build())
        reader.try_load()
        reference = fmt.read_doc_lengths(os.path.join(reader.index_dir, "doc_lengths.bin"))
        for doc_id, length in reference.items():
            self.assertEqual(reader.doc_length(doc_id), length)

    def test_missing_metadata_means_not_ready(self):
        index_dir = self.build(write_metadata=False)
        reader = self.make_reader(index_dir)
        self.assertFalse(reader.try_load())
        self.assertFalse(reader.ready)

    def test_missing_directory_means_not_ready(self):
        reader = self.make_reader(os.path.join(self.root, "nonexistent"))
        self.assertFalse(reader.try_load())

    def test_wrong_tokenizer_version_rejected(self):
        index_dir = self.build(metadata={"tokenizer_version": "2"})
        with self.assertRaises(UnsupportedIndexError):
            self.make_reader(index_dir).try_load()

    def test_wrong_format_version_rejected(self):
        index_dir = self.build(metadata={"format_version": 99})
        with self.assertRaises(UnsupportedIndexError):
            self.make_reader(index_dir).try_load()

    def test_corrupted_magic_rejected(self):
        index_dir = self.build()
        path = os.path.join(index_dir, "lexicon.bin")
        with open(path, "r+b") as f:
            f.write(b"XXXX")
        with self.assertRaises(UnsupportedIndexError):
            self.make_reader(index_dir).try_load()

    def test_lazy_reload_after_index_appears(self):
        index_dir = os.path.join(self.root, "index")
        reader = self.make_reader(index_dir)
        self.assertFalse(reader.try_load())
        build_index(self.root, DOCS)
        self.assertTrue(reader.try_load())
        self.assertEqual(reader.n, 3)


if __name__ == "__main__":
    unittest.main()