# Gateway & MCP Service

> **Роль 3: Backend, RAG & MCP Architect**  
> Центральный бэкенд-шлюз, gRPC-клиент к поисковому ядру, RAG-оркестратор (LLM), сервер протокола Model Context Protocol (Remote MCP) и модуль авторизации (SSO/JWT).

---

## 1. Назначение и зона ответственности

Сервис выступает единой точкой входа для фронтенда и внешних AI-агентов (Cursor, Claude Code, Windsurf). Он абстрагирует низкоуровневые gRPC-вызовы поискового ядра и объединяет лексический поиск с генеративным AI.

### Основные задачи:
1. **gRPC-клиент к Core (`:50051`):**
   - Вызовы `SearchService.Search`, `SearchService.GetStats`, `SearchService.ReloadIndex`, `SearchService.HealthCheck`.
   - Вызовы `GraphService.GetThreadGraph`, `GraphService.GetThreadDetails`.
2. **RAG-пайплайн (Retrieval-Augmented Generation):**
   - Агрегация и очистка HTML-контекста из Top-K документов (где $K \in [3, 10]$).
   - Формирование структурированного промпта (первопричина, готовый блок кода, профилактика).
   - Обращение к LLM: `openai` (OpenAI-совместимый Chat Completions, vLLM, Ollama), `gigachat` (с сертификатом НУЦ Минцифры), `stub` (детерминированный шаблон).
   - Потоковый стриминг ответа клиенту по протоколу Server-Sent Events (SSE) с `WriteTimeout = 0`.
   - Отказоустойчивость: деградация LLM до шаблонного синтеза при исчерпании лимитов, ошибках сети или таймаутах (поиск не ломается).
3. **Remote MCP Server (`/sse`):**
   - Реализация стандарта Model Context Protocol (открытый доступ без токенов по ADR-012):
     - `errorium_search_solution(query, tags, limit)` — поиск релевантных тем и кода.
     - `errorium_explain_error(error_message, stack_trace)` — детальный RAG-разбор ошибки.
     - `errorium_get_related_errors(error_name)` — получение связанных тредов (дубликатов).
   - Защита in-memory лимитами: 60 вызовов/мин на сессию, 100 сессий с одного IP, 200 глобальных сессий.
4. **Аутентификация, SSO и безопасность:**
   - Вход через GitHub OAuth2, Google OIDC и Yandex OAuth на Go stdlib.
   - Выпуск и верификация пары JWT (Access + Refresh) с ротацией и отзывом сессий.
   - Безопасное хэширование паролей (bcrypt cost 12) с прозрачной миграцией legacy SHA-256.
   - Сброс пароля по ссылке через email: `LogMailer` для разработки, `SMTPMailer` (TLS/STARTTLS) для продакшна.
   - Смена пароля и сброс отзывают все существующие refresh-сессии пользователя.
   - Разделение прав: `user`, `admin`. Идемпотентный bootstrap первого администратора (`FIRST_ADMIN_*`).
5. **Чаты и конвейер сообщений:**
   - Персистентное хранение диалогов и сообщений в PostgreSQL 18.
   - Проверка владения чатами (возврат 404 для чужих ресурсов).
   - Извлечение тегов из скобок в запросе (`[Python]`).
   - Автоматическая генерация заголовков чатов (до 80 рун).
   - Поддержка SSE-стриминга (`event: user_message`, `event: delta`, `event: done`) и JSON-режима.
6. **Admin API и мониторинг:**
   - Метрики задержек поиска (p50, p95, p99) и общее количество запросов.
   - Горячая перезагрузка индекса поискового ядра через gRPC (`POST /api/v1/admin/index/reload`).
   - Управление пользователями (`GET /api/v1/admin/users`, `PATCH /api/v1/admin/users/{id}`).
   - Конфигурация RAG-пайплайна на лету через БД (`GET/PUT /api/v1/admin/rag/config`).
