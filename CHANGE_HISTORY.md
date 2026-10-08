# CHANGE_HISTORY.md — История изменений проекта Errorium

## [2026-10-08] — Точное пиксель-перфект соответствие Chat-screen.png

### Затронутые компоненты и спецификации:
- `docs/Product/04-interface-and-behavior.md`
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/mascot.png`
- `tests/unit/test_frontend_layout.py`
- `STATUS.md`

### Суть изменений:
1. Из оригинального векторного ассета `message.svg` извлечён и сохранён в `services/frontend/static/mascot.png` официальный 3D-маскот Errorium с прозрачным фоном.
2. Вёрстка `index.html` и `styles.css` доведена до точного соответствия скрину `Chat-screen.png`:
   - Оригинальный маскот с естественной мягкой тенью и плавающим позиционированием.
   - Двойной световой эффект: глубокое космическое синее свечение на фоне (`chat-main::before`) и яркий овальный рассеянный прожектор под слоганом (`.spotlight`).
   - Моноширинный типографический блок слогана: `Element 404. Reacts with developers` (с ярко-красным акцентом `404.`) и `Stack overflow. Issue overknown.`.
   - Точный капсульный инпут (composer) со скруглёнными краями, стекломорфизмом, иконкой прикрепления `+`, полем «What went wrong?» и синей круглой кнопкой отправки со стрелкой вверх `↑`.
   - Дисклеймер «AI can make mistakes. Check important information.» внизу.
   - Векторные SVG-иконки для гамбургер-меню, профиля, прикрепления и отправки.
3. Дополнены тесты в `tests/unit/test_frontend_layout.py` на наличие маскота, прожектора и капсулы ввода.

### Результаты верификации:
- `python3 -m unittest discover tests/unit`: **PASS (5 тестов)**.

## [2026-10-01] — Настраиваемые Docker-порты и frontend dev-сервер

### Затронутые компоненты и спецификации:
- `docs/Tech/02-running.md`
- `.env.example`
- `docker-compose.yml`
- `docker-compose.dev.yml`
- `services/frontend/Dockerfile.dev`
- `Makefile`
- `tests/unit/test_compose_ports.py`
- `STATUS.md`

### Суть изменений:
1. Host-порты production Compose берутся из `GATEWAY_PORT`, `FRONTEND_HTTP_PORT` и `FRONTEND_HTTPS_PORT` с совместимыми значениями по умолчанию.
2. Добавлен изолированный frontend dev-сервер с bind mount статических файлов и live reload.
3. Команда `make frontend-dev-server` поднимает dev-сервер на `FRONTEND_DEV_PORT`, по умолчанию `3000`.

### Результаты верификации:
- Unit-тесты frontend и Compose: **PASS (5 тестов)**.
- `docker compose config` с переопределёнными портами: **PASS**.
- Сборка dev-образа и HTTP-проверка `http://127.0.0.1:3000/`: **PASS**.
- `git diff --check`: **PASS**.

## [2026-10-01] — Адаптивная верстка Chat-screen и бокового меню

### Затронутые компоненты и спецификации:
- `docs/Product/04-interface-and-behavior.md`
- `services/frontend/static/index.html`
- `services/frontend/static/styles.css`
- `services/frontend/static/app.js`
- `tests/unit/test_frontend_layout.py`
- `STATUS.md`

### Суть изменений:
1. По фреймам Figma `Chat-screen` и `Chat-screen-menu-not-authorized` реализован интерфейс на чистых HTML, CSS и JavaScript.
2. При ширине от 800 px боковое меню шириной 496 px отображается постоянно; на меньшей ширине оно работает как drawer.
3. Добавлены мобильное открытие и закрытие меню, затемнение, закрытие клавишей Escape и адаптивное поле запроса.
4. Системные панели телефона и браузера исключены. MCP-лендинг не реализовывался.

### Результаты верификации:
- `python -m unittest tests/unit/test_frontend_layout.py`: **PASS (3 теста)**.
- `node --check services/frontend/static/app.js`: **PASS**.
- `docker compose build frontend`: **PASS**.
- `git diff --check`: **PASS**.

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
