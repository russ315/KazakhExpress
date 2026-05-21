create table if not exists users (
    id varchar(36) primary key,
    email varchar(255) unique not null,
    password_hash varchar(255) not null,
    first_name varchar(100) not null,
    last_name varchar(100) not null,
    phone varchar(20),
    address text,
    reset_token_hash varchar(255),
    reset_token_expires_at timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);
create index if not exists idx_users_email on users(email);
