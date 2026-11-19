# Промпты для команды Gemini Flash: Gateway «Ядро + стабы»

Два режима:
- **A. Оркестратор (/teamwork)** — одна инструкция ниже, оркестратор сам
  ведёт сессии, гейты, ретраи и ревью.
- **B. Ручной (fallback)** — 8 готовых промптов, по одному на сессию.
  Копируй целиком, порядок строго T1→T8, без параллелизма.

Правила намеренно продублированы в каждом промпте: не рассчитывай, что
исполнитель прочитает их по ссылке.

---

## A. Инструкция оркестратору (/teamwork)

```
Ты оркестратор команды исполнителей (Gemini Flash) в репозитории Errorium
(ветка feat/significant-backend-update). Пакет задач: handoff/gateway-backend/TASKS.md.
Контракт API: services/gateway/api/openapi.yaml (операции x-status: core).

ГРАФ ЗАДАЧ — строго линейно:
T1 → T2 → T3 → T4 → T5 → T6 → T7 → T8
(Параллелить нельзя: T3 и T4 оба правят internal/app/app.go.
 T5 требует T2+T4; T6 требует T2+T4; T7 после T6; T8 последняя.)

ПРАВИЛА ДЛЯ КАЖДОГО ВОРКЕРА (вставляй в промпт дословно):
1. Один воркер = одна задача = один коммит. Коммит только после зелёных гейтов.
   Сообщение коммита указано в задаче; стиль — простой русский текст без
   префиксов feat:/fix:.
2. Сначала тесты (красные), потом код (зелёные). Unit-тесты: *_test.go в тех же
   пакетах, httptest + фейки интерфейсов, без БД и Docker.
3. Скоуп: только файлы задачи (+ новые тесты). Не трогать frontend, indexer,
   search_engine_core, protos/, docs/, openapi.yaml, compose-файлы, Makefile,
   STATUS.md, CHANGE_HISTORY.md (их обновляет только T8).
4. Разрешённые новые зависимости: golang.org/x/crypto, aws-sdk-go-v2
   (config/credentials/service/s3). Запрещены ORM, веб-фреймворки, любые
   поисковые библиотеки (Elasticsearch/Lucene/rank-bm25 и т.п. — академический
   запрет), swagger-генераторы. Роутинг — stdlib net/http.ServeMux.
5. Контракт openapi.yaml неизменен. Реализуются ТОЛЬКО x-status: core.
   planned (SSO, пароли по email, MCP, SSE, admin) не реализовывать.
6. Ошибки — {"error":"msg"}; чужой ресурс → 404; password_hash никогда в JSON.
7. Непонятно, противоречие, чужие незакоммиченные правки в файлах задачи,
   гейт не чинится за 3 попытки — СТОП и эскалация человеку. Не угадывать.

ГЕЙТЫ ПОСЛЕ КАЖДОЙ ЗАДАЧИ (проверяешь ты, не воркер со слов):
  cd services/gateway
  gofmt -l .            # пусто
  go build ./...
  go vet ./...
  go test ./...
  cd ../.. && python3 -m unittest discover -s tests/unit -p "test_*.py"
  git diff --stat HEAD~1  # только файлы задачи (+ go.mod/go.sum где разрешено)
  plus критерии готовности из TASKS.md §Tn.

РЕТРАИ: до 3 на задачу. Каждый ретрай — новый промпт с текстом ошибки гейта
и требованием исправить, не переписывая задачу заново. После 3 — эскалация
человеку с логом гейта и диффом. Не принимай «тесты прошли» со слов — запускай сам.

ПЕРЕДАЧА ВОРКЕРУ: промпт = правила выше + полный текст задачи Tn из TASKS.md
(копируй раздел целиком) + требование первым шагом прочитать TASKS.md §0–§2
и core-операции openapi.yaml и пересказать критерии готовности своими словами.

ФИНАЛ (T8): прогони dev-стек и интеграционные тесты сам, проверь запись
CHANGE_HISTORY на честность (реальные цифры, непроверенное помечено),
обнови STATUS.md. Отчёт человеку: список коммитов, результаты гейтов,
что не проверено и почему.
```

---

## B. Ручные промпты (по одному на сессию)

### Общая шапка (уже включена в каждый промпт ниже)

---

### Промпт T1

