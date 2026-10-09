# Search Engine Core

> **Роль 2: Search Core & Performance Engineer**  
> Высокопроизводительное поисковое ядро: реализация алгоритма BM25 с нуля, MMap-чтение индекса, gRPC-сервер ядра (`:50051`).

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
```

---

## 3. Академические ограничения (ИУ5)

- **СТРОГО ЗАПРЕЩЕНО:** использовать `rank-bm25`, `bm25s`, `tantivy`, `scikit-learn`, `Elasticsearch` и т.д.
- Весь расчет ранжирования и поиск по постингам реализуется **собственным кодом**.
