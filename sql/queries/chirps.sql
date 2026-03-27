-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
    gen_random_uuid(), NOW(), NOW(), $1, $2
)
RETURNING *;



-- name: GetAllChirps :many
select * from chirps
order by 
case when $1 = 'asc' then created_at end asc,
case when $1 = 'desc' then created_at end desc;


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


-- name: GetChirpsbyUserID :many
select * from chirps
where user_id = $1
order by 
case when $2 = 'asc' then created_at end asc,
case when $2 = 'desc' then created_at end desc;