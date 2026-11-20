"""BM25 строго по формуле кафедры ИУ5 (AGENTS.md п. 2.1; docs/Tech/03-data.md п. 6.3).

Только стандартная библиотека; готовые поисковые/ранжирующие библиотеки запрещены.
"""
import heapq
import math
from typing import Dict, Iterable, List, Mapping, Sequence, Tuple


def idf(n: int, df: int) -> float:
    """IDF(t) = ln(1 + (N - DF + 0.5) / (DF + 0.5))."""
    return math.log(1 + (n - df + 0.5) / (df + 0.5))


def term_score(tf: int, idf_value: float, doc_len: int, avgdl: float,
               k1: float, b: float) -> float:
    """Вклад одного терма: IDF * TF*(k1+1) / (TF + k1*(1-b+b*|d|/avgdl))."""
    norm = k1 * (1 - b + b * doc_len / avgdl)
    return idf_value * tf * (k1 + 1) / (tf + norm)


def score_documents(term_postings: Iterable[Tuple[int, Sequence[Tuple[int, int]]]],
                    lengths: Mapping[int, int], n: int, avgdl: float,
                    k1: float, b: float) -> Dict[int, float]:
    """Накопление баллов по уникальным термам запроса: ``(df, [(doc_id, tf), ...])``."""
    scores: Dict[int, float] = {}
    for df, postings in term_postings:
        if not postings:
            continue
        weight = idf(n, df)
        for doc_id, tf in postings:
            scores[doc_id] = scores.get(doc_id, 0.0) + term_score(
                tf, weight, lengths[doc_id], avgdl, k1, b)
    return scores


def top_k(scores: Mapping[int, float], k: int) -> List[Tuple[int, float]]:
    """Min-heap размера k, ключ ``(score, -doc_id)``.

    Итог: убывание балла, равные баллы — по возрастанию doc_id (занятие 3).
    """
    if k <= 0 or not scores:
        return []
    heap: List[Tuple[float, int]] = []
    for doc_id, score in scores.items():
        item = (score, -doc_id)
        if len(heap) < k:
            heapq.heappush(heap, item)
        elif item > heap[0]:
            heapq.heapreplace(heap, item)
    return [(-neg_doc, score) for score, neg_doc in sorted(heap, reverse=True)]