7. **Профиль пользователя и аватары:**
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
| **Auth** | `POST /api/v1/auth/register`<br>`POST /api/v1/auth/login`<br>`POST /api/v1/auth/refresh`<br>`POST /api/v1/auth/logout`<br>`GET /api/v1/auth/me` | Регистрация, аутентификация, ротация сессий, завершение сессии, текущий профиль. |
| **Password** | `POST /api/v1/auth/password/forgot`<br>`POST /api/v1/auth/password/reset`<br>`POST /api/v1/auth/password/change` | Запрос ссылки сброса (202 Accepted), установка нового пароля по токену, смена пароля с отзывом всех активных refresh-сессий. |
| **SSO** | `GET /api/v1/auth/sso/login/{provider}`<br>`GET /api/v1/auth/sso/callback/{provider}` | OAuth2/OIDC вход через `github`, `google` и `yandex`. Автоматическое связывание с пользователем в БД, редирект на фронтенд с access и refresh токенами. |
| **Users** | `GET /api/v1/users/me`<br>`PATCH /api/v1/users/me`<br>`POST /api/v1/users/avatar`<br>`GET /api/v1/users/{id}/avatar` | Профиль пользователя, обновление полей (profession, gender, language), загрузка аватара в Garage S3, публичная выдача аватара. Поле `role` неизменяемо (400). |
| **Chats** | `GET /api/v1/chats`<br>`POST /api/v1/chats`<br>`GET /api/v1/chats/{id}`<br>`PATCH /api/v1/chats/{id}`<br>`DELETE /api/v1/chats/{id}`<br>`POST /api/v1/chats/{id}/messages` | CRUD диалогов, строгая изоляция (404 на чужой чат), отправка сообщения с извлечением `[tag]`, поиск и RAG-синтез (JSON или SSE), автозаголовок до 80 рун. |
| **Search & RAG** | `POST /api/v1/search`<br>`POST /api/v1/search/rag` | Лексический BM25 поиск и RAG-синтез. Поддержка SSE-стриминга (`Accept: text/event-stream`) с событиями `delta` и `done` либо синхронного JSON. |
| **Graph** | `GET /api/v1/graph/threads?doc_ids=...`<br>`GET /api/v1/graph/threads/{doc_id}` | Подграф связей тредов (PRIMARY, OTHER, DUPLICATE, LINKED) и детали треда через gRPC `GraphService`. |
| **Tags** | `GET /api/v1/tags` | Каталог 23 стандартных тегов фронтенда в фиксированном порядке. |
| **Status** | `GET /api/v1/status` | Состояние сервисов (Gateway, Search Core connected via gRPC, RAG provider) для модального окна настроек. |
| **Admin** | `GET /api/v1/admin/stats`<br>`POST /api/v1/admin/index/reload`<br>`GET /api/v1/admin/users`<br>`PATCH /api/v1/admin/users/{id}`<br>`GET /api/v1/admin/rag/config`<br>`PUT /api/v1/admin/rag/config` | Админ-панель: статистика поискового ядра и метрики задержек (p50/p95/p99), горячая перезагрузка индекса ядра, управление ролями и блокировкой пользователей, настройка параметров RAG в реальном времени. |
| **MCP Server** | `GET /sse`<br>`POST /mcp/messages` | Model Context Protocol по HTTP Server-Sent Events. Открытый доступ, sliding-window rate limiting, выполнение инструментов AI-агентов. |
| **Health & Docs**| `GET /health`<br>`GET /docs`<br>`GET /openapi.yaml` | Проверка жизнеспособности и встроенная документация API (Swagger UI). |

---

## 3. Конфигурация и переменные окружения

