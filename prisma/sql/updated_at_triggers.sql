-- Ensure updated_at is set to current epoch seconds on every row update for all tables
-- Idempotent: safely re-creatable

BEGIN;

-- 1) Create or replace the trigger function
CREATE OR REPLACE FUNCTION set_updated_at_unix()
RETURNS trigger AS $$
BEGIN
  NEW.updated_at := (extract(epoch from now()))::bigint;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 2) Helper to (re)create triggers safely
-- Drops an existing trigger if it exists, then re-creates it.
-- Usage: SELECT ensure_updated_at_trigger('table_name');
CREATE OR REPLACE FUNCTION ensure_updated_at_trigger(tbl regclass)
RETURNS void AS $$
DECLARE
  trg_name text := 'set_updated_at_unix_trg';
BEGIN
  -- Drop existing trigger if present
  IF EXISTS (
    SELECT 1
    FROM pg_trigger t
    JOIN pg_class c ON c.oid = t.tgrelid
    WHERE t.tgname = trg_name AND c.oid = tbl
  ) THEN
    EXECUTE format('DROP TRIGGER %I ON %s;', trg_name, tbl);
  END IF;

  -- Create trigger
  EXECUTE format(
    'CREATE TRIGGER %I BEFORE UPDATE ON %s FOR EACH ROW EXECUTE FUNCTION set_updated_at_unix();',
    trg_name,
    tbl
  );
END;
$$ LANGUAGE plpgsql;

-- 3) Apply to all tables we manage
SELECT ensure_updated_at_trigger('user_account');
SELECT ensure_updated_at_trigger('invalid_subscription');
SELECT ensure_updated_at_trigger('free_credit');
SELECT ensure_updated_at_trigger('spending_unit');

COMMIT;
