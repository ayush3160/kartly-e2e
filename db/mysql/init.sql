-- Legacy accounts database from the old monolith.
CREATE TABLE accounts (
  id        BIGINT PRIMARY KEY,
  email     VARCHAR(255) NOT NULL UNIQUE,
  full_name VARCHAR(255) NOT NULL,
  tier      VARCHAR(16) NOT NULL DEFAULT 'standard'
);

CREATE TABLE addresses (
  id         BIGINT PRIMARY KEY AUTO_INCREMENT,
  account_id BIGINT NOT NULL,
  line1      VARCHAR(255) NOT NULL,
  city       VARCHAR(64) NOT NULL,
  pincode    CHAR(6) NOT NULL,
  is_default TINYINT(1) NOT NULL DEFAULT 0,
  INDEX (account_id)
);

INSERT INTO accounts (id, email, full_name, tier) VALUES
  (101, 'asha.rao@example.com',    'Asha Rao',     'gold'),
  (102, 'vikram.s@example.com',    'Vikram Singh', 'standard'),
  (103, 'meera.k@example.com',     'Meera Kapoor', 'standard'),
  (104, 'arjun.n@example.com',     'Arjun Nair',   'silver'),
  (900, 'support@kartly.example',  'Kartly Support','staff'),
  (901, 'ops-admin@kartly.example','Kartly Admin', 'staff');

INSERT INTO addresses (id, account_id, line1, city, pincode, is_default) VALUES
  (1, 101, '12 MG Road',          'Bengaluru', '560001', 1),
  (2, 101, '44 Residency Road',   'Bengaluru', '560025', 0),
  (3, 102, '9 Linking Road',      'Mumbai',    '400050', 1),
  (4, 103, '221 Park Street',     'Kolkata',   '700016', 1),
  (5, 104, '7 Anna Salai',        'Chennai',   '600002', 1),
  (6, 104, '18 Banjara Hills',    'Hyderabad', '500034', 0);
