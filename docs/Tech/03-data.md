# 03-data.md — Хранение данных, форматы индексов и графа знаний

## 1. Структура инвертированного индекса (`data/index/`)

Индекс формируется офлайн сервисом `search_engine_indexer` и читается сервисом `search_engine_core` в режиме `read-only` с использованием механизма `mmap`.

```text
data/index/
├── lexicon.bin         # Словарь терминов: [term_id (uint32), term_str, offset (uint64), df (uint32)]
├── postings.bin        # Списки постингов: [doc_id (uint32), tf (uint16), [positions...]]
├── doc_lengths.bin     # Таблица длин документов: [doc_id -> length (uint16)], а также N и avgdl
├── metadata.json       # Общая статистика: N, avgdl, timestamp, версия токенизатора
└── graph.db            # Локальная база данных связей графа тредов (SQLite)
```

### Формат записи Posting List
Для каждого термина $t$:
$$\text{Posting} = \langle doc\_id, tf \rangle$$
При сжатии используется дельта-кодирование $\Delta doc\_id = doc\_id_{i} - doc\_id_{i-1}$ с переменной длиной байта (VByte / Varint) для минимизации размера на диске.

---

## 2. Граф связанных тредов (`graph.db`)

Связи графа формируются индексатором и сохраняются в легковесной встраиваемой базе данных SQLite (`data/index/graph.db`).
Такой подход устраняет необходимость ручного управления бинарными смещениями и гарантирует мгновенную точечную выборку 1-hop/2-hop окрестностей по B-Tree индексу ($< 1$ мс).

### Схема базы данных `graph.db`

#### Таблица связей `thread_links` (Ребра графа)
```sql
CREATE TABLE thread_links (
    source_doc_id INTEGER NOT NULL,
    target_doc_id INTEGER NOT NULL,
    link_type     TEXT NOT NULL,    -- 'DUPLICATE_OF' | 'LINKED_IN_ANSWER'
    weight        REAL DEFAULT 1.0,
    PRIMARY KEY (source_doc_id, target_doc_id, link_type)
);

CREATE INDEX idx_thread_links_source ON thread_links(source_doc_id);
CREATE INDEX idx_thread_links_target ON thread_links(target_doc_id);
```

#### Таблица метаданных `thread_nodes` (Узлы графа и детали)
```sql
CREATE TABLE thread_nodes (
    doc_id              INTEGER PRIMARY KEY,
    title               TEXT NOT NULL,
    snippet             TEXT NOT NULL,
    has_accepted_answer BOOLEAN DEFAULT FALSE,
    tags                TEXT,         -- сериализованный JSON-список, например '["java", "spring"]'
    question_score      INTEGER DEFAULT 0,
    best_answer_score   INTEGER DEFAULT 0,
    best_answer_body    TEXT          -- полный текст ответа (для GetThreadDetails)
);
```

- **Запись (Роль 1, `search_engine_indexer`):** 
  - Пакетная вставка связей транзакциями при потоковом разборе `PostLinks.xml`.
  - Заполнение метаданных узлов при разборе `Posts.xml`.
- **Чтение (Роль 2, `search_engine_core`):** 
  - Выборка подграфа в режиме read-only (`file:data/index/graph.db?mode=ro`):
    1. Нахождение смежных ID:
       ```sql
       SELECT target_doc_id, link_type, weight FROM thread_links WHERE source_doc_id IN (?, ?, ...);
       ```
    2. Извлечение метаданных для всех узлов окрестности:
       ```sql
       SELECT doc_id, title, snippet, has_accepted_answer, tags, question_score, best_answer_score 
       FROM thread_nodes WHERE doc_id IN (?, ?, ...);
       ```
    3. Выборка деталей треда по клику (`GetThreadDetails`):
       ```sql
       SELECT title, best_answer_body FROM thread_nodes WHERE doc_id = ?;
       ```

---

## 3. База данных пользователей и авторизации

Для хранения учетных записей, SSO-профилей и ключей MCP в Gateway используется встроенная БД (SQLite по умолчанию или PostgreSQL для продакшна):
- **Таблица `users`:**
  - `id` (UUID), `email` (string, unique), `hashed_password` (string, nullable для SSO), `role` (`guest`, `developer`, `admin`), `is_active` (boolean), `created_at` (timestamp).
- **Таблица `sso_accounts`:**
  - `id` (UUID), `user_id` (FK), `provider` (`github`, `google`, `oidc`), `provider_user_id` (string), `profile_data` (JSON).
- **Таблица `mcp_tokens`:**
  - `id` (UUID), `user_id` (FK), `token_hash` (string), `name` (string), `expires_at` (timestamp), `last_used_at` (timestamp).
- **Таблица `audit_logs`:**
  - `id` (UUID), `user_id` (FK, nullable), `action` (string), `ip_address` (string), `details` (JSON), `timestamp` (timestamp).
