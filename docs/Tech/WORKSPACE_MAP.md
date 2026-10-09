# WORKSPACE_MAP.md — Карта кодовой базы и сервисов

## 1. Структура директорий

```text
.
├── Makefile                          # Точка входа для сборки, тестов, protoc и запуска
├── docker-compose.yml                # Описание мультиконтейнерного стека
├── AGENTS.md                         # Инструкции и ограничения для AI-ассистентов
├── STATUS.md                         # Текущий статус готовности и ближайший шаг
├── CHANGE_HISTORY.md                 # Журнал изменений архитектуры и реализации
│
├── protos/                           # Межсервисные gRPC спецификации
│   ├── search.proto                  # Контракт поискового ядра и ранжирования
│   └── graph.proto                   # Контракт извлечения графа знаний и сущностей
│
├── docs/                             # Полный комплект документации SDD
│   ├── README.md                     # Карта разделов документации
│   ├── document-manifest.json        # Реестр документов и версий
│   ├── Product/                      # Продуктовая спецификация
│   ├── Tech/                         # Техническая спецификация
│   └── Service/                      # Журнал решений и источники требований
│
├── services/                         # Микросервисы системы
│   ├── search_engine_indexer/        # [Роль 1] Потоковый парсинг 100 ГБ, токенизатор, инвертированный индекс
│   │   ├── Dockerfile
│   │   ├── README.md
│   │   └── src/
│   │
│   ├── search_engine_core/           # [Роль 2] BM25 с нуля, MMap индекс, gRPC-сервер (:50051)
│   │   ├── Dockerfile
│   │   ├── README.md
│   │   └── src/
│   │
│   ├── gateway/                      # [Роль 3] REST API, SSO, MCP Server, RAG
│   │   ├── Dockerfile
│   │   ├── README.md
│   │   └── src/
│   │
│   └── frontend/                     # [Роль 4] Web UI поиска, граф знаний, админ-панель
│       ├── Dockerfile
│       ├── README.md
│       └── src/
│
├── data/                             # Хранилище сырых дампов и индексов (в .gitignore)
│   ├── raw/                          # Posts.xml, Tags.xml, PostLinks.xml
│   └── index/                        # postings.bin, lexicon.bin, doc_lengths.bin, graph.db
│
├── benchmark/                        # Метрики качества ранжирования (Recall@k, MRR)
│   └── evaluate.py
│
└── tests/                            # Автотесты
    ├── unit/                         # Тесты токенизатора, BM25, Auth, графа тредов
    └── integration/                  # Тесты gRPC, REST API, MCP
```

---

## 2. Распределение ролей и зон ответственности

| Модуль | Роль | Основная зона ответственности |
| :--- | :--- | :--- |
| **`services/search_engine_indexer/`** | Роль 1 | Парсер Stack Overflow, токенизатор кода, построение инвертированного индекса, построение связей тредов. |
| **`services/search_engine_core/`** | Роль 2 | Реализация формулы BM25, хранение posting lists в mmap, Top-K выборка, реализация gRPC сервисов `SearchService` и `GraphService`. |
| **`services/gateway/`** | Роль 3 | REST API (:8000), SSO авторизация (OAuth2/OIDC), JWT, Remote MCP Server (:8000/sse), RAG-оркестрация с LLM. |
| **`services/frontend/`** | Роль 4 | Пользовательский Web UI (:3000), визуализация графа связанных тредов, интерфейс админ-панели. |
