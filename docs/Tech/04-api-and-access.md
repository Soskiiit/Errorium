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

Каждая операция помечена `x-status: core`. Итерация 1 («Ядро + стабы», пакет `handoff/gateway-backend/`, **выполнена**): auth, профиль/аватар, чаты, стабы search/graph/tags/status, Swagger UI. Итерация 2 («Финал до КТ3», пакет `handoff/gateway-final/`, ADR-011/012): волна A — поисковое ядро + benchmark, волна B — gRPC-адаптеры вместо стабов, LLM (OpenAI/GigaChat) + SSE, SSO, сброс/смена пароля, открытый MCP с лимитами, Admin API, волна C — интеграция фронтенда.

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
| SSO | `GET /api/v1/auth/sso/login/{provider}`, `callback/{provider}` (`github\|google\|yandex`) | core (итерация 2) |
| Пароль | `POST /api/v1/auth/password/forgot`, `reset`, `change` | core (итерация 2) |
| Профиль | `GET/PATCH /api/v1/users/me`; `POST /api/v1/users/avatar`; `GET /api/v1/users/{id}/avatar` (публичный) | core |
| Чаты | `GET/POST /api/v1/chats`; `GET/PATCH/DELETE /api/v1/chats/{id}`; `POST /api/v1/chats/{id}/messages` | core |
| Поиск | `POST /api/v1/search`, `POST /api/v1/search/rag` (публичные; gRPC к ядру, LLM, SSE) | core |
| Граф | `GET /api/v1/graph/threads?doc_ids=…`, `GET /api/v1/graph/threads/{doc_id}` (публичные; gRPC GraphService) | core |
| Теги | `GET /api/v1/tags` — курируемый список пилюль фронтенда; при наличии Bearer-токена порядок тегов персонализируется по полю `profession` из JWT (см. §2.4) | core |
| Статус | `GET /api/v1/status` — состояние ядра/RAG для модалки настроек | core |
| MCP | `GET /sse`, `POST /mcp/messages` (открытый доступ, лимиты; токенов нет — ADR-012) | core (итерация 2) |
| Admin | `GET /api/v1/admin/stats`, `POST /api/v1/admin/index/reload`, `GET /api/v1/admin/users`, `PATCH /api/v1/admin/users/{id}`, `GET/PUT /api/v1/admin/rag/config` | core (итерация 2) |
| Служебные | `GET /health`, `GET /docs`, `GET /openapi.yaml` | core |

> **Режим моков (FR-3.5):** до интеграции с `GraphService` (gRPC) экран Star Map (`/star-map`) во фронтенде строит граф из мок-данных: источники RAG-ответа становятся узлами `PRIMARY_RESULT` / `OTHER_RESULT`, пул связанных тредов — узлами `DUPLICATE` / `LINKED`. Структура узлов и ребер повторяет контракты `ThreadGraphResponse` / `ThreadDetailsResponse` из `protos/graph.proto`, поэтому переход на реальный API не требует изменения слоя визуализации. REST-стаб gateway повторяет ту же структуру.

> **SSE-стриминг** (итерация 2): `Accept: text/event-stream` для `/search/rag` и `/chats/{id}/messages`. Формат: `event: delta` / `data: {"text":"…"}` → `event: done` / `data: {полный JSON-ответ}`; ошибки — `event: error` / `data: {"error":"…"}`. Для сообщений чата первым идёт `event: user_message`. Заголовки: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `X-Accel-Buffering: no` (против буферизации nginx). Без заголовка Accept — синхронный JSON, как в итерации 1.

### 2.3. LLM-провайдеры RAG (итерация 2, ADR-011)

