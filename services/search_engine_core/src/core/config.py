"""Конфигурация ядра из переменных окружения (docs/Tech/03-data.md, п. 6.6)."""
import os
from dataclasses import dataclass
from typing import Mapping, Optional


@dataclass(frozen=True)
class Config:
    index_dir: str = "/app/data/index"
    port: int = 50051
    k1: float = 1.5
    b: float = 0.75
    max_top_k: int = 100

    @classmethod
    def from_env(cls, env: Optional[Mapping[str, str]] = None) -> "Config":
        env = os.environ if env is None else env
        k1 = float(env.get("BM25_K1", "1.5"))
        b = float(env.get("BM25_B", "0.75"))
        port = int(env.get("SEARCH_GRPC_PORT", "50051"))
        max_top_k = int(env.get("SEARCH_MAX_TOP_K", "100"))
        if not 1.2 <= k1 <= 2.0:
            raise ValueError(f"BM25_K1={k1} вне допустимого диапазона [1.2, 2.0] (методичка ИУ5)")
        if not 0.0 <= b <= 1.0:
            raise ValueError(f"BM25_B={b} вне допустимого диапазона [0, 1]")
        if port <= 0:
            raise ValueError(f"SEARCH_GRPC_PORT={port} должен быть > 0")
        if max_top_k < 1:
            raise ValueError(f"SEARCH_MAX_TOP_K={max_top_k} должен быть >= 1")
        return cls(index_dir=env.get("INDEX_DIR", "/app/data/index"),
                   port=port, k1=k1, b=b, max_top_k=max_top_k)