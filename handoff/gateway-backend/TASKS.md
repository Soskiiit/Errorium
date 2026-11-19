# Пакет задач: Gateway «Ядро + стабы» (итерация 1)

> Самодостаточный файл задач для команды исполнителей. Всё необходимое — внутри
> этого файла и репозитория. Внешних ссылок нет. Непонятно — остановись и спроси.

---

## 0. Контекст проекта (прочитай обязательно)

**Errorium** — поисковый и RAG-сервис по базе знаний Stack Overflow (учебный
проект НИРС, МГТУ им. Баумана, кафедра ИУ5). Четыре микросервиса:

| Сервис | Директория | Состояние |
| :--- | :--- | :--- |
| search_engine_indexer (Роль 1) | `services/search_engine_indexer/` | Готов (Python). НЕ ТРОГАТЬ. |
| search_engine_core (Роль 2) | `services/search_engine_core/` | Не реализован коллегами. НЕ ТРОГАТЬ. |
| **gateway (Роль 3)** | `services/gateway/` | **Твоя зона работы.** Go 1.27.1. |
| frontend (Роль 4) | `services/frontend/` | Готов на моках (Vanilla JS). НЕ ТРОГАТЬ. |

Текущее состояние gateway: хендлеры auth/chats/users/search написаны, но
**не подключены** — `internal/app/app.go` регистрирует только `/health`.
Известные баги, которые ты исправишь в задаче T1:
1. В `internal/middleware/auth.go` и `internal/handler/chat_handler.go` объявлены
   **два разных типа** `contextKey` с одинаковым значением `"user_id"`. Middleware
   кладёт user_id своим типом, хендлеры чатов/пользователей читают своим —
   не находят и всегда возвращают 401.
2. Чаты не проверяют владельца: любой пользователь может прочитать/удалить
   чужой чат по id.
3. `PATCH /api/v1/users/me` позволяет назначить себе роль `admin`.

**Контракт API (источник истины):** `services/gateway/api/openapi.yaml`
(OpenAPI 3.0.3). Каждая операция помечена `x-status`:
- `core` — реализуется в этом пакете (задачи T1–T8);
- `planned` — контракт для будущих итераций. **НЕ реализовывать.**

Спецификации проекта (для справки, менять не нужно, кроме T8):
`docs/Tech/04-api-and-access.md`, `docs/Tech/03-data.md` (§3),
`docs/Service/decisions.md` (ADR-010 — решения этой итерации).

**Академические запреты (нарушение = дисквалификация проекта):**
- Никаких готовых поисковых библиотек: Elasticsearch, OpenSearch, Lucene,
  Solr, Whoosh, Meilisearch, rank-bm25, bm25s, tantivy, scikit-learn (TF-IDF/BM25).
- BM25 и инвертированный индекс реализуются с нуля в `search_engine_core`
  (не твоя зона). В gateway поиск — **стаб** за интерфейсом, без ранжирования.
- Gateway ↔ search_engine_core — только gRPC (`protos/search.proto`,
  `protos/graph.proto`). В этой итерации gRPC-клиент НЕ пишется: ядро не готово.
  Стабы должны быть изолированы интерфейсами так, чтобы позже замена на gRPC
  не меняла REST-контракт и хендлеры.

---

## 1. Окружение и команды

### Требования к машине
- Go >= 1.27.1 (`go version`), доступ к proxy.golang.org (разрешено `go get`).
- Docker + Docker Compose v2, GNU Make, python3 (для тестов проекта).
- Клон репозитория, ветка `feat/significant-backend-update`.

### Разрешённые зависимости (только эти, новые — спроси)
- Уже в `go.mod`: `github.com/google/uuid`, `github.com/golang-migrate/migrate/v4`,
  `github.com/lib/pq`.
- Можно добавить: `golang.org/x/crypto` (bcrypt), `github.com/aws/aws-sdk-go-v2`
  (+ `config`, `credentials`, `service/s3`) — для Garage S3.
- Запрещено: ORM, веб-фреймворки (gin/echo/fiber), готовые поисковые библиотеки,
  swagger-генераторы из аннотаций (swaggo). Роутинг — stdlib `net/http.ServeMux`
  с паттернами Go 1.22+ (`"POST /api/v1/..."`, `r.PathValue("id")`).

### Подготовка (один раз)
```bash
cp .env.example .env        # dev-значения по умолчанию подходят
cd services/gateway && go build ./... && go test ./...   # должно быть зелено
```

### Гейты качества (после КАЖДОЙ задачи, все обязаны пройти)
```bash
cd services/gateway
gofmt -l .                  # вывод ДОЛЖЕН быть пустым
go build ./...
go vet ./...
go test ./...
```
Дополнительно из корня (не должны сломаться чужие тесты):
```bash
python3 -m unittest discover -s tests/unit -p "test_*.py"
```

### Dev-стек (нужен для T8; для T1–T7 не обязателен)
```bash
docker compose -f docker-compose.dev.yml up --build -d
# postgres:18, garage (S3), gateway (:8000), frontend (:3000)
curl http://localhost:8000/health          # {"status":"ok"}
docker compose -f docker-compose.dev.yml down
```
Если порты заняты — меняй `POSTGRES_PORT` / `GARAGE_PORT` / `GATEWAY_PORT` /
`FRONTEND_PORT` в `.env`, не в compose-файлах.

