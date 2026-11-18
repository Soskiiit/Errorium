"""Конвейеры индексации: ``sample`` (выборка из дампа SO) и ``corpus`` (малый JSONL-корпус).

docs/Tech/03-data.md, п. 4.5 и 4.9. Только стандартная библиотека (ADR-008).
"""
import hashlib
import json
import os
import re
import shutil
import time
from typing import Callable, Dict, Iterable, Iterator, List, Optional, Set, Tuple

from .config import Config
from .graph_db import node_from_corpus_doc, node_from_thread, write_graph_db
from .sampler import (
    Thread,
    build_edges,
    build_neighbor_set,
    build_threads,
    in_sample,
    iter_lines,
    parse_link_row,
    parse_row,
    select_rows,
)
from .sources import archive_info, open_member
from .spimi import build_index
from .tokenizer import TOKENIZER_VERSION, clean_html, tokenize

SAMPLE_DIRNAME = "sample"
POSTS_SAMPLE = "Posts.sample.xml"
MANIFEST_NAME = "sample_manifest.json"
STAGING_DIRNAME = ".staging"

_POST_TYPE = re.compile(r' PostTypeId="(\d+)"')
Log = Callable[..., None]


def document_text(thread: Thread) -> str:
    """Индексируемый текст треда: заголовок + вопрос + принятый ответ (п. 4.3)."""
    return " ".join((thread.title, clean_html(thread.question_body), clean_html(thread.best_answer_body)))


def _links(lines: Iterable[str]) -> Iterator[Tuple[int, int, int]]:
    for line in lines:
        parsed = parse_link_row(line)
        if parsed is not None:
            yield parsed


def _write_json(path: str, data: dict) -> None:
    with open(path, "w", encoding="utf-8") as f:
        f.write(json.dumps(data, ensure_ascii=False, sort_keys=True, indent=2) + "\n")


def _fresh_dir(path: str) -> str:
    shutil.rmtree(path, ignore_errors=True)
    os.makedirs(path)
    return path


# ------------------------------------------------------------------------ sample

def _write_sample_posts(cfg: Config, in_corpus: Callable[[int], bool], dest: str) -> Dict[str, object]:
    """Проход по ``Posts.xml``: пишет отобранные строки (LF, без BOM) и считает статистику."""
    counts = {"questions": 0, "answers": 0}
    digest = hashlib.sha256()
    size = 0
    tmp = dest + ".tmp"
    with open_member(cfg, "Posts.xml") as stream, open(tmp, "wb", buffering=1 << 20) as out:
        for line in select_rows(iter_lines(stream), in_corpus):
            data = (line + "\n").encode("utf-8")
            out.write(data)
            digest.update(data)
            size += len(data)
            m = _POST_TYPE.search(line, 0, 120)
            if m:
                counts["questions" if m.group(1) == "1" else "answers"] += 1
    os.replace(tmp, dest)
    return {"questions": counts["questions"], "answers": counts["answers"],
            "bytes": size, "sha256": digest.hexdigest()}


def _copy_member(cfg: Config, member: str, dest: str) -> None:
    tmp = dest + ".tmp"
    with open_member(cfg, member) as stream, open(tmp, "wb") as out:
        shutil.copyfileobj(stream, out)
    os.replace(tmp, dest)


def _iter_rows(path: str) -> Iterator[dict]:
    with open(path, "rb") as f:
        for line in iter_lines(f):
            row = parse_row(line)
            if row is not None:
                yield row


def _load_threads(posts_path: str, min_score: int) -> List[Thread]:
    """Два прохода по ``Posts.sample.xml``: в ОЗУ остаются только валидные треды (п. 4.5)."""
    valid: Dict[int, int] = {}
    for row in _iter_rows(posts_path):
        if row["post_type"] == 1 and row["accepted_answer_id"] and row["score"] >= min_score:
            valid[row["id"]] = row["accepted_answer_id"]
    accepted_ids: Set[int] = set(valid.values())
    needed = (r for r in _iter_rows(posts_path)
              if (r["post_type"] == 1 and r["id"] in valid) or (r["post_type"] == 2 and r["id"] in accepted_ids))
    return build_threads(needed, min_score)


