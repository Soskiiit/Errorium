-- Память чата для RAG (ADR-014): компактные факты, обновляемые моделью через remember-блок.
ALTER TABLE chats ADD COLUMN IF NOT EXISTS memory TEXT NOT NULL DEFAULT '';