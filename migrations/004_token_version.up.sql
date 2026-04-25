-- token_version используется для немедленной инвалидации ранее выпущенных
-- access-токенов после смены пароля или блокировки учётной записи (ТЗ §4.1).
-- При login/refresh новый access-токен подписывается актуальным значением;
-- Auth middleware отклоняет любой access-токен с устаревшим token_version.
ALTER TABLE users ADD COLUMN IF NOT EXISTS token_version INT NOT NULL DEFAULT 1;