```
Ты backend-исполнитель в репозитории Errorium (Go-сервис services/gateway).
Выполни задачу T1 из handoff/gateway-backend/TASKS.md и закоммить результат.

ШАГ 0 (обязательно): прочитай handoff/gateway-backend/TASKS.md (разделы 0, 1, 2
и T1) и services/gateway/api/openapi.yaml. Перескажи своими словами критерии
готовности T1, затем работай.

ПРАВИЛА:
- Одна задача = один коммит, только после зелёных гейтов. Сообщение коммита:
  «Подключить хендлеры gateway к роутеру и исправить ключи контекста»
  (без префиксов feat:/fix:).
- TDD: сначала тесты (красные), потом код. Unit-тесты — httptest + фейки
  интерфейсов, без БД/Docker.
- Меняй только: internal/app/app.go, internal/handler/helpers.go,
  internal/handler/chat_handler.go, internal/handler/user_handler.go,
  pkg/config/config.go (+config_test.go), cmd/main.go, новые тесты
  internal/app/app_test.go и internal/handler/context_test.go.
  Не трогай другие сервисы, docs/, openapi.yaml, compose, Makefile,
  STATUS.md, CHANGE_HISTORY.md.
- Новые зависимости запрещены в этой задаче. Роутинг — stdlib ServeMux.
- Контракт: ошибки {"error":"msg"}; без Bearer /api/v1/chats и /api/v1/users/me
  → 401; /health и /api/v1/search публичные.
- Непонятно/противоречие/гейт не чинится за 3 попытки — остановись и спроси.

ЗАДАЧА T1:
1. Баг: в internal/middleware/auth.go и internal/handler/chat_handler.go два
   РАЗНЫХ типа contextKey со значением "user_id" — хендлеры чатов/юзеров не
   видят пользователя и всегда 401. Удали локальный contextKey/ctxKeyUserID/
   userIDFromCtx из chat_handler.go; везде используй middleware.UserIDFromCtx
   и middleware.RoleFromCtx. В helpers.go оставь respondJSON/respondError/
   parseUUIDParam/queryInt.
2. app.New(cfg): если cfg.DB.Host != "" — пул db.New(cfg.DB), репозитории
   NewUserRepository/NewSessionRepository/NewChatRepository/NewTagRepository,
   jwt.NewManager(cfg.JWT); зарегистрируй health, NewAuthHandler(users,
   sessions, jwtMgr), NewChatHandler(chatRepo, tagRepo), NewUserHandler(userRepo),
   NewSearchHandler(). ВСЕ маршруты /api/v1/chats... и /api/v1/users/me,
   POST /api/v1/users/avatar оберни в middleware.Auth(jwtMgr). Весь mux —
   в middleware.CORS(). Если DB.Host == "" — стартуй с /health и search,
   auth/chats/users не регистрируй (зафиксируй тестом). Graceful shutdown
   сохрани, добавь закрытие пула БД.
3. pkg/config: поле SearchEngineAddr = getEnv("SEARCH_ENGINE_ADDR",
   SEARCH_ENGINE_HOST+":"+SEARCH_ENGINE_PORT), дефолт search_engine_core:50051.
4. Поведение auth-хендлеров не меняй (кроме ключа контекста).

ТЕСТЫ (сначала): middleware+chat handler — с валидным JWT 200, без/мусор 401
(фейковые ChatStore/TagStore); app без БД — /health 200, /api/v1/chats 404,
CORS-заголовки есть, OPTIONS 204; config — дефолт и переопределение
SearchEngineAddr.

ГЕЙТЫ (все обязаны пройти, покажи вывод):
  cd services/gateway && gofmt -l . && go build ./... && go vet ./... && go test ./...
  (gofmt -l . должен вывести пусто)
  из корня: python3 -m unittest discover -s tests/unit -p "test_*.py"
Затем git add только своих файлов и коммит с указанным сообщением.
```

---

### Промпт T2

