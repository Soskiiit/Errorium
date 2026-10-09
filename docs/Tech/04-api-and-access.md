# 04-api-and-access.md — Спецификация API и протоколов доступа

## 1. Межсервисный протокол gRPC (Search Engine Core)

Связь между Gateway и Search Engine Core осуществляется строго по gRPC на порту `:50051`.
Контракты определены в файлах:
- `protos/search.proto` — сервис поиска и ранжирования `SearchService`.
- `protos/graph.proto` — определяет `GraphService` с методами `GetThreadGraph` и `GetThreadDetails`.

Подробная сигнатура методов описана в разделе [Protobuf контракты](#3-protobuf-контракты).

---

## 2. Внешний REST API (Gateway :8000)

Интерактивная спецификация OpenAPI (Swagger) доступна по адресу `http://localhost:8000/docs`.

### 2.1. Аутентификация и SSO (`/api/v1/auth`)
- `GET /api/v1/auth/sso/login/{provider}` — Инициация OAuth2 авторизации (`google`, `yandex`, `github`).
- `GET /api/v1/auth/sso/callback/{provider}` — Обработка callback с кодом авторизации, выпуск пары JWT токенов (`access_token`, `refresh_token`).
- `POST /api/v1/auth/login` — Локальная аутентификация по логину и паролю.
- `POST /api/v1/auth/refresh` — Обновление access-токена по refresh-токену с ротацией.
- `POST /api/v1/auth/logout` — Отзыв текущей сессии (`is_revoked = true`).
- `GET /api/v1/auth/me` — Получение данных профиля текущего пользователя и его роли (`user` | `admin`).

### 2.2. Пользователи и медиа (`/api/v1/users`)
- `POST /api/v1/users/avatar` — Загрузка и конвертация аватара пользователя в формат WebP с сохранением в MinIO (`errorium-avatars`).

### 2.3. Чаты и сообщения (`/api/v1/chats`)
- `GET /api/v1/chats` — Получение списка последних диалогов пользователя (`Recent chats`) с пагинацией и поиском по заголовку.
- `POST /api/v1/chats` — Создание нового диалога.
- `GET /api/v1/chats/{id}` — Получение истории сообщений диалога, привязанных тегов и RAG-ответов.
- `PATCH /api/v1/chats/{id}` — Переименование диалога, закрепление (`is_pinned`).
- `DELETE /api/v1/chats/{id}` — Удаление диалога со всеми сообщениями (CASCADE).
- `POST /api/v1/chats/{id}/messages` — Отправка сообщения с ошибкой, запуск BM25-поиска и RAG-генерации (поддерживает SSE-стриминг).

### 2.4. Поиск и RAG (`/api/v1/search`)
- `POST /api/v1/search` — Выполнение поиска по BM25:
  ```json
  {
    "query": "NullPointerException Spring Boot autowired service is null",
    "top_k": 10,
    "tags": ["java", "spring-boot"]
  }
  ```
- `POST /api/v1/search/rag` — Запуск RAG-пайплайна с синтезом решения (поддерживает SSE-стриминг `Accept: text/event-stream`).

### 2.5. Граф связей тредов (`/api/v1/graph`)
- `GET /api/v1/graph/threads` — Получение данных подграфа тредов (дубликаты, связанные ответы) для визуализации во фронтенде.

### 2.6. Панель управления (Admin API, `/api/v1/admin`)
*(Требуется заголовок `Authorization: Bearer <TOKEN>` пользователя с ролью `admin`)*
- `GET /api/v1/admin/stats` — Метрики поискового ядра: $N$, $avgdl$, объем памяти, количество запросов.
- `POST /api/v1/admin/index/reload` — Сигнал ядру поиска на горячую перезагрузку файлов индекса.
- `GET /api/v1/admin/users` — Список пользователей, изменение ролей (`user` $\leftrightarrow$ `admin`), блокировка.
- `GET /api/v1/admin/rag/config` / `PUT /api/v1/admin/rag/config` — Настройка системного промпта, выбора LLM-модели и параметров генерации.

---

## 3. Remote MCP Protocol (`/sse`)

Сервис Gateway реализует спецификацию **Model Context Protocol (MCP)** по протоколу HTTP Server-Sent Events (SSE).
**Авторизация для Remote MCP не требуется** — подключение открыто для всех локальных AI-агентов (Cursor, Claude Code, Windsurf) без передачи API-токенов.

### Инструменты MCP (Tools):
1. **`errorium_search_solution`**: Поиск релевантных тем и кода решений по описанию ошибки или стектрейсу.
   - Параметры: `query` (string, required), `tags` (array of strings, optional), `limit` (integer, default: 5).
2. **`errorium_explain_error`**: Получение детального RAG-разбора ошибки с анализом причин и рекомендациями.
   - Параметры: `error_message` (string, required), `stack_trace` (string, optional).
3. **`errorium_get_related_errors`**: Запрос связанных тредов (дубликатов, ссылок) для выявленной проблемы.
   - Параметры: `error_name` (string, required).
