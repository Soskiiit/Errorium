"""Проверка структуры золотого набора benchmark (docs/Tech/05-verification.md, раздел 3)."""
import json
import os
import re
import unittest

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
DOCS_PATH = os.path.join(REPO_ROOT, "benchmark", "corpus", "documents.jsonl")
QUESTIONS_PATH = os.path.join(REPO_ROOT, "benchmark", "questions.json")
REFUSAL = "В документах нет этой информации."
MAX_SHARED_WORDS = 7


def load_documents():
    with open(DOCS_PATH, encoding="utf-8") as f:
        return [json.loads(line) for line in f if line.strip()]


def load_questions():
    with open(QUESTIONS_PATH, encoding="utf-8") as f:
        return json.load(f)


def words(text):
    return re.findall(r"[a-z0-9_]+", text.lower())


def longest_shared_run(a, b):
    """Длина самой длинной общей непрерывной последовательности слов."""
    best = 0
    prev = [0] * (len(b) + 1)
    for i in range(1, len(a) + 1):
        cur = [0] * (len(b) + 1)
        for j in range(1, len(b) + 1):
            if a[i - 1] == b[j - 1]:
                cur[j] = prev[j - 1] + 1
                best = max(best, cur[j])
        prev = cur
    return best


class TestDocuments(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.docs = load_documents()

    def test_thirty_documents_with_contiguous_ids(self):
        self.assertEqual([d["doc_id"] for d in self.docs], list(range(1, 31)))

    def test_required_fields(self):
        for d in self.docs:
            with self.subTest(doc_id=d["doc_id"]):
                self.assertIsInstance(d["title"], str)
                self.assertTrue(d["title"].strip())
                self.assertTrue(d["text"].startswith("Question: "))
                self.assertIn("\n\nAnswer: ", d["text"])
                self.assertTrue(500 <= len(d["text"]) <= 6000, len(d["text"]))
                self.assertTrue(d["tags"] and all(isinstance(t, str) for t in d["tags"]))

    def test_source_links_unique_and_consistent(self):
        so_ids = [d["so_id"] for d in self.docs]
        self.assertEqual(len(set(so_ids)), 30)
        for d in self.docs:
            self.assertEqual(d["url"], f"https://stackoverflow.com/questions/{d['so_id']}")

    def test_text_is_plain(self):
        for d in self.docs:
            with self.subTest(doc_id=d["doc_id"]):
                self.assertNotRegex(d["text"], r"</?(p|pre|code|a|li|ul|strong|em)\b")
                self.assertNotIn("&lt;", d["text"])
                self.assertNotIn("&amp;", d["text"])

    def test_code_indentation_preserved_somewhere(self):
        self.assertTrue(any(re.search(r"\n    \S", d["text"]) for d in self.docs))


class TestQuestions(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.docs = load_documents()
        cls.data = load_questions()
        cls.questions = cls.data["questions"]
        cls.labels = {f"D{d['doc_id']:02d}" for d in cls.docs}

    def test_fifteen_questions_ten_plus_five(self):
        self.assertEqual([q["id"] for q in self.questions], [f"Q{i:02d}" for i in range(1, 16)])
        self.assertEqual([q["type"] for q in self.questions], ["answerable"] * 10 + ["trap"] * 5)

    def test_every_question_has_text_and_reference(self):
        for q in self.questions:
            with self.subTest(id=q["id"]):
                self.assertTrue(q["question"].strip().endswith("?"))
                self.assertTrue(q["reference_answer"].strip())

    def test_answerable_have_at_least_three_valid_distinct_relevant_docs(self):
        for q in self.questions[:10]:
            with self.subTest(id=q["id"]):
                rel = q["relevant_docs"]
                self.assertGreaterEqual(len(rel), 3)
                self.assertEqual(len(rel), len(set(rel)))
                self.assertTrue(set(rel) <= self.labels)

    def test_traps_have_no_relevant_docs_and_expect_refusal(self):
        for q in self.questions[10:]:
            with self.subTest(id=q["id"]):
                self.assertEqual(q["relevant_docs"], [])
                self.assertEqual(q["reference_answer"], REFUSAL)

    def test_every_document_is_relevant_to_some_question(self):
        covered = {label for q in self.questions for label in q["relevant_docs"]}
        self.assertEqual(covered, self.labels)

    def test_questions_are_unique(self):
        texts = [q["question"] for q in self.questions]
        self.assertEqual(len(texts), len(set(texts)))

    def test_questions_do_not_copy_document_fragments(self):
        for q in self.questions:
            qw = words(q["question"])
            for d in self.docs:
                shared = longest_shared_run(qw, words(d["title"] + " " + d["text"]))
                with self.subTest(question=q["id"], doc=d["doc_id"]):
                    self.assertLessEqual(shared, MAX_SHARED_WORDS)

    def test_trap_topics_absent_from_corpus(self):
        corpus_words = set()
        for d in self.docs:
            corpus_words.update(words(d["title"] + " " + d["text"]))
        for term in ("rust", "kubernetes", "crashloopbackoff", "gradle", "postgresql", "react", "useeffect", "borrow"):
            self.assertNotIn(term, corpus_words, term)


if __name__ == "__main__":
    unittest.main()
