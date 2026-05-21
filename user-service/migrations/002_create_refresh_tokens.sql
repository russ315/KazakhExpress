create table if not exists refresh_tokens (
    id serial primary key,
    user_id varchar(36) not null references users(id) on delete cascade,
    token_hash varchar(255) not null unique,
    expires_at timestamptz not null,
    created_at timestamptz not null default now()
);
create index if not exists idx_refresh_tokens_user_id on refresh_tokens(user_id);
create index if not exists idx_refresh_tokens_token_hash on refresh_tokens(token_hash);
