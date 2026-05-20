alter table payments
	add column if not exists provider_transaction_id text not null default '',
	add column if not exists idempotency_key text,
	add column if not exists failure_reason text not null default '';

update payments
set idempotency_key = id
where idempotency_key is null or idempotency_key = '';

alter table payments
	alter column idempotency_key set not null;

create unique index if not exists idx_payments_idempotency_key on payments(idempotency_key);

do $$
begin
	if exists (
		select 1
		from pg_constraint
		where conrelid = 'payments'::regclass
		  and conname = 'payments_status_check'
	) then
		alter table payments drop constraint payments_status_check;
	end if;
end $$;

update payments
set status = 'succeeded'
where status = 'paid';

alter table payments
	add constraint payments_status_check check (status in ('pending', 'succeeded', 'failed', 'refunded', 'cancelled'));

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
