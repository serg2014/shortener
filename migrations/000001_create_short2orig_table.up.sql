CREATE TABLE IF NOT EXISTS short2orig (
		short_url char(8) PRIMARY KEY,
		orig_url text UNIQUE,
		user_id text,
		is_deleted bool NOT NULL DEFAULT false
);