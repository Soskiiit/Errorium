-- 000001_init_gateway_schema.up.sql
-- Инициализация схемы PostgreSQL 18 для Gateway (Errorium)
-- Использование идиоматичного типа TEXT с CHECK-ограничениями длины и формата

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- 1. Таблица пользователей
CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    email           TEXT NOT NULL UNIQUE CHECK (length(email) <= 255),
    username        TEXT NOT NULL CHECK (length(username) <= 100),
    avatar_url      TEXT,
    password_hash   TEXT CHECK (password_hash IS NULL OR length(password_hash) <= 255), -- NULL для чистого SSO входа
    role            TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin')),
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at   TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- 2. Таблица привязанных SSO аккаунтов (Google, Yandex, GitHub)
CREATE TABLE IF NOT EXISTS user_sso_accounts (
    id                  UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider            TEXT NOT NULL CHECK (provider IN ('google', 'yandex', 'github')),
    provider_user_id    TEXT NOT NULL CHECK (length(provider_user_id) <= 255),
    email               TEXT CHECK (email IS NULL OR length(email) <= 255),
    profile_data        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_sso_provider_user UNIQUE (provider, provider_user_id)
);

CREATE INDEX IF NOT EXISTS idx_sso_user_id ON user_sso_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_sso_provider_user ON user_sso_accounts(provider, provider_user_id);

-- 3. Таблица сессий пользователей (Refresh-токены с безопасным хэшированием)
CREATE TABLE IF NOT EXISTS user_sessions (
    id                  UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash  TEXT NOT NULL UNIQUE CHECK (length(refresh_token_hash) = 64), -- SHA-256 хэш refresh токена (64 hex-символа)
    user_agent          TEXT,
    client_ip           INET,
    is_revoked          BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at          TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_active_expiry ON user_sessions(expires_at) WHERE is_revoked = FALSE;

-- 4. Таблица чатов / диалогов пользователя
CREATE TABLE IF NOT EXISTS chats (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title           TEXT NOT NULL DEFAULT 'Новый чат' CHECK (length(title) <= 255),
    is_pinned       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chats_user_updated ON chats(user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_chats_title_trgm ON chats USING gin (title gin_trgm_ops);

-- 5. Таблица сообщений чата (включая RAG-ответы и детали ошибок)
CREATE TABLE IF NOT EXISTS chat_messages (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    chat_id         UUID NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    sender_role     TEXT NOT NULL CHECK (sender_role IN ('user', 'assistant', 'system')),
    message_type    TEXT NOT NULL DEFAULT 'error_report' CHECK (message_type IN ('text', 'error_report', 'rag_solution')),
    error_context   JSONB, -- {language, stack_trace, error_type, raw_query}
    rag_sources     JSONB, -- [{doc_id, title, score, url, snippet}]
    token_usage     JSONB, -- {prompt_tokens, completion_tokens, total_tokens}
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_chat_created ON chat_messages(chat_id, created_at ASC);

-- 6. Нормализованный справочник тегов
CREATE TABLE IF NOT EXISTS tags (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    name            TEXT NOT NULL UNIQUE CHECK (length(name) <= 50),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name);

-- 7. Связующая таблица сообщений и тегов (M2M)
CREATE TABLE IF NOT EXISTS chat_message_tags (
    chat_message_id UUID NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
    tag_id          UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (chat_message_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_cmt_tag_id ON chat_message_tags(tag_id);

-- 8. Журнал аудита действий
CREATE TABLE IF NOT EXISTS audit_logs (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id         UUID REFERENCES users(id) ON DELETE SET NULL,
    action          TEXT NOT NULL CHECK (length(action) <= 100),
    client_ip       INET,
    details         JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
