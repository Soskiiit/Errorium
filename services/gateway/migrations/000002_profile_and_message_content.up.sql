ALTER TABLE users ADD COLUMN profession TEXT NOT NULL DEFAULT 'Backend developer' CHECK (length(profession) <= 100);
ALTER TABLE users ADD COLUMN gender TEXT NOT NULL DEFAULT 'Not specified' CHECK (gender IN ('Male', 'Female', 'Not specified'));
ALTER TABLE users ADD COLUMN language TEXT NOT NULL DEFAULT 'English' CHECK (language IN ('English', 'Русский'));

ALTER TABLE chat_messages ADD COLUMN content TEXT;
ALTER TABLE chat_messages ADD COLUMN code TEXT;
