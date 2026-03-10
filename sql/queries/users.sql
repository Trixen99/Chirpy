-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email)
VALUES (
    gen_random_uuid(), NOW(), NOW(), $1
)
RETURNING *;



-- name: ClearUsers :exec
delete from users;




-- name: AddPassword :exec
update users
set hashed_password = $2
where id = $1;


-- name: GetPassword :one
select * from users where email = $1;