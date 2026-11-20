#!/usr/bin/env python3
"""Расчёт метрик поиска НИРС (Recall@k, MRR@k) на бенчмарк-корпусе.

Контрольные метрики качества поиска:
- Recall@3, Recall@5, Recall@10
- MRR@10 (Mean Reciprocal Rank)

Требования NFR:
- Recall@10 >= 0.75
- MRR@10 >= 0.50
"""

import argparse
import json
import os
import sys
from typing import Any, Dict, Iterable, List, Optional, Sequence, Tuple, Union

# Добавляем пути к сервисам ядра и индексатора
PROJECT_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
CORE_SRC = os.path.join(PROJECT_ROOT, "services", "search_engine_core", "src")
INDEXER_SRC = os.path.join(PROJECT_ROOT, "services", "search_engine_indexer", "src")

if CORE_SRC not in sys.path:
    sys.path.insert(0, CORE_SRC)
if INDEXER_SRC not in sys.path:
    sys.path.insert(0, INDEXER_SRC)

try:
    from core.engine import Engine
except ImportError:
    Engine = None  # type: ignore[assignment,misc]

# NFR пороги качества
NFR_RECALL_10 = 0.75
NFR_MRR_10 = 0.50


def parse_doc_id(label: Union[str, int]) -> int:
    """Преобразует строковый идентификатор документа ('D01', 'D11', 1) в int (1, 11, 1)."""
    if isinstance(label, int):
        return label
    label_str = str(label).strip()
    if label_str.startswith(("D", "d")):
        return int(label_str[1:])
    return int(label_str)


def parse_relevant_docs(labels: Iterable[Union[str, int]]) -> List[int]:
    """Преобразует список меток релевантных документов в список int doc_id."""
    return [parse_doc_id(label) for label in labels]


def recall_at_k(relevant_docs: Sequence[int], retrieved_docs: Sequence[int], k: int) -> float:
    """Вычисляет Recall@k = |relevant ∩ top-k| / |relevant|.

    Если relevant_docs пуст или k <= 0, возвращает 0.0.
    """
    if not relevant_docs or k <= 0:
        return 0.0
    relevant_set = set(relevant_docs)
    retrieved_k = set(retrieved_docs[:k])
    hits = relevant_set & retrieved_k
    return len(hits) / float(len(relevant_set))


def mrr_at_k(relevant_docs: Sequence[int], retrieved_docs: Sequence[int], k: int = 10) -> float:
    """Вычисляет Reciprocal Rank для одного запроса в пределах top-k.

    Возвращает 1/rank первого найденного релевантного документа (rank начинается с 1).
    Если ни один релевантный документ не найден в пределах top-k, возвращает 0.0.
    """
    if not relevant_docs or not retrieved_docs or k <= 0:
        return 0.0
    relevant_set = set(relevant_docs)
    for idx, doc_id in enumerate(retrieved_docs[:k]):
        if doc_id in relevant_set:
            return 1.0 / float(idx + 1)
    return 0.0


