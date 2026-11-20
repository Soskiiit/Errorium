"""Точка входа: python -m core (docs/Tech/02-running.md)."""
import logging

from .config import Config
from .server import serve


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")
    serve(Config.from_env())


if __name__ == "__main__":
    main()