# Search Engine Core

> **Роль 2: Search Core & Performance Engineer**  
> Высокопроизводительное поисковое ядро: реализация алгоритма BM25 с нуля, потокобезопасное чтение индекса, gRPC-сервер ядра (`:50051`).

---

## 1. Назначение и зона ответственности

Сервис предоставляет сетевой gRPC-интерфейс для ранжированного лексического поиска по заранее построенному инвертированному индексу и извлечения графа связанных тредов.

### Основные задачи:
1. **Чтение бинарного индекса v1:**
   - Чтение файлов `lexicon.bin`, `postings.bin`, `doc_lengths.bin` через функции `indexer.index_format`.
   - Проверка `metadata.json` (`format_version == 1`, `tokenizer_version == "1"`).
   - Чтение связей тредов из `graph.db` (SQLite в режиме `mode=ro`).
2. **Расчет BM25 строго по формуле кафедры ИУ5:**
   $$IDF(t) = \ln\left(1 + \frac{N - DF(t) + 0.5}{DF(t) + 0.5}\right)$$
   $$BM25(q, d) = \sum_{t \in q} IDF(t) \cdot \frac{TF(t, d) \cdot (k_1 + 1)}{TF(t, d) + k_1 \cdot \left(1 - b + b \cdot \frac{|d|}{avgdl}\right)}$$
   где $k_1 = 1.5$ (допустимо $[1.2, 2.0]$), $b = 0.75$. При равных баллах сортировка по `doc_id` ASC.
3. **Top-K отбор кандидатов:**
   - Сбор документов, содержащих термины запроса, и выборка $K$ кандидатов (clamping в диапазон $[1, 50]$, по умолчанию 10).
4. **Реализация gRPC-сервера (`:50051`):**
   - `SearchService` (`protos/search.proto`):
     - `Search(SearchRequest) -> SearchResponse` (Top-K ранжирование, замер latency_ms, tokenized_query).
     - `GetDocument(GetDocumentRequest) -> GetDocumentResponse`.
     - `BatchGetDocuments(BatchGetDocumentsRequest) -> BatchGetDocumentsResponse`.
     - `GetStats(GetStatsRequest) -> GetStatsResponse` ($N$, $avgdl$, объем индекса).
     - `ReloadIndex(ReloadIndexRequest) -> ReloadIndexResponse` (потокобезопасная горячая замена индекса).
     - `HealthCheck(HealthCheckRequest) -> HealthCheckResponse` (`SERVING` при загруженном индексе).
   - `GraphService` (`protos/graph.proto`):
     - `GetThreadGraph(ThreadGraphRequest) -> ThreadGraphResponse` (выборка окрестности дубликатов и связанных тредов из `graph.db`, до 20 соседей).
     - `GetThreadDetails(ThreadDetailsRequest) -> ThreadDetailsResponse` (полный текст лучшего ответа).
5. **Оценка качества ранжирования (Бенчмарк НИРС):**
   - Расчет метрик $Recall@k$ ($k=3, 5, 10$) и $MRR@10$ по `benchmark/questions.json`:
     - $Recall@10 \ge 0.75$ (NFR).
     - $MRR@10 \ge 0.50$ (NFR).

---

## 2. Формат индекса v1

Директория индекса (`INDEX_DIR`) содержит:
- `metadata.json` — метаданные индекса (`format_version=1`, `tokenizer_version="1"`, $N$, $avgdl$, `build_timestamp`).
- `lexicon.bin` — бинарный словарь термов (`term`, `df`, `offset`, `length`).
- `postings.bin` — бинарные списки постингов (`doc_id`, `tf`).
- `doc_lengths.bin` — бинарная таблица длин документов (`doc_id`, $|d|$).
- `graph.db` — база данных SQLite с таблицами `thread_nodes` и `thread_links`.

---

## 3. Конфигурация и переменные окружения

```bash
GRPC_PORT=50051              # Порт gRPC-сервера ядра (по умолчанию 50051)
INDEX_DIR=/app/data/index    # Директория с готовыми файлами индекса (монтируется :ro)
BM25_K1=1.5                  # Коэффициент k1 (1.2 .. 2.0)
BM25_B=0.75                  # Коэффициент b (0.75)
```

---

## 4. Локальный запуск и смоук-тест

### Запуск сервера ядра:
```bash
# Предварительно построить тестовый индекс корпуса:
make index-corpus

# Локальный запуск:
PYTHONPATH=services/search_engine_core/src:services/search_engine_indexer/src \
  INDEX_DIR=data/index-benchmark \
  python3 -m core.server
```

### Запуск смоук-теста:
```bash
PYTHONPATH=services/search_engine_core/src:services/search_engine_indexer/src \
  python3 -m core.smoke localhost:50051
```
Смоук проверяет `HealthCheck` (`SERVING`), выполняет тестовый поиск `Search("python IndexError")`, запрашивает системную статистику `GetStats` и строит граф `GetThreadGraph`.

### Запуск в Docker:
```bash
docker compose -f docker-compose.dev.yml up --build search_engine_core
```

---

## 5. Оценка качества ранжирования (Бенчмарк)

```bash
make benchmark
```
Или напрямую:
```bash
PYTHONPATH=services/search_engine_core/src:services/search_engine_indexer/src \
  INDEX_DIR=data/index-benchmark \
  python3 benchmark/evaluate.py
```

### Результаты бенчмарка волны A:
- **Recall@3:** `0.9333`
- **Recall@5:** `0.9333`
- **Recall@10:** `0.9667` (порог NFR $\ge 0.75$ — выполнен)
- **MRR@10:** `1.0000` (порог NFR $\ge 0.50$ — выполнен)

---

## 6. Академические ограничения (ИУ5)

- **СТРОГО ЗАПРЕЩЕНО:** использовать `rank-bm25`, `bm25s`, `tantivy`, `scikit-learn`, `Elasticsearch` и т.д.
- Весь расчет ранжирования и поиск по постингам реализуется **собственным кодом**.