| Переменная | Значение по умолчанию | Описание |
| :--- | :--- | :--- |
| `GATEWAY_PORT` | `8000` | Порт HTTP-сервера REST API |
| `SEARCH_ENGINE_ADDR` | `search_engine_core:50051` | Адрес gRPC-сервиса Search Engine Core (при пустом значении — fallback на in-memory стабы) |
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
| `FRONTEND_URL` | `http://localhost:3000` | Базовый URL веб-интерфейса фронтенда для редиректов SSO и ссылок сброса пароля |
| `FIRST_ADMIN_EMAIL` | `admin@errorium.ru` | Email для автоматического bootstrap первого администратора при старте |
| `FIRST_ADMIN_PASSWORD` | `AdminSecurePassword2026!` | Пароль для bootstrap первого администратора |
| `SMTP_HOST` | `""` | Хост SMTP сервера. Если не задан, используется `LogMailer` (ссылка пишется в stdout) |
| `SMTP_PORT` | `587` | Порт SMTP (465 — TLS, 587 — STARTTLS) |
| `SMTP_USER` | `""` | Имя пользователя SMTP |
| `SMTP_PASSWORD` | `""` | Пароль SMTP |
| `SMTP_FROM` | `no-reply@errorium.ru` | Адрес отправителя email |
| `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` | `""` | Учетные данные OAuth приложения GitHub |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | `""` | Учетные данные OAuth приложения Google |
| `YANDEX_CLIENT_ID` / `YANDEX_CLIENT_SECRET` | `""` | Учетные данные OAuth приложения Яндекс |
| `SSO_REDIRECT_BASE` | `http://localhost:8000` | Базовый URL шлюза для формирования redirect_uri в SSO-провайдерах |
| `LLM_PROVIDER` | `openai` | Провайдер LLM: `openai`, `gigachat` или `stub` |
| `LLM_TEMPERATURE` | `0.2` | Температура генерации ответов модели (0.0 — 2.0) |
| `LLM_MAX_TOKENS` | `1500` | Максимальное количество токенов генерации |
| `LLM_MAX_CONCURRENT`| `8` | Глобальный лимит одновременных запросов к LLM (ожидание 2 с, затем fallback на шаблон) |
| `LLM_SYSTEM_PROMPT` | `""` | Системный промпт для RAG-синтеза |
| `OPENAI_BASE_URL` | `https://api.openai.com/v1` | Базовый URL OpenAI-совместимого API (OpenAI, proxy, vLLM, Ollama) |
| `OPENAI_API_KEY` | `""` | Ключ доступа API OpenAI |
| `OPENAI_MODEL_NAME` | `gpt-4o-mini` | Идентификатор модели OpenAI |
| `GIGACHAT_CLIENT_ID` | `""` | Client ID для авторизации в GigaChat OAuth |
| `GIGACHAT_CLIENT_SECRET` | `""` | Client Secret для GigaChat OAuth |
| `GIGACHAT_SCOPE` | `GIGACHAT_API_PERS` | Scope доступа (по умолчанию `GIGACHAT_API_PERS` для физлиц) |
| `GIGACHAT_CA_CERT` | `/app/certs/russian_trusted_root_ca.crt` | Путь к корневому сертификату НУЦ Минцифры внутри контейнера (compose монтирует `./deploy/certs` → `/app/certs`) |
| `MCP_MAX_SESSIONS_PER_IP` | `100` | Максимум одновременных активных MCP сессий с одного IP (429) |
| `MCP_MAX_SESSIONS` | `200` | Максимум одновременных активных MCP сессий глобально (503) |
| `MCP_CALLS_PER_MIN` | `60` | Максимум вызовов `tools/call` в минуту на одну MCP сессию (429) |
| `S3_ENDPOINT` / `GARAGE_ENDPOINT` | `localhost:3900` | S3 API эндпоинт Garage |
| `S3_ACCESS_KEY` / `GARAGE_ACCESS_KEY` | `garageadmin` | Ключ доступа S3 |
| `S3_SECRET_KEY` / `GARAGE_SECRET_KEY` | `garageadminpassword` | Секретный ключ S3 |
| `S3_BUCKET_NAME` / `GARAGE_BUCKET_NAME` | `errorium-avatars` | Имя бакета для аватаров пользователей |

---

## 4. Открытый Remote MCP Server и лимиты (ADR-012)

Gateway предоставляет полнофункциональный сервер протокола **Model Context Protocol (MCP)** для подключения внешних AI-агентов (Cursor, Claude Code, Windsurf) по транспорту HTTP SSE.

### Специфика подключения:
- **Открытый доступ без токенов:** модальное окно фронтенда выдает прямой URL подключения (`http://localhost:8000/sse`) и готовый фрагмент конфигурации `mcpServers` без необходимости генерации токенов или секретов.
- **Двухфазный транспорт:**
  1. Клиент выполняет `GET /sse`. Сервер открывает поток SSE и немедленно отправляет первое событие `event: endpoint` с адресом для сообщений: `data: /mcp/messages?sessionId=<uuid>`.
  2. Все запросы протокола JSON-RPC 2.0 (например, `initialize`, `tools/list`, `tools/call`) отправляются клиентом методом `POST /mcp/messages?sessionId=<uuid>`.
- **Доступные инструменты (Tools):**
  - `errorium_search_solution(query, tags, limit)`: поиск тем и сниппетов кода по ошибке.
  - `errorium_explain_error(error_message, stack_trace)`: детальный RAG-анализ первопричины ошибки и шагов исправления.
  - `errorium_get_related_errors(error_name)`: извлечение дубликатов и связанных тем из графа тредов Stack Overflow.
