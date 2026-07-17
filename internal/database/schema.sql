CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY,
    username      TEXT    NOT NULL UNIQUE,
    email         TEXT    NOT NULL UNIQUE,
    password_hash TEXT    NOT NULL,
    created_at    TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    token      TEXT    PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) UNIQUE,
    created_at TEXT    NOT NULL,
    expires_at TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS posts (
    id         INTEGER PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id),
    title      TEXT    NOT NULL,
    body       TEXT    NOT NULL,
    created_at TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS comments (
    id         INTEGER PRIMARY KEY,
    post_id    INTEGER NOT NULL REFERENCES posts(id),
    user_id    INTEGER NOT NULL REFERENCES users(id),
    body       TEXT    NOT NULL,
    created_at TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS categories (
    id   INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    kind TEXT NOT NULL CHECK (kind IN ('demographic', 'genre', 'theme', 'discussion'))
);

CREATE TABLE IF NOT EXISTS post_categories (
    post_id     INTEGER NOT NULL REFERENCES posts(id),
    category_id INTEGER NOT NULL REFERENCES categories(id),
    PRIMARY KEY (post_id, category_id)
);

CREATE TABLE IF NOT EXISTS reactions (
    id          INTEGER PRIMARY KEY,
    user_id     INTEGER NOT NULL REFERENCES users(id),
    target_id   INTEGER NOT NULL,
    target_type TEXT    NOT NULL CHECK (target_type IN ('post', 'comment')),
    value       INTEGER NOT NULL CHECK (value = 1 OR value = -1),
    created_at  TEXT    NOT NULL,
    UNIQUE (user_id, target_id, target_type)
);

INSERT OR IGNORE INTO categories (name, kind) VALUES
    ('Shonen',           'demographic'),
    ('Shoujo',           'demographic'),
    ('Seinen',           'demographic'),
    ('Josei',            'demographic'),
    ('Action',           'genre'),
    ('Fantasy',          'genre'),
    ('Romance',          'genre'),
    ('Isekai',           'genre'),
    ('Mecha',            'genre'),
    ('Slice of Life',    'genre'),
    ('Horror',           'genre'),
    ('Comedy',           'genre'),
    ('Sports',           'genre'),
    ('School',           'theme'),
    ('Music',            'theme'),
    ('Military',         'theme'),
    ('Supernatural',     'theme'),
    ('Historical',       'theme'),
    ('Anime Discussion', 'discussion'),
    ('Manga',            'discussion'),
    ('Recommendations',  'discussion'),
    ('News',             'discussion'),
    ('Fan Creations',    'discussion'),
    ('General',          'discussion');
