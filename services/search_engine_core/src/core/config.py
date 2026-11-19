"""Configuration for search_engine_core."""
from dataclasses import dataclass
import os

MIN_BM25_K1 = 1.2
MAX_BM25_K1 = 2.0


@dataclass(frozen=True)
class CoreConfig:
    index_dir: str = "/app/data/index"
    bm25_k1: float = 1.5
    bm25_b: float = 0.75
    grpc_port: int = 50051

    def __post_init__(self) -> None:
        if not (MIN_BM25_K1 <= self.bm25_k1 <= MAX_BM25_K1):
            raise ValueError(
                f"BM25_K1 must be within [{MIN_BM25_K1}, {MAX_BM25_K1}], got {self.bm25_k1}"
            )
        if self.bm25_b < 0:
            raise ValueError(f"BM25_B must be non-negative, got {self.bm25_b}")
        if self.grpc_port <= 0 or self.grpc_port > 65535:
            raise ValueError(f"GRPC_PORT must be between 1 and 65535, got {self.grpc_port}")

    @classmethod
    def from_env(cls) -> "CoreConfig":
        index_dir = os.environ.get("INDEX_DIR", "/app/data/index")
        k1_raw = os.environ.get("BM25_K1", "1.5")
        b_raw = os.environ.get("BM25_B", "0.75")
        port_raw = os.environ.get("GRPC_PORT", "50051")

        try:
            bm25_k1 = float(k1_raw)
        except ValueError as err:
            raise ValueError(f"Invalid BM25_K1 in environment: {k1_raw!r}") from err

        try:
            bm25_b = float(b_raw)
        except ValueError as err:
            raise ValueError(f"Invalid BM25_B in environment: {b_raw!r}") from err

        try:
            grpc_port = int(port_raw)
        except ValueError as err:
            raise ValueError(f"Invalid GRPC_PORT in environment: {port_raw!r}") from err

        return cls(
            index_dir=index_dir,
            bm25_k1=bm25_k1,
            bm25_b=bm25_b,
            grpc_port=grpc_port,
        )
