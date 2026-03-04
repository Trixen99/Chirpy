-- +goose Up
CREATE TABLE chirps (
    id uuid primary key,
    created_at timestamp not null,
    Updated_at timestamp not null,
    body text not null,
    user_id uuid not null,
    foreign key (user_id) references users(id) on delete cascade
);

-- +goose Down
DROP TABLE chirps;