# 04-api-and-access.md — Спецификация API и протоколов доступа

## 1. Межсервисный протокол gRPC (Search Engine Core)

Связь между Gateway и Search Engine Core осуществляется строго по gRPC на порту `:50051`.
Контракты определены в файлах:
- `protos/search.proto` — сервис поиска и ранжирования `SearchService`.
- `protos/graph.proto` — определяет `GraphService` с методами `GetThreadGraph` и `GetThreadDetails`.

Подробная сигнатура методов описана в разделе [Protobuf контракты](#3-protobuf-контракты).

---

## 2. Внешний REST API (Gateway :8000)

**Машиночитаемый контракт (источник истины):** [`services/gateway/api/openapi.yaml`](../../services/gateway/api/openapi.yaml) — OpenAPI 3.0.3, составлен по фактическому поведению фронтенда (`static/app.js`, `index.html`). Gateway встраивает файл через `go:embed` и отдаёт:
- `GET /openapi.yaml` — спека;
- `GET /docs` — Swagger UI (HTML-страница, читает `/openapi.yaml`).

Каждая операция помечена `x-status`:
- **`core`** — итерация «Ядро + стабы» (текущий пакет задач, см. `handoff/gateway-backend/`): auth (register/login/refresh/logout/me), профиль и аватар, чаты CRUD + сообщения, search/RAG/graph/tags/status как детерминированные стабы, Swagger UI.
- **`planned`** — контракт зафиксирован, реализация в следующих итерациях: SSO, сброс/смена пароля, MCP-токены и `/sse`, SSE-стриминг, Admin API.

### 2.1. Ключевые решения контракта (ADR-010)
1. **Ошибки** — единый JSON `{"error": "<сообщение>"}`; коды: 400 валидация, 401 нет/невалиден токен, 403 нет прав/аккаунт заблокирован, 404 не найдено **или чужой ресурс** (существование не раскрывается), 409 конфликт (email занят).
2. **`role` ≠ «роль» в UI.** `role` (`user|admin`) — RBAC, клиенту недоступен для изменения (убран из `PATCH /api/v1/users/me`). Профессиональная «роль» из профиля UI (Backend developer, QA engineer…) — отдельное поле `profession`. Поля профиля `gender` (`Male|Female|Not specified`) и `language` (`English|Русский`) — значения дропдаунов фронтенда.
3. **Сообщения чата.** В `chat_messages` добавлены `content` (текст: запрос пользователя / решение ассистента) и `code` (блок кода решения) — миграция `000002`. Карточка `.assistant-card` фронтенда = `content` + `code` + `rag_sources`.
4. **Два score у источника.** `question_score` — голоса вопроса SO (целое, «▲ 4180» в UI, размер звезды Star Map); `bm25_score` — релевантность ранжирования (дробное). Поля не смешивать.
5. **Теги в запросе.** Фронтенд вставляет теги пилюлями как `[Python]` в текст запроса. Gateway извлекает скобки, объединяет с явным полем `tags`, нормализует в lowercase, вырезает скобки из текста для поиска; исходный ввод сохраняется в `error_context.raw_query`.
6. **Аватар.** Конвертация в WebP отменена (ADR-010): оригинал PNG/JPEG/WebP/GIF ≤ 5 МиБ кладётся в Garage с ключом `avatars/{user_id}`, выдаётся публичным прокси-эндпоинтом `GET /api/v1/users/{id}/avatar` (тег `<img>` не умеет слать Bearer); `users.avatar_url` = прокси-путь.
7. **Стабы поиска/графа.** До готовности `search_engine_core` (Роль 2) gateway отвечает детерминированными фикстурами через интерфейсы `SearchClient` / `GraphClient`. Замена на gRPC (`protos/search.proto`, `protos/graph.proto`) не меняет REST-контракт. REST-узел графа обогащён `best_answer_body` / `original_url` относительно proto `ThreadNode`.
8. **Пароли.** bcrypt (cost 12) для новых паролей; legacy-формат `salt:sha256` принимается при входе и прозрачно перехэшируется. `remember_me=true` → refresh-сессия 30 дней, иначе 24 часа.

### 2.2. Карта эндпоинтов (полные схемы — в openapi.yaml)

| Группа | Эндпоинты | Статус |
| :--- | :--- | :--- |
| Auth | `POST /api/v1/auth/register`, `login`, `refresh`, `logout`; `GET /api/v1/auth/me` | core |
| SSO | `GET /api/v1/auth/sso/login/{provider}`, `callback/{provider}` (`github\|google\|yandex`) | planned |
| Пароль | `POST /api/v1/auth/password/forgot`, `reset`, `change` | planned |
| Профиль | `GET/PATCH /api/v1/users/me`; `POST /api/v1/users/avatar`; `GET /api/v1/users/{id}/avatar` (публичный) | core |
| Чаты | `GET/POST /api/v1/chats`; `GET/PATCH/DELETE /api/v1/chats/{id}`; `POST /api/v1/chats/{id}/messages` | core |
| Поиск | `POST /api/v1/search`, `POST /api/v1/search/rag` (публичные; стаб) | core |
| Граф | `GET /api/v1/graph/threads?doc_ids=…`, `GET /api/v1/graph/threads/{doc_id}` (публичные; стаб) | core |
| Теги | `GET /api/v1/tags` — курируемый список пилюль фронтенда | core |
| Статус | `GET /api/v1/status` — состояние ядра/RAG для модалки настроек | core |
| MCP | `GET /api/v1/mcp/token`, `POST /api/v1/mcp/token/regenerate`, `GET /sse` | planned |
| Admin | `GET /api/v1/admin/stats`, `POST /api/v1/admin/index/reload`, `GET /api/v1/admin/users`, `PATCH /api/v1/admin/users/{id}`, `GET/PUT /api/v1/admin/rag/config` | planned |
| Служебные | `GET /health`, `GET /docs`, `GET /openapi.yaml` | core |

> **Режим моков (FR-3.5):** до интеграции с `GraphService` (gRPC) экран Star Map (`/star-map`) во фронтенде строит граф из мок-данных: источники RAG-ответа становятся узлами `PRIMARY_RESULT` / `OTHER_RESULT`, пул связанных тредов — узлами `DUPLICATE` / `LINKED`. Структура узлов и ребер повторяет контракты `ThreadGraphResponse` / `ThreadDetailsResponse` из `protos/graph.proto`, поэтому переход на реальный API не требует изменения слоя визуализации. REST-стаб gateway повторяет ту же структуру.

> **SSE-стриминг** (`Accept: text/event-stream` для `/search/rag` и `/chats/{id}/messages`) — planned: в текущей итерации ответы синхронные JSON.

---

## 3. Remote MCP Protocol (`/sse`)

Сервис Gateway реализует спецификацию **Model Context Protocol (MCP)** по протоколу HTTP Server-Sent Events (SSE).
**Авторизация для Remote MCP не требуется** — подключение открыто для всех локальных AI-агентов (Cursor, Claude Code, Windsurf) без передачи API-токенов.

> **Нерешённое противоречие (planned):** FR-4.4 и модалка MCP во фронтенде предполагают персональные Bearer-токены (`err_live_…`), а `docs/Tech/03-data.md` §3.4 фиксирует открытый доступ без таблицы `mcp_tokens`. Эндпоинты `/api/v1/mcp/token*` описаны в openapi.yaml как `planned`; реализация — только после решения владельца продукта.

### Инструменты MCP (Tools):
1. **`errorium_search_solution`**: Поиск релевантных тем и кода решений по описанию ошибки или стектрейсу.
   - Параметры: `query` (string, required), `tags` (array of strings, optional), `limit` (integer, default: 5).
2. **`errorium_explain_error`**: Получение детального RAG-разбора ошибки с анализом причин и рекомендациями.
   - Параметры: `error_message` (string, required), `stack_trace` (string, optional).
3. **`errorium_get_related_errors`**: Запрос связанных тредов (дубликатов, ссылок) для выявленной проблемы.
   - Параметры: `error_name` (string, required).
