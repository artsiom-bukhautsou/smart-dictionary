GRANT ALL PRIVILEGES ON DATABASE postgres TO admin;
GRANT ALL PRIVILEGES ON SCHEMA public TO admin;
GRANT ALL PRIVILEGES ON TABLE public.translations TO admin;

CREATE TABLE IF NOT EXISTS public.translations (
    id SERIAL PRIMARY KEY,
    word VARCHAR(255) NOT NULL,
    meaning VARCHAR(255) NOT NULL,
    examples VARCHAR(255)[],
    russian_translation VARCHAR(255) NOT NULL,
    meaning_russian VARCHAR(255) NOT NULL,
    examples_russian VARCHAR(255)[]
    );

-- Index for word to improve the performance
CREATE INDEX idx_word ON translations (word);

CREATE TABLE IF NOT EXISTS public.decks (
                                            id SERIAL PRIMARY KEY,
                                            deck_name VARCHAR(255) NOT NULL,
    user_id INT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE
    );

CREATE INDEX idx_user_id_deck ON decks (user_id);

CREATE TABLE IF NOT EXISTS public.deck_translations (
                                                        id SERIAL PRIMARY KEY,
                                                        deck_id INT NOT NULL REFERENCES public.decks(id) ON DELETE CASCADE,
    translation_id INT NOT NULL REFERENCES public.translations(id) ON DELETE CASCADE
    );

CREATE INDEX idx_deck_id ON deck_translations (deck_id);
CREATE INDEX idx_translation_id ON deck_translations (translation_id);d