---

## 2. Общие правила для каждой задачи

1. **Одна задача = один коммит.** Коммит — только после зелёных гейтов.
   Сообщение коммита — простое, на русском, БЕЗ префиксов вида `feat:`/`fix:`.
   Точное сообщение указано в задаче.
2. **TDD:** сначала тесты (красные), потом реализация (зелёные). Unit-тесты —
   в тех же пакетах (`*_test.go`), table-driven, `net/http/httptest`,
   фейковые реализации интерфейсов хранилищ. Unit-тесты НЕ требуют БД/Docker.
3. **Скоуп:** меняй только файлы, перечисленные в задаче (плюс новые файлы
   и тесты к ним). Не трогай frontend, indexer, core, `protos/`, `docs/`,
   `openapi.yaml`, compose-файлы, Makefile. Не реформатируй чужой код.
   Корневые `STATUS.md` / `CHANGE_HISTORY.md` обновляет только T8.
4. **Стиль существующего кода:** хелперы `respondJSON`/`respondError`
   (`internal/handler/helpers.go`), ошибки `{"error": "msg"}`, доменные типы
   в `domain/`, репозитории в `internal/repository/` (database/sql + lib/pq,
   без ORM), комментарии — только когда «почему» неочевидно из кода.
5. **Контракт неизменен:** формы запросов/ответов, коды ошибок и пути — строго
   по `openapi.yaml` (`x-status: core`). Если спека кажется ошибочной —
   остановись и спроси, не правь спеку сам.
6. **Безопасность:** секреты не логируются и не коммитятся; пароли/токены
   в ответах не возвращаются (кроме выдачи пары токенов при auth);
   `password_hash` никогда не сериализуется в JSON (тег `json:"-"` уже есть).
7. **Непонятно / противоречие / гейт не чинится за 3 попытки — остановись
   и спроси человека.** Не угадывай, не обходи проверки, не удаляй чужой код.

### Форма ошибок (единая для всех эндпоинтов)
```json
{"error": "invalid credentials"}
```
Коды: 400 — валидация; 401 — нет/невалиден Bearer; 403 — нет прав или аккаунт
заблокирован; 404 — не найдено ИЛИ чужой ресурс; 409 — конфликт (email занят);
502 — недоступен S3. Тело 204 — пустое.

### Ключевые схемы (выжимка из openapi.yaml; полные — в файле спеки)

**User** (JSON): `id` (uuid), `email`, `username`, `avatar_url` (string|null),
`role` (`user|admin`, readOnly), `profession` (string, default "Backend developer"),
`gender` (`Male|Female|Not specified`, default "Not specified"),
`language` (`English|Русский`, default "English"), `is_active` (bool),
`created_at`, `updated_at`, `last_login_at` (RFC3339|null).

**TokenPair**: `access_token` (JWT HS256), `refresh_token` (opaque), `user` (User).

**Chat**: `id`, `user_id`, `title`, `is_pinned`, `created_at`, `updated_at`.

**ChatMessage**: `id`, `chat_id`, `sender_role` (`user|assistant|system`),
`message_type` (`text|error_report|rag_solution`), `content` (string|null),
`code` (string|null), `error_context` (object|null: `language`, `stack_trace`,
`error_type`, `raw_query`, `tags[]`), `rag_sources` (Source[]|null),
`token_usage` (object|null), `tags` (Tag[]: `{id?, name}`), `created_at`.

**Source** (он же результат поиска): `doc_id` (int64, id вопроса SO), `title`,
`url`, `snippet`, `tags[]` (string), `question_score` (int — голоса SO, «▲ 4180»
в UI), `bm25_score` (float — релевантность), `is_accepted` (bool).
**Это два разных score, не смешивать.**

**ThreadNode** (Star Map): `doc_id`, `category`
(`PRIMARY_RESULT|OTHER_RESULT|DUPLICATE|LINKED`), `score` (float — голоса SO,
размер звезды), `title`, `snippet`, `has_accepted_answer`, `tags[]`,
`best_answer_score` (int), `best_answer_body` (string), `original_url`.
**ThreadEdge**: `source_doc_id`, `target_doc_id`, `type`
(`DUPLICATE_OF|LINKED_IN_ANSWER`).

---

## 3. Граф задач

```
T1 (wiring, context key, CORS)
 └─ T2 (миграция 000002, domain, репозитории)
     └─ T3 (auth: bcrypt, remember_me)
         └─ T4 (стабы search/graph/tags/status)
             └─ T5 (чаты: владелец, конвейер сообщений)   [нужны T2+T4]
                 └─ T6 (профиль + аватар S3)              [нужны T2+T4]
                     └─ T7 (Swagger UI /docs, /openapi.yaml)
                         └─ T8 (интеграционные тесты, документация, финальный прогон)
```
Режим — **строго линейно** T1→T2→…→T8. Параллелить T3/T4 нельзя:
обе задачи правят `internal/app/app.go` (конструкторы хендлеров).

