# CHANGE_HISTORY.md — История изменений проекта Errorium

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
