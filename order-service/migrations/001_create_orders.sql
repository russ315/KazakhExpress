create table if not exists orders (
	id text primary key,
	customer_id text not null,
	items jsonb not null,
	status text not null check (status in ('created', 'paid', 'shipped', 'completed', 'canceled')),
	total_kzt bigint not null check (total_kzt >= 0),
	created_at timestamptz not null,
	updated_at timestamptz not null
);

create index if not exists idx_orders_customer_id on orders(customer_id);
create index if not exists idx_orders_status on orders(status);
create index if not exists idx_orders_created_at on orders(created_at);
