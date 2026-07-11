CREATE TABLE IF NOT EXISTS users (
id INTEGER PRIMARY KEY,
username TEXT NOT NULL UNIQUE,
email TEXT NOT NULL UNIQUE,
password_hash TEXT NOT NULL,
created_at TEXT NOT NULL
);
CREATE TABLE sessions(
id INTEGER PRIMARY KEY,
user_id INTEGER REFERENCES users(id) NOT NULL,
created_at TEXT NOT NULL,
expires_at TEXT NOT NULL
);
CREATE TABLE posts(
id INTEGER PRIMARY KEY,
user_id INTEGER REFERENCES users(id) NOT NULL,
title TEXT NOT NULL,
body TEXT NOT NULL,
created_at TEXT NOT NULL
);
CREATE TABLE comments(
id INTEGER PRIMARY KEY,
post_id INTEGER REFERENCES posts(id) NOT NULL,
user_id INTEGER REFERENCES users(id) NOT NULL,
body TEXT NOT NULL,
created_at TEXT NOT NULL
);
CREATE TABLE categories(
id INTEGER PRIMARY KEY,
name TEXT NOT NULL UNIQUE,
kind TEXT NOT NULL
);
CREATE TABLE post_categories(
post_id INTEGER REFERENCES posts(id) NOT NULL,
category_id INTEGER REFERENCES categories(id) NOT NULL,
PRIMARY KEY(post_id, category_id)
);
CREATE TABLE reactions(
id INTEGER PRIMARY KEY,
user_id INTEGER REFERENCES users(id) NOT NULL,
target_id INTEGER NOT NULL,
target_type TEXT NOT NULL,
value INTEGER CHECK(value = 1 OR value = -1) NOT NULL,
created_at TEXT NOT NULL,
UNIQUE(user_id, target_id, target_type)
);