def run_sample(cfg: Config, log: Log = print, timestamp: Optional[int] = None) -> dict:
    """Строит индекс по хеш-выборке тредов из дампа и публикует его в ``INDEX_OUTPUT_PATH``."""
    started = time.time()
    ts = int(started) if timestamp is None else int(timestamp)
    denom, residue = cfg.sample_denom, cfg.sample_residue

    log(f"[1/5] PostLinks.xml: соседи выборки (1/{denom}, остаток {residue}, лимит {cfg.graph_max_neighbors})")
    with open_member(cfg, "PostLinks.xml") as stream:
        neighbors = build_neighbor_set(_links(iter_lines(stream)), denom, residue, cfg.graph_max_neighbors)
    log(f"      соседей вне выборки: {len(neighbors)}")

    def in_corpus(thread_id: int) -> bool:
        return in_sample(thread_id, denom, residue) or thread_id in neighbors

    log("[2/5] Posts.xml: отбор строк выборки")
    sample_dir = os.path.join(cfg.raw_data_path, SAMPLE_DIRNAME)
    os.makedirs(sample_dir, exist_ok=True)
    posts_path = os.path.join(sample_dir, POSTS_SAMPLE)
    stats = _write_sample_posts(cfg, in_corpus, posts_path)
    _copy_member(cfg, "Tags.xml", os.path.join(sample_dir, "Tags.xml"))
    log(f"      вопросов: {stats['questions']}, ответов: {stats['answers']}, байт: {stats['bytes']}")

    log("[3/5] Сборка валидных тредов (вопрос + принятый ответ)")
    threads = _load_threads(posts_path, cfg.sample_min_score)
    valid_ids = {t.doc_id for t in threads}
    log(f"      валидных тредов: {len(threads)}")

    log("[4/5] PostLinks.xml: рёбра графа между валидными тредами")
    with open_member(cfg, "PostLinks.xml") as stream:
        edges = build_edges(_links(iter_lines(stream)), valid_ids)
    log(f"      рёбер: {len(edges)}")

    log("[5/5] Токенизация, SPIMI-индекс и graph.db")
    os.makedirs(cfg.index_output_path, exist_ok=True)
    staging = _fresh_dir(os.path.join(cfg.index_output_path, STAGING_DIRNAME))
    try:
        graph_counts = write_graph_db(os.path.join(staging, "graph.db"),
                                      (node_from_thread(t) for t in threads), edges)
        manifest = {
            "mode": "sample",
            "tokenizer_version": TOKENIZER_VERSION,
            "sample_denom": denom,
            "sample_residue": residue,
            "sample_min_score": cfg.sample_min_score,
            "graph_max_neighbors": cfg.graph_max_neighbors,
            "archive": archive_info(cfg),
            "neighbors": len(neighbors),
            "questions_selected": stats["questions"],
            "answers_selected": stats["answers"],
            "posts_sample_bytes": stats["bytes"],
            "posts_sample_sha256": stats["sha256"],
            "valid_threads": len(threads),
            "edges": graph_counts["edges"],
            "build_timestamp": ts,
        }
        _write_json(os.path.join(staging, MANIFEST_NAME), manifest)
        docs = ((t.doc_id, tokenize(document_text(t))) for t in threads)
        meta = build_index(
            docs, cfg.index_output_path, index_mode="sample",
            block_postings=cfg.spimi_block_postings, timestamp=ts,
            source={"manifest": MANIFEST_NAME, "sample_denom": denom, "sample_residue": residue,
                    "sample_min_score": cfg.sample_min_score, "graph_max_neighbors": cfg.graph_max_neighbors},
            extra_files={"graph.db": os.path.join(staging, "graph.db"),
                         MANIFEST_NAME: os.path.join(staging, MANIFEST_NAME)})
    finally:
        shutil.rmtree(staging, ignore_errors=True)
    log(f"Готово за {time.time() - started:.1f} с: документов {meta['num_documents']}, "
        f"термов {meta['vocabulary_size']}, средняя длина {meta['avg_doc_length']:.1f}")
    return manifest


# ------------------------------------------------------------------------ corpus

def _parse_corpus(path: str, text: str) -> Iterator[dict]:
    for lineno, line in enumerate(text.splitlines(), start=1):
        if not line.strip():
            continue

        def fail(reason: str) -> ValueError:
            return ValueError(f"{path}:{lineno}: {reason}")

        try:
            doc = json.loads(line)
        except ValueError as exc:
            raise fail(f"некорректный JSON ({exc})") from None
        if not isinstance(doc, dict):
            raise fail("ожидается JSON-объект")
        doc_id = doc.get("doc_id")
        if isinstance(doc_id, bool) or not isinstance(doc_id, int) or doc_id <= 0:
            raise fail("doc_id должен быть целым числом > 0")
        if not isinstance(doc.get("text"), str):
            raise fail("поле text обязательно и должно быть строкой")
        title = doc.get("title", "")
        if not isinstance(title, str):
            raise fail("поле title должно быть строкой")
        tags = doc.get("tags", [])
        if not isinstance(tags, list) or not all(isinstance(t, str) for t in tags):
            raise fail("поле tags должно быть списком строк")
        yield {"doc_id": doc_id, "title": title, "text": doc["text"], "tags": tags}


def run_corpus(cfg: Config, log: Log = print, timestamp: Optional[int] = None) -> dict:
    """Индекс малого учебного корпуса из JSONL за секунды (режим benchmark в рантайме)."""
    started = time.time()
    with open(cfg.corpus_path, "rb") as f:
        raw = f.read()
    docs = list(_parse_corpus(cfg.corpus_path, raw.decode("utf-8")))
    log(f"Корпус {os.path.basename(cfg.corpus_path)}: документов {len(docs)}")

    os.makedirs(cfg.index_output_path, exist_ok=True)
    staging = _fresh_dir(os.path.join(cfg.index_output_path, STAGING_DIRNAME))
    try:
        write_graph_db(os.path.join(staging, "graph.db"),
                       (node_from_corpus_doc(d["doc_id"], d["title"], d["text"], d["tags"]) for d in docs), [])
        meta = build_index(
            ((d["doc_id"], tokenize(f"{d['title']} {d['text']}")) for d in docs),
            cfg.index_output_path, index_mode="corpus", block_postings=cfg.spimi_block_postings,
            timestamp=timestamp,
            source={"corpus_file": os.path.basename(cfg.corpus_path),
                    "corpus_sha256": hashlib.sha256(raw).hexdigest()},
            extra_files={"graph.db": os.path.join(staging, "graph.db")})
    finally:
        shutil.rmtree(staging, ignore_errors=True)
    log(f"Готово за {time.time() - started:.2f} с: термов {meta['vocabulary_size']}")
    return meta
