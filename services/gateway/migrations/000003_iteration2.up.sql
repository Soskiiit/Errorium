-- 000003_iteration2.up.sql
-- Итерация 2: сброс пароля, SSO-состояния и конфигурация RAG
-- (MCP-токенов нет — ADR-012: открытый MCP без токенов)

-- 1. Токены сброса пароля
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id          UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE CHECK (length(token_hash) = 64),
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_password_reset_user_id ON password_reset_tokens(user_id);

-- 2. Состояния авторизации SSO (OAuth2 authorization-code flow)
CREATE TABLE IF NOT EXISTS sso_auth_states (
    state       TEXT PRIMARY KEY,
    provider    TEXT NOT NULL CHECK (provider IN ('github', 'google', 'yandex')),
    redirect_to TEXT,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Конфигурация параметров RAG/LLM (синглтон id=1)
CREATE TABLE IF NOT EXISTS rag_config (
    id            INTEGER PRIMARY KEY CHECK (id = 1),
    provider      TEXT NOT NULL DEFAULT 'stub' CHECK (provider IN ('openai', 'gigachat', 'stub')),
    base_url      TEXT,
    model         TEXT NOT NULL DEFAULT '',
    system_prompt TEXT NOT NULL DEFAULT '',
    temperature   REAL NOT NULL DEFAULT 0.2 CHECK (temperature >= 0 AND temperature <= 2),
    max_tokens    INTEGER NOT NULL DEFAULT 1500 CHECK (max_tokens > 0),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Начальная запись id=1 (системный промпт из .env.example)
INSERT INTO rag_config (id, provider, base_url, model, system_prompt, temperature, max_tokens)
VALUES (
    1,
    'stub',
    NULL,
    '',
    'Ты Errorium — ассистент по диагностике программных ошибок. Отвечай на русском: суть и первопричина, готовый исправленный код, профилактика. Опирайся только на переданные источники Stack Overflow.',
    0.2,
    1500
)
ON CONFLICT (id) DO NOTHING;