---

## T1. Подключение хендлеров, единый ключ контекста, CORS

**Цель:** все существующие хендлеры начинают работать; исправить баг №1
(разные типы context key); закрыть чаты/профиль авторизацией; CORS.

**Файлы:**
- `services/gateway/internal/app/app.go`
- `services/gateway/internal/handler/helpers.go`
- `services/gateway/internal/handler/chat_handler.go`
- `services/gateway/internal/handler/user_handler.go`
- `services/gateway/pkg/config/config.go` (+ `config_test.go`)
- `services/gateway/cmd/main.go` (закрытие БД при shutdown)
- новые тесты: `internal/app/app_test.go`, `internal/handler/context_test.go`

**Требования:**
1. Удали локальный `contextKey`/`ctxKeyUserID`/`userIDFromCtx` из
   `chat_handler.go`. Все чтения user_id — через `middleware.UserIDFromCtx`,
   роли — через `middleware.RoleFromCtx`. В `helpers.go` оставь
   `respondJSON`, `respondError`, `parseUUIDParam`, `queryInt`.
2. `app.New(cfg)`:
   - если `cfg.DB.Host != ""` — открой пул `db.New(cfg.DB)`, создай
     репозитории `NewUserRepository`, `NewSessionRepository`,
     `NewChatRepository`, `NewTagRepository`;
   - создай `jwt.NewManager(cfg.JWT)`;
   - зарегистрируй: health (как сейчас), `NewAuthHandler(users, sessions, jwtMgr)`,
     `NewChatHandler(chatRepo, tagRepo)`, `NewUserHandler(userRepo)`,
     `NewSearchHandler()`;
   - **все** маршруты `/api/v1/chats...` и `/api/v1/users/me`,
     `POST /api/v1/users/avatar` оберни в `middleware.Auth(jwtMgr)`
     (без Bearer → 401). `/api/v1/search...` и `/health` — публичные.
     Обёртку делай там, где регистрируются маршруты (в `RegisterRoutes`
     чатов/пользователей передавай middleware параметром либо оборачивай
     в app.go тем же паттернам — поведение важнее способа);
   - весь mux оберни в `middleware.CORS()`;
   - сохрани graceful shutdown; добавь закрытие пула БД в `Shutdown`.
   - Если БД не настроена (`DB.Host == ""`), сервис должен стартовать
     и отдавать `/health` (auth/chats/users при этом недоступны — 503
     или не регистрируются; выбери «не регистрируются» и зафиксируй в тесте).
3. `pkg/config`: добавь поле `SearchEngineAddr string` —
   `getEnv("SEARCH_ENGINE_ADDR", SEARCH_ENGINE_HOST + ":" + SEARCH_ENGINE_PORT)`,
   хост по умолчанию `search_engine_core`, порт `50051`.
4. Поведение auth-хендлеров НЕ меняй (это T3), кроме замены ключа контекста.

**Тесты (напиши сначала):**
- `context_test.go`: middleware.Auth с валидным JWT кладёт user_id, который
  читается через `middleware.UserIDFromCtx` в хендлере чатов (httptest +
  фейковые ChatStore/TagStore): с токеном — 200, без — 401, с мусором — 401.
- `app_test.go`: `New(cfg)` с пустым DB.Host — `/health` 200,
  `/api/v1/chats` не зарегистрирован (404); CORS-заголовки присутствуют
  на произвольном ответе; OPTIONS → 204.
- `config_test.go`: дефолт `SearchEngineAddr` = `search_engine_core:50051`;
  переопределение через `SEARCH_ENGINE_ADDR`.

**Критерии готовности:** гейты §1 зелёные; `go test ./...` включает новые
тесты; без токена `/api/v1/chats` → 401 (при настроенной БД) либо маршрут
отсутствует (без БД); чужие тесты проекта не сломаны.

**Коммит:** `Подключить хендлеры gateway к роутеру и исправить ключи контекста`

---

## T2. Миграция 000002: поля профиля и текст сообщений

**Цель:** схема БД покрывает UI: profession/gender/language у пользователя,
content/code у сообщения (баг схемы: RAG-ответу негде храниться).

**Файлы:**
- `services/gateway/migrations/000002_profile_and_message_content.up.sql` (новый)
- `services/gateway/migrations/000002_profile_and_message_content.down.sql` (новый)
- `services/gateway/migrations/migrations_test.go`
- `services/gateway/domain/user.go`, `domain/chat.go`
- `services/gateway/internal/repository/user_repo.go`, `chat_repo.go`
- новые тесты: `domain/user_test.go`, `domain/chat_test.go`

**Требования:**
1. SQL up (идемпотентность не нужна — мигратор версионный; стиль 000001):
   ```sql
   ALTER TABLE users ADD COLUMN profession TEXT NOT NULL DEFAULT 'Backend developer' CHECK (length(profession) <= 100);
   ALTER TABLE users ADD COLUMN gender TEXT NOT NULL DEFAULT 'Not specified' CHECK (gender IN ('Male', 'Female', 'Not specified'));
   ALTER TABLE users ADD COLUMN language TEXT NOT NULL DEFAULT 'English' CHECK (language IN ('English', 'Русский'));
   ALTER TABLE chat_messages ADD COLUMN content TEXT;
   ALTER TABLE chat_messages ADD COLUMN code TEXT;
   ```
   down — `DROP COLUMN` всех пяти (в обратном порядке).
