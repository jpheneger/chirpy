-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: DeleteUsers :exec
DELETE FROM users;

-- name: GetUserById :one
SELECT *
FROM users
WHERE id = $1
;

-- name: GetUserByEmail :one
SELECT *
FROM users
where email = $1
;

-- name: UpdateUser :one
UPDATE users
SET email = $2, hashed_password = $3
WHERE id = $1
RETURNING *;

-- name: UpgradeUser :exec
UPDATE users
SET is_chirpy_red = true
WHERE id = $1
;