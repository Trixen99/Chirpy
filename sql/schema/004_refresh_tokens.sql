-- +goose Up
CREATE TABLE refresh_tokens (
    token text primary key,
    created_at timestamp not null,
    updated_at timestamp not null,
    user_id uuid not null,
    expires_at timestamp not null,
    revoked_at timestamp,
    foreign key (user_id) references users(id) on delete cascade
);


-- +goose Down
DROP TABLE refresh_tokens;