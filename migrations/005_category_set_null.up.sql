-- Идемпотентная миграция: при удалении категории обнуляем ссылку в наборах/карточках.
-- Используем DO $$-блок, чтобы безопасно пересоздать FK при повторном запуске.
DO $$
BEGIN
    -- decks.category_id
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'decks_category_id_fkey') THEN
        ALTER TABLE decks DROP CONSTRAINT decks_category_id_fkey;
    END IF;
    ALTER TABLE decks ADD CONSTRAINT decks_category_id_fkey
        FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL;

    -- cards.category_id
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'cards_category_id_fkey') THEN
        ALTER TABLE cards DROP CONSTRAINT cards_category_id_fkey;
    END IF;
    ALTER TABLE cards ADD CONSTRAINT cards_category_id_fkey
        FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL;
END $$;