```
Ты backend-исполнитель в репозитории Errorium (Go-сервис services/gateway).
Выполни задачу T2 из handoff/gateway-backend/TASKS.md и закоммить результат.

ШАГ 0 (обязательно): прочитай handoff/gateway-backend/TASKS.md (разделы 0, 1, 2
и T2), migrations/000001_init_gateway_schema.up.sql и openapi.yaml (схемы User,
ChatMessage). Перескажи критерии готовности T2, затем работай.

ПРАВИЛА:
- Одна задача = один коммит после зелёных гейтов. Сообщение:
  «Добавить поля профиля и текст сообщений в схему gateway».
- TDD: сначала тесты. Unit-тесты без БД/Docker.
- Меняй только: migrations/000002_profile_and_message_content.up.sql (новый),
  .down.sql (новый), migrations/migrations_test.go, domain/user.go,
  domain/chat.go, internal/repository/user_repo.go, chat_repo.go,
  interfaces.go, новые domain/user_test.go, domain/chat_test.go.
  Не трогай другие сервисы, docs/, openapi.yaml, compose, Makefile,
  STATUS.md, CHANGE_HISTORY.md. Новые зависимости запрещены.
- Непонятно/противоречие/3 неудачных попытки фикса — остановись и спроси.

ЗАДАЧА T2:
1. Миграция 000002 up:
   ALTER TABLE users ADD COLUMN profession TEXT NOT NULL DEFAULT 'Backend developer' CHECK (length(profession) <= 100);
   ALTER TABLE users ADD COLUMN gender TEXT NOT NULL DEFAULT 'Not specified' CHECK (gender IN ('Male','Female','Not specified'));
   ALTER TABLE users ADD COLUMN language TEXT NOT NULL DEFAULT 'English' CHECK (language IN ('English','Русский'));
   ALTER TABLE chat_messages ADD COLUMN content TEXT;
   ALTER TABLE chat_messages ADD COLUMN code TEXT;
   down — DROP COLUMN всех пяти в обратном порядке. embed уже есть (*.sql).
2. domain.User += Profession/Gender/Language string (json profession/gender/
   language). domain.ChatMessage += Content *string (json:"content"),
   Code *string (json:"code") — указатели, чтобы nil давал null в JSON.
3. user_repo: Create/scanUser/UpdateProfile включают новые поля;
   UpdateProfile обновляет username, email, avatar_url, profession, gender,
   language, updated_at WHERE id. Добавь UpdatePasswordHash(ctx, id, hash)
   (UPDATE users SET password_hash=$1, updated_at=NOW() WHERE id=$2)
   и внеси его в интерфейс repository.UserStore (interfaces.go).
4. chat_repo: AddMessage/GetMessages включают content, code.

ТЕСТЫ (сначала): migrations_test — 000002 up/down встроены, непустые, up
содержит 5 ADD COLUMN, down — 5 DROP COLUMN; domain — JSON User содержит
profession/gender/language и НЕ содержит password_hash; ChatMessage с nil
Content/Code → "content":null,"code":null, с заполненными → строки.

ГЕЙТЫ (покажи вывод):
  cd services/gateway && gofmt -l . && go build ./... && go vet ./... && go test ./...
  из корня: python3 -m unittest discover -s tests/unit -p "test_*.py"
Затем коммит только своих файлов с указанным сообщением.
```

---

### Промпт T3

```
Ты backend-исполнитель в репозитории Errorium (Go-сервис services/gateway).
Выполни задачу T3 из handoff/gateway-backend/TASKS.md и закоммить результат.

ШАГ 0 (обязательно): прочитай handoff/gateway-backend/TASKS.md (разделы 0, 1, 2
и T3), internal/handler/auth_handler.go, password.go, pkg/jwt/jwt.go и
openapi.yaml (auth-операции core, схемы RegisterRequest/LoginRequest/TokenPair).
Перескажи критерии готовности T3, затем работай.

ПРАВИЛА:
- Одна задача = один коммит после зелёных гейтов. Сообщение:
  «Перевести пароли gateway на bcrypt и добавить remember_me».
- TDD: сначала тесты (фейковые UserStore/SessionStore, httptest, без БД).
- Меняй только: internal/handler/password.go, auth_handler.go,
  internal/repository/interfaces.go, user_repo.go (если UpdatePasswordHash
  ещё нет), internal/app/app.go (конструктор AuthHandler), go.mod/go.sum,
  новые password_test.go, auth_handler_test.go.
  Не трогай другие сервисы, docs/, openapi.yaml, compose, Makefile,
  STATUS.md, CHANGE_HISTORY.md.
- Разрешена зависимость: golang.org/x/crypto (bcrypt). Другие — запрещены.
- Контракт: ошибки {"error":"msg"}; password_hash никогда в JSON;
  неверный email и неверный пароль → одинаковый 401 "invalid credentials".
- Непонятно/противоречие/3 неудачных попытки — остановись и спроси.

ЗАДАЧА T3:
1. hashPassword → bcrypt cost 12. checkPassword: stored начинается с "$2" →
   bcrypt.CompareHashPassword; иначе legacy "<32hex salt>:<64hex sha256>" →
   constant-time сравнение (crypto/subtle). Добавь needsRehash(stored) bool
   (true для legacy).
2. login: при успешном входе с legacy-хэшем — перехэшируй bcrypt и сохрани
   через users.UpdatePasswordHash (ошибка сохранения не ломает вход,
   логируется). is_active=false → 403 "account is disabled".
3. LoginRequest += RememberMe bool (json remember_me). TTL refresh-сессии:
   true → cfg.JWT.RefreshTokenExpireDays суток (30), false → 24 часа.
   NewAuthHandler принимает refreshExpDays int; app.go передаёт из cfg.
4. register: username 1..100 (trim), email формат+lowercase, password 8..128;
   дефолты профиля Profession="Backend developer", Gender="Not specified",
   Language="English"; duplicate → 409 "email already registered";
   ответ 201 TokenPair {access_token, refresh_token, user}.
5. refresh: ротация сохранена (старая сессия отозвана, новая создана);
   отозванный/истёкший/неизвестный → 401. logout → 204. me → User/404.

ТЕСТЫ (сначала): password — bcrypt roundtrip, legacy verify, needsRehash,
неверный пароль, битый формат. register — 201+токены+дефолты, 400 (короткий
пароль/пустой username/битый email), 409. login — 200; 401 одинаковый для
неверного пароля и неизвестного email; 403 неактивный; legacy → вход +
UpdatePasswordHash вызван с bcrypt; remember_me TTL ~30д/~24ч (±1 мин).
refresh — ротация и 401-кейсы. logout 204 с токеном и без; me 200/401/404.

ГЕЙТЫ (покажи вывод):
  cd services/gateway && gofmt -l . && go build ./... && go vet ./... && go test ./...
  из корня: python3 -m unittest discover -s tests/unit -p "test_*.py"
Затем коммит только своих файлов с указанным сообщением.
```

