"""Unit tests for benchmark metrics and evaluation logic (A4)."""
import io
import json
import os
import sys
import tempfile
import unittest

PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
if PROJECT_ROOT not in sys.path:
    sys.path.insert(0, PROJECT_ROOT)

from benchmark.evaluate import (
    NFR_MRR_10,
    NFR_RECALL_10,
    evaluate_dataset,
    mrr_at_k,
    parse_doc_id,
    parse_relevant_docs,
    recall_at_k,
)


class TestBenchmarkMetrics(unittest.TestCase):
    def test_parse_doc_id(self):
        self.assertEqual(parse_doc_id("D01"), 1)
        self.assertEqual(parse_doc_id("D11"), 11)
        self.assertEqual(parse_doc_id("D30"), 30)
        self.assertEqual(parse_doc_id("d05"), 5)
        self.assertEqual(parse_doc_id(42), 42)
        self.assertEqual(parse_doc_id("15"), 15)

    def test_parse_relevant_docs(self):
        self.assertEqual(parse_relevant_docs(["D01", "D11", "D21"]), [1, 11, 21])
        self.assertEqual(parse_relevant_docs([]), [])
        self.assertEqual(parse_relevant_docs([5, "D06"]), [5, 6])

    def test_recall_at_k_perfect(self):
        # All relevant retrieved within top-k
        relevant = [1, 2]
        retrieved = [1, 2, 3, 4]
        self.assertAlmostEqual(recall_at_k(relevant, retrieved, k=3), 1.0)
        self.assertAlmostEqual(recall_at_k(relevant, retrieved, k=2), 1.0)

    def test_recall_at_k_partial(self):
        # 3 relevant, only 1 in top-3, 2 in top-4
        relevant = [1, 2, 3]
        retrieved = [1, 4, 5, 2, 6]
        self.assertAlmostEqual(recall_at_k(relevant, retrieved, k=3), 1.0 / 3.0)
        self.assertAlmostEqual(recall_at_k(relevant, retrieved, k=4), 2.0 / 3.0)
        self.assertAlmostEqual(recall_at_k(relevant, retrieved, k=5), 2.0 / 3.0)

    def test_recall_at_k_zero(self):
        # No relevant retrieved
        relevant = [1, 2]
        retrieved = [3, 4, 5]
        self.assertAlmostEqual(recall_at_k(relevant, retrieved, k=3), 0.0)

    def test_recall_at_k_boundaries(self):
        # k > length of retrieved
        relevant = [1, 2]
        retrieved = [1]
        self.assertAlmostEqual(recall_at_k(relevant, retrieved, k=5), 0.5)

        # Empty relevant
        self.assertAlmostEqual(recall_at_k([], [1, 2], k=5), 0.0)

        # Empty retrieved
        self.assertAlmostEqual(recall_at_k([1, 2], [], k=5), 0.0)

        # k = 0
        self.assertAlmostEqual(recall_at_k([1, 2], [1, 2], k=0), 0.0)

    def test_mrr_at_k_first_hit(self):
        # First retrieved is relevant -> 1.0
        relevant = [1, 2]
        retrieved = [1, 3, 4]
        self.assertAlmostEqual(mrr_at_k(relevant, retrieved, k=10), 1.0)

    def test_mrr_at_k_second_hit(self):
        # Second retrieved is first relevant -> 1/2 = 0.5
        relevant = [2, 3]
        retrieved = [1, 2, 3]
        self.assertAlmostEqual(mrr_at_k(relevant, retrieved, k=10), 0.5)

    def test_mrr_at_k_third_hit(self):
        # Third retrieved is first relevant -> 1/3
        relevant = [3]
        retrieved = [1, 2, 3]
        self.assertAlmostEqual(mrr_at_k(relevant, retrieved, k=10), 1.0 / 3.0)

    def test_mrr_at_k_beyond_k(self):
        # First relevant is at rank 4, but k=3 -> 0.0
        relevant = [4]
        retrieved = [1, 2, 3, 4]
        self.assertAlmostEqual(mrr_at_k(relevant, retrieved, k=3), 0.0)

    def test_mrr_at_k_boundaries(self):
        # No relevant
        self.assertAlmostEqual(mrr_at_k([9], [1, 2, 3], k=10), 0.0)
        # Empty relevant
        self.assertAlmostEqual(mrr_at_k([], [1, 2], k=10), 0.0)
        # Empty retrieved
        self.assertAlmostEqual(mrr_at_k([1], [], k=10), 0.0)
        # k = 0
        self.assertAlmostEqual(mrr_at_k([1], [1], k=0), 0.0)


class MockEngine:
    def __init__(self, query_to_results):
        self.query_to_results = query_to_results

    def search(self, query, top_k=10):
        results = self.query_to_results.get(query, [])[:top_k]
        # Return object with hits having doc_id and score
        class MockHit:
            def __init__(self, doc_id, score):
                self.doc_id = doc_id
                self.score = score

        class MockSearchResult:
            def __init__(self, hits):
                self.hits = hits

        hits = [MockHit(doc_id, score) for doc_id, score in results]
        return MockSearchResult(hits)


