"""Unit-тесты потокового сэмплера корпуса (docs/Tech/03-data.md, раздел 4).

Фикстуры — маленькие синтетические XML-строки в формате дампа Stack Overflow:
BOM, окончания CRLF, экранированные сущности, `Tags` вида `<c++><c#>`.
"""
import io
import os
import sys
import unittest
from datetime import datetime, timezone

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
sys.path.insert(0, os.path.join(REPO_ROOT, "services", "search_engine_indexer", "src"))

from indexer.sampler import (  # noqa: E402
    build_edges,
    build_neighbor_set,
    build_threads,
    in_sample,
    iter_lines,
    parse_link_row,
    parse_row,
    parse_tags,
    select_rows,
    thread_hash,
)

DENOM = 100
IN_S = [i for i in range(1, 5000) if in_sample(i, DENOM, 0)]
OUT_S = [i for i in range(1, 5000) if not in_sample(i, DENOM, 0)]


def row(**attrs):
    """Строка `<row .../>` в формате дампа (значения уже экранированы)."""
    body = " ".join(f'{k}="{v}"' for k, v in attrs.items())
    return f"  <row {body} />"


def question(qid, accepted=None, score=5, title="Title", body="Body", tags="&lt;python&gt;"):
    attrs = dict(Id=qid, PostTypeId=1, CreationDate="2008-08-07T00:45:51.980", Score=score)
    if accepted is not None:
        attrs["AcceptedAnswerId"] = accepted
    attrs.update(Body=body, Title=title, Tags=tags)
    return row(**attrs)


def answer(aid, parent, score=3, body="Answer"):
    return row(Id=aid, PostTypeId=2, ParentId=parent, CreationDate="2008-08-08T10:00:00.000",
               Score=score, Body=body)


class TestHash(unittest.TestCase):
    def test_known_values(self):
        # h(id) = (id * 2654435761) mod 2^32
        self.assertEqual(thread_hash(1), 2654435761)
        self.assertEqual(thread_hash(2), 1013904226)
        self.assertEqual(thread_hash(0), 0)

    def test_range(self):
        for i in (1, 12345, 2**31, 2**33 + 7):
            self.assertTrue(0 <= thread_hash(i) < 2**32)

    def test_denom_one_selects_everything(self):
        self.assertTrue(all(in_sample(i, 1, 0) for i in range(1, 500)))

    def test_fraction_close_to_one_over_denom(self):
        n = sum(in_sample(i, DENOM, 0) for i in range(1, 200001))
        self.assertTrue(1700 <= n <= 2300, n)

    def test_residues_partition_ids(self):
        for i in range(1, 1000):
            hits = [r for r in range(DENOM) if in_sample(i, DENOM, r)]
            self.assertEqual(len(hits), 1, i)

    def test_deterministic(self):
        self.assertEqual([in_sample(i, DENOM, 0) for i in range(1, 300)],
                         [in_sample(i, DENOM, 0) for i in range(1, 300)])


class TestParsing(unittest.TestCase):
    def test_parse_question_row(self):
        line = question(145, accepted=150, score=60, title="Compressing &amp; Files",
                        body="&lt;p&gt;a &amp;amp; b&lt;/p&gt;&#xA;", tags="&lt;c#&gt;&lt;.net&gt;")
        r = parse_row(line)
        self.assertEqual(r["id"], 145)
        self.assertEqual(r["post_type"], 1)
        self.assertEqual(r["accepted_answer_id"], 150)
        self.assertEqual(r["score"], 60)
        self.assertEqual(r["title"], "Compressing & Files")
        # XML-сущности раскрыты один раз: тело остаётся HTML с экранированным &amp;
        self.assertEqual(r["body"], "<p>a &amp; b</p>\n")
        self.assertEqual(r["tags"], ["c#", ".net"])
        self.assertEqual(r["creation_date"], "2008-08-07T00:45:51.980")

    def test_parse_answer_row(self):
        r = parse_row(answer(150, 145, score=7))
        self.assertEqual((r["id"], r["post_type"], r["parent_id"], r["score"]), (150, 2, 145, 7))
        self.assertIsNone(r["accepted_answer_id"])

    def test_parse_accepts_bytes_with_bom_and_crlf(self):
        raw = ("﻿" + question(1) + "\r\n").encode("utf-8")
        r = parse_row(raw)
        self.assertEqual(r["id"], 1)

    def test_non_row_lines_return_none(self):
        for line in ('<?xml version="1.0" encoding="utf-8"?>', "<posts>", "</posts>",
                     "<!--", "  ContentLicense", "  -->", ""):
            with self.subTest(line=line):
                self.assertIsNone(parse_row(line))

    def test_parse_tags(self):
        self.assertEqual(parse_tags("&lt;c++&gt;&lt;c#&gt;&lt;.net&gt;"), ["c++", "c#", ".net"])
        self.assertEqual(parse_tags("<c++><c#><.net>"), ["c++", "c#", ".net"])
        self.assertEqual(parse_tags(""), [])
        self.assertEqual(parse_tags(None), [])

    def test_parse_link_row(self):
        line = row(Id=1, CreationDate="2009-01-01T00:00:00.000", PostId=10, RelatedPostId=20, LinkTypeId=3)
        self.assertEqual(parse_link_row(line), (10, 20, 3))
        self.assertIsNone(parse_link_row("<links>"))