2. `domain.User`: добавь `Profession string` (`json:"profession"`),
   `Gender string` (`json:"gender"`), `Language string` (`json:"language"`).
   `domain.ChatMessage`: добавь `Content *string` (`json:"content"`),
   `Code *string` (`json:"code"`) — указатели, чтобы в JSON был `null`.
3. `user_repo.go`: Create/scanUser/UpdateProfile включают новые поля;
   **UpdateProfile расширяется до полного профиля**:
   `username, email, avatar_url, profession, gender, language, updated_at`
   (WHERE id). Добавь метод `UpdatePasswordHash(ctx, id uuid.UUID, hash string) error`
   (UPDATE users SET password_hash=$1, updated_at=NOW() WHERE id=$2) —
   понадобится T3; обнови интерфейс `repository.UserStore` в `interfaces.go`.
4. `chat_repo.go`: AddMessage/GetMessages включают `content`, `code`.
5. Миграции встраиваются автоматически (`go:embed *.sql` уже есть).

**Тесты:**
- `migrations_test.go`: 000002 up/down встроены и непустые; up содержит все
  пять ADD COLUMN, down — все пять DROP COLUMN.
- `domain` тесты: JSON User содержит profession/gender/language и НЕ содержит
  password_hash; JSON ChatMessage с nil Content/Code даёт `"content":null`,
  `"code":null`; с заполненными — строки.

**Критерии готовности:** гейты зелёные; `migrator.Up` на реальном PostgreSQL
применяет 000002 (проверяется в T8; здесь — SQL-ревью и embed-тесты).

**Коммит:** `Добавить поля профиля и текст сообщений в схему gateway`

---

## T3. Auth: bcrypt, remember_me, legacy-перехэширование

**Цель:** пароли хранятся в bcrypt; чекбокс «Запомнить меня» управляет TTL
refresh-сессии; старые sha256-хэши продолжают работать и мигрируют при входе.

**Файлы:**
- `services/gateway/internal/handler/password.go`
- `services/gateway/internal/handler/auth_handler.go`
- `services/gateway/internal/repository/interfaces.go` (если не сделано в T2)
- `services/gateway/internal/repository/user_repo.go` (UpdatePasswordHash)
- `services/gateway/internal/app/app.go` (конструктор AuthHandler)
- новые тесты: `internal/handler/password_test.go`, `internal/handler/auth_handler_test.go`

**Требования:**
1. `go get golang.org/x/crypto/bcrypt`.
2. `hashPassword` → bcrypt cost 12. `checkPassword(stored, plain)`:
   - если stored начинается с `$2` — `bcrypt.CompareHashPassword`;
   - иначе legacy-формат `<32 hex salt>:<64 hex sha256>` (см. текущий код) —
     сравнение constant-time (`crypto/subtle`);
   - добавь `needsRehash(stored string) bool` (true для legacy).
3. `login`: после успешной проверки, если `needsRehash` — перехэшируй bcrypt
   и сохрани через `users.UpdatePasswordHash` (ошибка сохранения не ломает
   вход, но логируется). Неверный email и неверный пароль → одинаковый
   ответ 401 `invalid credentials`. `is_active=false` → 403 `account is disabled`.
4. `LoginRequest` += `RememberMe bool` (`json:"remember_me"`).
   `issueTokenPair` получает TTL refresh-сессии: `remember_me=true` →
   `cfg.JWT.RefreshTokenExpireDays` суток (30), иначе 24 часа.
   Конструктор `NewAuthHandler` принимает `refreshExpDays int` (из cfg).
5. `register`: валидация по спеке (username 1..100 после trim, email формат +
   lowercase, password 8..128); новые поля профиля — дефолты
   (`Backend developer`, `Not specified`, `English`); duplicate email → 409
   `email already registered`; ответ 201 TokenPair.
6. `refresh`: ротация (старая сессия отзывается, новая создаётся) — уже есть,
   сохрани; истёкшая/отозванная/неизвестная → 401. `logout` → 204.
   `me` → User или 404.

**Тесты (фейковые UserStore/SessionStore, httptest):**
- password: bcrypt roundtrip; legacy-хэш проверяется; needsRehash; неверный
  пароль false; битый формат false.
- register: 201 + пара токенов + user с дефолтами; 400 (короткий пароль,
  пустой username, битый email); 409 duplicate.
- login: 200; 401 неверный пароль и неизвестный email (одинаковое тело);
  403 неактивный; legacy-хэш → вход успешен + UpdatePasswordHash вызван
  с bcrypt-хэшем; remember_me=true → сессия ~30 суток, false → ~24 часа
  (проверка ExpiresAt в фейке, допуск ±1 мин).
- refresh: ротация (старый хэш отозван, новая сессия создана); 401 для
  отозванного/истёкшего/неизвестного.
