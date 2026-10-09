# 02-running.md — Запуск и настройка окружения

## 1. Системные требования

- **Операционная система:** Linux (Ubuntu 22.04+, Debian 12+, Fedora, Arch) или macOS.
- **Инструменты:**
  - Docker $\ge 24.0$
  - Docker Compose $\ge 2.20$
  - GNU Make $\ge 4.3$
- **Рекомендуемые аппаратные ресурсы:**
  - CPU: $\ge 4$ ядра
  - RAM: $\ge 8$ ГБ (для работы с демонстрационным индексом), $\ge 16$ ГБ (для полной индексации 100 ГБ дампа)
  - Диск: SSD со свободным местом от 20 ГБ (мини-выборка) до 250 ГБ (полный архив и индекс).

---

## 2. Сетевые порты и сервисы

| Сервис | Порт хоста | Внутренний протокол | Назначение |
| :--- | :--- | :--- | :--- |
| `frontend` | `3000` | HTTP | Веб-интерфейс пользователя и админ-панель |
| `gateway` | `8000` | HTTP / SSE | REST API, Swagger (`/docs`), SSO Callbacks, Remote MCP |
| `search_engine_core` | `50051` | gRPC (HTTP/2) | Поисковое ядро и построение графа связей |

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

# === Search Engine Core (Роль 2) ===
SEARCH_GRPC_PORT=50051
INDEX_DIR=/app/data/index

# === Gateway & MCP Service (Роль 3) ===
GATEWAY_PORT=8000
SEARCH_ENGINE_HOST=search_engine_core
SEARCH_ENGINE_PORT=50051

# === Web Frontend (Роль 4) ===
FRONTEND_PORT=3000
GATEWAY_API_URL=http://gateway:8000

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

# 1. Локальный запуск дев-сервера фронтенда (порт :3000):
make dev

# 2. Локальный запуск полного стека в Docker (Frontend :3000 + Gateway :8000):
make dev-docker

# 3. Остановка локального дев-стека в Docker:
make dev-down

# 4. Продакшн запуск с SSL-сертификатами (порты 80/443):
make up

# 5. Остановка продакшн-контейнеров:
make down

# 6. Компиляция Protobuf контрактов:
make proto

# 7. Запуск тестов:
make test
```

