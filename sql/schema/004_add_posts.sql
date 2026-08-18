-- +goose Up
CREATE TABLE posts (
  id INTEGER PRIMARY KEY,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  title TEXT NOT NULL,
  url TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL,
  published_at TIMESTAMP NOT NULL,
  feed_id INTEGER NOT NULL REFERENCES feeds ON DELETE CASCADE
);
-- +goose Down
DROP TABLE posts;
