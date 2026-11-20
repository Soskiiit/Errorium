"""Top-K отбор через min-heap (docs/Tech/03-data.md, п. 6.3).

Поведение обязано совпадать с полной сортировкой: убывание балла,
равные баллы — по возрастанию doc_id (занятие 3, п. 5.4).
"""
import random
import unittest

from _core_fixture import CORE_SRC  # noqa: F401  (sys.path setup)

from core.bm25 import top_k


def reference(scores, k):
    return sorted(scores.items(), key=lambda kv: (-kv[1], kv[0]))[:k]


class TestTopK(unittest.TestCase):
    def test_tie_break_by_doc_id(self):
        scores = {3: 1.0, 1: 1.0, 2: 1.0}
        self.assertEqual(top_k(scores, 2), [(1, 1.0), (2, 1.0)])
        self.assertEqual(top_k(scores, 3), [(1, 1.0), (2, 1.0), (3, 1.0)])

    def test_matches_full_sort_on_random_data(self):
        rng = random.Random(42)
        for _ in range(20):
            scores = {d: rng.choice([0.5, 1.0, 1.5, rng.random()])
                      for d in rng.sample(range(1, 10000), 500)}
            for k in (1, 3, 10, 100):
                self.assertEqual(top_k(scores, k), reference(scores, k))

    def test_k_larger_than_candidate_count(self):
        scores = {5: 2.0, 2: 1.0}
        self.assertEqual(top_k(scores, 100), [(5, 2.0), (2, 1.0)])

    def test_empty_and_nonpositive_k(self):
        self.assertEqual(top_k({}, 10), [])
        self.assertEqual(top_k({1: 1.0}, 0), [])
        self.assertEqual(top_k({1: 1.0}, -3), [])

    def test_single_candidate(self):
        self.assertEqual(top_k({7: 0.123}, 10), [(7, 0.123)])

    def test_heap_keeps_best_not_first(self):
        # минимум вытесняется: поздние кандидаты с большим баллом попадают в выдачу
        scores = {i: float(i) for i in range(1, 101)}
        self.assertEqual(top_k(scores, 3), [(100, 100.0), (99, 99.0), (98, 98.0)])


if __name__ == "__main__":
    unittest.main()