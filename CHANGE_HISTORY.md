# CHANGE_HISTORY.md — История изменений проекта Errorium

## [2026-10-10] — Миграция объектного хранилища S3 с MinIO на Garage

### Затронутые компоненты и спецификации:
- `deploy/garage/garage.toml`
- `docker-compose.yml`
- `docker-compose.dev.yml`
- `.env.example`
- `services/gateway/pkg/config/config.go`
- `services/gateway/pkg/config/config_test.go`
- `docs/Tech/02-running.md`
- `docs/Tech/03-data.md`
- `docs/Tech/04-api-and-access.md`
- `tests/unit/test_docker_ports.py`
- `STATUS.md`

### Суть изменений:
1. **Инфраструктура Garage S3 (`deploy/garage/garage.toml`, docker-compose):**
   - В связи с прекращением поддержки и удалением образов MinIO осуществлен переход на легковесное распределенное S3-совместимое хранилище **Garage** (образ `dxflrs/garage:v2.4.1`).
   - Добавлен базовый конфигурационный файл `deploy/garage/garage.toml` (sqlite движок метаданных, S3 API на порту 3900, Admin API на порту 3903, replication_factor = 1).
   - В `docker-compose.yml` и `docker-compose.dev.yml` сервис `minio` заменен на `garage` с флагами `--single-node --default-bucket`, автоматически создающими ноду, ключ доступа и бакет `errorium-avatars` при первом запуске.
   - Настроен healthcheck на основе нативной утилиты `/garage node id` (`test: ["CMD", "/garage", "node", "id"]`).
   - Тома обновлены на `errorium_garage_meta` и `errorium_garage_data` (и соответствующие dev-версии).
2. **Параметризация портов и `.env.example`:**
   - Введены переменные `GARAGE_PORT` (по умолчанию 3900) и `GARAGE_ADMIN_PORT` (по умолчанию 3903).
   - Обновлен `.env.example` с полным набором настроек Garage S3 (`GARAGE_DEFAULT_ACCESS_KEY`, `GARAGE_DEFAULT_SECRET_KEY`, `GARAGE_DEFAULT_BUCKET`, `GARAGE_REGION=garage`, `GARAGE_RPC_SECRET`, `GARAGE_ADMIN_TOKEN`).
3. **Конфигурация Gateway (`services/gateway/pkg/config`):**
   - Структура `MinIOConfig` генерализована в `S3Config` (с сохранением псевдонима `MinIOConfig = S3Config` и поля `cfg.MinIO` для обратной совместимости).
   - Добавлена поддержка региона S3 (`Region: "garage"`), а также чтение переменных окружения `S3_*`, `GARAGE_*` и fallback на `MINIO_*`.
   - Написаны модульные тесты конфигуратора в `services/gateway/pkg/config/config_test.go`.
4. **Тесты и спецификации:**
   - Обновлен тест `tests/unit/test_docker_ports.py` для валидации портов Garage S3.
   - В документации `docs/Tech/02-running.md`, `docs/Tech/03-data.md` и `docs/Tech/04-api-and-access.md` все упоминания MinIO скорректированы на Garage S3.
   - Все unit-тесты (Python и Go) пройдены успешно, конфигурации Docker Compose валидированы.

## [2026-10-09] — Параметризация портов сервисов для dev и prod окружений

### Затронутые компоненты и спецификации:
- `docker-compose.dev.yml`
- `docker-compose.yml`
- `.env.example`
- `docs/Tech/02-running.md`
- `tests/unit/test_docker_ports.py`
- `STATUS.md`

### Суть изменений:
1. **Параметризация портов в `docker-compose.dev.yml`:**
   - Для всех сервисов дев-окружения добавлен проброс портов через переменные окружения с дефолтными значениями:
     - `minio`: `127.0.0.1:${MINIO_PORT:-9000}:9000` и `127.0.0.1:${MINIO_CONSOLE_PORT:-9001}:9001` (позволяет сменить порт, если порт 9000 или 9001 занят на хост-машине).
     - `postgres`: `127.0.0.1:${POSTGRES_PORT:-5432}:5432`.
     - `gateway`: `127.0.0.1:${GATEWAY_PORT:-8000}:8000`.
     - `frontend`: `${FRONTEND_PORT:-3000}:80`.
   - Внутри Docker-сети сервисы продолжают взаимодействовать по фиксированным внутренним адресам (`minio:9000`, `postgres:5432`, `gateway:8000`), благодаря чему изменение портов хоста не ломает межсервисные связи.
