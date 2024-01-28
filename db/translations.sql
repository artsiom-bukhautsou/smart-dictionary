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

-- Insert sample data
INSERT INTO public.translation (meaning, examples, russian_translation, meaning_russian, examples_russian)