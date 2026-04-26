-- Откат: восстанавливаем FK без ON DELETE SET NULL
ALTER TABLE decks DROP CONSTRAINT IF EXISTS decks_category_id_fkey;
ALTER TABLE decks ADD CONSTRAINT decks_category_id_fkey
    FOREIGN KEY (category_id) REFERENCES categories(id);

ALTER TABLE cards DROP CONSTRAINT IF EXISTS cards_category_id_fkey;
ALTER TABLE cards ADD CONSTRAINT cards_category_id_fkey
    FOREIGN KEY (category_id) REFERENCES categories(id);
