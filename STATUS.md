# STATUS.md — Текущее состояние и следующий шаг проекта Errorium

## 1. Текущий статус проекта
- **Этап:** КТ 1 (Каркасы сервисов, базовые эндпоинты, CI/CD автодеплой).
- **Инфраструктура автодеплоя:** Развернут пайплайн в `.github/workflows/deploy.yml` (прогон тестов -> сборка и пуш в GHCR -> SSH-деплой на dev-сервер `errorium.ru`).
- **Сетевой контур:**
  - `frontend` на Nginx 1.26-alpine (порты 80 с редиректом на 443 HTTPS в проде, порт 3000 в dev; порты хоста гибко настраиваются через `FRONTEND_HTTP_PORT`/`FRONTEND_HTTPS_PORT` и `FRONTEND_PORT`).
  - `gateway` на Go 1.27.1 (порт 8000, чистая слоистая архитектура, `GET /health` с тестами, graceful shutdown; порт хоста настраивается через `GATEWAY_PORT` как в dev, так и в prod).
    - `postgres` и `garage`: в dev-стеке порты хоста вынесены в переменные `.env` (`POSTGRES_PORT`, `GARAGE_PORT`, `GARAGE_ADMIN_PORT`) для бесконфликтного запуска при занятых портах (например, если 3900 или 5432 заняты локально).
- **Поисковое ядро:** `search_engine_core` и `search_engine_indexer` временно выведены из compose-файла до реализации коллегами.

## 2. Матрица готовности сервисов

| Компонент | Роль | Статус | Ближайшая задача |
| :--- | :--- | :--- | :--- |
| **Search Engine Indexer** | Роль 1 | Специфицирован: стратегия корпуса (`docs/Tech/03-data.md`, п. 4; ADR-007), токенизатор (п. 5), язык — Python (ADR-008) | Токенизатор и сэмплер реализованы (`src/indexer/`, 95 unit-тестов зелёные); формат индекса и режим `corpus` специфицированы (п. 1.1, 4.9); далее тесты и SPIMI-построитель, запись `graph.db`, Dockerfile с `7zz`, цель Makefile |
| **Search Engine Core** | Роль 2 | Специфицирован (gRPC) | Реализация чтения инвертированного индекса и BM25 с нуля |
| **Gateway & MCP Service** | Роль 3 | Спроектирована БД PostgreSQL 18 (UUIDv7, SSO, сессии, чаты, теги), настроен Garage S3, встроены автомиграции через embed.FS (golang-migrate) | Реализация слоя репозиториев (pgxpool) и интеграция с Garage S3 SDK |
| **Web Frontend** | Роль 4 | Реализован адаптивный SPA без фреймворков; реализованы экраны и модальные окна по макетам (Auth, Register, Reset Password, Profile, Tags popover, Authorized drawer); восстановлен оригинальный дизайн сообщений чата (.user-bubble и .assistant-card RAG Synthesis); разработан изолированный полноэкранный лендинг (/mcp, /about); реализован Star Map (/star-map) — интерактивная карта связей SO-тредов в стиле Obsidian на Vanilla Canvas с собственной force-симуляцией (FR-3, ADR-006): кнопка входа в карточке RAG-ответа, анимированный уход чата и prompt bar, узлы — компактные четырёхлучевые звёзды со свечением своего цвета, мягкой белой мини-звездой внутри и мерцанием (цветовое кодирование FR-3.2), прогрессивные подписи по зуму (сокращённые → полные, crossfade, без рейтинга и галочки), hover-тултип, панель деталей ответа, звёздчатая легенда, плавный zoom кнопками / wheel / pinch / drag, мок-данные по контракту graph.proto (FR-3.5); 38 unit-тестов пройдены успешно (включая 14 новых test_star_map.py), синтаксис JS верифицирован (node --check) | Интеграция с API Gateway (/api/v1/search) и замена мок-данных Star Map на реальный GraphService (gRPC) |

## 3. Следующий шаг (Next Step)
- **Роль 1 (индексатор):** токенизатор и потоковый сэмплер реализованы и покрыты тестами; далее (формат и режим `corpus` уже специфицированы) тесты и SPIMI-построитель индекса (`lexicon.bin`, `postings.bin`, `doc_lengths.bin`, `metadata.json`), запись `graph.db`, оркестрация режима `sample` (проход по `PostLinks.xml`, затем по `Posts.xml`), Dockerfile и цель Makefile; согласовать с Ролью 2 способ подключения общего токенизатора в ядро (п. 5.6).
- Реализация подключения к PostgreSQL (`pgxpool`) и слоя репозиториев для `users`, `user_sso_accounts`, `user_sessions`, `chats`, `chat_messages`, `tags` в сервисе `gateway`.
- Реализация S3-клиента для Garage (загрузка и получение аватарок).
- Подключение сетевых вызовов из Web Frontend к REST эндпоинтам Gateway (`/api/v1/search`).
- Реализация `GraphService` (gRPC) в `search_engine_core` и замена мок-билдера Star Map на реальные `GetThreadGraph` / `GetThreadDetails` (FR-3.5).

