-- 000001_init_gateway_schema.down.sql
-- Откат схемы базы данных Gateway

DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS chat_message_tags;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS chats;
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS user_sso_accounts;
DROP TABLE IF EXISTS users;

