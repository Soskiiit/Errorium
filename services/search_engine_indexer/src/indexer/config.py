"""Конфигурация индексатора из переменных окружения (docs/Tech/03-data.md, п. 4.7)."""
import os
from dataclasses import dataclass
from typing import Mapping, Optional


def _int(env: Mapping[str, str], name: str, default: int) -> int:
    raw = env.get(name)
    if raw is None or raw == "":
        return default
    try:
        return int(raw)
    except ValueError:
        raise ValueError(f"{name}: ожидается целое число, получено {raw!r}") from None


@dataclass(frozen=True)
class Config:
    raw_data_path: str
    index_output_path: str
    so_dump_archive: Optional[str]
    corpus_path: str
    sample_denom: int
    sample_residue: int
    sample_min_score: int
    graph_max_neighbors: int
    spimi_block_postings: int
    sevenzip_bin: str
    shard_count: int

    @classmethod
    def from_env(cls, env: Optional[Mapping[str, str]] = None) -> "Config":
        env = os.environ if env is None else env
        cfg = cls(
            raw_data_path=env.get("RAW_DATA_PATH") or "/app/data/raw",
            index_output_path=env.get("INDEX_OUTPUT_PATH") or "/app/data/index",
            so_dump_archive=env.get("SO_DUMP_ARCHIVE") or None,
            corpus_path=env.get("CORPUS_PATH") or "benchmark/corpus/documents.jsonl",
            sample_denom=_int(env, "SAMPLE_DENOM", 100),
            sample_residue=_int(env, "SAMPLE_RESIDUE", 0),
            sample_min_score=_int(env, "SAMPLE_MIN_SCORE", 1),
            graph_max_neighbors=_int(env, "GRAPH_MAX_NEIGHBORS", 20),
            spimi_block_postings=_int(env, "SPIMI_BLOCK_POSTINGS", 2_000_000),
            sevenzip_bin=env.get("SEVENZIP_BIN") or "7zz",
            shard_count=_int(env, "SHARD_COUNT", 64),
        )
        cfg.validate()
        return cfg

    def validate(self) -> None:
        if self.sample_denom < 1:
            raise ValueError("SAMPLE_DENOM должен быть >= 1")
        if not 0 <= self.sample_residue < self.sample_denom:
            raise ValueError("SAMPLE_RESIDUE должен лежать в диапазоне [0, SAMPLE_DENOM)")
        if self.graph_max_neighbors < 0:
            raise ValueError("GRAPH_MAX_NEIGHBORS должен быть >= 0")
        if self.spimi_block_postings < 1:
            raise ValueError("SPIMI_BLOCK_POSTINGS должен быть >= 1")
        if self.shard_count < 1:
            raise ValueError("SHARD_COUNT должен быть >= 1")
