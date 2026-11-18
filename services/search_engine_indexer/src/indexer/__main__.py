"""Точка входа: ``python -m indexer sample|corpus`` (docs/Tech/03-data.md, п. 4.10)."""
import argparse
import os
import sys
from typing import Callable, List, Mapping, Optional

from .config import Config
from .pipeline import run_corpus, run_sample


def _log(*args) -> None:
    print(*args, flush=True)


def main(argv: Optional[List[str]] = None, env: Optional[Mapping[str, str]] = None,
         log: Callable[..., None] = _log) -> int:
    parser = argparse.ArgumentParser(prog="indexer", description="Индексатор Errorium (Роль 1)")
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("sample", help="выборка из дампа Stack Overflow (КТ2)")
    sub.add_parser("corpus", help="индекс малого корпуса из JSONL (benchmark)")
    args = parser.parse_args(argv)
    try:
        cfg = Config.from_env(os.environ if env is None else env)
        if args.command == "sample":
            run_sample(cfg, log=log)
        else:
            run_corpus(cfg, log=log)
    except (ValueError, OSError, RuntimeError) as exc:
        print(f"ошибка: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
