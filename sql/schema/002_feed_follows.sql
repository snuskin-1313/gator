-- +goose Up
CREATE TABLE feed_follows (
  id INTEGER PRIMARY KEY,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  user_id INTEGER NOT NULL REFERENCES users ON DELETE CASCADE,
  feed_id INTEGER NOT NULL REFERENCES feeds ON DELETE CASCADE,
  UNIQUE (user_id, feed_id)
);

-- +goose Down
DROP TABLE feed_follows;
