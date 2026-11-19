# Gateway & MCP Service

> **Роль 3: Backend, RAG & MCP Architect**  
> Центральный бэкенд-шлюз, gRPC-клиент к поисковому ядру, RAG-оркестратор (LLM), сервер протокола Model Context Protocol (Remote MCP) и модуль авторизации (SSO/JWT).

---

## 1. Назначение и зона ответственности

Сервис выступает единой точкой входа для фронтенда и внешних AI-агентов (Cursor, Claude Code, Windsurf). Он абстрагирует низкоуровневые gRPC-вызовы поискового ядра и объединяет лексический поиск с генеративным AI.

### Основные задачи:
1. **gRPC-клиент к Core (`:50051`):**
   - Вызовы `SearchService.Search`, `SearchService.GetStats`, `SearchService.HealthCheck`.
   - Вызовы `GraphService.GetThreadGraph`, `GraphService.GetThreadDetails`.
2. **RAG-пайплайн (Retrieval-Augmented Generation):**
   - Агрегация и очистка HTML-контекста из Top-K документов (где $K \in [3, 10]$).
   - Формирование структурированного промпта (первопричина, готовый блок кода, профилактика).
   - Обращение к LLM (OpenAI, Anthropic, локальные через Ollama/vLLM).
   - Потоковый стриминг ответа клиенту по протоколу Server-Sent Events (SSE).
3. **Remote MCP Server (`/sse`):**
   - Реализация стандарта Model Context Protocol:
     - `errorium_search_solution(query, tags, limit)` — поиск релевантных тем и кода.
     - `errorium_explain_error(error_message, stack_trace)` — детальный RAG-разбор ошибки.
     - `errorium_get_related_errors(error_name)` — получение связанных тредов (дубликатов).
4. **Аутентификация, SSO и RBAC:**
   - Вход через GitHub OAuth2, Google OIDC и корпоративные IdP.
   - Выпуск и верификация пары JWT (Access + Refresh).
   - Ротация refresh-токенов с отзывом старых сессий.
   - Безопасное хэширование паролей (bcrypt cost 12) с прозрачной миграцией legacy SHA-256.
   - Поддержка `remember_me` (30 дней против 24 часов).
   - Разделение прав: `guest`, `user`, `admin`.
5. **Чаты и конвейер сообщений:**
   - Персистентное хранение диалогов и сообщений в PostgreSQL 18.
   - Проверка владения чатами (возврат 404 для чужих ресурсов).
   - Извлечение тегов из скобок в запросе (`[Python]`).
   - Автоматическая генерация заголовков чатов (до 80 рун).
6. **Профиль пользователя и аватары:**
   - Хранение аватаров в Garage S3 (`avatars/{userID}`).
   - Публичный прокси-роут выдачи с кэширующими заголовками `Cache-Control` и `ETag`.

---

## 2. Спецификация REST эндпоинтов

Полный контракт OpenAPI 3.0.3 зафиксирован в файле [`api/openapi.yaml`](api/openapi.yaml) и встроен в сервис:
- `GET /docs` — интерактивная OpenAPI документация (Swagger UI на базе `swagger-ui-dist@5.35.0`).
- `GET /openapi.yaml` — исходная OpenAPI-спецификация в формате YAML.

### Сводная карта групп эндпоинтов (`x-status: core`):

| Группа | Маршруты | Описание |
| :--- | :--- | :--- |
| **Auth** | `POST /api/v1/auth/register`<br>`POST /api/v1/auth/login`<br>`POST /api/v1/auth/refresh`<br>`POST /api/v1/auth/logout`<br>`GET /api/v1/auth/me` | Регистрация, аутентификация, ротация сессий, завершение сессии, текущий пользователь. |
| **Users** | `GET /api/v1/users/me`<br>`PATCH /api/v1/users/me`<br>`POST /api/v1/users/avatar`<br>`GET /api/v1/users/{id}/avatar` | Профиль пользователя, обновление полей (profession, gender, language), загрузка аватара в Garage S3, публичная выдача аватара. Поле `role` неизменяемо (400). |
| **Chats** | `GET /api/v1/chats`<br>`POST /api/v1/chats`<br>`GET /api/v1/chats/{id}`<br>`PATCH /api/v1/chats/{id}`<br>`DELETE /api/v1/chats/{id}`<br>`POST /api/v1/chats/{id}/messages` | CRUD диалогов, строгая проверка владельца (404 на чужой чат), отправка сообщения с извлечением `[tag]`, поиск и RAG-синтез, автозаголовок до 80 рун. |
| **Search & RAG** | `POST /api/v1/search`<br>`POST /api/v1/search/rag` | Лексический поиск и RAG-синтез (детерминированный стаб на границе gRPC). |
| **Graph** | `GET /api/v1/graph/threads?doc_ids=...`<br>`GET /api/v1/graph/threads/{doc_id}` | Подграф связей тредов (PRIMARY, OTHER, DUPLICATE, LINKED) и детали треда. |
| **Tags** | `GET /api/v1/tags` | Каталог 23 стандартных тегов фронтенда в фиксированном порядке. |
| **Status** | `GET /api/v1/status` | Состояние сервисов (Gateway, Search Core, RAG) для модального окна настроек. |
| **Health & Docs**| `GET /health`<br>`GET /docs`<br>`GET /openapi.yaml` | Проверка жизнеспособности и документация API. |

