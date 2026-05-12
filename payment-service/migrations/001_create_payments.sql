create table if not exists payments (
	id text primary key,
	order_id text not null,
	customer_id text not null,
	customer_email text not null,
	amount_kzt bigint not null check (amount_kzt > 0),
	method text not null check (method in ('card', 'kaspi', 'wallet')),
	status text not null check (status in ('paid', 'refunded', 'failed')),
	refund_reason text not null default '',
	created_at timestamptz not null,
	updated_at timestamptz not null
);

create index if not exists idx_payments_order_id on payments(order_id);
create index if not exists idx_payments_customer_id on payments(customer_id);
create index if not exists idx_payments_status on payments(status);
