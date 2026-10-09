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
   - Генерация персональных долгоживущих API-ключей для MCP в IDE.
   - Разделение прав: `guest`, `developer`, `admin`.
5. **Admin REST API:**
   - Статистика индекса и поисковых задержек.
   - Настройка системных промптов и температурных режимов LLM.
   - Управление учетными записями пользователей.

---

## 2. Спецификация REST эндпоинтов

- `GET /docs` — интерактивная OpenAPI документация (Swagger UI).
- `POST /api/v1/search` — выполнение лексического поиска через Core.
- `POST /api/v1/search/rag` — запуск RAG-генерации (поддержка SSE стриминга).
- `GET /api/v1/graph/threads` — проксирование подграфа связей тредов для Web UI.
- `GET /api/v1/auth/sso/login/{provider}` / `GET /api/v1/auth/sso/callback/{provider}` — SSO авторизация.
- `GET /sse` — точка подключения Remote MCP сервера.

---

## 3. Конфигурация и переменные окружения

```bash
GATEWAY_PORT=8000                  # Порт HTTP REST API и MCP
SEARCH_ENGINE_HOST=search_engine_core
SEARCH_ENGINE_PORT=50051

JWT_SECRET_KEY=secret_key
JWT_ALGORITHM=HS256
ACCESS_TOKEN_EXPIRE_MINUTES=60
REFRESH_TOKEN_EXPIRE_DAYS=30

GITHUB_CLIENT_ID=...
GITHUB_CLIENT_SECRET=...
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...

LLM_PROVIDER=openai               # openai | anthropic | local
OPENAI_API_KEY=sk-...
OPENAI_MODEL_NAME=gpt-4o-mini
LLM_TEMPERATURE=0.2
LLM_MAX_TOKENS=1500
```