---

### Промпт T4

```
Ты backend-исполнитель в репозитории Errorium (Go-сервис services/gateway).
Выполни задачу T4 из handoff/gateway-backend/TASKS.md и закоммить результат.

ШАГ 0 (обязательно): прочитай handoff/gateway-backend/TASKS.md (разделы 0, 1, 2
и T4), internal/handler/search_handler.go и openapi.yaml (search, graph, tags,
status; схемы Source/SearchResponse/RagResponse/ThreadNode/ThreadEdge/
ThreadGraph/ThreadDetails/StatusResponse). Перескажи критерии готовности T4.

ПРАВИЛА:
- Одна задача = один коммит после зелёных гейтов. Сообщение:
  «Добавить стабы поиска, графа, тегов и статуса в gateway».
- TDD: сначала тесты, без БД/Docker.
- Меняй только: internal/search/client.go, stub.go (новые),
  internal/graph/client.go, stub.go (новые), internal/handler/search_handler.go
  (переписать), tags_handler.go, status_handler.go (новые),
  internal/app/app.go (wiring), новые тесты к ним.
  Не трогай другие сервисы, docs/, openapi.yaml, compose, Makefile,
  STATUS.md, CHANGE_HISTORY.md. Новые зависимости запрещены.
- АКАДЕМИЧЕСКИЙ ЗАПРЕТ: никаких поисковых библиотек и настоящего BM25.
  Стаб ранжирует детерминированным пересечением слов — это заглушка, не поиск.
- Хендлеры зависят только от интерфейсов SearchClient/GraphClient;
  конкретные стабы подключаются в app.go. planned-эндпоинты не реализовывать.
- Непонятно/противоречие/3 неудачных попытки — остановись и спроси.

ЗАДАЧА T4:
1. Интерфейсы: SearchClient{Search(ctx, SearchRequest) (*SearchResponse, error);
   RAG(ctx, SearchRequest) (*RAGResult, error)}, GraphClient{GetThreadGraph(ctx,
   []int64) (*ThreadGraph, error); GetThreadDetails(ctx, int64) (*ThreadDetails,
   error)}. SearchRequest{Query, TopK, Tags, Language}. DTO — строго по спеке
   (snake_case json). Source: doc_id int64, title, url, snippet, tags[],
   question_score int (голоса SO), bm25_score float, is_accepted bool.
2. Стаб поиска: >=10 фикстур реальных SO-тредов про ошибки (придумай сам,
   doc_id — настоящие id, url https://stackoverflow.com/questions/{id},
   у каждого теги, голоса, тело ответа с кодом). Скоринг: lowercase-слова
   запроса (split по не-буквам/цифрам) против title+tags+snippet;
   bm25_score = совпадения + 0.1*question_score/1000; сортировка DESC;
   обрезка top_k; фильтр tags (хотя бы один общий, lowercase); без rand/time
   в скоринге — одинаковый запрос даёт идентичный ответ. total_hits — число
   ненулевых; took_ms — замер.
3. Стаб RAG: synthesis — шаблон 2–4 предложения про запрос и лучший источник;
   code — блок кода лучшего треда; sources — результаты Search.
4. Стаб графа: пул >=12 связанных тредов с answer-текстом. GetThreadGraph:
   первый запрошенный id → PRIMARY_RESULT, остальные OTHER_RESULT (из фикстур
   поиска, неизвестные пропускать); связанные ранжируются пересечением тегов,
   до 12, первые 3 DUPLICATE, прочие LINKED, без дублей doc_id; рёбра
   OTHER→PRIMARY LINKED_IN_ANSWER, DUPLICATE→PRIMARY DUPLICATE_OF,
   LINKED→PRIMARY/OTHER LINKED_IN_ANSWER, без петель; узлы содержат
   best_answer_body и original_url. GetThreadDetails: фикстура или not found.
5. GET /api/v1/tags: константа, ровно 23 в порядке: Go, Python, Java, C#, C++,
   JS, SQL, Rust, TypeScript, Docker, Linux, Kubernetes, PostgreSQL, Redis,
   GraphQL, React, Vue, Swift, Kotlin, PHP, Git, Nginx, gRPC.
   Ответ {"items":[{"name":"Go"},...]}.
6. GET /api/v1/status: {"gateway":{"status":"ok","version":"1.0.0"},
   "search_core":{"connected":false,"address":"<cfg.SearchEngineAddr>"},
   "rag":{"enabled":false,"provider":"stub"}}.
7. Хендлеры публичные: POST /api/v1/search и /search/rag — query 1..8000
   (trim) иначе 400, top_k default 10 clamp 1..50; GET /api/v1/graph/threads
   — doc_ids через запятую, 1..20 int64 иначе 400; GET /api/v1/graph/threads/
   {doc_id} — 200/404.

ТЕСТЫ (сначала): детерминизм (два вызова → идентично), сортировка DESC,
top_k, фильтр tags; граф — категории, 3 дубликата, рёбра без петель/дублей,
неизвестные id, details not found; хендлеры — 400-кейсы, обязательные поля
ответов, tags ровно 23 и порядок, status address из конфига.

ГЕЙТЫ (покажи вывод):
  cd services/gateway && gofmt -l . && go build ./... && go vet ./... && go test ./...
  из корня: python3 -m unittest discover -s tests/unit -p "test_*.py"
Затем коммит только своих файлов с указанным сообщением.
```

