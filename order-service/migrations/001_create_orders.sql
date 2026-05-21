create table if not exists orders (
	id text primary key,
	customer_id text not null,
	status text not null check (status in ('created', 'paid', 'payment_failed', 'shipped', 'completed', 'canceled')),
	total_kzt bigint not null check (total_kzt >= 0),
	created_at timestamptz not null,
	updated_at timestamptz not null
);

create table if not exists order_items (
	id bigserial primary key,
	order_id text not null references orders(id) on delete cascade,
	product_id text not null,
	name text not null,
	quantity integer not null check (quantity > 0),
	price_kzt bigint not null check (price_kzt >= 0)
);

create table if not exists order_status_history (
	id bigserial primary key,
	order_id text not null references orders(id) on delete cascade,
	from_status text,
	to_status text not null,
	reason text,
	created_at timestamptz not null
);

create index if not exists idx_orders_customer_id on orders(customer_id);
create index if not exists idx_orders_status on orders(status);
create index if not exists idx_orders_created_at on orders(created_at);
create index if not exists idx_order_items_order_id on order_items(order_id);
create index if not exists idx_order_status_history_order_id on order_status_history(order_id);
