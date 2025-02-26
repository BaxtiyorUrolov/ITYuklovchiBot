CREATE TABLE users (
    id BIGINT NOT NULL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE admins (
    id BIGINT NOT NULL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE channels (
    username VARCHAR(100)
);

CREATE TABLE configs (
    instagram_api TEXT,
    tiktok_api TEXT
)