---

### Промпт T5

```
Ты backend-исполнитель в репозитории Errorium (Go-сервис services/gateway).
Выполни задачу T5 из handoff/gateway-backend/TASKS.md и закоммить результат.
Предусловие: T1–T4 уже влиты (проверь git log).

ШАГ 0 (обязательно): прочитай handoff/gateway-backend/TASKS.md (разделы 0, 1, 2
и T5), internal/handler/chat_handler.go, internal/repository/chat_repo.go,
internal/search/client.go и openapi.yaml (chats-операции, схемы Chat/
ChatMessage/SendMessageRequest/Source). Перескажи критерии готовности T5.

ПРАВИЛА:
- Одна задача = один коммит после зелёных гейтов. Сообщение:
  «Добавить проверку владельца чатов и конвейер сообщений в gateway».
- TDD: сначала тесты (фейковые ChatStore/TagStore + стаб SearchClient, без БД).
- Меняй только: internal/handler/chat_handler.go,
  internal/repository/chat_repo.go (и интерфейс ChatStore),
  internal/app/app.go (передача SearchClient), новый chat_handler_test.go.
  Не трогай другие сервисы, docs/, openapi.yaml, compose, Makefile,
  STATUS.md, CHANGE_HISTORY.md. Новые зависимости запрещены.
- Безопасность: чужой чат и несуществующий → ОДИНАКОВЫЙ 404 "chat not found"
  (существование не раскрывать). Все chat-маршруты за middleware.Auth.
- Непонятно/противоречие/3 неудачных попытки — остановись и спроси.

ЗАДАЧА T5:
1. ChatStore += GetByIDForUser(ctx, id, userID) (SELECT WHERE id AND user_id;
   нет строки → not found). GET/PATCH/DELETE /chats/{id} и POST messages
   используют его; не-владелец → 404.
2. SendMessage: (a) теги regex \[([^\[\]]{1,50})\] из query + поле tags →
   lowercase, дедуп, ≤50 симв; cleanQuery — без скобок, пробелы схлопнуты;
   пустой → 400. (b) user-сообщение: error_report, content = исходный query
   со скобками, error_context {"raw_query": исходный, "tags": merged};
   AttachToMessage. (c) SearchClient.Search(cleanQuery, topK default 5
   clamp 1..20, tags) + RAG; ошибка клиента → 502 "search backend unavailable"
   (user-сообщение уже сохранено — допустимо, зафиксируй тестом).
   (d) assistant-сообщение: rag_solution, content=synthesis, code=code,
   rag_sources=JSON массив Source. (e) если title=="Новый чат" → первые
   80 РУН cleanQuery; chats.updated_at=now всегда. (f) 201
   {"user_message":..., "assistant_message":...}, у user_message — tags.
3. GetChat: сообщения ASC, у user-сообщений теги через TagStore.GetByMessage.
4. ListChats: limit default 20 clamp 1..100, offset>=0, {"items":[],"total":N}.
5. CreateChat: пустой title → «Новый чат», >255 → 400. UpdateChat: title
   1..255 иначе 400, is_pinned, updated_at. DeleteChat → 204.
   Захардкоженные mock-источники из SendMessage удалить полностью.

ТЕСТЫ (сначала): матрица владения (GET/PATCH/DELETE/messages: чужой 404,
свой успех); теги — "[Python] TypeError: x" → ["python"] + clean "TypeError: x",
несколько скобок, merge, дедуп, "[C++] [c++]" → один, пустой cleanQuery 400,
raw_query со скобками; автозаголовок — 80 рун unicode, переименованный не
трогается, updated_at растёт; content/code/rag_sources/tags сохранены;
ошибка SearchClient → 502; clamp пагинации; валидация create/update.

ГЕЙТЫ (покажи вывод):
  cd services/gateway && gofmt -l . && go build ./... && go vet ./... && go test ./...
  из корня: python3 -m unittest discover -s tests/unit -p "test_*.py"
Затем коммит только своих файлов с указанным сообщением.
```