class TestEvaluateDataset(unittest.TestCase):
    def setUp(self):
        self.questions = [
            {
                "id": "Q01",
                "type": "answerable",
                "question": "q1",
                "relevant_docs": ["D01", "D02"],
            },
            {
                "id": "Q02",
                "type": "answerable",
                "question": "q2",
                "relevant_docs": ["D03"],
            },
            {
                "id": "Q03",
                "type": "trap",
                "question": "q3_trap",
                "relevant_docs": [],
            },
        ]

    def test_trap_questions_excluded_from_metrics(self):
        # Q01: retrieved [1, 2] -> R@3=1.0, R@5=1.0, R@10=1.0, MRR@10=1.0
        # Q02: retrieved [4, 3] -> R@3=1.0, R@5=1.0, R@10=1.0, MRR@10=0.5 (rank 2)
        # Q03 (trap): retrieved [5, 6] -> should NOT affect averages!
        engine = MockEngine({
            "q1": [(1, 10.0), (2, 8.0)],
            "q2": [(4, 9.0), (3, 7.0)],
            "q3_trap": [(5, 12.0), (6, 5.0)],
        })

        summary = evaluate_dataset(engine, self.questions, top_k=10)

        self.assertEqual(summary["total_questions"], 3)
        self.assertEqual(summary["answerable_count"], 2)
        self.assertEqual(summary["trap_count"], 1)

        # Means over 2 answerable questions
        expected_recall_10 = (1.0 + 1.0) / 2.0
        expected_mrr_10 = (1.0 + 0.5) / 2.0
        self.assertAlmostEqual(summary["metrics"]["recall_at_10"], expected_recall_10)
        self.assertAlmostEqual(summary["metrics"]["mrr_at_10"], expected_mrr_10)

        # Check trap question recording
        q3_res = next(q for q in summary["questions"] if q["id"] == "Q03")
        self.assertEqual(q3_res["type"], "trap")
        self.assertEqual(q3_res["top1_doc_id"], 5)
        self.assertAlmostEqual(q3_res["top1_score"], 12.0)
        self.assertIsNone(q3_res["recall_at_10"])
        self.assertIsNone(q3_res["mrr_at_10"])

    def test_nfr_thresholds_pass_and_fail(self):
        # Passing case
        engine_pass = MockEngine({
            "q1": [(1, 10.0), (2, 8.0)],
            "q2": [(3, 9.0)],
            "q3_trap": [(5, 12.0)],
        })
        summary_pass = evaluate_dataset(engine_pass, self.questions, top_k=10)
        self.assertTrue(summary_pass["nfr"]["passed"])
        self.assertEqual(len(summary_pass["nfr"]["violations"]), 0)

        # Failing case: no relevant found for Q02
        engine_fail = MockEngine({
            "q1": [(1, 10.0)],      # R@10 = 0.5, MRR = 1.0
            "q2": [(9, 9.0)],      # R@10 = 0.0, MRR = 0.0
            "q3_trap": [(5, 12.0)],
        })
        # Mean R@10 = (0.5 + 0.0)/2 = 0.25 (< 0.75 NFR)
        # Mean MRR@10 = (1.0 + 0.0)/2 = 0.50 (>= 0.50 NFR)
        summary_fail = evaluate_dataset(engine_fail, self.questions, top_k=10)
        self.assertFalse(summary_fail["nfr"]["passed"])
        self.assertFalse(summary_fail["nfr"]["recall_at_10_passed"])
        self.assertTrue(summary_fail["nfr"]["mrr_at_10_passed"])
        self.assertTrue(any("Recall@10" in v for v in summary_fail["nfr"]["violations"]))

    def test_print_evaluation_report(self):
        engine = MockEngine({
            "q1": [(1, 10.0), (2, 8.0)],
            "q2": [(3, 9.0)],
            "q3_trap": [(5, 12.0)],
        })
        summary = evaluate_dataset(engine, self.questions, top_k=10)
        out = io.StringIO()
        from unittest.mock import patch
        from benchmark.evaluate import print_evaluation_report
        with patch("sys.stdout", out):
            print_evaluation_report(summary)
        output = out.getvalue()
        self.assertIn("BENCHMARK EVALUATION", output)
        self.assertIn("Q01", output)
        self.assertIn("Q03", output)
        self.assertIn("PASS", output)

    def test_main_cli_missing_files(self):
        from benchmark.evaluate import main
        # Non-existent questions file
        code = main(["--questions", "/tmp/non_existent_questions.json"])
        self.assertEqual(code, 1)

        # Non-existent index dir
        with tempfile.NamedTemporaryFile("w", suffix=".json", delete=False) as f:
            json.dump({"questions": []}, f)
            q_path = f.name
        try:
            code = main(["--questions", q_path, "--index-dir", "/tmp/non_existent_index_dir"])
            self.assertEqual(code, 1)
        finally:
            os.remove(q_path)


if __name__ == "__main__":
    unittest.main()
