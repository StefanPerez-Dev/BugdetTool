CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS accounts (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  short_name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS categories (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  hue INTEGER,
  is_income BOOLEAN NOT NULL DEFAULT false,
  is_savings BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS budgets (
  category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
  amount_cents INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (category_id)
);

CREATE TABLE IF NOT EXISTS transactions (
  id BIGSERIAL PRIMARY KEY,
  booked_on DATE NOT NULL,
  description TEXT NOT NULL,
  amount_cents INTEGER NOT NULL,
  account_id BIGINT NOT NULL REFERENCES accounts(id),
  category_id BIGINT NOT NULL REFERENCES categories(id),
  reviewed BOOLEAN NOT NULL DEFAULT false,
  recurring BOOLEAN NOT NULL DEFAULT false,
  flag TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS transactions_booked_on_idx ON transactions (booked_on DESC);
CREATE INDEX IF NOT EXISTS transactions_reviewed_idx ON transactions (reviewed);
CREATE INDEX IF NOT EXISTS transactions_account_idx ON transactions (account_id);
CREATE INDEX IF NOT EXISTS transactions_category_idx ON transactions (category_id);

CREATE TABLE IF NOT EXISTS app_meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