---

### Промпт T6

```
Ты backend-исполнитель в репозитории Errorium (Go-сервис services/gateway).
Выполни задачу T6 из handoff/gateway-backend/TASKS.md и закоммить результат.
Предусловие: T1–T5 уже влиты (проверь git log).

ШАГ 0 (обязательно): прочитай handoff/gateway-backend/TASKS.md (разделы 0, 1, 2
и T6), internal/handler/user_handler.go, pkg/config/config.go (S3Config) и
openapi.yaml (users-операции, схемы User/UpdateMeRequest). Перескажи критерии
готовности T6.

ПРАВИЛА:
- Одна задача = один коммит после зелёных гейтов. Сообщение:
  «Добавить профиль и аватары через Garage S3 в gateway».
- TDD: сначала тесты (фейковый AvatarStore, без реального S3/БД).
- Меняй только: pkg/storage/s3.go (новый), internal/handler/user_handler.go,
  internal/app/app.go (wiring), go.mod/go.sum, новые user_handler_test.go,
  pkg/storage/s3_test.go.
  Не трогай другие сервисы, docs/, openapi.yaml, compose, Makefile,
  STATUS.md, CHANGE_HISTORY.md.
- Разрешённые зависимости: github.com/aws/aws-sdk-go-v2 (config, credentials,
  service/s3). Другие — запрещены.
- БЕЗОПАСНОСТЬ: поле role в PATCH /users/me → 400 "role is not editable"
  (самоназначение admin запрещено). GET аватара — публичный (img не шлёт
  Bearer). Секреты S3 не логируются.
- Непонятно/противоречие/3 неудачных попытки — остановись и спроси.

ЗАДАЧА T6:
1. pkg/storage: интерфейс AvatarStore{Put(ctx,key,contentType,data) error;
   Get(ctx,key) ([]byte,string,error)}; типизированная ErrNotFound.
   S3Store: static credentials из cfg.S3, Region, endpoint cfg.S3.Endpoint,
   UsePathStyle=true (обязательно для Garage), UseSSL из cfg.
   New(cfg config.S3Config) (*S3Store, error).
2. PATCH /api/v1/users/me: username/email/gender/language/profession, все
   опциональны, минимум одно (иначе 400). username 1..100; email формат+
   lowercase+уникальность (занят другим → 409 "email already taken", свой —
   ок); gender ∈ Male|Female|Not specified; language ∈ English|Русский;
   profession 1..100; role в теле → 400. Ответ 200 — User.
3. POST /api/v1/users/avatar: multipart "avatar", MaxBytesReader 5 МиБ,
   http.DetectContentType → только image/png|jpeg|webp|gif иначе 400;
   нет поля → 400; Put бакет cfg.S3.BucketName ключ "avatars/{userID}";
   ошибка S3 → 502 "storage unavailable"; avatar_url =
   "/api/v1/users/{id}/avatar" сохранить UpdateProfile; ответ 200
   {"avatar_url":...}.
4. GET /api/v1/users/{id}/avatar — БЕЗ auth: не-uuid → 400; ErrNotFound →
   404; иная ошибка S3 → 502; успех → байты, Content-Type из S3,
   Content-Length, Cache-Control "public, max-age=3600", ETag.
5. GET /users/me — без изменений.

ТЕСТЫ (сначала): PATCH — каждое поле, 400 (битый email/enum/пустой username/
role/пустое тело), 409 чужой email, 200 свой email; avatar POST — минимальный
валидный PNG (сгенерируй в тесте) → 200+url+Put с ключом avatars/{id},
текст 400, >5МиБ 400, без поля 400, ошибка Put 502; avatar GET — байты+
Content-Type, 404, 502, 400 не-uuid, доступ без токена.

ГЕЙТЫ (покажи вывод):
  cd services/gateway && gofmt -l . && go build ./... && go vet ./... && go test ./...
  из корня: python3 -m unittest discover -s tests/unit -p "test_*.py"
Затем коммит только своих файлов с указанным сообщением.
```

