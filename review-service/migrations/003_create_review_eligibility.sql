CREATE TABLE IF NOT EXISTS review_eligibility (
    user_id TEXT NOT NULL,
    product_id TEXT NOT NULL,
    order_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, product_id, order_id)
);

CREATE INDEX IF NOT EXISTS idx_review_eligibility_user_product ON review_eligibility (user_id, product_id);