class TestIterLines(unittest.TestCase):
    def test_strips_bom_and_crlf_keeps_indentation(self):
        raw = ("﻿<?xml version=\"1.0\"?>\r\n" + question(1) + "\r\n</posts>\r\n").encode("utf-8")
        lines = list(iter_lines(io.BytesIO(raw)))
        self.assertEqual(lines[0], '<?xml version="1.0"?>')
        self.assertEqual(lines[1], question(1))
        self.assertEqual(lines[2], "</posts>")
        self.assertEqual(len(lines), 3)

    def test_works_on_any_binary_iterable(self):
        self.assertEqual(list(iter_lines([b"a\n", b"b\r\n"])), ["a", "b"])


class TestSelectRows(unittest.TestCase):
    def test_selects_questions_by_id_and_answers_by_parent(self):
        corpus = {10, 20}
        lines = [
            "<posts>",
            question(10, accepted=11),
            question(30, accepted=31),          # вне корпуса
            answer(11, 10),                     # ответ на вопрос из корпуса
            answer(31, 30),                     # ответ на вопрос вне корпуса
            answer(21, 20),                     # ответ на вопрос из корпуса (вопрос позже в файле)
            question(20, accepted=21),
            row(Id=99, PostTypeId=5, Score=0),  # tag-wiki
            row(Id=10, PostTypeId=4, Score=0),  # не вопрос: совпадение Id не считается
            "</posts>",
        ]
        out = list(select_rows(lines, lambda i: i in corpus))
        self.assertEqual([parse_row(x)["id"] for x in out], [10, 11, 21, 20])

    def test_hash_predicate_keeps_thread_together_in_any_order(self):
        qid = IN_S[3]
        aid = OUT_S[0]
        for order in ((question(qid, accepted=aid), answer(aid, qid)),
                      (answer(aid, qid), question(qid, accepted=aid))):
            out = list(select_rows(list(order), lambda i: in_sample(i, DENOM, 0)))
            self.assertEqual(len(out), 2, order)

    def test_output_is_normalized(self):
        lines = ["﻿" + question(10) + "\r"]
        out = list(select_rows(lines, lambda i: i == 10))
        self.assertEqual(out, [question(10)])


class TestNeighbors(unittest.TestCase):
    def test_outside_end_added_for_sampled_source(self):
        s, o = IN_S[0], OUT_S[0]
        self.assertEqual(build_neighbor_set([(s, o)], DENOM, 0, 20), {o})

    def test_outside_end_added_for_sampled_target(self):
        s, o = IN_S[0], OUT_S[0]
        self.assertEqual(build_neighbor_set([(o, s)], DENOM, 0, 20), {o})

    def test_link_without_sampled_end_ignored(self):
        self.assertEqual(build_neighbor_set([(OUT_S[0], OUT_S[1])], DENOM, 0, 20), set())

    def test_both_ends_sampled_adds_nothing(self):
        self.assertEqual(build_neighbor_set([(IN_S[0], IN_S[1])], DENOM, 0, 20), set())

    def test_denom_one_has_no_neighbors(self):
        self.assertEqual(build_neighbor_set([(1, 2), (3, 4)], 1, 0, 20), set())

    def test_hub_capped_deterministically(self):
        hub = IN_S[0]
        outsiders = OUT_S[:50]
        links = [(hub, o) for o in outsiders]
        got = build_neighbor_set(links, DENOM, 0, 20)
        self.assertEqual(got, set(outsiders[:20]))

    def test_cap_is_per_sampled_node(self):
        a, b = IN_S[0], IN_S[1]
        links = [(a, o) for o in OUT_S[:30]] + [(b, o) for o in OUT_S[100:130]]
        got = build_neighbor_set(links, DENOM, 0, 20)
        self.assertEqual(len(got), 40)


