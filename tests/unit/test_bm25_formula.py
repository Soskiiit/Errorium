"""BM25 по формуле кафедры ИУ5 (docs/Tech/03-data.md, п. 6.3; занятие 3).

Эталон — ручной расчёт из методички: N=3, avgdl=4, df=2, tf=1, L=4, k1=1.2, b=0.75:
IDF = ln(1 + (3-2+0.5)/(2+0.5)) = ln(1.6) ≈ 0.470; документ A = 0.940, B и C = 0.470.
"""
import math
import unittest

from _core_fixture import CORE_SRC  # noqa: F401  (sys.path setup)

from core import bm25
from core.config import Config


def lesson3_scores(k1=1.2, b=0.75):
    """Пример занятия 3 через API ядра: термы запроса «Как происходит возврат оплаты?»."""
    term_postings = [
        (0, []),                      # «как» — нет в корпусе
        (0, []),                      # «происходит»
        (2, [(1, 1), (3, 1)]),        # «возврат»: A и C
        (2, [(1, 1), (2, 1)]),        # «оплаты»: A и B
    ]
    lengths = {1: 4, 2: 4, 3: 4}
    return bm25.score_documents(term_postings, lengths, n=3, avgdl=4.0, k1=k1, b=b)


class TestIdf(unittest.TestCase):
    def test_lesson3_value(self):
        self.assertAlmostEqual(bm25.idf(3, 2), math.log(1.6), places=12)
        self.assertAlmostEqual(bm25.idf(3, 2), 0.470, places=3)

    def test_rarer_term_weighs_more(self):
        self.assertGreater(bm25.idf(1000, 1), bm25.idf(1000, 100))

    def test_term_in_all_documents_still_positive(self):
        # ln(1 + 0.5/(N+0.5)) > 0 — формула методички не даёт отрицательных IDF
        self.assertGreater(bm25.idf(10, 10), 0.0)


class TestTermScore(unittest.TestCase):
    def test_lesson3_single_term_contribution(self):
        idf = bm25.idf(3, 2)
        # tf=1, L=avgdl => дробь равна 1, вклад = IDF
        self.assertAlmostEqual(bm25.term_score(1, idf, 4, 4.0, 1.2, 0.75), idf, places=12)

    def test_tf_saturation_lesson3_multiplier(self):
        # занятие 3: при равной длине tf=2 даёт множитель 1.38 (k1=1.2)
        s1 = bm25.term_score(1, 1.0, 4, 4.0, 1.2, 0.75)
        s2 = bm25.term_score(2, 1.0, 4, 4.0, 1.2, 0.75)
        self.assertAlmostEqual(s2 / s1, 1.375, places=3)
        self.assertLess(s2, 2 * s1)

    def test_longer_document_scores_lower(self):
        short = bm25.term_score(1, 1.0, 4, 4.0, 1.5, 0.75)
        long = bm25.term_score(1, 1.0, 8, 4.0, 1.5, 0.75)
        self.assertLess(long, short)

    def test_b_zero_disables_length_normalization(self):
        self.assertAlmostEqual(
            bm25.term_score(1, 1.0, 100, 4.0, 1.5, 0.0),
            bm25.term_score(1, 1.0, 4, 4.0, 1.5, 0.0), places=12)


class TestScoreDocuments(unittest.TestCase):
    def test_lesson3_full_example(self):
        scores = lesson3_scores()
        idf = math.log(1.6)
        self.assertAlmostEqual(scores[1], 2 * idf, places=9)   # A: два совпадения
        self.assertAlmostEqual(scores[2], idf, places=9)       # B
        self.assertAlmostEqual(scores[3], idf, places=9)       # C
        self.assertEqual(bm25.top_k(scores, 2), [(1, scores[1]), (2, scores[2])])

    def test_missing_terms_contribute_nothing(self):
        scores = bm25.score_documents([(0, [])], {}, n=3, avgdl=4.0, k1=1.2, b=0.75)
        self.assertEqual(scores, {})

    def test_scores_accumulate_across_terms(self):
        scores = bm25.score_documents(
            [(1, [(1, 2)]), (1, [(1, 1)])], {1: 4}, n=1, avgdl=4.0, k1=1.5, b=0.75)
        expected = (bm25.term_score(2, bm25.idf(1, 1), 4, 4.0, 1.5, 0.75)
                    + bm25.term_score(1, bm25.idf(1, 1), 4, 4.0, 1.5, 0.75))
        self.assertAlmostEqual(scores[1], expected, places=12)

    def test_default_parameters_match_iu5(self):
        cfg = Config.from_env({})
        self.assertEqual(cfg.k1, 1.5)
        self.assertEqual(cfg.b, 0.75)
        # при L=avgdl и tf=1 вклад не зависит от k1 — проверяем на tf=2
        s_default = bm25.score_documents([(1, [(1, 2)])], {1: 4}, 1, 4.0, cfg.k1, cfg.b)
        s_12 = bm25.score_documents([(1, [(1, 2)])], {1: 4}, 1, 4.0, 1.2, cfg.b)
        self.assertNotAlmostEqual(s_default[1], s_12[1], places=6)


class TestConfigValidation(unittest.TestCase):
    def test_k1_range_enforced(self):
        for bad in ("1.1", "2.1"):
            with self.assertRaises(ValueError):
                Config.from_env({"BM25_K1": bad})
        self.assertEqual(Config.from_env({"BM25_K1": "1.2"}).k1, 1.2)
        self.assertEqual(Config.from_env({"BM25_K1": "2.0"}).k1, 2.0)

    def test_b_range_enforced(self):
        for bad in ("-0.1", "1.1"):
            with self.assertRaises(ValueError):
                Config.from_env({"BM25_B": bad})

    def test_defaults(self):
        cfg = Config.from_env({})
        self.assertEqual(cfg.port, 50051)
        self.assertEqual(cfg.index_dir, "/app/data/index")
        self.assertEqual(cfg.max_top_k, 100)


if __name__ == "__main__":
    unittest.main()