- logout 204 (с refresh_token и без); me 200/401/404.

**Критерии готовности:** гейты зелёные; `go.mod` содержит только разрешённые
зависимости; все сценарии выше покрыты тестами.

**Коммит:** `Перевести пароли gateway на bcrypt и добавить remember_me`

---

## T4. Стабы поиска, графа, тегов и статуса

**Цель:** публичные эндпоинты search/search-rag/graph/tags/status отвечают
детерминированными данными по контракту; граница стаба — интерфейсы
`SearchClient`/`GraphClient`, готовые к замене на gRPC.

**Файлы:**
- `services/gateway/internal/search/client.go` (новый: интерфейсы + DTO)
- `services/gateway/internal/search/stub.go` (новый: фикстуры + реализация)
- `services/gateway/internal/graph/client.go`, `stub.go` (новые)
- `services/gateway/internal/handler/search_handler.go` (переписать)
- `services/gateway/internal/handler/tags_handler.go`, `status_handler.go` (новые)
- `services/gateway/internal/app/app.go` (wiring новых конструкторов)
- тесты: `internal/search/stub_test.go`, `internal/graph/stub_test.go`,
  `internal/handler/search_handler_test.go`, `tags_handler_test.go`,
  `status_handler_test.go`

**Требования:**
1. Интерфейсы (DTO строго по openapi.yaml, схемы Source/SearchResponse/
   RagResponse/ThreadGraph/ThreadDetails):
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
   `SearchRequest{Query string; TopK int; Tags []string; Language string}`.
   Хендлеры зависят ТОЛЬКО от интерфейсов.
2. Стаб поиска: фикстуры >= 10 реальных SO-тредов про ошибки (придумай сам:
   Python/Java/C++/Go и т.п.; doc_id — настоящие id вопросов SO, url —
   `https://stackoverflow.com/questions/{doc_id}`; у каждого title, snippet,
   tags, question_score (голоса, целое), is_accepted, тело ответа с кодом).
   Ранжирование стаба — детерминированное пересечение слов запроса (lowercase,
   split по не-буквам/цифрам) с title+tags+snippet: `bm25_score` = float на
   основе числа совпадений (например, совпадения + 0.1*question_score/1000),
   сортировка по убыванию, обрезка до top_k. Никакого rand/time в скоринге.
   Одинаковый запрос → байт-в-байт одинаковый ответ. `total_hits` — число
   документов с ненулевым скором. Фильтр tags: если переданы — оставь только
   треды, имеющие хотя бы один из тегов (lowercase). `took_ms` — замер времени.
3. Стаб RAG: `synthesis` — шаблонный текст про запрос и лучший источник
   (2–4 предложения), `code` — блок кода из фикстуры лучшего треда,
   `sources` — результаты Search. Детерминированно.
4. Стаб графа: пул >= 12 связанных тредов (отдельные фикстуры с answer-текстом).
   `GetThreadGraph(docIDs)`: узлы-запросы — первый `PRIMARY_RESULT`, остальные
   `OTHER_RESULT` (данные из фикстур поиска; неизвестный id — пропусти);
   связанные из пула ранжируются пересечением тегов с узлами-запросами,
   берутся до 12, первые 3 — `DUPLICATE`, остальные `LINKED`; без дублей
   doc_id. Рёбра: OTHER→PRIMARY `LINKED_IN_ANSWER`; DUPLICATE→PRIMARY
   `DUPLICATE_OF`; LINKED→PRIMARY или OTHER `LINKED_IN_ANSWER`; без петель.
   Узлы содержат best_answer_body и original_url.
   `GetThreadDetails(docID)`: фикстура или ошибка «not found» (хендлер → 404).
5. `GET /api/v1/tags`: константа — ровно эти 23 имени в этом порядке:
   Go, Python, Java, C#, C++, JS, SQL, Rust, TypeScript, Docker, Linux,
   Kubernetes, PostgreSQL, Redis, GraphQL, React, Vue, Swift, Kotlin,
   PHP, Git, Nginx, gRPC. Ответ `{"items":[{"name":"Go"},...]}`.
6. `GET /api/v1/status`:
   `{"gateway":{"status":"ok","version":"1.0.0"},"search_core":{"connected":false,"address":"<cfg.SearchEngineAddr>"},"rag":{"enabled":false,"provider":"stub"}}`.
7. Хендлеры: `POST /api/v1/search` и `/search/rag` — публичные, валидация
   query (1..8000 после trim) → 400; top_k default 10, clamp 1..50;
   `GET /api/v1/graph/threads?doc_ids=1,2` — парсинг через запятую, 1..20
   штук, int64 → иначе 400; `GET /api/v1/graph/threads/{doc_id}` → 200/404.
   Ответы — строго по схемам спеки (имена полей snake_case).

**Тесты:**
- детерминизм: два вызова Search/RAG/GetThreadGraph с одинаковым входом →
  идентичные результаты; порядок по bm25_score DESC; top_k соблюдён;
  фильтр tags работает; пустой результат для бессмысленного запроса — OK.
- граф: категории (первичный/остальные/3 дубликата/прочие linked), рёбра без
  петель и без дублей узлов, неизвестные id пропускаются, details 404-кейс.
