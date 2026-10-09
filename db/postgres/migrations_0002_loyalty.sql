-- KART-405: loyalty points ledger.
CREATE TABLE IF NOT EXISTS loyalty_ledger (
  id         bigserial PRIMARY KEY,
  order_id   text NOT NULL,
  points     bigint NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS loyalty_ledger_order ON loyalty_ledger (order_id);
