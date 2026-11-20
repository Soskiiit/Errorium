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

> [!NOTE]
> **Продакшн-фронтенд:** в `docker-compose.yml` nginx слушает порты `80`/`443`, они переопределяются переменными `FRONTEND_HTTP_PORT` / `FRONTEND_HTTPS_PORT`. Переменная `FRONTEND_PORT=3000` действует только в dev-стеке (`docker-compose.dev.yml`).

> [!TIP]
> **Конфликт портов на хост-машине:**  
> Если порт (например, `3900` или `5432`) уже занят другим процессом на рабочей машине, переопределите его в файле `.env` (`GARAGE_PORT=3910`, `POSTGRES_PORT=5433`). Внутри Docker-сети сервисы продолжат общаться по внутренним портам по умолчанию, конфликты на хосте будут устранены.

---

## 3. Переменные окружения (`.env`)

Создайте файл `.env` в корне репозитория (или используйте `.env.example`).

**Как переменные попадают в контейнеры:**
- `gateway` получает весь `.env` через `env_file` (ключи SSO, LLM, SMTP, JWT, лимиты MCP); явные записи `environment:` в compose имеют приоритет над `env_file`.
- `postgres`, `garage`, `search_engine_core` получают только свои переменные через подстановку `${...}` в compose.
- Credentials БД задаются переменными `POSTGRES_DB` / `POSTGRES_USER` / `POSTGRES_PASSWORD` (их читает контейнер postgres); compose пробрасывает их в gateway как `DB_NAME` / `DB_USER` / `DB_PASSWORD`. Переменные `DB_*` в `.env` действуют только при запуске gateway вне Docker.

```bash
# === Общие настройки системы ===
# Зарезервировано: текущие сервисы не читают эти переменные
# ENVIRONMENT=development
# LOG_LEVEL=INFO

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
FRONTEND_HTTP_PORT=80                       # Продакшн-порт HTTP (docker-compose.yml)
FRONTEND_HTTPS_PORT=443                     # Продакшн-порт HTTPS (docker-compose.yml)
GATEWAY_API_URL=http://localhost:8000       # Только для `make dev` вне Docker; в Docker nginx проксирует /api/ сам

# === База данных Gateway (PostgreSQL 18+) ===
POSTGRES_PORT=5432
POSTGRES_DB=errorium                        # Основные credentials для контейнера postgres
POSTGRES_USER=errorium_user
POSTGRES_PASSWORD=errorium_password
DB_HOST=postgres                            # DB_* — только для запуска gateway вне Docker
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
# JWT_ALGORITHM не читается: gateway всегда подписывает HS256 (pkg/jwt)
ACCESS_TOKEN_EXPIRE_MINUTES=60
REFRESH_TOKEN_EXPIRE_DAYS=30

# OAuth2 / SSO провайдеры (GitHub / Google / Yandex)
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
YANDEX_CLIENT_ID=your_yandex_client_id
YANDEX_CLIENT_SECRET=your_yandex_client_secret
SSO_REDIRECT_BASE=http://localhost:8000
FRONTEND_URL=http://localhost:3000

# Отправка email (сброс пароля)
SMTP_HOST=                                   # Пусто -> dev-режим (LogMailer в stdout)
SMTP_PORT=587                                # 587 (STARTTLS) или 465 (SSL/TLS)
SMTP_USER=
SMTP_PASSWORD=
SMTP_FROM=no-reply@errorium.ru

# Стартовая учетная запись администратора (bootstrap)
FIRST_ADMIN_EMAIL=admin@errorium.ru
FIRST_ADMIN_PASSWORD=AdminSecurePassword2026!

# === RAG-пайплайн и LLM Провайдер ===
LLM_PROVIDER=openai                          # openai | gigachat | stub
LLM_TEMPERATURE=0.2
LLM_MAX_TOKENS=1500
LLM_MAX_CONCURRENT=8                         # Глобальный семафор одновременных запросов к LLM

# OpenAI-совместимый провайдер (OpenAI, vLLM, Ollama)
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_API_KEY=sk-...
OPENAI_MODEL_NAME=gpt-4o-mini

# GigaChat провайдер
GIGACHAT_CLIENT_ID=
GIGACHAT_CLIENT_SECRET=
GIGACHAT_SCOPE=GIGACHAT_API_PERS
GIGACHAT_CA_CERT=/app/certs/russian_trusted_root_ca.crt  # путь в контейнере; compose монтирует ./deploy/certs -> /app/certs

# === Лимиты открытого MCP-сервера ===
MCP_MAX_SESSIONS_PER_IP=100                  # Максимум одновременных сессий с одного IP (429)
MCP_MAX_SESSIONS=200                         # Максимум одновременных сессий глобально (503)
MCP_CALLS_PER_MIN=60                         # Максимум tools/call в минуту на сессию (429)
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

---

## 5. Настройка компонентов и интеграций волны B

### 5.1. Настройка SSO-провайдеров (GitHub, Google, Яндекс)

Сервис Gateway поддерживает бесшовный вход через внешних провайдеров идентификации по протоколам OAuth2 и OIDC. Вся логика реализована на стандартной библиотеке Go (`net/http`) без сторонних SDK.

1. **Регистрация OAuth-приложений:**
   - **GitHub:** `Settings` $\to$ `Developer settings` $\to$ `OAuth Apps` $\to$ `New OAuth App`.
     - *Authorization callback URL:* `http://localhost:8000/api/v1/auth/sso/callback/github` (или значение переменной `{SSO_REDIRECT_BASE}/api/v1/auth/sso/callback/github`).
   - **Google:** `Google Cloud Console` $\to$ `APIs & Services` $\to$ `Credentials` $\to$ `OAuth 2.0 Client IDs`.
     - *Authorized redirect URI:* `http://localhost:8000/api/v1/auth/sso/callback/google`.
   - **Яндекс:** `Яндекс OAuth` $\to$ `Создать приложение` $\to$ `Веб-сервисы`.
     - *Redirect URI:* `http://localhost:8000/api/v1/auth/sso/callback/yandex`.
