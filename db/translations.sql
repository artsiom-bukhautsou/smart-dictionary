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

CREATE INDEX idx_word ON translations (word);

CREATE TABLE IF NOT EXISTS public.users (
    id SERIAL PRIMARY KEY,
    user_name VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL
    );

CREATE INDEX idx_user_name ON users (user_name);

-- Create decks table
CREATE TABLE IF NOT EXISTS public.decks (
    id SERIAL PRIMARY KEY,
    deck_name VARCHAR(255) NOT NULL,
    user_id INT NOT NULL REFERENCES public.users(id) ON DELETE CASCADE
);

-- Create index on user_id column in decks table
CREATE INDEX idx_user_id_deck ON decks (user_id);

-- Create deck_translations join table
CREATE TABLE IF NOT EXISTS public.deck_translations (
    id SERIAL PRIMARY KEY,
    deck_id INT NOT NULL REFERENCES public.decks(id) ON DELETE CASCADE,
    translation_id INT NOT NULL REFERENCES public.translations(id) ON DELETE CASCADE
);

-- Create index on deck_id and translation_id columns in deck_translations table
CREATE INDEX idx_deck_id ON deck_translations (deck_id);
CREATE INDEX idx_translation_id ON deck_translations (translation_id);