create table if not exists payments (
	id text primary key,
	order_id text not null,
	customer_id text not null,
	customer_email text not null,
	amount_kzt bigint not null check (amount_kzt > 0),
	method text not null check (method in ('card', 'kaspi', 'wallet')),
	status text not null check (status in ('pending', 'succeeded', 'failed', 'refunded', 'cancelled')),
	provider_transaction_id text not null default '',
	idempotency_key text not null unique,
	refund_reason text not null default '',
	failure_reason text not null default '',
	created_at timestamptz not null,
	updated_at timestamptz not null
);

create index if not exists idx_payments_order_id on payments(order_id);
create index if not exists idx_payments_customer_id on payments(customer_id);
create index if not exists idx_payments_status on payments(status);

create table if not exists payment_events (
	id bigserial primary key,
	payment_id text not null references payments(id) on delete cascade,
	order_id text not null,
	customer_id text not null,
	amount_kzt bigint not null,
	status text not null check (status in ('pending', 'succeeded', 'failed', 'refunded', 'cancelled')),
	reason text not null default '',
	provider_transaction_id text not null default '',
	occurred_at timestamptz not null
);

create table if not exists refunds (
	id bigserial primary key,
	payment_id text not null references payments(id) on delete cascade,
	reason text not null,
	amount_kzt bigint not null,
	created_at timestamptz not null
);

create index if not exists idx_payment_events_payment_id on payment_events(payment_id);
create index if not exists idx_payment_events_status on payment_events(status);
create index if not exists idx_refunds_payment_id on refunds(payment_id);
