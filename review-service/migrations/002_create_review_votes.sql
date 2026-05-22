CREATE TABLE IF NOT EXISTS review_votes (
    review_id TEXT NOT NULL REFERENCES reviews (id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    vote INT NOT NULL CHECK (vote IN (-1, 1)),
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (review_id, user_id)
);