- **Многоуровневая защита от абьюза (In-Memory Limiter):**
  - Не более **100** активных сессий с одного IP (заголовок `X-Real-IP` от nginx или `RemoteAddr`) $\to$ `429 Too Many Requests`. Зал защиты за общим NAT не блокируется.
  - Не более **200** активных сессий суммарно в системе $\to$ `503 Service Unavailable`.
  - Не более **60** вызовов `tools/call` в минуту на сессию (скользящее окно 60 с) $\to$ `429 Too Many Requests` с заголовком `Retry-After: 5`.
  - Закрытие SSE-соединения клиентом автоматически освобождает слоты и счетчики в лимитере.

---

## 5. RAG-оркестратор, LLM-провайдеры и SSE-стриминг

- **OpenAI-совместимый провайдер (`openai`):** работает с любым сервисом, поддерживающим Chat Completions (`/chat/completions`), включая оригинальный OpenAI, vLLM, LocalAI, Ollama (`/v1`).
- **GigaChat провайдер (`gigachat`):**
  - Получение токена: `POST https://ngw.devices.sberbank.ru:9443/api/v2/oauth` с Basic Auth и заголовком `RqUID`.
  - Вызов чата: `https://api.giga.chat/v1/chat/completions` (OpenAI-совместимый контракт).
  - **Безопасность TLS:** обязательное использование публичного корневого сертификата НУЦ Минцифры (`GIGACHAT_CA_CERT`). Отключение проверки TLS (`InsecureSkipVerify`) строго запрещено.
- **Глобальный семафор (`LLM_MAX_CONCURRENT=8`):** защищает внешние API от превышения rate-limit. При исчерпании слотов запрос ожидает до 2 секунд; если слот не освободился — происходит автоматическая деградация до шаблонного синтеза.
- **WriteTimeout = 0:** в `http.Server` таймаут на запись отключен, что обеспечивает стабильное удержание SSE-потоков любой длительности.
- **Формат SSE-событий:**
  - `/api/v1/search/rag`: `event: delta` $\to$ `event: done`.
  - `/api/v1/chats/{id}/messages`: `event: user_message` $\to$ `event: delta` $\to$ `event: done`.
  - Ошибки транслируются как `event: error`. При отсутствии заголовка `Accept: text/event-stream` отдается стандартный JSON.

---

## 6. Авторизация, SSO, пароли и Admin API

- **Bcrypt (cost 12):** надежное хэширование паролей с прозрачной миграцией устаревших хэшей `salt:sha256`.
- **Сброс и смена пароля:**
  - `POST /api/v1/auth/password/forgot`: всегда отвечает 202 Accepted против enumeration-атак. Токен сброса живет 1 час.
  - В режиме разработки (без `SMTP_HOST`) ссылка логируется через `LogMailer`: `[DEV MAILER] password reset link for ...`.
  - Успешный сброс или смена пароля (`/auth/password/change`) немедленно отзывают все существующие refresh-сессии пользователя.
- **SSO (GitHub, Google, Yandex):**
  - Реализован на стандартной библиотеке Go (`net/http`) без сторонних OAuth-библиотек.
  - Одноразовый криптографический `state` сохраняется в PostgreSQL (срок жизни 10 минут) и удаляется при первом обращении.
  - Профиль IdP нормализуется в `domain.User` и связывается в таблице `user_sso_accounts`.
- **Bootstrap первого администратора:**
  - При старте шлюз проверяет `FIRST_ADMIN_EMAIL`. Если администратор не существует, создается учетная запись с ролью `admin` и паролем из `FIRST_ADMIN_PASSWORD`. Операция строго идемпотентна.
- **Admin API:**
  - Защищен middleware проверки роли `admin`.
  - `GET /api/v1/admin/stats`: общая статистика документов, среднего размера и перцентили задержек поиска (`p50`, `p95`, `p99`).
  - `POST /api/v1/admin/index/reload`: запуск горячей перезагрузки индекса ядра по gRPC без остановки сервиса.
  - `PATCH /api/v1/admin/users/{id}`: блокировка учетных записей (`is_active = false`) и смена ролей.
  - `GET/PUT /api/v1/admin/rag/config`: динамическое переключение LLM-провайдера (`openai`, `gigachat`, `stub`), системного промпта и параметров генерации на лету.
