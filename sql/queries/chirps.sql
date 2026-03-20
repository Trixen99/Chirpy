-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
    gen_random_uuid(), NOW(), NOW(), $1, $2
)
RETURNING *;



-- name: GetAllChirps :many
select * from chirps
order by created_at asc;


-- name: GetChirpbyID :one
select * from chirps
where id = $1
order by created_at asc;


-- name: GetChirpbyIDCorrectUser :one
select * from chirps
where id = $1 and user_id = $2
order by created_at asc;

-- name: DeleteChirpByIDAndUser :exec
delete from chirps where id = $1;