2. **Переменные окружения:**
   Внесите полученные `Client ID` и `Client Secret` в файл `.env` (`GITHUB_CLIENT_ID`, `GOOGLE_CLIENT_ID`, `YANDEX_CLIENT_ID` и соответствующие секреты).
3. **Поведение при входе:**
   - Фронтенд перенаправляет пользователя на `/api/v1/auth/sso/login/{provider}`.
   - Gateway генерирует криптографический одноразовый параметр `state`, сохраняет его в таблице `sso_states` (срок жизни 10 мин) и выполняет редирект 302 на страницу авторизации провайдера.
   - После успешного входа провайдер возвращает пользователя на callback-эндпоинт шлюза. Шлюз обменивает авторизационный код на access токен, запрашивает профиль пользователя (UserInfo), создает или находит пользователя в БД и связывает его в таблице `user_sso_accounts`.
   - Шлюз перенаправляет браузер на фронтенд:
     `{FRONTEND_URL}/auth/callback#access_token=<JWT>&refresh_token=<JWT>`

### 5.2. Настройка почтового отправителя (SMTP и dev-режим LogMailer)

Для отправки писем со ссылками восстановления доступа используется абстракция `mailer.Mailer`:

1. **Режим разработки (по умолчанию):**
   - Если переменная `SMTP_HOST` не заполнена (или пуста), Gateway автоматически активирует `LogMailer`.
   - При запросе `POST /api/v1/auth/password/forgot` письмо не отправляется по сети, а ссылка сброса выводится в стандартный поток вывода Gateway:
     ```text
     [DEV MAILER] password reset link for user@example.com: http://localhost:3000/reset-password?token=...
     ```
   - Это позволяет тестировать весь сценарий восстановления пароля локально без реального почтового сервера.
2. **Продакшн-режим (SMTP):**
   - Задайте переменные:
     ```bash
     SMTP_HOST=smtp.yandex.ru
     SMTP_PORT=465                  # 465 для прямого TLS или 587 для STARTTLS
     SMTP_USER=no-reply@errorium.ru
     SMTP_PASSWORD=your_app_password
     SMTP_FROM=no-reply@errorium.ru
     ```
   - Шлюз поддерживает как прямое шифрование SSL/TLS (порт 465), так и протокол STARTTLS (порт 587) с аутентификацией `PLAIN`.
   - Ссылка сброса пароля формируется на основе `FRONTEND_URL` и одноразового токена (срок действия 1 час). При успешном сбросе или смене пароля все существующие сессии пользователя немедленно отзываются.