Конфигурация хранится в БД (`rag_config`, редактируется через Admin API) и в env:
- **`openai`** — OpenAI-совместимый Chat Completions API с настраиваемым `OPENAI_BASE_URL` (OpenAI, прокси, vLLM, Ollama `/v1`), стриминг `stream: true`.
- **`gigachat`** — бесплатный для физлиц scope `GIGACHAT_API_PERS`. OAuth: `POST https://ngw.devices.sberbank.ru:9443/api/v2/oauth` (Basic base64(client_id:client_secret), заголовок `RqUID: <uuid>`, тело `scope=GIGACHAT_API_PERS`), токен живёт 30 минут. API: `https://api.giga.chat/v1/chat/completions` (OpenAI-совместимый; новый адрес для подключений после 17.07.2026, старый `gigachat.devices.sberbank.ru` выводится из использования). TLS требует корневого сертификата НУЦ Минцифры (`russian_trusted_root_ca.crt`, путь через `GIGACHAT_CA_CERT`).
- **`stub`** — детерминированный шаблон без LLM (поведение итерации 1).
- **Деградация (NFR):** недоступность LLM не ломает поиск: `/search` работает всегда; `/search/rag` и сообщения чата при ошибке LLM используют шаблонный синтез по лучшим источникам.

### 2.4. Персонализация тегов по роли/профессии

При первой регистрации пользователя (или первом входе через SSO) отображается окно онбординга с выбором профессиональной роли (`profession`: Backend developer, Frontend developer, DevOps engineer и др.).
1. Выбранная роль сохраняется через `PATCH /api/v1/users/me` (`profession`).
2. Поле `profession` кодируется в access JWT токен пользователя.
3. Эндпоинт `GET /api/v1/tags` при наличии заголовка `Authorization: Bearer <token>` ранжирует курируемый список из 23 тегов так, чтобы первыми отображались наиболее релевантные технологии для данной профессии (например, Go/Python/SQL для Backend, JS/TypeScript/React для Frontend, Docker/Kubernetes/Linux для DevOps), а остальные теги следовали далее в стандартном порядке.
4. При отсутствии авторизации теги возвращаются в базовом порядке.

---

## 3. Remote MCP Protocol (`/sse`)

Сервис Gateway реализует спецификацию **Model Context Protocol (MCP)** по протоколу HTTP Server-Sent Events (SSE): `GET /sse` (сервер→клиент, первым событием `endpoint` с URL `/mcp/messages?sessionId=…`) и `POST /mcp/messages` (клиент→сервер, JSON-RPC 2.0).

**Доступ — открытый, без токенов (ADR-012):** подключение открыто для всех локальных AI-агентов (Cursor, Claude Code, Windsurf); персональные MCP-токены не выпускаются, таблицы `mcp_tokens` нет. Модалка MCP во фронтенде показывает URL подключения и готовый JSON-конфиг без заголовков авторизации. Защита от абьюза — лимиты (не по IP-запросам, чтобы общий NAT зала на защите не блокировался): ≤100 одновременных сессий с IP, ≤200 сессий глобально, ≤60 `tools/call`/мин на сессию, ≤8 одновременных LLM-вызовов глобально (сверх — деградация до шаблонного синтеза). Значения настраиваются env (`MCP_MAX_SESSIONS_PER_IP`, `MCP_MAX_SESSIONS`, `MCP_CALLS_PER_MIN`, `LLM_MAX_CONCURRENT`). Прежние формулировки «гибрид с токенами» (ADR-011) и противоречие с FR-4.4 сняты: FR-4.4 переписан под открытый доступ.

### Инструменты MCP (Tools):
1. **`errorium_search_solution`**: Поиск релевантных тем и кода решений по описанию ошибки или стектрейсу.
   - Параметры: `query` (string, required), `tags` (array of strings, optional), `limit` (integer, default: 5).
2. **`errorium_explain_error`**: Получение детального RAG-разбора ошибки с анализом причин и рекомендациями.
   - Параметры: `error_message` (string, required), `stack_trace` (string, optional).
3. **`errorium_get_related_errors`**: Запрос связанных тредов (дубликатов, ссылок) для выявленной проблемы.
   - Параметры: `error_name` (string, required).
