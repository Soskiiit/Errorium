# CHANGE_HISTORY.md — Журнал изменений (SDD)

### [2026-09-27] — Инициализация структуры сервисов под 4 роли

- **Затронутые файлы/спецификации:**
  - `AGENTS.md`
  - `README.md`
  - `.gitignore`
  - `services/search_engine_indexer/`
  - `services/search_engine_core/`
  - `services/gateway/`
  - `services/frontend/`
  - `data/`
  - `benchmark/`

- **Суть изменений:**
  - Разделена архитектура поискового ядра на два отдельных сервиса: офлайн-индексатор (`search_engine_indexer`, Роль 1) и онлайн поисковое ядро (`search_engine_core`, Роль 2).
  - Структурировано 4 микросервиса под каждого участника команды:
    1. `services/search_engine_indexer/` — Потоковый парсинг 100GB дампа, токенизатор, инвертированный индекс.
    2. `services/search_engine_core/` — Реализация BM25 с нуля, Top-K отбор, gRPC-сервер ядра (`:50051`).
    3. `services/gateway/` — FastAPI REST API, RAG-оркестрация (LLM), Remote MCP Server.
    4. `services/frontend/` — Веб-интерфейс пользователя (Роль 4).
  - Внутри каждого сервиса созданы каркасные `Dockerfile` и `README.md`.
  - Созданы директории `data/` (добавлена в `.gitignore` с сохранением `.gitkeep`) и `benchmark/`.
  - Обновлены инструкции и схемы потоков данных в `AGENTS.md` и `README.md`.
