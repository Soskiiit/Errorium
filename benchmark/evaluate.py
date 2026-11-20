"""Бенчмарк качества ранжирования НИРС (занятие 3; docs/Tech/05-verification.md, раздел 3).

Прогоняет 10 обычных вопросов золотого корпуса через поисковое ядро (core.engine,
in-process) по индексу режима `corpus` и считает Recall@{3,5,10} и MRR (глубина 10).
Вопросы-ловушки в метрики поиска не входят (ожидаемый ответ системы — отказ, зона RAG).

Запуск: `make benchmark` (собирает индекс корпуса и вызывает этот скрипт в контейнере)
или `python benchmark/evaluate.py` на хосте после `make index-corpus`.
Пороги приёмки (README ядра): Recall@10 >= 0.75, MRR >= 0.50, P95 <= 150 мс.
"""
import argparse
import json
import math
import os
import sys
import time

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
for _p in (os.path.join(REPO_ROOT, "services", "search_engine_core", "src"),
           os.path.join(REPO_ROOT, "services", "search_engine_indexer", "src")):
    if os.path.isdir(_p) and _p not in sys.path:
        sys.path.insert(0, _p)

from core.config import Config  # noqa: E402
from core.engine import SearchEngine  # noqa: E402
from core.reader import IndexReader  # noqa: E402

DEPTH = 10
RECALL_KS = (3, 5, 10)
THRESHOLDS = {"recall@10": 0.75, "mrr": 0.50, "p95_ms": 150.0}


def doc_id_from_label(label: str) -> int:
    return int(label[1:])


def percentile(values, p):
    if not values:
        return 0.0
    ordered = sorted(values)
    idx = max(0, math.ceil(p / 100.0 * len(ordered)) - 1)
    return ordered[idx]


def evaluate(index_dir: str, questions_path: str, k1: float, b: float):
    with open(questions_path, encoding="utf-8") as f:
        questions = json.load(f)["questions"]
    answerable = [q for q in questions if q["type"] == "answerable"]
    traps = [q for q in questions if q["type"] == "trap"]

    engine = SearchEngine(IndexReader(index_dir), k1=k1, b=b)
    engine.search("warmup")  # загрузка индекса не входит в латентность запросов

    per_question = []
    latencies = []
    for q in answerable:
        relevant = {doc_id_from_label(label) for label in q["relevant_docs"]}
        result = engine.search(q["question"], top_k=DEPTH)
        latencies.append(result.latency_ms)
        ranked = [d for d, _ in result.hits]
        rr = 0.0
        for rank, doc_id in enumerate(ranked, start=1):
            if doc_id in relevant:
                rr = 1.0 / rank
                break
        recalls = {k: len(relevant & set(ranked[:k])) / len(relevant) for k in RECALL_KS}
        per_question.append({
            "id": q["id"], "question": q["question"],
            "relevant_docs": sorted(relevant),
            "retrieved": [{"doc_id": d, "score": s} for d, s in result.hits],
            "recall": {f"@{k}": v for k, v in recalls.items()},
            "reciprocal_rank": rr, "latency_ms": result.latency_ms,
        })

    n = len(per_question)
    metrics = {f"recall@{k}": sum(q["recall"][f"@{k}"] for q in per_question) / n
               for k in RECALL_KS}
    metrics["mrr"] = sum(q["reciprocal_rank"] for q in per_question) / n
    metrics["p50_ms"] = percentile(latencies, 50)
    metrics["p95_ms"] = percentile(latencies, 95)
    metrics["max_ms"] = max(latencies)
    return {
        "index_dir": index_dir, "k1": k1, "b": b, "depth": DEPTH,
        "num_questions": n, "num_traps_excluded": len(traps),
        "timestamp": int(time.time()),
        "metrics": metrics, "questions": per_question,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Benchmark Recall@k / MRR поискового ядра")
    parser.add_argument("--index-dir", default=os.environ.get(
        "INDEX_DIR", os.path.join(REPO_ROOT, "data", "index-benchmark")))
    parser.add_argument("--questions", default=os.environ.get(
        "BENCHMARK_QUESTIONS", os.path.join(REPO_ROOT, "benchmark", "questions.json")))
    parser.add_argument("--output", help="Сохранить JSON-отчёт (выдача по каждому вопросу)")
    args = parser.parse_args()

    cfg = Config.from_env({**os.environ, "INDEX_DIR": args.index_dir})
    report = evaluate(args.index_dir, args.questions, cfg.k1, cfg.b)
    metrics = report["metrics"]

    print(f"Индекс: {args.index_dir}  (k1={cfg.k1}, b={cfg.b}, глубина {DEPTH})")
    print(f"Вопросов: {report['num_questions']} обычных; ловушек исключено: {report['num_traps_excluded']}")
    print()
    print(f"{'ID':<5} {'R@3':>6} {'R@5':>6} {'R@10':>6} {'RR':>6} {'мс':>8}  Top-10")
    for q in report["questions"]:
        top = ",".join(str(h["doc_id"]) for h in q["retrieved"])
        print(f"{q['id']:<5} {q['recall']['@3']:>6.3f} {q['recall']['@5']:>6.3f} "
              f"{q['recall']['@10']:>6.3f} {q['reciprocal_rank']:>6.3f} "
              f"{q['latency_ms']:>8.2f}  {top}")
    print()
    for k in RECALL_KS:
        print(f"Recall@{k:<2} = {metrics[f'recall@{k}']:.3f}")
    print(f"MRR@{DEPTH}  = {metrics['mrr']:.3f}")
    print(f"Латентность: P50={metrics['p50_ms']:.2f} мс, P95={metrics['p95_ms']:.2f} мс, "
          f"max={metrics['max_ms']:.2f} мс")

    failed = []
    for name, threshold in THRESHOLDS.items():
        value = metrics[name]
        ok = value >= threshold if name != "p95_ms" else value <= threshold
        print(f"Порог {name} {'>= ' if name != 'p95_ms' else '<= '}{threshold}: "
              f"{'OK' if ok else 'ПРОВАЛ'} ({value:.3f})")
        if not ok:
            failed.append(name)

    if args.output:
        report["thresholds_passed"] = not failed
        with open(args.output, "w", encoding="utf-8") as f:
            json.dump(report, f, ensure_ascii=False, indent=2)
        print(f"Отчёт сохранён: {args.output}")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())