### 5.3. Сертификат НУЦ Минцифры для GigaChat

Для подключения к сервисам Сбера (GigaChat API) по протоколу HTTPS используется корневой сертификат Национального удостоверяющего центра (НУЦ) Минцифры России.

> [!CAUTION]
> Отключение проверки TLS (`InsecureSkipVerify: true`) в кодовой базе проекта категорически запрещено требованиями информационной безопасности!

1. **Публичный сертификат в репозитории:**
   Корневой сертификат НУЦ (публичный, не секрет) лежит в `deploy/certs/russian_trusted_root_ca.crt` и доставляется на сервер автодеплоем вместе с папкой `deploy/`. Скачать/обновить его можно с официального портала Госуслуг (PEM, срок действия до 27.02.2032):
   ```bash
   curl -o deploy/certs/russian_trusted_root_ca.crt https://gu-st.ru/content/lending/russian_trusted_root_ca_pem.crt
   ```
   Прежний путь `https://gu-st.ru/content/Other/doc/russian_trusted_root_ca.crt` с 2026 года отдаёт 404 — не использовать.
2. **Конфигурация:**
   - Каталог `deploy/certs/` монтируется в контейнер gateway как `/app/certs` (read-only) в обоих compose-файлах. Путь `/app/certs/russian_trusted_root_ca.crt` — **дефолт в `pkg/config` gateway**, отдельная переменная не обязательна; переопределение через `GIGACHAT_CA_CERT` (путь **внутри контейнера**):
     ```bash
     GIGACHAT_CA_CERT=/app/certs/russian_trusted_root_ca.crt
     ```
   - При инициализации клиента `llm.NewGigaChat` Gateway загружает этот сертификат в системный пул `x509.CertPool` и безопасно валидирует TLS-соединение как к OAuth-шлюзу (`ngw.devices.sberbank.ru:9443`), так и к API чата (`api.giga.chat`).
   - Если сертификат не найден по указанному пути, провайдер `gigachat` завершает инициализацию с понятной ошибкой в логе и не выполняет небезопасных соединений.

### 5.4. Bootstrap первого администратора (FIRST_ADMIN)

Для начального администрирования сервиса без необходимости ручных вставок в БД предусмотрен механизм автоматической инициализации:

1. **Параметры запуска:**
   ```bash
   FIRST_ADMIN_EMAIL=admin@errorium.ru
   FIRST_ADMIN_PASSWORD=AdminSecurePassword2026!
   ```
2. **Принцип работы (`app.BootstrapFirstAdmin`):**
   - При старте сервиса шлюз проверяет наличие пользователя с указанным `FIRST_ADMIN_EMAIL`.
   - Если пользователь не найден, создается учетная запись с ролью `RoleAdmin` (`admin`), заполненным именем и паролем, захэшированным с помощью `bcrypt` (cost 12).
   - Если учетная запись уже существует, операция завершается успешно без изменения существующего хэша пароля или данных пользователя.
   - Механизм строго идемпотентен и безопасен при горизонтальном масштабировании контейнеров Gateway.

### 5.5. Настройка лимитов открытого MCP-сервера

В соответствии с архитектурным решением **ADR-012**, сервер Remote MCP (`/sse` и `/mcp/messages`) работает в открытом режиме без персональных токенов. Для предотвращения перегрузки и атак отказа в обслуживании на защите НИРС действуют in-memory лимиты:

```bash
MCP_MAX_SESSIONS_PER_IP=100    # Лимит сессий с одного IP (X-Real-IP от nginx / RemoteAddr)
MCP_MAX_SESSIONS=200           # Глобальный лимит одновременных SSE-соединений
MCP_CALLS_PER_MIN=60           # Лимит вызовов tools/call в минуту на одну сессию
LLM_MAX_CONCURRENT=8           # Глобальный семафор одновременных обращений к LLM
```