- хендлеры: 400 (пустой query, кривой doc_ids, >20 id), формы ответов
  (наличие всех обязательных полей), tags — ровно 23 и порядок, status —
  address из конфига.

**Критерии готовности:** гейты зелёные; хендлеры не импортируют конкретные
стабы напрямую (только интерфейсы, wiring в app.go); `planned`-эндпоинты
НЕ реализованы.

**Коммит:** `Добавить стабы поиска, графа, тегов и статуса в gateway`

---

## T5. Чаты: проверка владельца и конвейер сообщений

**Цель:** закрыть баг №2 (чужие чаты), реализовать sendMessage по контракту
(теги из скобок, поиск+RAG через клиент T4, content/code, автозаголовок).

**Файлы:**
- `services/gateway/internal/handler/chat_handler.go`
- `services/gateway/internal/repository/chat_repo.go` (+ интерфейс ChatStore)
- `services/gateway/internal/app/app.go` (передача SearchClient в ChatHandler)
- тесты: `internal/handler/chat_handler_test.go` (новый)

**Требования:**
1. Владение: добавь в ChatStore/репозиторий
   `GetByIDForUser(ctx, id, userID uuid.UUID) (*domain.Chat, error)`
   (SELECT ... WHERE id=$1 AND user_id=$2; нет строки → ошибка not found).
   GET/PATCH/DELETE `/chats/{id}` и POST `/chats/{id}/messages` используют
   его; не-владелец и несуществующий id → одинаковый 404 `chat not found`.
2. `SendMessage` (POST `/chats/{id}/messages`), порядок:
   a. Извлечь теги regex `\[([^\[\]]{1,50})\]` из `query`; объединить с
      полем `tags`; lowercase; дедуплицировать; обрезать до 50 символов.
      `cleanQuery` — query без скобок, пробелы схлопнуты, trim.
      Пустой cleanQuery → 400.
   b. Сохранить user-сообщение: `sender_role=user`,
      `message_type=error_report`, `content` = исходный query (со скобками),
      `error_context` = `{"raw_query": <исходный query>, "tags": <merged>}`.
      Привязать теги через TagStore.AttachToMessage (если есть).
   c. Вызвать `SearchClient.Search(cleanQuery, topK default 5 clamp 1..20, tags)`
      и `SearchClient.RAG(...)`. Ошибка клиента → 502 `search backend unavailable`
      (user-сообщение при этом уже сохранено — это допустимо, зафиксируй в тесте).
   d. Сохранить assistant-сообщение: `sender_role=assistant`,
      `message_type=rag_solution`, `content` = synthesis, `code` = code
      (nullable), `rag_sources` = JSON массив Source.
   e. Если `chat.Title == "Новый чат"` — заменить первыми 80 **рунами**
      cleanQuery; обновить `chat.UpdatedAt = now` (всегда, при каждом сообщении).
   f. Ответ 201 `{"user_message": ..., "assistant_message": ...}`;
      у user_message в JSON — привязанные `tags`.
3. `GetChat`: сообщения created_at ASC; у user-сообщений подтянуть теги
   (TagStore.GetByMessage) в поле `tags`.
4. `ListChats`: limit default 20 clamp 1..100, offset >= 0; ответ
   `{"items":[...],"total":N}`; сортировка уже в репозитории (pinned, updated_at).
5. `CreateChat`: пустой title → «Новый чат»; title > 255 → 400.
   `UpdateChat`: title (если передан) 1..255 → иначе 400; is_pinned;
   обновляет updated_at. `DeleteChat` → 204.

**Тесты (фейковые ChatStore/TagStore + стаб SearchClient):**
- матрица владения: для GET/PATCH/DELETE/messages чужой чат → 404, свой → успех.
- извлечение тегов: `"[Python] TypeError: x"` → tags=["python"],
  cleanQuery="TypeError: x"; несколько скобок; merge с явными tags;
  дедупликат; `"[C++] [c++]"` → один тег; пустой после очистки → 400;
  raw_query сохраняет скобки.
- автозаголовок: «Новый чат» → первые 80 рун (проверь на длинной строке
  с unicode); уже переименованный чат не трогается; updated_at растёт.
- сообщения: content/code/rag_sources сохранены и возвращаются; тип и роль
  корректны; tags привязаны; ошибка SearchClient → 502.
- пагинация clamp; create/update/delete валидация.

**Критерии готовности:** гейты зелёные; все сценарии покрыты;
`SendMessage` больше не содержит захардкоженных mock-источников.

**Коммит:** `Добавить проверку владельца чатов и конвейер сообщений в gateway`

---

## T6. Профиль и аватары через Garage S3

**Цель:** PATCH профиля по спеке (без role!), загрузка аватара в Garage,
публичная выдача аватара прокси-роутом.

**Файлы:**
- `services/gateway/pkg/storage/s3.go` (новый)
- `services/gateway/internal/handler/user_handler.go`
- `services/gateway/internal/app/app.go` (wiring storage)
- `services/gateway/go.mod` / `go.sum` (aws-sdk-go-v2)
- тесты: `internal/handler/user_handler_test.go`, `pkg/storage/s3_test.go`

