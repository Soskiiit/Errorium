"""Токенизатор кода и сообщений об ошибках (docs/Tech/03-data.md, раздел 5).

Один и тот же токенизатор используется для документов и для запросов
(методичка, замечание 1.1). Только стандартная библиотека (ADR-008).

Ключевые свойства:
  * идентификаторы (``NullPointerException``), символы языков (``C++``, ``C#``,
    ``.NET``), пути и утилиты (``os.path.join``, ``pip3``, ``docker-compose``)
    попадают в индекс целиком, в нижнем регистре;
  * подтокены (``null``, ``pointer``, ``exception``) добавляются *дополнительно*;
  * нет стемминга, лемматизации и списка стоп-слов.
"""
import html
import re
from typing import List

TOKENIZER_VERSION = "1"

MAX_TOKEN_LEN = 64
MIN_TOKEN_LEN = 2

_URL = re.compile(r"https?://\S+", re.IGNORECASE)
_TAG = re.compile(r"<[^>]*>")

# Базовый токен (п. 5.2):
#   * необязательная ведущая точка перед буквой в начале слова (``.net``);
#   * ядро: символы слова, соединители «.» и «-» только между буквенно-цифровыми;
#   * необязательный суффикс «++» / «#», если за ним не идёт буква или цифра.
_TOKEN = re.compile(
    r"""
    (?:(?<!\w)\.(?=[^\W\d_]))?
    \w+(?:[.\-]\w+)*
    (?:\+\+(?!\w)|\#(?!\w))?
    """,
    re.VERBOSE,
)

_CONNECTORS = re.compile(r"[._\-]+")


def clean_html(markup: str) -> str:
    """HTML тела поста -> обычный текст.

    Теги заменяются пробелом (чтобы соседние абзацы не склеивались), содержимое
    ``<code>`` и ``<pre>`` сохраняется; затем раскрываются HTML-сущности
    (порядок важен: ``&lt;int&gt;`` не должно стать тегом).
    """
    return html.unescape(_TAG.sub(" ", markup))


def _split_camel(segment: str) -> List[str]:
    """Делит сегмент по границам ``нижний→ВЕРХНИЙ`` и концу акронима (``HTTPServer``).

    На цифрах сегмент не делится (``pip3``, ``utf8``); подходит для Unicode.
    """
    if not segment:
        return []
    parts: List[str] = []
    start = 0
    n = len(segment)
    for i in range(1, n):
        prev, cur = segment[i - 1], segment[i]
        boundary = False
        if prev.islower() and cur.isupper():
            boundary = True
        elif prev.isupper() and cur.isupper() and i + 1 < n and segment[i + 1].islower():
            boundary = True
        if boundary:
            parts.append(segment[start:i])
            start = i
    parts.append(segment[start:])
    return parts


def _subtokens(raw: str) -> List[str]:
    """Части составного токена в исходном регистре (до casefold)."""
    core = raw[1:] if raw.startswith(".") else raw
    if core.endswith("++"):
        core = core[:-2]
    elif core.endswith("#"):
        core = core[:-1]
    parts: List[str] = []
    for segment in _CONNECTORS.split(core):
        parts.extend(_split_camel(segment))
    return parts


def _keep(token: str) -> bool:
    return MIN_TOKEN_LEN <= len(token) <= MAX_TOKEN_LEN and any(ch.isalnum() for ch in token)


def tokenize(text: str) -> List[str]:
    """Текст -> список токенов в порядке эмиссии (с повторами, для TF).

    После каждого составного токена сразу идут его подтокены.
    """
    text = _URL.sub(" ", text)
    out: List[str] = []
    for match in _TOKEN.finditer(text):
        raw = match.group(0)
        whole = raw.casefold()
        if _keep(whole):
            out.append(whole)
        for part in _subtokens(raw):
            folded = part.casefold()
            if folded != whole and _keep(folded):
                out.append(folded)
    return out
