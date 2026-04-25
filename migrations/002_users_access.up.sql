-- Расширение таблицы users полями для блокировки и мягкого удаления.
-- Реализует функциональные требования ТЗ §4.1: «блокировка и удаление учётных записей».
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_blocked BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS blocked_at TIMESTAMP NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP NULL;

-- Частичный индекс для быстрых запросов по активным пользователям.
CREATE INDEX IF NOT EXISTS idx_users_active ON users(id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role) WHERE deleted_at IS NULL;