**Требования:**
1. `go get github.com/aws/aws-sdk-go-v2/config github.com/aws/aws-sdk-go-v2/credentials github.com/aws/aws-sdk-go-v2/service/s3`.
2. `pkg/storage`: интерфейс
   ```go
   type AvatarStore interface {
       Put(ctx context.Context, key, contentType string, data []byte) error
       Get(ctx context.Context, key string) (data []byte, contentType string, err error)
   }
   ```
   Реализация S3: static credentials из cfg.S3 (AccessKey/SecretKey),
   Region, endpoint из cfg.S3.Endpoint, **UsePathStyle=true** (Garage не
   поддерживает virtual-host стиль), UseSSL из cfg. Ошибка «объект не найден»
   — типизированная (например `storage.ErrNotFound`), чтобы хендлер отличал
   404 от 502. Конструктор `New(cfg config.S3Config) (*S3Store, error)`.
3. `PATCH /api/v1/users/me`: поля username/email/gender/language/profession
   (все опциональны, минимум одно). Валидация: username 1..100; email формат +
   lowercase + уникальность (GetByEmail: занят другим → 409 `email already taken`);
   gender/language — только enum-значения спеки → иначе 400; profession 1..100.
   **Если в теле есть поле `role` → 400 `role is not editable`** (декодируй
   в map или отдельную структуру с RawMessage). RBAC-роль не меняется.
   Ответ 200 — обновлённый User.
4. `POST /api/v1/users/avatar`: multipart поле `avatar`;
   `http.MaxBytesReader` 5 МиБ; тип — `http.DetectContentType(data)`, допустимы
   image/png, image/jpeg, image/webp, image/gif → иначе 400; нет поля → 400.
   Put в бакет cfg.S3.BucketName ключ `avatars/{userID}` (перезапись).
   Ошибка S3 → 502 `storage unavailable`. Сохранить
   `avatar_url = "/api/v1/users/{id}/avatar"` через UpdateProfile.
   Ответ 200 `{"avatar_url": "..."}`.
5. `GET /api/v1/users/{id}/avatar` — **публичный** (без auth middleware):
   id не uuid → 400; Get из S3; ErrNotFound → 404; иная ошибка → 502;
   успех → тело изображения, заголовки Content-Type (из S3),
   Content-Length, `Cache-Control: public, max-age=3600`, ETag
   (например, sha256 от данных или S3 ETag).
6. `GET /api/v1/users/me` — без изменений (User уже с новыми полями).

**Тесты (фейковый AvatarStore):**
- PATCH: каждое поле по отдельности и вместе; 400 на битый email/enum/пустой
  username/role в теле; 409 на занятый email; 200 на свой email (не конфликт).
- avatar POST: валидный PNG (сгенерируй минимальный PNG в тесте) → 200 +
  avatar_url + Put вызван с ключом avatars/{id}; текст → 400; >5 МиБ → 400;
  без поля → 400; ошибка Put → 502.
- avatar GET: отдаёт байты и Content-Type; 404 на ErrNotFound; 502 на иную
  ошибку; 400 на не-uuid; маршрут доступен без токена.

**Критерии готовности:** гейты зелёные; `go.mod` — только разрешённые
зависимости; реальный Garage проверяется в T8.

**Коммит:** `Добавить профиль и аватары через Garage S3 в gateway`

---

## T7. Swagger UI и выдача спеки

**Цель:** `GET /docs` — Swagger UI, `GET /openapi.yaml` — встроенная спека.

**Файлы:**
- `services/gateway/api/embed.go` (новый, package api)
- `services/gateway/internal/handler/docs_handler.go` (новый)
- `services/gateway/internal/app/app.go` (регистрация)
- тесты: `internal/handler/docs_handler_test.go`
- **`services/gateway/api/openapi.yaml` НЕ МЕНЯТЬ.**

**Требования:**
1. `api/embed.go`: `//go:embed openapi.yaml` → `var OpenAPIYAML []byte`.
2. `GET /openapi.yaml`: 200, `Content-Type: application/yaml; charset=utf-8`,
   тело — встроенный файл.
3. `GET /docs`: 200, `text/html; charset=utf-8`, самодостаточная HTML-страница:
   - CSS `https://unpkg.com/swagger-ui-dist@5.35.0/swagger-ui.css`;
   - JS `https://unpkg.com/swagger-ui-dist@5.35.0/swagger-ui-bundle.js`;
   - `SwaggerUIBundle({url: '/openapi.yaml', dom_id: '#swagger-ui',
     deepLinking: true, persistAuthorization: true})`;
   - `<title>Errorium API</title>`, тёмный фон не нужен — стандартный UI.
   Оба маршрута публичные.

**Тесты:** `/openapi.yaml` → 200 + содержит `openapi: 3.0.3` + верный
Content-Type; `/docs` → 200 + содержит `swagger-ui-dist@5.35.0` и `/openapi.yaml`;
embed непустой.

**Критерии готовности:** гейты зелёные; спека не изменена (`git diff` по
`api/openapi.yaml` пуст).

