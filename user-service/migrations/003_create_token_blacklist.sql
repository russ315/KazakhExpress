create table if not exists token_blacklist (
    id serial primary key,
    jti varchar(255) not null unique,
    expires_at timestamptz not null,
    created_at timestamptz not null default now()
);
create index if not exists idx_token_blacklist_jti on token_blacklist(jti);
create index if not exists idx_token_blacklist_expires_at on token_blacklist(expires_at);
