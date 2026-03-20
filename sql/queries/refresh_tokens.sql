-- name: AddRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at, revoked_at)
VALUES (
    $1, NOW(), NOW(), $2, $3, null
)
RETURNING *;


-- name: CheckTokenAndGetUser :one
select users.id as user_id, refresh_tokens.expires_at as tokenExpiresAt, refresh_tokens.revoked_at as tokenRevokedAt from users 
inner join refresh_tokens on users.id = refresh_tokens.user_id
where refresh_tokens.token = $1 and refresh_tokens.revoked_at is null and refresh_tokens.expires_at > NOW();



-- name: RevokeToken :exec
update refresh_tokens
set revoked_at = NOW(), updated_at = NOW()
where token = $1;

