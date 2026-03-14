-- name: AddRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at, revoked_at)
VALUES (
    $1, NOW(), NOW(), $2, $3, null
)
RETURNING *;


-- name: CheckTokenAndGetUser :one
select user_id, refresh_tokens.expires_at as tokenExpiresAt, refresh_tokens.revoked_at as tokenRevokedAt from users 
full join refresh_tokens on users.user_id = refresh_tokens.user_id
where refresh_tokens.token = $1;

