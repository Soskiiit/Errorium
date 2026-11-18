# Search Engine Indexer

> **Роль 1: Data & Indexing Engineer**  
> Офлайн-пайплайн потокового парсинга дампа Stack Overflow, токенизации и построения поисковых индексов.

---

## 1. Назначение и зона ответственности

Сервис отвечает за автономную офлайн-обработку исходных дампов Stack Overflow (~100 ГБ) и формирование артефактов, необходимых для работы поискового ядра `search_engine_core`.

### Основные задачи:
1. **Потоковый парсинг архива XML:**
   - Чтение `Posts.xml`, `Tags.xml`, `PostLinks.xml` из каталога `data/raw/` без загрузки всего файла в RAM (лимит памяти $\le 4$ ГБ, SAX/итераторы).
   - Выделение вопросов (`PostTypeId=1`) и ответов (`PostTypeId=2`), привязка принятого/лучшего ответа по `AcceptedAnswerId` или максимальному `Score`.
2. **Специализированная токенизация кода и ошибок:**
   - Сохранение неделимыми идентификаторов исключений (`NullPointerException`, `IndexOutOfBoundsException`).
   - Сохранение языковых конструкций и библиотек (`C++`, `C#`, `.NET`, `fmt.Println`, `os.path.join`).
   - Сохранение имен утилит и пакетов (`pip3`, `g++`, `docker-compose`).
3. **Построение инвертированного индекса (SPIMI):**
   - Single-Pass In-Memory Indexing: накопление блоков в оперативной памяти, сброс на диск и последующее многопутевое слияние (Multi-way merge).
   - Подсчет общего числа документов $N$, частот терминов $TF(t, d)$, документных частот $DF(t)$ и средней длины $avgdl$.
4. **Сжатие и запись артефактов в `data/index/`:**
   - Бинарные структуры постингов и словаря с дельта-кодированием (VByte/Varint).
   - База данных графа связей и метаданных тредов `graph.db` (SQLite).

---

## 1.1. Стек

Python 3 + стандартная библиотека (ADR-008 в `docs/Service/decisions.md`). Правила токенизации и эталонные примеры — `docs/Tech/03-data.md`, раздел 5; код и тесты: `src/` и `tests/unit/test_tokenizer.py`, `tests/unit/test_sampler.py`.

---

## 2. Входные и выходные данные

### Входные данные:
Основной источник — архив `stackoverflow.com.7z` (64 ГиБ, несолидный), который читается **потоком** (`7zz x -so`), без распаковки на диск. Альтернатива — уже распакованные файлы в `data/raw/`.
- `Posts.xml` — все посты (105.6 ГБ; вопросы и ответы перемешаны, порядок не по `Id`).
- `Tags.xml` — словарь тегов.
- `PostLinks.xml` — связи между постами (дубликаты `LinkTypeId=3`, связанные темы `LinkTypeId=1`).

Корпус формируется **хеш-выборкой тредов** (`INDEX_MODE=sample`, КТ2) либо по шардам (`INDEX_MODE=full`, КТ3). Префиксный срез «первые N ГБ» не используется: в нём почти нет принятых ответов. Полная спецификация — `docs/Tech/03-data.md`, раздел 4; обоснование — ADR-007 в `docs/Service/decisions.md`.

### Выходные артефакты (`data/index/`):
- `lexicon.bin` — бинарный словарь уникальных терминов с оффсетами и $DF(t)$.
- `postings.bin` — сжатые списки постингов $\langle doc\_id, tf \rangle$.
- `doc_lengths.bin` — длины документов в токенах.
- `metadata.json` — параметры индекса ($N$, $avgdl$, timestamp).
- `graph.db` — SQLite база данных связей графа (`thread_links`) и метаданных узлов (`thread_nodes`).

---

## 3. Конфигурация и переменные окружения

```bash
RAW_DATA_PATH=/app/data/raw          # Путь к сырым XML-дампам (и к sample/)
INDEX_OUTPUT_PATH=/app/data/index    # Путь для сохранения готовых индексов
SO_DUMP_ARCHIVE=/dump/stackoverflow.com.7z   # Архив дампа (монтируется :ro), читается потоком
INDEX_MODE=sample                    # sample (КТ2) | full (КТ3) | corpus (benchmark, секунды)
CORPUS_PATH=benchmark/corpus/documents.jsonl   # только для INDEX_MODE=corpus
SPIMI_BLOCK_POSTINGS=2000000         # размер блока SPIMI в постингах
SAMPLE_DENOM=100                     # доля выборки 1/DENOM тредов
SAMPLE_RESIDUE=0
SAMPLE_MIN_SCORE=1
GRAPH_MAX_NEIGHBORS=20
SHARD_COUNT=64                       # только для INDEX_MODE=full
```

Описание переменных и критериев приёмки — `docs/Tech/03-data.md`, п. 4.7–4.8.

---

## 4. Академические ограничения (ИУ5)

- **СТРОГО ЗАПРЕЩЕНО:** использовать готовые движки и библиотеки (`Elasticsearch`, `Lucene`, `rank-bm25`, `Whoosh`).
- Алгоритм инвертированного индекса и постингов пишется **полностью вручную**.