**Коммит:** `Добавить Swagger UI и выдачу OpenAPI-спеки в gateway`

---

## T8. Интеграционные тесты, документация, финальный прогон

**Цель:** доказать, что всё работает вместе на реальном PostgreSQL и dev-стеке;
зафиксировать итерацию в документации проекта.

**Файлы:**
- `services/gateway/internal/app/app_integration_test.go` (новый)
- `tests/integration/test_gateway_api.py` (новый)
- `services/gateway/README.md`
- `STATUS.md`, `CHANGE_HISTORY.md` (корень)

**Требования:**
1. Go интеграционный тест (пропускается через `t.Skip`, если не задан
   `TEST_DB_DSN`): `migrator.Up(dsn)` → репозитории → mux как в app.New →
   полный сценарий через httptest:
   register → login → me → PATCH profile (profession/gender/language) →
   create chat → send message (`"[Python] TypeError: unsupported operand"`) →
   get chat (content/code/rag_sources/tags на месте, автозаголовок) →
   patch (pin+rename) → list (search находит) → второй пользователь →
   чужой чат 404 → delete → get 404. Refresh-ротация: старый токен 401.
   Запуск: `TEST_DB_DSN='postgres://errorium_user:errorium_password@localhost:5432/errorium?sslmode=disable' go test ./internal/app/ -run Integration -v`
   (порт из `.env`, если менял).
2. Python e2e `tests/integration/test_gateway_api.py` — unittest, только
   stdlib (urllib.request, json, uuid, os). Базовый URL из env `GATEWAY_URL`
   (default `http://localhost:8000`); если недоступен — `skipTest`.
   Сценарии: health; docs/openapi.yaml; register/login/me; chats CRUD +
   message (проверка content/code/rag_sources, тегов из скобок); search
   (детерминизм: два одинаковых запроса → одинаковые результаты);
   graph/threads (категории, рёбра); graph details 404; tags (23); status.
   Каждый тест создаёт своего пользователя (uuid в email) — тесты независимы.
   Запуск: `GATEWAY_URL=http://localhost:8000 python3 -m unittest tests.integration.test_gateway_api -v`
   или `python3 -m unittest discover -s tests/integration -p "test_*.py"`.
3. Прогон dev-стека:
   ```bash
   docker compose -f docker-compose.dev.yml up --build -d
   curl -s http://localhost:8000/health
   curl -s http://localhost:8000/openapi.yaml | head -5
   GATEWAY_URL=http://localhost:8000 python3 -m unittest discover -s tests/integration -p "test_*.py"
   TEST_DB_DSN=... go test ./internal/app/ -run Integration -v   # из services/gateway
   docker compose -f docker-compose.dev.yml down
   ```
   Аватар против реального Garage проверь вручную curl-ом (multipart PNG)
   и опиши результат в CHANGE_HISTORY; если Garage недоступен в окружении —
   честно зафиксируй «не проверено», не выдумывай.
4. `services/gateway/README.md`: замени устаревший список эндпоинтов ссылкой
   на `api/openapi.yaml` + краткая таблица core-групп; добавь env
   `SEARCH_ENGINE_ADDR`, bcrypt-примечание, раздел «Стабы и граница gRPC».
5. `STATUS.md`: обнови строку Gateway в матрице готовности и «Следующий шаг»
   (реализовано ядро+стабы; дальше: gRPC-клиент к ядру, LLM/SSE, SSO,
   сброс пароля, MCP-политика, admin).
6. `CHANGE_HISTORY.md`: запись в стиле существующих (дата фактическая,
   затронутые спеки, суть по задачам T1–T7, **реальные** результаты тестов
   и прогонов; непроверенное помечай честно).

**Критерии готовности:** все гейты §1 + `make test-unit` зелёные;
интеграционные тесты пройдены на реальном стеке (или честно зафиксировано,
что недоступно); документация обновлена; `git status` без лишних файлов.

**Коммит:** `Добавить интеграционные тесты gateway и обновить документацию`

---

## 4. Что НЕ входит в пакет (не реализовывать, не спрашивать повторно)

- SSO (github/google/yandex), сброс/смена пароля, MCP-токены и `/sse`,
  SSE-стриминг, Admin API — всё это `x-status: planned` в спеке.
  Политика MCP-авторизации не урегулирована владельцем (FR-4.4 против
  docs/Tech/03-data.md §3.4) — реализация запрещена до его решения.
- gRPC-клиент к search_engine_core, реальная LLM, реальный BM25.
- Любые изменения фронтенда (замена моков на fetch — отдельная итерация
  Роли 4, контракт уже готов).
- Изменения `openapi.yaml`, compose-файлов, Makefile, protos, docs
  (кроме T8: README gateway, STATUS.md, CHANGE_HISTORY.md).

## 5. Эскалация

Остановись и спроси человека, если:
- спека противоречит коду или сама себе;
- нужный файл уже изменён кем-то другим (не перезатирй — спроси);
- гейт не чинится за 3 попытки;
- требуется зависимость вне разрешённого списка;
- окружение не даёт запустить обязательную проверку (Docker занят, порты,
  нет сети для go get) — сообщи, что именно не проверено.