class TestBuildThreads(unittest.TestCase):
    def rows(self, *lines):
        return [parse_row(x) for x in lines]

    def test_joins_question_with_accepted_answer_any_order(self):
        rows = self.rows(
            answer(11, 10, score=4, body="&lt;p&gt;fix it&lt;/p&gt;"),
            question(10, accepted=11, score=9, title="NPE", body="&lt;p&gt;why&lt;/p&gt;", tags="&lt;java&gt;"),
        )
        (t,) = build_threads(rows, min_score=1)
        self.assertEqual(t.doc_id, 10)
        self.assertEqual(t.title, "NPE")
        self.assertEqual(t.question_body, "<p>why</p>")
        self.assertEqual(t.tags, ["java"])
        self.assertEqual(t.question_score, 9)
        self.assertEqual(t.accepted_answer_id, 11)
        self.assertEqual(t.best_answer_body, "<p>fix it</p>")
        self.assertEqual(t.best_answer_score, 4)
        expected = int(datetime(2008, 8, 7, 0, 45, 51, tzinfo=timezone.utc).timestamp())
        self.assertEqual(t.created_at, expected)

    def test_non_accepted_answer_ignored_even_with_higher_score(self):
        rows = self.rows(
            question(10, accepted=11),
            answer(12, 10, score=100, body="better?"),
            answer(11, 10, score=1, body="accepted"),
        )
        (t,) = build_threads(rows)
        self.assertEqual(t.best_answer_body, "accepted")

    def test_dropped_without_accepted_answer_id(self):
        rows = self.rows(question(10), answer(11, 10))
        self.assertEqual(build_threads(rows), [])

    def test_dropped_when_accepted_answer_row_missing(self):
        rows = self.rows(question(10, accepted=11))
        self.assertEqual(build_threads(rows), [])

    def test_accepted_answer_must_belong_to_question(self):
        rows = self.rows(question(10, accepted=11), answer(11, 999))
        self.assertEqual(build_threads(rows), [])

    def test_min_score_filter(self):
        rows = self.rows(question(10, accepted=11, score=0), answer(11, 10),
                         question(20, accepted=21, score=1), answer(21, 20))
        self.assertEqual([t.doc_id for t in build_threads(rows, min_score=1)], [20])
        self.assertEqual([t.doc_id for t in build_threads(rows, min_score=0)], [10, 20])

    def test_sorted_by_doc_id_and_ignores_other_post_types(self):
        rows = self.rows(
            question(30, accepted=31), answer(31, 30),
            question(10, accepted=11), answer(11, 10),
            row(Id=7, PostTypeId=5, Score=0),
        )
        self.assertEqual([t.doc_id for t in build_threads(rows)], [10, 30])


class TestEdges(unittest.TestCase):
    def test_types_mapped_and_filtered_to_valid_ids(self):
        links = [(1, 2, 3), (2, 3, 1), (1, 9, 3), (8, 1, 1), (1, 3, 5)]
        edges = build_edges(links, {1, 2, 3})
        self.assertEqual(edges, [
            (1, 2, "DUPLICATE_OF", 1.0),
            (2, 3, "LINKED_IN_ANSWER", 1.0),
        ])

    def test_duplicates_collapsed_and_sorted(self):
        links = [(5, 6, 1), (1, 2, 3), (5, 6, 1)]
        edges = build_edges(links, {1, 2, 5, 6})
        self.assertEqual(edges, [(1, 2, "DUPLICATE_OF", 1.0), (5, 6, "LINKED_IN_ANSWER", 1.0)])

    def test_self_links_dropped(self):
        self.assertEqual(build_edges([(1, 1, 1)], {1}), [])


if __name__ == "__main__":
    unittest.main()
