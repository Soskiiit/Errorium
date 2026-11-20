# Web Frontend

> **Роль 4: Frontend & Data Visualization Engineer**  
> Пользовательский веб-интерфейс, визуализация интерактивной звездной карты (Star Map) связанных тредов Stack Overflow и интеграция с Remote MCP.

---

## 1. Назначение и архитектура

Сервис реализует современный Single Page Application (SPA) веб-интерфейс без сторонних JS-фреймворков и библиотек визуализации (Pure Vanilla JS / HTML5 / CSS3 / Canvas 2D), строго соответствующий утвержденной дизайн-системе (`references/Chat-screen.png` и `references/Chat-screen-menu-not-authorized.png`) и архитектурным решениям проекта:
- **Hero-экран поиска:** центрированный парящий маскот, фирменные слоганы в моноширинном начертании (`Element 404. Reacts with developers` / `Stack overflow. Issue overknown.`), синяя световая аура и нижний эллиптический спотлайт.
- **Интерактивный Prompt Bar:** капсульный темный контейнер со стеклянным эффектом Liquid Glass (`backdrop-filter: blur(24px)`), кнопкой быстрого поповера тегов `+`, автоматическим увеличением высоты ввода, синей круглой кнопкой отправки `↑` и предупреждением `AI can make mistakes`.
- **Выдвижная навигация (Drawer):** полнофункциональное боковое меню со строкой поиска по истории, списком недавних чатов (`Recent chats`) с возможностью переименования, закрепления и удаления, кнопками `MCP`, `About us`, плавающей белой пилюлей `New chat`, авторизацией и настройками.
- **RAG & SERP выдача:** динамический вывод карточки синтезированного решения с кнопкой открытия Star Map, блоков кода с кнопкой копирования и ранжированных источников Stack Overflow.
- **Star Map (`/star-map`, ADR-006):** интерактивная карта связанных SO-тредов на чистом Canvas 2D с собственной физической симуляцией (force-directed graph). Построена на реальных данных эндпоинта `GET /api/v1/graph/threads?doc_ids=...`. Без внешних библиотек (D3.js, React Flow, Cytoscape категорически исключены).
- **Интеграция с Remote MCP (ADR-012):** открытый доступ без персональных токенов. Модальное окно и страница лендинга предоставляют URL подключения `{location.origin}/sse` и готовый JSON-конфиг для вставки в `~/.cursor/mcp.json` и `claude_desktop_config.json` без заголовков авторизации.
- **Авторизация и профиль:** JWT-аутентификация с прозрачным обновлением токенов через `POST /api/v1/auth/refresh`, вход через Яндекс ID (OAuth2), сброс пароля, смена профессии, пола, языка и загрузка пользовательского аватара (`POST /api/v1/users/avatar`).

---

## 2. Структура статических файлов

Все клиентские файлы размещаются в директории `services/frontend/static/`:

```text
services/frontend/static/
├── index.html            # Главная страница SPA со всеми контейнерами представлений и модалок
├── app.js                # Основная логика интерфейса, роутинг, Star Map Canvas, UI-события
├── api.js                # Клиент API Gateway: JWT-аутентификация, авто-рефреш, SSE ReadableStream
├── styles.css            # Корневой файл стилей (@import модулей)
├── mascot.png            # Фирменный персонаж-маскот
├── blue-fire.png         # Фоновый графический элемент свечения
├── css/
│   ├── base.css          # CSS переменные, сброс стилей, типографика
│   ├── layout.css        # Сетка основного контейнера и экранов
│   ├── sidebar.css       # Стили боковой выдвижной панели (Drawer)
│   ├── hero.css          # Стили центральной зоны Hero
│   ├── landing.css       # Стили секций лендинга (#landing-mcp, #landing-about)
│   ├── chat.css          # Сообщения пользователя и карточки RAG-ответа
│   ├── prompt.css        # Капсульный контейнер ввода Liquid Glass
│   ├── modals.css        # Стили модальных окон (MCP, Settings, About)
│   ├── profile.css       # Экран редактирования профиля и загрузки аватара
│   ├── auth.css          # Окна входа, регистрации и сброса пароля
│   ├── star-map.css      # Полноэкранный оверлей Star Map, легенда, панель деталей
│   └── responsive.css    # Адаптивные медиа-запросы
```

