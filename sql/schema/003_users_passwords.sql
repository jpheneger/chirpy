-- +goose Up
ALTER TABLE users ADD COLUMN hashed_password TEXT NOT NULL;

-- +goose Down
ALTER TABLE users DROP COLUMN IF ExISTS hashed_password;