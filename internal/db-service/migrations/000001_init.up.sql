-- создание таблицы users
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user',  -- 'user' или 'user_premium'
    created_at TIMESTAMP DEFAULT now()
);

-- создание таблицы tasks
CREATE TABLE tasks (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT DEFAULT 'not_done',   -- например: pending, done, in_progress
    created_at TIMESTAMP DEFAULT now()
);
