# 02-running.md — Запуск и настройка окружения

## 1. Системные требования

- **Операционная система:** Linux (Ubuntu 22.04+, Debian 12+, Fedora, Arch) или macOS.
- **Инструменты:**
  - Docker $\ge 24.0$
  - Docker Compose $\ge 2.20$
  - GNU Make $\ge 4.3$
  - Go $\ge 1.27.1$
  - Python $\ge 3.11$ (рекомендуется 3.13)
  - `protoc` v36.1 (для кодогенерации `make proto`)
    - Go-плагины: `protoc-gen-go@v1.36.6`, `protoc-gen-go-grpc@v1.5.1`
    - Python-пакет: `grpcio-tools==1.76.0`
- **Рекомендуемые аппаратные ресурсы:**
  - CPU: $\ge 4$ ядра
  - RAM: $\ge 8$ ГБ (для работы с демонстрационным индексом), $\ge 16$ ГБ (для полной индексации 100 ГБ дампа)
  - Диск: SSD со свободным местом от 20 ГБ (мини-выборка) до 250 ГБ (полный архив и индекс).
    - Для индексатора в режиме `sample` (КТ2) сырой дамп **не распаковывается**: архив `stackoverflow.com.7z` (≈ 64 ГиБ) читается потоком, дополнительно нужно ≈ 3 ГиБ под `data/raw/sample/` и `data/index/`. Подробности и бюджет режима `full` — `docs/Tech/03-data.md`, раздел 4.

---

## 2. Сетевые порты и сервисы

| Сервис | Порт хоста (по умолчанию) | Переменная окружения | Внутренний протокол | Назначение |
| :--- | :--- | :--- | :--- | :--- |
| `frontend` | `3000` | `FRONTEND_PORT` | HTTP | Веб-интерфейс пользователя и админ-панель |
| `gateway` | `8000` | `GATEWAY_PORT` | HTTP / SSE | REST API, Swagger (`/docs`), SSO Callbacks, Remote MCP |
| `postgres` | `5432` | `POSTGRES_PORT` | PostgreSQL TCP | СУБД Gateway (доступ с хоста для DBeaver / psql) |
| `garage` (S3 API) | `3900` | `GARAGE_PORT` | S3 API (HTTP) | Хранилище объектов / аватарок (S3 клиент с хоста) |
| `garage` (Admin) | `3903` | `GARAGE_ADMIN_PORT` | HTTP | Garage Admin & Metrics API |
| `search_engine_core` | `50051` | `SEARCH_GRPC_PORT` | gRPC (HTTP/2) | Поисковое ядро и построение графа связей |

> [!TIP]
> **Конфликт портов на хост-машине:**  
> Если порт (например, `3900` или `5432`) уже занят другим процессом на рабочей машине, переопределите его в файле `.env` (`GARAGE_PORT=3910`, `POSTGRES_PORT=5433`). Внутри Docker-сети сервисы продолжат общаться по внутренним портам по умолчанию, конфликты на хосте будут устранены.

---

## 3. Переменные окружения (`.env`)

Создайте файл `.env` в корне репозитория (или используйте `.env.example`):

```bash
# === Общие настройки системы ===
ENVIRONMENT=development
LOG_LEVEL=INFO

# === Search Engine Indexer (Роль 1) ===
RAW_DATA_PATH=/app/data/raw
INDEX_OUTPUT_PATH=/app/data/index
SO_DUMP_ARCHIVE=/dump/stackoverflow.com.7z   # читается потоком, не распаковывается на диск
SAMPLE_DENOM=100
SAMPLE_MIN_SCORE=1

# === Search Engine Core (Роль 2) ===
SEARCH_GRPC_PORT=50051
CORE_INDEX_DIR=/app/data/index              # В dev для корпуса benchmark: /app/data/index-benchmark
BM25_K1=1.5                                 # Параметр k1 формулы BM25 [1.2, 2.0]
BM25_B=0.75                                 # Параметр b формулы BM25

# === Gateway & MCP Service (Роль 3) ===
GATEWAY_PORT=8000
SEARCH_ENGINE_ADDR=search_engine_core:50051 # gRPC-адрес поискового ядра (пусто -> fallback на стабы)
SEARCH_ENGINE_HOST=search_engine_core
SEARCH_ENGINE_PORT=50051

# === Web Frontend (Роль 4) ===
FRONTEND_PORT=3000
GATEWAY_API_URL=http://gateway:8000

# === База данных Gateway (PostgreSQL 18+) ===
POSTGRES_PORT=5432
DB_HOST=postgres
DB_PORT=5432
DB_NAME=errorium
DB_USER=errorium_user
DB_PASSWORD=errorium_password
DB_SSLMODE=disable

# === Объектное хранилище аватарок (Garage / S3) ===
GARAGE_PORT=3900
GARAGE_ADMIN_PORT=3903
GARAGE_ENDPOINT=garage:3900
GARAGE_DEFAULT_ACCESS_KEY=garageadmin
GARAGE_DEFAULT_SECRET_KEY=garageadminpassword
GARAGE_DEFAULT_BUCKET=errorium-avatars
GARAGE_REGION=garage
GARAGE_USE_SSL=false

# === Аутентификация, JWT и SSO ===
JWT_SECRET_KEY=change-this-in-production-to-strong-secret-key-64chars-min
JWT_ALGORITHM=HS256
ACCESS_TOKEN_EXPIRE_MINUTES=60
REFRESH_TOKEN_EXPIRE_DAYS=30

# OAuth2 / SSO провайдеры (GitHub / Google / OIDC)
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret

# Стартовая учетная запись администратора
FIRST_ADMIN_EMAIL=admin@errorium.ru
FIRST_ADMIN_PASSWORD=AdminSecurePassword2026!

# === RAG-пайплайн и LLM Провайдер ===
LLM_PROVIDER=openai # openai | anthropic | local
OPENAI_API_KEY=sk-...
OPENAI_MODEL_NAME=gpt-4o-mini
LLM_TEMPERATURE=0.2
LLM_MAX_TOKENS=1500
```

---

## 4. Команды запуска и сборки

Все операции производятся через `Makefile`:

```bash
# 1. Компиляция Protobuf контрактов (Go и Python):
make proto

# 2. Построение индекса малого корпуса benchmark (30 документов):
make index-corpus

# 3. Выборка из дампа Stack Overflow (требуется SO_DUMP_ARCHIVE):
make index-sample

# 4. Бенчмарк качества поиска (Recall@k, MRR@10 на корпусе benchmark):
make benchmark

# 5. Локальный запуск дев-сервера фронтенда (порт :3000):
make dev

# 6. Локальный запуск полного dev-стека в Docker (Core :50051, Gateway :8000, Frontend :3000, Postgres, Garage):
make dev-docker

# 7. Остановка локального дев-стека в Docker:
make dev-down

# 8. Продакшн запуск с SSL-сертификатами (порты 80/443):
make up

# 9. Остановка продакшн-контейнеров:
make down

# 10. Запуск модульных тестов:
make test-unit

# 11. Запуск интеграционных тестов:
make test-integration

# 12. Полный тестовый набор:
make test
```