def evaluate_dataset(engine: Any, questions: Sequence[Dict[str, Any]], top_k: int = 10) -> Dict[str, Any]:
    """Прогоняет вопросы через поисковый движок и вычисляет метрики."""
    question_results: List[Dict[str, Any]] = []

    for q in questions:
        q_id = q.get("id", "")
        q_type = q.get("type", "answerable")
        question_text = q.get("question", "")
        raw_relevant = q.get("relevant_docs", [])
        relevant_docs = parse_relevant_docs(raw_relevant)

        search_res = engine.search(question_text, top_k=top_k)
        hits = getattr(search_res, "hits", search_res)

        retrieved_docs: List[int] = []
        scores: List[float] = []
        for hit in hits:
            doc_id = getattr(hit, "doc_id", None)
            if doc_id is None and isinstance(hit, dict):
                doc_id = hit.get("doc_id")
            score = getattr(hit, "score", None)
            if score is None and isinstance(hit, dict):
                score = hit.get("score")
            retrieved_docs.append(int(doc_id))
            scores.append(float(score) if score is not None else 0.0)

        top1_doc_id = retrieved_docs[0] if retrieved_docs else None
        top1_score = scores[0] if scores else None

        if q_type == "answerable":
            r3 = recall_at_k(relevant_docs, retrieved_docs, 3)
            r5 = recall_at_k(relevant_docs, retrieved_docs, 5)
            r10 = recall_at_k(relevant_docs, retrieved_docs, 10)
            mrr10 = mrr_at_k(relevant_docs, retrieved_docs, 10)
        else:
            # Для ловушек (trap) метрики Recall и MRR НЕ рассчитываются
            r3 = None
            r5 = None
            r10 = None
            mrr10 = None

        question_results.append({
            "id": q_id,
            "type": q_type,
            "question": question_text,
            "relevant_docs": relevant_docs,
            "retrieved_docs": retrieved_docs,
            "top1_doc_id": top1_doc_id,
            "top1_score": top1_score,
            "recall_at_3": r3,
            "recall_at_5": r5,
            "recall_at_10": r10,
            "mrr_at_10": mrr10,
        })

    answerable_questions = [q for q in question_results if q["type"] == "answerable"]
    trap_questions = [q for q in question_results if q["type"] == "trap"]
    n_ans = len(answerable_questions)

    if n_ans > 0:
        mean_r3 = sum(q["recall_at_3"] for q in answerable_questions) / float(n_ans)  # type: ignore[arg-type]
        mean_r5 = sum(q["recall_at_5"] for q in answerable_questions) / float(n_ans)  # type: ignore[arg-type]
        mean_r10 = sum(q["recall_at_10"] for q in answerable_questions) / float(n_ans)  # type: ignore[arg-type]
        mean_mrr10 = sum(q["mrr_at_10"] for q in answerable_questions) / float(n_ans)  # type: ignore[arg-type]
    else:
        mean_r3 = 0.0
        mean_r5 = 0.0
        mean_r10 = 0.0
        mean_mrr10 = 0.0

    r10_passed = mean_r10 >= NFR_RECALL_10
    mrr_passed = mean_mrr10 >= NFR_MRR_10
    all_passed = r10_passed and mrr_passed

    violations: List[str] = []
    if not r10_passed:
        violations.append(f"Recall@10 = {mean_r10:.4f} < порога NFR {NFR_RECALL_10:.4f}")
    if not mrr_passed:
        violations.append(f"MRR@10 = {mean_mrr10:.4f} < порога NFR {NFR_MRR_10:.4f}")

    return {
        "total_questions": len(question_results),
        "answerable_count": n_ans,
        "trap_count": len(trap_questions),
        "metrics": {
            "recall_at_3": mean_r3,
            "recall_at_5": mean_r5,
            "recall_at_10": mean_r10,
            "mrr_at_10": mean_mrr10,
        },
        "nfr": {
            "recall_at_10_threshold": NFR_RECALL_10,
            "mrr_at_10_threshold": NFR_MRR_10,
            "recall_at_10_passed": r10_passed,
            "mrr_at_10_passed": mrr_passed,
            "passed": all_passed,
            "violations": violations,
        },
        "questions": question_results,
    }


