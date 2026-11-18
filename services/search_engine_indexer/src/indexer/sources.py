"""Источники сырых XML: архив 7z (потоком, без распаковки на диск) или готовые файлы.

docs/Tech/03-data.md, п. 4.2. Только стандартная библиотека (ADR-008).
"""
import os
import subprocess
import tempfile
from contextlib import contextmanager
from typing import Dict, Iterator, Optional

from .config import Config

_PIPE_BUFFER = 1 << 20


@contextmanager
def open_member(cfg: Config, member: str) -> Iterator:
    """Контекст, отдающий двоичный поток содержимого ``member`` (``Posts.xml`` и т.п.).

    Если задан ``SO_DUMP_ARCHIVE``, поток — stdout ``7zz x -so``; иначе файл из
    ``RAW_DATA_PATH``. Ненулевой код возврата 7-Zip после чтения даёт ``RuntimeError``
    (индекс при этом не публикуется — см. ``build_index``).
    """
    if not cfg.so_dump_archive:
        with open(os.path.join(cfg.raw_data_path, member), "rb") as f:
            yield f
        return

    if not os.path.exists(cfg.so_dump_archive):
        raise FileNotFoundError(f"архив не найден: {cfg.so_dump_archive}")
    cmd = [cfg.sevenzip_bin, "x", "-so", cfg.so_dump_archive, member]
    with tempfile.TemporaryFile() as err:
        try:
            proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=err, bufsize=_PIPE_BUFFER)
        except FileNotFoundError:
            raise FileNotFoundError(f"7-Zip не найден: {cfg.sevenzip_bin}") from None
        completed = False
        try:
            yield proc.stdout
            completed = True
        finally:
            if completed:
                proc.stdout.read()  # дочитать остаток (потребитель обычно читает всё)
            else:
                proc.kill()
            proc.stdout.close()
            returncode = proc.wait()
        if returncode != 0:
            err.seek(0)
            message = err.read().decode("utf-8", errors="replace").strip()[-500:]
            raise RuntimeError(f"7-Zip завершился с кодом {returncode}: {message}")


def sevenzip_version(binary: str) -> Optional[str]:
    """Первая строка баннера 7-Zip (например, ``7-Zip (z) 26.06 ...``) или ``None``."""
    try:
        result = subprocess.run([binary], capture_output=True, timeout=15)
    except (OSError, subprocess.SubprocessError):
        return None
    for line in result.stdout.decode("utf-8", errors="replace").splitlines():
        if line.strip():
            return line.strip()
    return None


def archive_info(cfg: Config) -> Optional[Dict[str, object]]:
    """Сведения об архиве для манифеста выборки (``None`` для режима готовых файлов)."""
    if not cfg.so_dump_archive:
        return None
    return {
        "name": os.path.basename(cfg.so_dump_archive),
        "bytes": os.path.getsize(cfg.so_dump_archive),
        "sevenzip_version": sevenzip_version(cfg.sevenzip_bin),
    }