2. **Параметризация портов в `docker-compose.yml`:**
   - Порты `gateway` (`127.0.0.1:${GATEWAY_PORT:-8000}:8000`) и `frontend` (`${FRONTEND_HTTP_PORT:-80}:80`, `${FRONTEND_HTTPS_PORT:-443}:443`) теперь также поддерживают переопределение через `.env`.
3. **Обновление `.env.example` и документации:**
   - В `.env.example` добавлены `POSTGRES_PORT=5432`, `MINIO_PORT=9000`, `MINIO_CONSOLE_PORT=9001` с поясняющими комментариями.
   - В `docs/Tech/02-running.md` дополнена сводная таблица портов и добавлена рекомендация по разрешению конфликтов портов на хосте.
4. **Тестирование:**
   - Добавлен модульный тест `tests/unit/test_docker_ports.py`, проверяющий корректность fallback-значений и регулярных выражений портов в конфигурациях compose и `.env.example`.
5. **Корректировка точки монтирования тома PostgreSQL:**
   - Путь монтирования тома PostgreSQL 18 в `docker-compose.dev.yml` и `docker-compose.yml` изменен с устаревшего `/var/lib/postgresql/data` на `/var/lib/postgresql` в соответствии с официальным образом `postgres:18-alpine`.

## [2026-10-09] — Проектирование схемы БД Gateway (PostgreSQL 18+), MinIO и миграций

### Затронутые компоненты и спецификации:
- `docs/Tech/03-data.md`
- `docs/Tech/04-api-and-access.md`
- `services/gateway/migrations/000001_init_gateway_schema.up.sql`
- `services/gateway/migrations/000001_init_gateway_schema.down.sql`
- `services/gateway/domain/` (`user.go`, `sso.go`, `session.go`, `chat.go`, `tag.go`, `audit.go`)
- `services/gateway/pkg/config/config.go`
- `docker-compose.yml`, `docker-compose.dev.yml`
- `.env.example`
- `STATUS.md`

### Суть изменений:
1. **Спецификация схемы PostgreSQL:**
   - Выбрана СУБД **PostgreSQL 18+**. Используется нативная встроенная функция ядра **`uuidv7()`** для первичных ключей (time-ordered идентификаторы без фрагментации B-Tree индексов).
   - Все строковые поля переведены с `VARCHAR(n)` на идиоматичный для PostgreSQL тип **`TEXT` с явными `CHECK`-ограничениями** длины (`length <= N`) и формата, что исключает накладные расходы на конвертацию и обеспечивает единый формат хранения `varlena`.

   - Сущности:
     - `users`: пользователи системы (роли `user` и `admin`, email, password_hash, avatar_url, статусы).
     - `user_sso_accounts`: OAuth/SSO профили (`google`, `yandex`, `github`), связь с пользователем, уникальность `(provider, provider_user_id)`.
     - `user_sessions`: сессии пользователей, SHA-256 хэши refresh-токенов, client_ip (INET), user_agent, статус отзыва (`is_revoked`), таймстемпы.
     - `chats`: темы диалогов пользователя (`Recent chats`), флаг закрепления `is_pinned`, быстрый композитный индекс `(user_id, updated_at DESC)` и триграммный поиск `pg_trgm`.
     - `chat_messages`: сообщения диалога, контекст ошибки `error_context` (JSONB), источники Stack Overflow `rag_sources` (JSONB), статистика токенов `token_usage` (JSONB).
     - `tags` и `chat_message_tags`: нормализованный каталог тегов и отношение Many-to-Many к сообщениям с каскадным удалением.
     - `audit_logs`: системный журнал критических действий.
   - **MCP:** Полностью исключена авторизация для MCP (удалены токены, эндпоинт `/sse` открыт без auth).
2. **Объектное хранилище аватарок (MinIO / S3):**
   - Добавлен сервис MinIO в `docker-compose.yml` и `docker-compose.dev.yml` для хранения аватарок пользователей (бакет `errorium-avatars`).