def print_evaluation_report(summary: Dict[str, Any]) -> None:
    """Выводит красивую форматированную таблицу результатов в stdout."""
    print("=" * 95)
    print("ОЦЕНКА КАЧЕСТВА ПОИСКА (BENCHMARK EVALUATION — BM25)")
    print("=" * 95)
    header = f"{'ID':<5} | {'Тип':<10} | {'Relevant':<14} | {'Top-1 (Score)':<18} | {'R@3':<7} | {'R@5':<7} | {'R@10':<7} | {'MRR@10':<7}"
    print(header)
    print("-" * 95)

    for q in summary["questions"]:
        q_id = q["id"]
        q_type = q["type"]
        rel_str = str(q["relevant_docs"]) if q["relevant_docs"] else "[]"
        if len(rel_str) > 14:
            rel_str = rel_str[:12] + ".."

        if q["top1_doc_id"] is not None:
            score_str = f"{q['top1_score']:.2f}" if q["top1_score"] is not None else "0.00"
            top1_str = f"D{q['top1_doc_id']:02d} ({score_str})"
        else:
            top1_str = "None"

        if q_type == "answerable":
            r3_str = f"{q['recall_at_3']:.4f}"
            r5_str = f"{q['recall_at_5']:.4f}"
            r10_str = f"{q['recall_at_10']:.4f}"
            mrr_str = f"{q['mrr_at_10']:.4f}"
        else:
            r3_str = "-"
            r5_str = "-"
            r10_str = "-"
            mrr_str = "-"

        row = f"{q_id:<5} | {q_type:<10} | {rel_str:<14} | {top1_str:<18} | {r3_str:<7} | {r5_str:<7} | {r10_str:<7} | {mrr_str:<7}"
        print(row)

    print("-" * 95)
    print("ИТОГИ:")
    metrics = summary["metrics"]
    nfr = summary["nfr"]
    print(f"  Всего вопросов:      {summary['total_questions']} (Answerable: {summary['answerable_count']}, Trap: {summary['trap_count']})")
    print(f"  Recall@3:            {metrics['recall_at_3']:.4f}")
    print(f"  Recall@5:            {metrics['recall_at_5']:.4f}")

    r10_status = "PASS" if nfr["recall_at_10_passed"] else "FAIL"
    mrr_status = "PASS" if nfr["mrr_at_10_passed"] else "FAIL"
    print(f"  Recall@10:           {metrics['recall_at_10']:.4f}  (NFR >= {nfr['recall_at_10_threshold']:.2f} -> {r10_status})")
    print(f"  MRR@10:              {metrics['mrr_at_10']:.4f}  (NFR >= {nfr['mrr_at_10_threshold']:.2f} -> {mrr_status})")
    print("=" * 95)

    if nfr["passed"]:
        print("[SUCCESS] Все требования NFR выполнены.")
    else:
        print("[FAILURE] Нарушены требования NFR:")
        for violation in nfr["violations"]:
            print(f"  - {violation}")
    print("=" * 95)


def main(argv: Optional[Sequence[str]] = None) -> int:
    parser = argparse.ArgumentParser(description="Расчет метрик качества поиска (Recall@k, MRR@k)")
    parser.add_argument(
        "--questions",
        default=os.path.join(PROJECT_ROOT, "benchmark", "questions.json"),
        help="Путь к файлу вопросов questions.json",
    )
    parser.add_argument(
        "--index-dir",
        default=os.environ.get("INDEX_DIR", os.path.join(PROJECT_ROOT, "data", "index-benchmark")),
        help="Путь к каталогу индекса (по умолчанию $INDEX_DIR или data/index-benchmark)",
    )
    parser.add_argument(
        "--json",
        dest="json_path",
        default=None,
        help="Опциональный путь для сохранения отчета в формате JSON",
    )
    parser.add_argument(
        "--top-k",
        type=int,
        default=10,
        help="Количество кандидатов top_k (по умолчанию 10)",
    )

    args = parser.parse_args(argv)

    if not os.path.isfile(args.questions):
        print(f"Ошибка: файл вопросов не найден: {args.questions}", file=sys.stderr)
        return 1

    if not os.path.isdir(args.index_dir):
        print(f"Ошибка: каталог индекса не найден: {args.index_dir}", file=sys.stderr)
        print("Постройте индекс: make index-corpus", file=sys.stderr)
        return 1

    if Engine is None:
        print("Ошибка: не удалось импортировать core.engine.Engine", file=sys.stderr)
        return 1

    with open(args.questions, "r", encoding="utf-8") as f:
        data = json.load(f)

    questions = data.get("questions", data if isinstance(data, list) else [])
    if not questions:
        print("Ошибка: список вопросов пуст", file=sys.stderr)
        return 1

    engine = Engine.load(args.index_dir)
    summary = evaluate_dataset(engine, questions, top_k=args.top_k)

    print_evaluation_report(summary)

    if args.json_path:
        with open(args.json_path, "w", encoding="utf-8") as f:
            json.dump(summary, f, ensure_ascii=False, indent=2)
        print(f"JSON-отчет сохранен в {args.json_path}")

    return 0 if summary["nfr"]["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
