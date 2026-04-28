-- Прогресс обучения по карточкам (алгоритм SM-2)
CREATE TABLE IF NOT EXISTS card_progress (
    id               SERIAL PRIMARY KEY,
    user_id          INT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    card_id          INT       NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    deck_id          INT       NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    ease_factor      FLOAT     NOT NULL DEFAULT 2.5,   -- коэффициент лёгкости (мин. 1.3)
    interval_days    INT       NOT NULL DEFAULT 0,      -- дней до следующего повторения
    repetitions      INT       NOT NULL DEFAULT 0,      -- кол-во успешных повторений подряд
    next_review_at   TIMESTAMP NOT NULL DEFAULT NOW(),  -- когда повторять следующий раз
    last_reviewed_at TIMESTAMP,
    status           VARCHAR(20) NOT NULL DEFAULT 'new', -- new | learning | review | mastered
    created_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, card_id)
);

-- Сессии обучения
CREATE TABLE IF NOT EXISTS study_sessions (
    id             SERIAL PRIMARY KEY,
    user_id        INT         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deck_id        INT         NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    started_at     TIMESTAMP   NOT NULL DEFAULT NOW(),
    ended_at       TIMESTAMP,
    cards_total    INT         NOT NULL DEFAULT 0,
    cards_reviewed INT         NOT NULL DEFAULT 0,
    cards_correct  INT         NOT NULL DEFAULT 0,
    status         VARCHAR(20) NOT NULL DEFAULT 'active', -- active | completed
    created_at     TIMESTAMP   NOT NULL DEFAULT NOW()
);

-- Избранные наборы
CREATE TABLE IF NOT EXISTS deck_favorites (
    user_id    INT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    deck_id    INT       NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, deck_id)
);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_card_progress_user_deck  ON card_progress(user_id, deck_id);
CREATE INDEX IF NOT EXISTS idx_card_progress_due        ON card_progress(user_id, next_review_at);
CREATE INDEX IF NOT EXISTS idx_study_sessions_user      ON study_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_deck_favorites_user      ON deck_favorites(user_id);
