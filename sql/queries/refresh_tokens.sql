-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- name: GetTokenByUserId :one
SELECT *
FROM refresh_tokens
WHERE user_id = $1
;

-- name: GetRefreshToken :one
SELECT *
FROM refresh_tokens
WHERE token = $1
;

-- name: RevokeToken :exec
UPDATE refresh_tokens
SET updated_at = NOW(), revoked_at = NOW()
WHERE token = $1
;