---

### Промпт T7

```
Ты backend-исполнитель в репозитории Errorium (Go-сервис services/gateway).
Выполни задачу T7 из handoff/gateway-backend/TASKS.md и закоммить результат.
Предусловие: T1–T6 уже влиты (проверь git log).

ШАГ 0 (обязательно): прочитай handoff/gateway-backend/TASKS.md (разделы 0, 1, 2
и T7) и services/gateway/api/openapi.yaml (первые 30 строк). Перескажи
критерии готовности T7.

ПРАВИЛА:
- Одна задача = один коммит после зелёных гейтов. Сообщение:
  «Добавить Swagger UI и выдачу OpenAPI-спеки в gateway».
- TDD: сначала тесты.
- Меняй только: api/embed.go (новый), internal/handler/docs_handler.go
  (новый), internal/app/app.go (регистрация), новый docs_handler_test.go.
  api/openapi.yaml НЕ МЕНЯТЬ (git diff по нему обязан быть пуст).
  Не трогай другие сервисы, docs/, compose, Makefile, STATUS.md,
  CHANGE_HISTORY.md. Новые зависимости запрещены (Swagger UI — с CDN).
- Непонятно/противоречие/3 неудачных попытки — остановись и спроси.

ЗАДАЧА T7:
1. api/embed.go (package api): //go:embed openapi.yaml → var OpenAPIYAML []byte.
2. GET /openapi.yaml: 200, Content-Type "application/yaml; charset=utf-8",
   тело — встроенный файл. Публичный.
3. GET /docs: 200, text/html; charset=utf-8, HTML-страница:
   <title>Errorium API</title>,
   CSS https://unpkg.com/swagger-ui-dist@5.35.0/swagger-ui.css,
   JS https://unpkg.com/swagger-ui-dist@5.35.0/swagger-ui-bundle.js,
   SwaggerUIBundle({url:'/openapi.yaml', dom_id:'#swagger-ui',
   deepLinking:true, persistAuthorization:true}). Публичный.

ТЕСТЫ (сначала): /openapi.yaml → 200 + содержит "openapi: 3.0.3" + верный
Content-Type; /docs → 200 + содержит "swagger-ui-dist@5.35.0" и "/openapi.yaml";
embed непустой.

ГЕЙТЫ (покажи вывод):
  cd services/gateway && gofmt -l . && go build ./... && go vet ./... && go test ./...
  git diff --stat -- api/openapi.yaml   # пусто
  из корня: python3 -m unittest discover -s tests/unit -p "test_*.py"
Затем коммит только своих файлов с указанным сообщением.
```

---

### Промпт T8

