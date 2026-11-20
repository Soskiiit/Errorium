# Search Engine Core

> **Роль 2: Search Core & Performance Engineer**  
> Высокопроизводительное поисковое ядро: реализация алгоритма BM25 с нуля, MMap-чтение индекса, gRPC-сервер ядра (`:50051`).

**Статус: реализован и верифицирован** (2026-11-20). Спецификация чтения и ранжирования — `docs/Tech/03-data.md`, раздел 6; решение о стеке — ADR-011. Бенчмарк на золотом корпусе: Recall@3 = 0.933, Recall@5 = 0.933, Recall@10 = 0.967, MRR@10 = 1.000, P95 = 0.25 мс (пороги: ≥ 0.75, ≥ 0.50, ≤ 150 мс).

## 0. Быстрый запуск

```bash
make index-corpus    # индекс учебного корпуса (секунды) → data/index-benchmark/
make core-up         # gRPC-сервер :50051 (docker-compose.core.yml)
make test-core       # unit + gRPC интеграционные тесты в контейнере
make benchmark       # Recall@k / MRR через ядро, отчёт data/benchmark_report.json
make core-down
```

Для индекса выборки SO: `make index-sample` (нужен дамп, см. `docs/Tech/02-running.md`). Без опубликованного индекса сервер стартует как `NOT_SERVING` и подхватывает индекс при появлении (lazy reload).

## 0.1. Состав кода (`src/core/`)

| Модуль | Назначение |
| :--- | :--- |
| `reader.py` | mmap-чтение `postings.bin`, словарь/длины в памяти, валидация `metadata.json` (версии формата и токенизатора) |
| `bm25.py` | Формула ИУ5: IDF, вклад терма, накопление баллов, Top-K min-heap (`heapq`) |
| `engine.py` | Токенизация запроса общим токенизатором → BM25 → фильтры tags/language → Top-K |
| `graph.py` | `graph.db` read-only: узлы, 1-hop окрестность, детали треда |
| `server.py` | gRPC `SearchService` + `GraphService`, коды `FAILED_PRECONDITION`/`NOT_FOUND` |
| `config.py` | Env-конфигурация с валидацией диапазонов k1/b |

Общие модули индексатора (`tokenizer.py`, `index_format.py`) вендорятся в образ при сборке; pb2 генерируются из `protos/` (`grpcio-tools`), в git не коммитятся. Единственные внешние зависимости — `grpcio`/`grpcio-tools` (ADR-011); весь поиск — собственный код на stdlib.

---

## 1. Назначение и зона ответственности

Сервис предоставляет сетевой gRPC-интерфейс для ранжированного лексического поиска по заранее построенному инвертированному индексу и извлечения графа связанных тредов.

### Основные задачи:
1. **Быстрое чтение индекса (`mmap`):**
   - Отображение файлов `lexicon.bin`, `postings.bin`, `doc_lengths.bin` в виртуальную память без накладных расходов на полную загрузку в RAM.
   - Микросекундный доступ к спискам постингов по оффсетам из словаря.
2. **Расчет BM25 строго по формуле ИУ5:**
   $$IDF(t) = \ln\left(1 + \frac{N - DF(t) + 0.5}{DF(t) + 0.5}\right)$$
   $$BM25(q, d) = \sum_{t \in q} IDF(t) \cdot \frac{TF(t, d) \cdot (k_1 + 1)}{TF(t, d) + k_1 \cdot \left(1 - b + b \cdot \frac{|d|}{avgdl}\right)}$$
   где $k_1 = 1.5, b = 0.75$.
3. **Top-K отбор кандидатов:**
   - Сбор документов, содержащих термины запроса, и выборка лучших $K$ кандидатов с использованием структуры данных Min-Heap (приоритетная очередь).
4. **Реализация gRPC-сервера (`:50051`):**
   - `SearchService` (описан в `protos/search.proto`):
     - `Search(SearchRequest) -> SearchResponse` (Top-K ранжирование, расчет латентности).
     - `GetDocument(GetDocumentRequest) -> GetDocumentResponse`.
     - `BatchGetDocuments(BatchGetDocumentsRequest) -> BatchGetDocumentsResponse`.
     - `GetStats(GetStatsRequest) -> GetStatsResponse` ($N$, $avgdl$, объем индекса).
     - `HealthCheck(HealthCheckRequest) -> HealthCheckResponse`.
   - `GraphService` (описан в `protos/graph.proto`):
     - `GetThreadGraph(ThreadGraphRequest) -> ThreadGraphResponse` (выборка окрестности дубликатов и связанных тредов из `graph.db`).
     - `GetThreadDetails(ThreadDetailsRequest) -> ThreadDetailsResponse` (полный текст ответа по клику).
5. **Оценка качества ранжирования (Бенчмарк):**
   - Реализация и прогон `benchmark/evaluate.py`:
     - $Recall@k \ge 0.75$ (при $k=10$).
     - $MRR \ge 0.50$.
     - Задержка ответа $P_{95} \le 150$ мс.

---

## 2. Конфигурация и переменные окружения

```bash
SEARCH_GRPC_PORT=50051         # Порт gRPC-сервера ядра
INDEX_DIR=/app/data/index    # Директория с готовыми файлами индекса (монтируется :ro)
BM25_K1=1.5                  # Параметр насыщения TF, допустимо [1.2, 2.0]
BM25_B=0.75                  # Поправка на длину, допустимо [0, 1]
SEARCH_MAX_TOP_K=100           # Верхняя граница top_k запроса
```

---

## 3. Академические ограничения (ИУ5)

- **СТРОГО ЗАПРЕЩЕНО:** использовать `rank-bm25`, `bm25s`, `tantivy`, `scikit-learn`, `Elasticsearch` и т.д.
- Весь расчет ранжирования и поиск по постингам реализуется **собственным кодом**.