3. **Миграции и инфраструктура:**
   - Написаны файлы миграции `000001_init_gateway_schema.up.sql` и `000001_init_gateway_schema.down.sql`.
   - **Элегантные встроенные миграции (embed.FS):** SQL-файлы встроены в бинарник через `migrations.FS` (`embed.FS`). Добавлен раннер `internal/migrator` на базе `golang-migrate/v4/source/iofs`, который при старте сервиса Gateway атомарно накатывает ожидающие миграции.
   - **Очистка Docker:** Устранен антипаттерн монтирования одного SQL-файла в `/docker-entrypoint-initdb.d/`. Контейнер PostgreSQL в `docker-compose.yml` и `docker-compose.dev.yml` теперь полностью чист.
   - Созданы Go-модели в `services/gateway/domain/` и обновлен `config.go` для подключения к Postgres и MinIO (`AUTO_MIGRATE`).
   - Написан юнит-тест `services/gateway/migrations/migrations_test.go` для валидации встроенных миграций.

## [2026-10-08] — Подключение веб-шрифтов Inter и JetBrains Mono для кроссплатформенного отображения


### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `STATUS.md`

### Суть изменений:
- В `<head>` [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html) добавлены директивы `preconnect` к `fonts.googleapis.com` и `fonts.gstatic.com` с подключением шрифтов `Inter` (начертания 400, 500, 600, 700) и `JetBrains Mono` (начертания 400, 500, 600, 700).
- В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) добавлен `@import` и обновлены системные переменные `--font-ui` и `--font-mono`.
- Теперь на мобильных устройствах (Android, iOS) и десктопах интерфейс и моноширинный слоган отображаются одинаково и аутентично макету.

## [2026-10-08] — Очистка и дедупликация тегов секции <head> в index.html

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `STATUS.md`

### Суть изменений:
- Устранены дублирующиеся теги: двойные `<meta charset="UTF-8">`, `<meta name="viewport">`, `<title>`, `<meta name="description">`, `<link rel="canonical">`, дублированные блоки Open Graph (`og:*`) и Twitter Card, повторные подключения фавиконов.
- Объединен расширенный список поисковых ключевых слов в единый тег `meta keywords` (`bm25, rag, mcp, java, python, go, c++`).
- Вычищены лишние пустые строки и сохранены все параметры адаптивного viewport (`viewport-fit=cover`), окрашивания системной шторки браузера (`#070a13`) и ранний анти-FOUC скрипт.

## [2026-10-08] — Отключение системного голубого выделения при нажатии на кнопки (Tap Highlight) на мобильных устройствах

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `STATUS.md`

### Суть изменений:
- Для всех интерактивных элементов (`button`, `a`, `[role="button"]`, `.icon-btn`, а также глобально `*`) задано свойство `-webkit-tap-highlight-color: transparent !important;`.
- Отключен браузерный прямоугольный контур фокуса при таче (`outline: none !important`), а также системный вызов контекстного меню (`-webkit-touch-callout: none`) и выделение текста на кнопках (`user-select: none`).

## [2026-10-08] — Стилизация строки ввода сообщений с эффектом Liquid Glass

### Затронутые компоненты и спецификации:
- `services/frontend/static/styles.css`
- `docs/Product/04-interface-and-behavior.md`
- `STATUS.md`

### Суть изменений:
1. **Эффект Liquid Glass (Apple VisionOS / High-end UI Glass) для капсулы ввода (`.prompt-bar-container`):**
   - Переход от темного глухого фона к кристальной оптической прозрачности (`rgba(255, 255, 255, 0.12)` $\rightarrow$ `rgba(255, 255, 255, 0.03)` $\rightarrow$ `rgba(15, 23, 42, 0.35)`), позволяющей фону преломляться сквозь линзу.
   - Оптическое размытие подложки: `backdrop-filter: blur(28px) saturate(210%) brightness(108%)`.
   - Полное бесшовное заполнение капсулы градиентом стекла от края до края (удалены искусственные половинчатые псевдоэлементы `::before` / `::after`, создававшие эффект несоразмерного блика).
   - 5-слойный стек теней и внутренних светорассеяний (яркая кромка `inset 0 1.5px 1.5px rgba(255, 255, 255, 0.75)`, глубинное свечение и рассеянный амбиент).