```
Ты backend-исполнитель в репозитории Errorium. Выполни задачу T8 из
handoff/gateway-backend/TASKS.md и закоммить результат.
Предусловие: T1–T7 уже влиты (проверь git log).

ШАГ 0 (обязательно): прочитай handoff/gateway-backend/TASKS.md (разделы 0, 1, 2
и T8), services/gateway/README.md, STATUS.md, CHANGE_HISTORY.md (последние
2 записи — стиль). Перескажи критерии готовности T8.

ПРАВИЛА:
- Одна задача = один коммит после зелёных гейтов. Сообщение:
  «Добавить интеграционные тесты gateway и обновить документацию».
- Меняй только: services/gateway/internal/app/app_integration_test.go (новый),
  tests/integration/test_gateway_api.py (новый), services/gateway/README.md,
  STATUS.md, CHANGE_HISTORY.md. Код сервисов НЕ менять (нашёл баг — стоп,
  спроси). Не трогай openapi.yaml, compose, Makefile, docs/.
- ЧЕСТНОСТЬ: в CHANGE_HISTORY только реальные результаты прогонов.
  Что не удалось проверить (нет Docker/порты/сеть) — помечай «не проверено»
  с причиной. Не выдумывай цифры.
- Python — только stdlib (unittest, urllib.request, json, uuid, os).
- Непонятно/противоречие/3 неудачных попытки — остановись и спроси.

ЗАДАЧА T8:
1. Go интеграционный тест (t.Skip без env TEST_DB_DSN): migrator.Up(dsn) →
   репозитории → mux как в app.New → httptest-сценарий: register → login →
   me → PATCH profile (profession/gender/language) → create chat → send
   message "[Python] TypeError: unsupported operand" → get chat (content/
   code/rag_sources/tags, автозаголовок) → patch pin+rename → list search →
   второй пользователь → чужой чат 404 → delete → get 404; refresh-ротация
   (старый токен 401).
2. tests/integration/test_gateway_api.py: unittest+urllib, GATEWAY_URL
   (default http://localhost:8000), недоступен → skipTest. Каждый тест —
   свой пользователь (uuid в email). Сценарии: health; docs/openapi.yaml;
   register/login/me; chats CRUD+message (content/code/rag_sources, теги
   из скобок); search детерминизм (два одинаковых запроса → идентичные
   результаты); graph/threads (категории, рёбра); details 404; tags 23; status.
3. Прогони (покажи вывод каждого):
   docker compose -f docker-compose.dev.yml up --build -d
   curl -s http://localhost:8000/health
   curl -s http://localhost:8000/openapi.yaml | head -5
   GATEWAY_URL=http://localhost:8000 python3 -m unittest discover -s tests/integration -p "test_*.py"
   cd services/gateway && TEST_DB_DSN='postgres://errorium_user:errorium_password@localhost:5432/errorium?sslmode=disable' go test ./internal/app/ -run Integration -v
   аватар против реального Garage: curl multipart с PNG (создай тестовый),
   затем GET avatar_url; результат опиши честно.
   docker compose -f docker-compose.dev.yml down
   (порты занятые — меняй в .env: POSTGRES_PORT/GARAGE_PORT/GATEWAY_PORT)
4. services/gateway/README.md: эндпоинты → ссылка на api/openapi.yaml +
   краткая таблица core-групп; env SEARCH_ENGINE_ADDR; bcrypt; раздел
   «Стабы и граница gRPC».
5. STATUS.md: строка Gateway в матрице + «Следующий шаг» (gRPC-клиент,
   LLM/SSE, SSO, сброс пароля, MCP-политика, admin).
6. CHANGE_HISTORY.md: запись в стиле существующих — дата, затронутые спеки
   (docs/Tech/04-api-and-access.md, 03-data.md, ADR-010, openapi.yaml),
   суть T1–T7, реальные результаты тестов/прогонов.

ГЕЙТЫ (покажи вывод):
  cd services/gateway && gofmt -l . && go build ./... && go vet ./... && go test ./...
  из корня: make test-unit
  интеграционные прогоны из п.3 (или честная пометка недоступности)
Затем коммит только своих файлов с указанным сообщением.
```

---

## Чек-лист приёмки пакета (для человека)

- [ ] 8 коммитов, сообщения — из задач, без conventional-префиксов.
- [ ] `git diff` каждого коммита — только файлы своей задачи.
- [ ] `openapi.yaml` не изменён ни одним коммитом.
- [ ] `planned`-эндпоинты не реализованы (SSO/password/MCP/SSE/admin отсутствуют).
- [ ] Гейты зелёные на каждом коммите (оркестратор проверял сам, не со слов).
- [ ] CHANGE_HISTORY содержит реальные цифры; непроверенное помечено.
- [ ] Поиск/граф — за интерфейсами, без готовых поисковых библиотек в go.mod.