---

## 3. Взаимодействие с Gateway API

Клиент `api.js` взаимодействует с REST API Gateway (`/api/v1`):

| Эндпоинт | Метод | Описание |
| :--- | :--- | :--- |
| `/api/v1/auth/register` | `POST` | Регистрация нового пользователя |
| `/api/v1/auth/login` | `POST` | Вход по логину и паролю |
| `/api/v1/auth/refresh` | `POST` | Прозрачное обновление JWT-токена |
| `/api/v1/auth/sso/login/yandex`| `GET` | Редирект на OAuth2 Яндекс ID |
| `/api/v1/auth/password/forgot` | `POST` | Запрос ссылки сброса пароля |
| `/api/v1/auth/password/reset`  | `POST` | Установка нового пароля по токену |
| `/api/v1/users/me` | `GET`, `PATCH` | Получение и обновление профиля |
| `/api/v1/users/avatar` | `POST` | Загрузка аватара (multipart/form-data) |
| `/api/v1/chats` | `GET`, `POST` | Список чатов и создание нового чата |
| `/api/v1/chats/{id}` | `GET`, `PATCH`, `DELETE` | Получение, закрепление/переименование и удаление чата |
| `/api/v1/chats/{id}/messages` | `POST` | Отправка сообщения (JSON или SSE поток) |
| `/api/v1/search` | `POST` | Поиск решений по BM25 (гостевой и авторизованный) |
| `/api/v1/search/rag` | `POST` | Потоковый RAG-ответ (SSE: `event: delta`, `event: done`) |
| `/api/v1/graph/threads` | `GET` | Получение узлов и связей для Star Map по `doc_ids` |
| `/api/v1/tags` | `GET` | Каталог популярных тегов для поповера `+` |
| `/api/v1/status` | `GET` | Статус ядра gRPC, шлюза и активного RAG-провайдера |
| `/sse`, `/mcp/messages` | `GET`, `POST` | Открытый Remote MCP сервер для Cursor / Claude Code |

---

## 4. Визуализация Star Map (ADR-006)

Интерактивная карта связей Stack Overflow тредов реализована на элементе `<canvas>` с собственной force-симуляцией:
- **Узлы (Nodes):** четырехлучевые искры с радиальным градиентом, центральным ядром и дифференциацией по цветам:
  - 🟢 **Зеленый (`#10b981`):** `PRIMARY_RESULT` (Top-1 результат поиска).
  - 🟡 **Желтый (`#f59e0b`):** `OTHER_RESULT` (Top-K результатов поиска).
  - 🔵 **Синий (`#38bdf8`):** `DUPLICATE` (треды-дубликаты по `graph.db`).
  - 🟣 **Фиолетовый (`#a855f7`):** `LINKED` (треды, упомянутые в ответах).
- **Связи (Edges):** пунктирные линии для дубликатов, сплошные линии для ссылок в ответах.
- **Интерактивность:** панорамирование (Pan), масштабирование (Zoom с плавным ease-out), перетаскивание узлов (Drag), тултип при наведении и боковая панель с полным текстом принятого ответа при клике.
- **Данные:** реальные графовые связи запрашиваются из `GET /api/v1/graph/threads?doc_ids=...`, возвращаемые gRPC-сервисом `GraphService`.

---

## 5. Конфигурация и запуск

### Запуск в Docker:
В составе полного dev-стека:
```bash
make dev-docker
# Фронтенд доступен по адресу: http://localhost:3000
```

### Локальная проверка синтаксиса и тесты:
```bash
# Синтаксическая проверка скриптов:
node --check services/frontend/static/app.js
node --check services/frontend/static/api.js

# Запуск модульных тестов фронтенда:
python3 -m unittest tests/unit/test_frontend_api.py
python3 -m unittest tests/unit/test_frontend_chats.py
python3 -m unittest tests/unit/test_frontend_profile.py
python3 -m unittest tests/unit/test_star_map.py
python3 -m unittest tests/unit/test_frontend_mcp.py
python3 -m unittest tests/unit/test_frontend_spa.py

# Сквозное e2e тестирование:
python3 -m unittest tests/integration/test_frontend_e2e.py
```