2. **Элементы управления:**
   - Кнопка `+` (`.prompt-plus-btn`) сделана чистой и свободной от круглой плашки/фона — аккуратный отдельный символ с плавной сменой цвета и микро-анимацией масштабирования.
   - Кнопка отправки `.prompt-send-btn` стилизована под сапфировую каплю с 3D-куполом, двойным бликом и неоновым свечением.
   - Текст плейсхолдера и поле ввода адаптированы для высокой читаемости на стекле.

## [2026-10-08] — Устранение эффекта передергивания (FOUC) при перезагрузке со свернутым меню

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `tests/unit/test_frontend_spa.py`
- `STATUS.md`

### Суть изменений:
1. **Ранняя синхронизация состояния в `<head>`:**
   - Добавлен компактный синхронный скрипт в `<head>` [index.html](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/index.html), который проверяет `localStorage` на сохраненное состояние `'collapsed'` до первого рендера DOM и навешивает класс `sidebar-collapsed` на `<html>`.
   - В [styles.css](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/styles.css) добавлены селекторы `html.sidebar-collapsed .sidebar-drawer` и шапки, чтобы браузер рендерил свернутое состояние с самого первого кадра без сдвига верстки (zero layout shift).
2. **Блокировка CSS-переходов на время начальной загрузки (`preload`):**
   - Добавлен класс `preload` на `<body>` и стили `.preload * { transition: none !important; }`.
   - В [app.js](file:///home/mariam/WebstormProjects/Errorium/services/frontend/static/app.js) класс `.preload` безопасно снимается после первой отрисовки через `requestAnimationFrame`, возвращая плавную анимацию для пользовательских кликов.
3. **Синхронизация в JS и фикс видимости кнопки разворачивания меню:**
   - Методы `collapseDesktopSidebar` и `expandDesktopSidebar` синхронно переключают класс на `document.documentElement`.
   - Кнопка разворачивания сайдбара в шапке `.header-sidebar-toggle-btn` (`#btn-expand-sidebar`) приведена к визуальному стилю мобильного гамбургер-меню (3 белые полоски через `.hamburger-bar`).
   - Исправлен конфликт каскада стилей, из-за которого кнопка открывания не исчезала при разворачивании меню, а съезжала в правую часть шапки: добавлено явное скрытие `display: none !important` при раскрытом сайдбаре.
4. **Тестирование (SDD):**
   - Расширен `test_collapsible_sidebar_desktop_feature` в [tests/unit/test_frontend_spa.py](file:///home/mariam/WebstormProjects/Errorium/tests/unit/test_frontend_spa.py).

### Результаты верификации:
- Unit-тесты `python3 -m unittest tests/unit/test_frontend_spa.py`: **PASS (11 unit tests PASS)**.

## [2026-10-08] — Возможность сворачивания бокового меню на десктопе по кнопке с фирменной SVG-иконкой

### Затронутые компоненты и спецификации:
- `docs/Product/04-interface-and-behavior.md`
- `tests/unit/test_frontend_spa.py`
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `STATUS.md`

### Суть изменений:
1. **Кнопка сворачивания/разворачивания бокового меню (`.sidebar-toggle-btn` и `.header-sidebar-toggle-btn`):**
   - Добавлена кнопка со специализированной SVG-иконкой панели (`M6.5 17.5L6.5 6.5...`) в шапку сайдбара и шапку приложения.
   - Стилизована обводка `stroke: currentColor !important`, обеспечивающая адаптацию под тему оформления и подсветку при наведении.
2. **Логика и стили сворачивания на широких экранах:**
   - По умолчанию боковое меню гарантированно открыто (раскрыто) на всех экранах от 768px и выше.
   - Состояние сохраняется в `localStorage` по ключу `errorium_sidebar_state` (`expanded` / `collapsed`).
   - Изначально (при первом визите или отсутствии явного сохранения) сайдбар всегда раскрыт. Состояние сворачивания восстанавливается только если пользователь явно свернул его ранее.
   - Очищен устаревший ключ `errorium_sidebar_collapsed`, чтобы исключить ошибочное открытие в свернутом виде из-за предыдущих тестовых кликов.
   - По клику на кнопку в сайдбаре меню плавно сворачивается влево (`margin-left: -var(--sidebar-width)`), расширяя рабочую область на 100% ширины экрана.
   - В шапке основного экрана при свернутом сайдбаре отображаются кнопка разворачивания (с аналогичной SVG-иконкой) и логотип `Errorium`.
   - Повторный клик разворачивает меню в исходное состояние. Добавлено сочетание клавиш `Ctrl+B` / `Cmd+B`.
3. **Тестирование (SDD):**
   - Добавлен тест `test_collapsible_sidebar_desktop_feature` в `tests/unit/test_frontend_spa.py`, проверяющий разметку, SVG, стили и логику управления состоянием в localStorage.

### Результаты верификации:
- Unit-тесты `python3 -m unittest discover tests/unit`: **PASS (11 unit tests PASS)**.

## [2026-10-08] — Адаптивное масштабирование синего огонька маскота на экранах с малой высотой (Samsung Galaxy Fold)

### Затронутые компоненты и спецификации:
- `docs/Product/04-interface-and-behavior.md`
- `tests/unit/test_frontend_spa.py`
- `services/frontend/static/styles.css`
- `STATUS.md`

### Суть изменений:
1. **Ограничение размеров `.hero-glow-aura` и `.blue-fire-bg`:**
   - По умолчанию максимальный размер ауры уменьшен с чрезмерных `900px / 140vw` до безопасных `min(520px, 120vw)` с ограничением `max-height: 56vh`.
   - Для фонового изображения `.blue-fire-bg` включено свойство `object-fit: contain; max-height: 100%; max-width: 100%;`, предотвращающее растяжение по вертикали.
2. **Медиа-запросы для экранов с малой высотой (Galaxy Fold cover screen / Landscape):**
   - Добавлен `@media (max-height: 720px)`: уменьшение ширины ауры до `380px` (`max-width: 90vw; max-height: 48vh;`).
   - Добавлен `@media (max-height: 600px)`: уменьшение ширины ауры до `290px` (`max-width: 80vw; max-height: 40vh;`), маскота до `84px` и компактные отступы секции.
3. **Тестирование (SDD):**
   - Добавлен модульный тест `test_short_viewport_height_scaling` в `tests/unit/test_frontend_spa.py`.

### Результаты верификации:
- Unit-тесты `make test-unit`: **PASS (Go PASS, Python SPA 9 tests PASS 0.56s)**.

## [2026-10-08] — Адаптация бокового меню для широких экранов: постоянный сайдбар и скрытие дублирующего логотипа

### Затронутые компоненты и спецификации:
- `docs/Product/04-interface-and-behavior.md`
- `tests/unit/test_frontend_spa.py`
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `STATUS.md`

### Суть изменений:
1. **Постоянная боковая панель недавних чатов на широких экранах (≥1024px):**
   - Сайдбар больше не сворачивается в мобильный Drawer, а отображается в статичном лейауте слева (`position: relative`, `transform: none`, `visibility: visible`, `width: var(--sidebar-width)`).
   - Затемняющий фон-бэкдроп (`.drawer-backdrop`) скрыт на широких экранах (`display: none`).
   - Кнопка закрытия `✕` (`.close-btn`) в шапке сайдбара скрыта, так как панель больше не закрывается.
2. **Исключение дублирования бренда Errorium в интерфейсе:**
   - Посередине экрана в шапке основного контента заголовок `Errorium` (`.header-logo`) скрыт, исключая повторение названия проекта (название остается единственным — в шапке боковой панели).
   - Кнопка гамбургер-меню (`.header-menu-btn`) скрыта на широких экранах.
   - Оставшаяся кнопка профиля/аутентификации аккуратно выровнена по правому краю шапки (`justify-content: flex-end`).
   - Логотип `Errorium` в шапке сайдбара сделан интерактивным и кликабельным (клик переключает на Hero-экран / создание нового чата).
3. **Безопасное управление состоянием в JavaScript:**
   - Добавлена проверка ширины окна в `openSidebar`, предотвращающая блокировку `overflow: hidden` на десктопе.
4. **Тестирование (SDD):**
   - Добавлен модульный тест `test_desktop_sidebar_and_header_rules` в `tests/unit/test_frontend_spa.py`.

### Результаты верификации:
- Unit-тесты `make test-unit`: **PASS (Go PASS, Python SPA 8 tests PASS 0.54s)**.

## [2026-10-08] — Реализация адаптивного Web Frontend SPA по дизайн-референсам (No-Frameworks)

### Затронутые компоненты и спецификации:
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `services/frontend/static/mascot.png`
- `services/frontend/static/blue-fire.png`
- `services/frontend/dev_server.py`
- `services/frontend/nginx.dev.conf`
- `docker-compose.dev.yml`
- `services/frontend/README.md`
- `docs/Tech/02-running.md`
- `docs/Product/04-interface-and-behavior.md`
- `tests/unit/test_frontend_spa.py`
- `Makefile`
- `STATUS.md`

### Суть изменений:
1. **Перенос дизайна точь-в-точь по макетам из `references/`:**
   - **Экран `Chat-screen.png`:**
     - Шапка с кнопкой меню (гамбургер), центрированным заголовком `Errorium` и иконкой профиля.
     - Центральный Hero-блок с парящим фирменным маскотом `mascot.png` на фоне оригинального арта `blue-fire.png` (интегрирующего синюю ауру и световой эллипс спотлайта), фирменными слоганами в моноширинном шрифте:
       `Element 404. Reacts with developers` (с красным акцентом на `404.`) и `Stack overflow. Issue overknown.`.
     - Капсульный нижний контейнер ввода с кнопкой `+`, автоувеличением высоты textarea, синей кнопкой отправки `↑` и предупреждением `AI can make mistakes. Check important information.`.
   - **Боковое меню `Chat-screen-menu-not-authorized.png`:**
     - Выдвижная панель (Drawer) с анимацией и блюр-бэкдропом.
     - Поле поиска по недавним чатам `Search`.
     - Секция `Recent chats` со стрелкой и 7 референсными ошибками (SyntaxError, IndentationError, TypeError, NameError, UnboundLocalError, KeyError) с меню `⋮`.
     - Навигационные пункты `MCP` и `About us`.
     - Плавающая белая пилюля `New chat` с иконкой пера/блокнота.
     - Нижняя строка профиля `Log in` / `Meet new features!` и иконка настроек `⚙`.
2. **Чистая архитектура SPA без сторонних фреймворков:**
   - Полное отсутствие React, Vue, Angular, Bootstrap, Tailwind или внешних CDN.
   - Модульный Vanilla JavaScript (`services/frontend/static/app.js`) с базой знаний типичных ошибок, генерацией ответов RAG, подсветкой синтаксиса и копированием блоков кода.
   - Модальные окна интеграции Model Context Protocol (MCP) с выдачей персонального Bearer-токена и готового сниппета `mcp.json` для Cursor / Claude Code.
3. **Адаптивность (Mobile-First):**
   - Пиксель-перфект верстка под мобильные устройства (390-430px) и плавное масштабирование на планшеты и десктопы.
4. **Тестирование (SDD):**
   - Создан тест `tests/unit/test_frontend_spa.py`, проверяющий наличие ассетов, отсутствие фреймворков, соответствие референсным текстам и адаптивность CSS.
   - Таргет `make test-unit` расширен для запуска тестов фронтенда.
5. **Устранение горизонтального скролла и окрашивание интерфейса браузера:**
   - Добавлены мета-теги `theme-color` (`#070a13`) для светлой и темной тем, `apple-mobile-web-app-status-bar-style: black-translucent` и `viewport-fit=cover` для бесшовной интеграции со статус-баром телефона.
   - Настроены ограничения `overflow-x: hidden` на `html, body, .app-container, .main-viewport, .view-content, .hero-screen`.
   - Закрытое боковое меню получило `visibility: hidden; pointer-events: none; box-shadow: none;` для предотвращения расширения виртуального скролла на мобильных устройствах.
   - Текст слогана и фон `blue-fire.png` адаптированы через `clamp()` и `max-width: 94vw`.

### Результаты верификации:
- Unit-тесты `make test-unit`: **PASS (Go tests PASS 1.014s, Python SPA tests PASS 0.001s)**.
- Docker build `services/frontend`: **Успешно (2.6s)**.

## [2026-10-01] — Настройка CI/CD автодеплоя (Gateway + Frontend)

### Затронутые компоненты и спецификации:
- `services/gateway/`
- `services/frontend/`
- `docker-compose.yml`
- `.github/workflows/deploy.yml`
- `Makefile`
- `STATUS.md`

### Суть изменений:
1. **Gateway Service (Go 1.27.1):**
   - Развернута модульная архитектура:
     - `pkg/config/config.go` — загрузка конфигурации окружения (`PORT`).
     - `domain/` — заглушка доменного слоя.
     - `internal/handler/health.go` — регистрация маршрутов и обработчик `GET /health` -> `{"status":"ok"}`.
     - `internal/app/app.go` — фабрика экземпляра приложения `App.New(cfg)` с регистрацией хэндлеров, запуск и `Shutdown(ctx)`.
     - `cmd/main.go` — точка входа с инициализацией конфига, приложения и graceful shutdown.
   - Написаны модульные тесты `internal/handler/health_test.go` (`GET /health` и метод `POST` -> 405).
   - Создан мультистейдж `Dockerfile` (`golang:1.27.1-alpine` -> `alpine:3.20`).

2. **Frontend Service (Nginx 1.26-alpine):**
   - Создан production-ready `nginx.conf`:
     - Редирект с HTTP (порт 80) на HTTPS (порт 443) для домена `errorium.ru`.
     - Настройка SSL (Mozilla Intermediate, ciphers, TLSv1.2/1.3, сессионный кэш).
     - Gzip-сжатие (уровень 6, типы json, js, css, xml, svg).
     - Проксирование API-запросов (`/api/` и `/health`) на шлюз `gateway:8000`.
   - Создана минималистичная заглушка `static/index.html` («Errorium. Скоро»).
   - Создан `Dockerfile` на базе `nginx:1.26-alpine`.

3. **Инфраструктура Docker Compose (`docker-compose.yml`):**
   - Сконфигурированы сервисы `gateway` и `frontend`.
   - Использованы теги образов GHCR (`ghcr.io/soskiiit/errorium-<service>:develop`) и локальные контексты сборки.
   - Настроено монтирование SSL-сертификатов хоста (`/etc/ssl/errorium.crt`, `/etc/ssl/errorium.key`) в контейнер фронтенда.
   - Поисковое ядро (`search_engine_core` и `search_engine_indexer`) временно исключено по согласованию.

4. **GitHub Actions CI/CD (`.github/workflows/deploy.yml`):**
   - `Stage 0 (test)`: запуск модульных тестов Go (`go test -v -race ./...`).
   - `Stage 1 (build-and-push)`: сборка и пуш Docker-образов в GHCR по триггеру `push` в `develop` или `workflow_dispatch` (запускается строго после тестов).
   - `Stage 2 (deploy)`: доставка `docker-compose.yml` на dev-сервер по SCP и перезапуск сервисов по SSH (`docker compose pull && docker compose up -d --remove-orphans`).

5. **Makefile:**
   - Обновлен таргет `test-unit` для запуска тестов `gateway` на Go.

### Результаты верификации:
- Unit-тесты `go test -v -race ./...`: **PASS (1.01s)**.
- Сборка Dockerfile `gateway`: **Успешно**.
- Сборка Dockerfile `frontend`: **Успешно**.

## [2026-10-01] — Настройка SEO, robots.txt, sitemap.xml и брендинга

### Затронутые компоненты:
- `services/frontend/static/robots.txt`
- `services/frontend/static/sitemap.xml`
- `services/frontend/static/favicon.svg`
- `services/frontend/static/index.html`
- `.gitignore`

### Суть изменений:
1. Создан `robots.txt` с разрешением индексации контента и запретом служебных путей (`/api/`, `/health`, `/admin/`).
2. Создан `sitemap.xml` для домена `https://errorium.ru/` с ежедневным приоритетом.
3. Добавлено исключение в `.gitignore` для `services/frontend/static/sitemap.xml`.
4. Создан брендовый векторный фавикон `favicon.svg`.
5. Обновлен `index.html` со слоганом «Errorium. Your error has been solved before.», полным набором мета-тегов для поисковиков, Open Graph и Twitter Cards.
