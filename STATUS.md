# STATUS.md — Текущее состояние и следующий шаг проекта Errorium

## 1. Текущий статус проекта
- **Этап:** **КТ 3 (Завершена).** Все три волны (Волна A, Волна B, Волна C) и 18 задач итерации 2 выполнены и верифицированы.
- **Инфраструктура и оркестрация:**
  - Полный стек поднимается через `docker compose -f docker-compose.dev.yml up --build` (`make dev-docker`).
  - Все сетевые порты вынесены в переменные `.env` (`FRONTEND_PORT=3000`, `GATEWAY_PORT=8000`, `SEARCH_GRPC_PORT=50051`, `POSTGRES_PORT=5432`, `GARAGE_PORT=3900`).
  - Контракт переменных окружения выверен (2026-11-20): `.env.example` синхронизирован с compose и `pkg/config` gateway (credentials `POSTGRES_*`, `SSO_REDIRECT_BASE`, порты продакшн-фронтенда); gateway получает ключи SSO/LLM/SMTP/JWT через `env_file: .env`, сертификат НУЦ монтируется в `/app/certs`. Регрессия закрыта тестом `tests/unit/test_env_contract.py` (307 unit-тестов OK).
  - Наблюдаемость и dev-среда (2026-11-20): каждый HTTP-запрос Gateway логируется middleware `RequestLogger` (`docker logs -f errorium_gateway_dev`); dev-nginx отдаёт статику с `Cache-Control: no-cache`, скрипты SPA подключены с версионным `?v=`; для dev-поиска построен индекс малого корпуса в `data/index/` (без индекса core отвечает `search backend unavailable`).
  - Пайплайн автодеплоя `.github/workflows/deploy.yml` готов к выкатке на `errorium.ru`.
- **Сквозная интеграция:**
  - Web Frontend (:3000) $\to$ Gateway (:8000) по REST API и SSE.
  - Gateway (:8000) $\to$ Search Engine Core (:50051) исключительно по gRPC (SearchService, GraphService).
  - Remote MCP Server (:8000/sse, :8000/mcp/messages) работает открыто без токенов (ADR-012) для подключения из Cursor и Claude Code.

---

## 2. Матрица готовности сервисов

| Компонент | Роль | Статус | Выполненный объем и подтверждение |
| :--- | :--- | :---: | :--- |
| **Search Engine Indexer** | Роль 1 | **Готов (КТ1–КТ3)** | Реализован потоковый парсер дампа Stack Overflow, токенизатор кода и ошибок (неделимые стектрейсы), SPIMI инвертированный индекс формата v1, SQLite база связей `graph.db`, режимы `sample` и `corpus`. 212 unit + 9 integration тестов пройдены успешно. |
| **Search Engine Core** | Роль 2 | **Готов (КТ2–КТ3)** | BM25 алгоритм с нуля по методичке ИУ5 ($k_1=1.5, b=0.75$), потокобезопасная горячая перезагрузка `ReloadIndex`, gRPC-сервисы `SearchService` и `GraphService` (`:50051`), бенчмарк `evaluate.py` подтвердил NFR: $Recall@10=0.9667$ ($\ge 0.75$), $MRR@10=1.0000$ ($\ge 0.50$). |
| **Gateway & MCP Service** | Роль 3 | **Готов (КТ2–КТ3)** | REST API на Go stdlib, gRPC-клиенты к ядру, LLM-провайдеры (OpenAI, GigaChat с сертификатом НУЦ Минцифры, stub), RAG-оркестратор с семафором `LLM_MAX_CONCURRENT=8` и потоковым SSE-стримингом, открытый Remote MCP сервер с in-memory rate limiting, SSO Яндекс ID, сброс пароля по email (`LogMailer`/`SMTPMailer`), Admin API, bootstrap первого администратора `FIRST_ADMIN_*`. |
| **Web Frontend** | Роль 4 | **Готов (КТ1–КТ3)** | Адаптивный SPA без внешних фреймворков (Vanilla JS, HTML5, CSS3); клиентский слой `api.js` с прозрачным обновлением токенов; чаты с уникальными адресами `/chats/<uuid>` и потоковой выдачей RAG; Star Map (`/star-map`) на чистом Canvas 2D (ADR-006) на реальных данных `/graph/threads`; открытый MCP без токенов (ADR-012), кнопка MCP ведёт на лендинг-якорь `#mcp`; профиль, загрузка аватара и теги. Все 307 unit и 39 e2e/integration тестов зеленые (2 skip — граф-фикстуры из дампа SO). |

---

## 3. Следующий шаг (Next Step)

1. **Защита проекта НИРС:**
   - Демонстрация поиска решений и стектрейсов через Web UI (`http://localhost:3000`).
   - Демонстрация интерактивного графа связей Star Map на Canvas 2D.
   - Демонстрация работы открытого Remote MCP сервера через внешнюю IDE (Cursor / Claude Code) по SSE-протоколу.
   - Подготовка финальной презентации и отчета по ГОСТ.
2. **Дальнейшее развитие (после КТ3):**
   - Перевод индексатора (Роль 1) в режим `full` на полный 100 ГБ архив Stack Overflow с хеш-шардированием.
   - Расширение панели администратора во фронтенде (графики задержек, управление RAG-конфигом через Web UI).
