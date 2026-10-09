-- Kartly orders database. Seeded so every recorded test case finds the
-- order, coupon or status it needs.
CREATE TABLE coupons (
  code        text PRIMARY KEY,
  kind        text NOT NULL CHECK (kind IN ('percent', 'fixed', 'free_shipping')),
  value       bigint NOT NULL DEFAULT 0,
  min_amount  bigint NOT NULL DEFAULT 0,
  expires_at  timestamptz NOT NULL
);

CREATE TABLE orders (
  id               text PRIMARY KEY,
  tenant_id        text NOT NULL,
  customer_id      text NOT NULL,
  status           text NOT NULL,
  total_amount     bigint NOT NULL,
  currency         text NOT NULL,
  coupon           text,
  shipping_fee     bigint NOT NULL DEFAULT 0,
  authorization_id text,
  capture_id       text,
  delivered_at     timestamptz,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX orders_tenant_customer ON orders (tenant_id, customer_id, created_at DESC);

CREATE TABLE order_items (
  order_id    text NOT NULL REFERENCES orders (id),
  sku         text NOT NULL,
  name        text NOT NULL,
  quantity    int NOT NULL,
  unit_amount bigint NOT NULL,
  PRIMARY KEY (order_id, sku)
);

CREATE TABLE returns (
  id            text PRIMARY KEY,
  order_id      text NOT NULL REFERENCES orders (id),
  status        text NOT NULL,
  refund_amount bigint NOT NULL,
  currency      text NOT NULL,
  reason        text NOT NULL,
  label         text,
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE outbox (
  id           bigserial PRIMARY KEY,
  aggregate_id text NOT NULL,
  topic        text NOT NULL,
  payload      jsonb NOT NULL,
  created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE webhook_events (
  provider    text NOT NULL,
  event_id    text NOT NULL,
  kind        text NOT NULL,
  received_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (provider, event_id)
);

CREATE TABLE audit_log (
  id     bigserial PRIMARY KEY,
  actor  text NOT NULL,
  action text NOT NULL,
  target text NOT NULL,
  at     timestamptz NOT NULL DEFAULT now()
);

INSERT INTO coupons (code, kind, value, min_amount, expires_at) VALUES
  ('SAVE10',    'percent',       10,    0,      '2030-01-01'),
  ('FLAT500',   'fixed',         50000, 100000, '2030-01-01'),
  ('FREESHIP',  'free_shipping', 0,     0,      '2030-01-01'),
  ('FESTIVE25', 'percent',       25,    200000, '2030-01-01'),
  ('EXPIRED20', 'percent',       20,    0,      '2024-01-01');

-- Fixed timestamps keep recorded responses stable.
INSERT INTO orders (id, tenant_id, customer_id, status, total_amount, currency, coupon, shipping_fee, authorization_id, capture_id, delivered_at, created_at) VALUES
  ('ord_seed_0001', 'acme', 'u-101', 'paid',           188682, 'INR', NULL,     0,    'auth_ord_seed_0001', 'cap_ord_seed_0001', NULL,                     '2026-09-01T09:00:00Z'),
  ('ord_seed_0002', 'acme', 'u-101', 'shipped',        94282,  'INR', 'SAVE10', 0,    'auth_ord_seed_0002', 'cap_ord_seed_0002', NULL,                     '2026-09-03T10:30:00Z'),
  ('ord_seed_0003', 'acme', 'u-101', 'delivered',      294882, 'INR', NULL,     0,    'auth_ord_seed_0003', 'cap_ord_seed_0003', now() - interval '5 days', '2026-08-20T08:15:00Z'),
  ('ord_seed_0004', 'acme', 'u-101', 'delivered',      58764,  'INR', NULL,     4900, 'auth_ord_seed_0004', 'cap_ord_seed_0004', now() - interval '45 days', '2026-07-01T12:00:00Z'),
  ('ord_seed_0005', 'acme', 'u-101', 'cancelled',      94282,  'INR', NULL,     0,    'auth_ord_seed_0005', 'cap_ord_seed_0005', NULL,                     '2026-08-10T16:45:00Z'),
  ('ord_seed_0006', 'acme', 'u-102', 'paid',           1061882,'INR', NULL,     0,    'auth_ord_seed_0006', 'cap_ord_seed_0006', NULL,                     '2026-09-05T11:00:00Z'),
  ('ord_seed_0007', 'acme', 'u-102', 'placed',         47082,  'INR', NULL,     4900, NULL,                 NULL,                NULL,                     '2026-09-06T07:20:00Z'),
  ('ord_seed_0008', 'acme', 'u-102', 'delivered',      412882, 'INR', 'FLAT500',0,    'auth_ord_seed_0008', 'cap_ord_seed_0008', now() - interval '2 days', '2026-08-28T13:10:00Z'),
  ('ord_seed_0009', 'acme', 'u-103', 'payment_failed', 248882, 'INR', NULL,     0,    'auth_ord_seed_0009', NULL,                NULL,                     '2026-09-07T18:05:00Z'),
  ('ord_seed_0010', 'acme', 'u-103', 'paid',           129882, 'INR', NULL,     0,    'auth_ord_seed_0010', 'cap_ord_seed_0010', NULL,                     '2026-09-08T09:40:00Z'),
  ('ord_seed_0011', 'acme', 'u-103', 'shipped',        353882, 'INR', NULL,     0,    'auth_ord_seed_0011', 'cap_ord_seed_0011', NULL,                     '2026-09-02T14:00:00Z'),
  ('ord_seed_0012', 'acme', 'u-104', 'delivered',      175582, 'INR', 'SAVE10', 0,    'auth_ord_seed_0012', 'cap_ord_seed_0012', now() - interval '29 days','2026-08-12T10:00:00Z'),
  ('ord_seed_0013', 'acme', 'u-104', 'paid',           70682,  'INR', NULL,     4900, 'auth_ord_seed_0013', 'cap_ord_seed_0013', NULL,                     '2026-09-09T19:30:00Z'),
  ('ord_seed_0014', 'globex','u-201','paid',           94282,  'INR', NULL,     0,    'auth_ord_seed_0014', 'cap_ord_seed_0014', NULL,                     '2026-09-04T09:00:00Z');

INSERT INTO order_items (order_id, sku, name, quantity, unit_amount) VALUES
  ('ord_seed_0001', 'SKU-TEE-BLK-M',  'Classic Tee, Black, M',  2, 79900),
  ('ord_seed_0002', 'SKU-TEE-WHT-L',  'Classic Tee, White, L',  1, 79900),
  ('ord_seed_0002', 'SKU-SOCK-3PK',   'Ankle Socks, 3-pack',    1, 39900),
  ('ord_seed_0003', 'SKU-HOOD-GRY-M', 'Zip Hoodie, Grey, M',    1, 249900),
  ('ord_seed_0004', 'SKU-MUG-CER',    'Ceramic Mug, 350ml',     1, 49900),
  ('ord_seed_0005', 'SKU-TEE-BLK-M',  'Classic Tee, Black, M',  1, 79900),
  ('ord_seed_0006', 'SKU-BUDS-PRO',   'Wireless Earbuds Pro',   1, 899900),
  ('ord_seed_0007', 'SKU-NOTE-A5',    'Dot-grid Notebook, A5',  1, 29900),
  ('ord_seed_0007', 'SKU-PEN-SET',    'Gel Pen Set',            1, 19900),
  ('ord_seed_0008', 'SKU-BOOK-DDIA',  'Designing Data-Intensive Applications', 1, 399900),
  ('ord_seed_0009', 'SKU-CHGR-65W',   '65W USB-C Charger',      1, 249900),
  ('ord_seed_0010', 'SKU-CABLE-2M',   'Braided USB-C Cable, 2m',1, 69900),
  ('ord_seed_0010', 'SKU-MUG-CER',    'Ceramic Mug, 350ml',     1, 49900),
  ('ord_seed_0011', 'SKU-SPKR-MINI',  'Mini Bluetooth Speaker', 1, 349900),
  ('ord_seed_0012', 'SKU-LAMP-DESK',  'LED Desk Lamp',          1, 189900),
  ('ord_seed_0013', 'SKU-PLANT-POT',  'Terracotta Planter',     1, 89900),
  ('ord_seed_0014', 'SKU-TEE-WHT-L',  'Classic Tee, White, L',  1, 79900);

INSERT INTO returns (id, order_id, status, refund_amount, currency, reason, label) VALUES
  ('ret_seed_0008', 'ord_seed_0008', 'refunded', 399900, 'INR', 'damaged', 'KRTRETSEED0008');