*Примечание: Эндпоинты SSO, сброса пароля, MCP SSE (`/sse`), персональных токенов MCP и Admin API имеют статус `x-status: planned` и будут реализованы в последующих итерациях.*

---

## 3. Конфигурация и переменные окружения

| Переменная | Значение по умолчанию | Описание |
| :--- | :--- | :--- |
| `GATEWAY_PORT` | `8000` | Порт HTTP-сервера REST API |
| `SEARCH_ENGINE_ADDR` | `search_engine_core:50051` | Адрес gRPC-сервиса Search Engine Core |
| `SEARCH_ENGINE_HOST` | `search_engine_core` | Хост gRPC ядра (если `SEARCH_ENGINE_ADDR` не задан) |
| `SEARCH_ENGINE_PORT` | `50051` | Порт gRPC ядра (если `SEARCH_ENGINE_ADDR` не задан) |
| `DB_HOST` | `localhost` | Хост PostgreSQL |
| `DB_PORT` | `5432` | Порт PostgreSQL |
| `DB_NAME` | `errorium` | Имя базы данных |
| `DB_USER` | `errorium_user` | Пользователь базы данных |
| `DB_PASSWORD` | `errorium_password` | Пароль базы данных |
| `DB_SSLMODE` | `disable` | Режим SSL для PostgreSQL |
| `AUTO_MIGRATE` | `true` | Автоматическое применение миграций при старте (`migrator.Up`) |
| `JWT_SECRET_KEY` | `change-this-in-production` | Секретный ключ подписи JWT токенов (HS256) |
| `ACCESS_TOKEN_EXPIRE_MINUTES` | `60` | Время жизни Access токена в минутах |
| `REFRESH_TOKEN_EXPIRE_DAYS` | `30` | Время жизни Refresh токена (при `remember_me=true`) в днях; без `remember_me` — 24 часа |
| `S3_ENDPOINT` / `GARAGE_ENDPOINT` | `localhost:3900` | S3 API эндпоинт Garage |
| `S3_ACCESS_KEY` / `GARAGE_ACCESS_KEY` | `garageadmin` | Ключ доступа S3 |
| `S3_SECRET_KEY` / `GARAGE_SECRET_KEY` | `garageadminpassword` | Секретный ключ S3 |
| `S3_BUCKET_NAME` / `GARAGE_BUCKET_NAME` | `errorium-avatars` | Имя бакета для аватаров пользователей |
| `S3_REGION` / `GARAGE_REGION` | `garage` | Регион S3 |
| `S3_USE_SSL` / `GARAGE_USE_SSL` | `false` | Использование HTTPS для подключения к S3 |

---

## 4. Хэширование паролей и безопасность

- **Алгоритм:** Все новые пароли хэшируются с использованием **bcrypt** с фактором сложности (cost) **12** (`golang.org/x/crypto/bcrypt`).
- **Обратная совместимость (Legacy SHA-256):** Поддерживается проверка устаревших хэшей формата `<32 hex salt>:<64 hex sha256>` через постоянное время (`crypto/subtle.ConstantTimeCompare`).
- **Автоматический рехэш:** При успешной аутентификации пользователя с legacy-хэшем сервис автоматически перехэширует пароль в bcrypt и обновляет запись в базе данных (`UpdatePasswordHash`) без нарушения пользовательского опыта.
- **Защита от перебора:** Для несуществующего email и неверного пароля возвращается идентичный унифицированный ответ `401 Unauthorized` (`{"error": "invalid credentials"}`).
- **Сессии и ротация:** Refresh-токены хранятся в БД в виде SHA-256 хэшей. При вызове `/api/v1/auth/refresh` старая сессия немедленно отзывается (`is_revoked = true`), выпускается новая пара токенов, а повторное использование старого refresh-токена пресекается (401).

---

## 5. Стабы и граница gRPC

В соответствии с методологией проекта и до готовности микросервиса `search_engine_core` (Роль 2), слой Gateway взаимодействует с поисковым ядром и графом связей исключительно через абстрактные интерфейсы Go:

```go
type SearchClient interface {
    Search(ctx context.Context, req SearchRequest) (*SearchResponse, error)
    RAG(ctx context.Context, req SearchRequest) (*RAGResult, error)
}

type GraphClient interface {
    GetThreadGraph(ctx context.Context, docIDs []int64) (*ThreadGraph, error)
    GetThreadDetails(ctx context.Context, docID int64) (*ThreadDetails, error)
}
```

### Архитектурные свойства стабов:
1. **Детерминированность:** Стаб поиска ранжирует пул фикстур реальных SO-тредов по пересечению термов и числу голосов. Идентичные запросы дают побайтово идентичные результаты и скоры BM25 без использования случайных величин.
2. **Изоляция контракта:** Хендлеры `SearchHandler` и `ChatHandler` зависят только от интерфейсов `SearchClient` и `GraphClient`.
3. **Бесшовная замена на gRPC:** При подключении `search_engine_core` реализация `search.Stub` и `graph.Stub` будет заменена на gRPC-клиент (по `protos/search.proto` и `protos/graph.proto`) исключительно на уровне фабрики в `internal/app/app.go`, без изменения обработчиков HTTP или REST-контракта.