- **Специфика бауманского контура:** лимиты наложены на количество одновременных сессий и частоту вызовов внутри сессии, а не на число HTTP-запросов с IP. Это гарантирует, что аудитория за общим NAT во время защиты проекта не будет заблокирована.
- **Освобождение ресурсов:** при закрытии клиентом SSE-стрима (или разрыве сетевого соединения) счетчик активных сессий немедленно декрементируется, освобождая слот для новых подключений.

### 5.6. Верификация Web Frontend и сквозное E2E тестирование (КТ3)

Фронтенд выполнен в архитектуре Single Page Application (SPA) на чистом Vanilla JS (HTML5, CSS3, Canvas 2D) без внешних фреймворков и библиотек визуализации (D3/React Flow исключены по ADR-006):

1. **Локальная проверка синтаксиса:**
   ```bash
   node --check services/frontend/static/app.js
   node --check services/frontend/static/api.js
   ```
2. **Модульное и статическое тестирование фронтенда:**
   ```bash
   python3 -m unittest discover -s tests/unit -p "test_*.py"
   ```
   Включает проверку отсутствия сторонних библиотек, целостности CSS-модулей, клиентской логики авторизации (`test_frontend_api.py`), чатов (`test_frontend_chats.py`), профиля и аватара (`test_frontend_profile.py`), открытого MCP без токенов (`test_frontend_mcp.py`) и визуализации Star Map на Canvas (`test_star_map.py`).
3. **Сквозное E2E тестирование (`tests/integration/test_frontend_e2e.py`):**
   При запущенном стеке (`GATEWAY_URL=http://localhost:8000`):
   ```bash
   python3 -m unittest tests/integration/test_frontend_e2e.py
   ```
   Сценарий автоматически верифицирует: регистрацию и логин пользователя, редактирование профиля и загрузку аватара, поиск ошибки и потоковый SSE RAG-синтез, получение графа Star Map (`/graph/threads`), полный CRUD чатов, гостевой поиск без токенов, прямое подключение к открытому Remote MCP (`/sse` + `errorium_search_solution`) и Admin API (stats, reload, rag config, блокировка).

### 5.7. Логирование HTTP-запросов Gateway и отладка dev-стека

1. **Журнал запросов:** каждый HTTP-запрос к Gateway (включая CORS-preflight и SSE-стримы) фиксируется middleware `middleware.RequestLogger` в stdout контейнера строкой формата:
   ```text
   2026/11/20 12:24:07 HTTP POST /api/v1/chats -> 201 (3.2ms) [172.18.0.1]
   ```
   Логируются метод, путь с query-строкой, итоговый статус, длительность и адрес клиента (`X-Real-IP` от nginx либо `RemoteAddr`). Тела запросов/ответов не логируются. Middleware прозрачен для SSE: обёртка `ResponseWriter` реализует `http.Flusher` и `http.Hijacker`, поэтому стриминг `/search/rag` и `/chats/{id}/messages` не буферизуется. Просмотр: `docker logs -f errorium_gateway_dev`.
2. **Индекс для dev-стека:** `search_engine_core` читает индекс из `data/index/` (`INDEX_DIR`). Если каталог пуст (нет `metadata.json`), core стартует в состоянии unready, а Gateway отвечает `{"error":"search backend unavailable"}` на поисковые запросы и сообщения чатов. Для локальной разработки достаточно малого корпуса benchmark:
   ```bash
   docker compose -f docker-compose.indexer.yml run --rm indexer corpus
   docker compose -f docker-compose.dev.yml restart search_engine_core
   ```
3. **Проксирование API во фронтенде:** dev-nginx (`nginx.dev.conf`) и prod-nginx (`nginx.conf`) проксируют `location /api/` в `http://gateway:8000/api/`; проверка доступности бэкенда через фронтенд: `curl -i http://localhost:3000/api/v1/chats` (ожидается `401` от Gateway без токена). Dev-nginx дополнительно отдаёт статику с `Cache-Control: no-cache`, исключая исполнение браузером устаревших версий `app.js`/`index.html`.



