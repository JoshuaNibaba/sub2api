-- Track the application user that owns an upstream account.
-- NULL keeps historical/shared accounts safe and redacted for restricted roles.
ALTER TABLE accounts
  ADD COLUMN IF NOT EXISTS owner_user_id BIGINT NULL;

ALTER TABLE accounts
  DROP CONSTRAINT IF EXISTS accounts_owner_user_id_fkey;

ALTER TABLE accounts
  ADD CONSTRAINT accounts_owner_user_id_fkey
  FOREIGN KEY (owner_user_id) REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS accounts_owner_user_id_idx
  ON accounts(owner_user_id);
