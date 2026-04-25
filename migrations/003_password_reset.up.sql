-- Таблица токенов восстановления пароля.
-- Реализует функциональное требование ТЗ §4.2 «Восстановление доступа»:
-- отправка ссылки и обработка состояний (успех/ошибка/истёкшая ссылка).
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id          SERIAL PRIMARY KEY,
    user_id     INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       VARCHAR(255) UNIQUE NOT NULL,
    expires_at  TIMESTAMP NOT NULL,
    used_at     TIMESTAMP NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_password_reset_tokens_active
    ON password_reset_tokens(expires_at) WHERE used_at IS NULL;
