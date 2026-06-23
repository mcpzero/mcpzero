-- Sample schema + data for the MCPZERO SQLite example.
-- Usage: sqlite3 demo.db < seed.sql

PRAGMA foreign_keys = ON;

DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS customers;

CREATE TABLE customers (
  id         INTEGER PRIMARY KEY,
  name       TEXT    NOT NULL,
  email      TEXT    NOT NULL UNIQUE,
  created_at TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE products (
  id          INTEGER PRIMARY KEY,
  sku         TEXT    NOT NULL UNIQUE,
  name        TEXT    NOT NULL,
  price_cents INTEGER NOT NULL CHECK (price_cents >= 0)
);

CREATE TABLE orders (
  id          INTEGER PRIMARY KEY,
  customer_id INTEGER NOT NULL REFERENCES customers(id),
  status      TEXT    NOT NULL DEFAULT 'pending',
  created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE order_items (
  id         INTEGER PRIMARY KEY,
  order_id   INTEGER NOT NULL REFERENCES orders(id),
  product_id INTEGER NOT NULL REFERENCES products(id),
  quantity   INTEGER NOT NULL CHECK (quantity > 0)
);

INSERT INTO customers (id, name, email) VALUES
  (1, 'Ada Lovelace',   'ada@example.com'),
  (2, 'Alan Turing',    'alan@example.com'),
  (3, 'Grace Hopper',   'grace@example.com');

INSERT INTO products (id, sku, name, price_cents) VALUES
  (1, 'SKU-001', 'Mechanical Keyboard', 12900),
  (2, 'SKU-002', 'USB-C Cable',          1500),
  (3, 'SKU-003', '27" Monitor',         29900);

INSERT INTO orders (id, customer_id, status) VALUES
  (1, 1, 'paid'),
  (2, 2, 'pending'),
  (3, 1, 'shipped');

INSERT INTO order_items (id, order_id, product_id, quantity) VALUES
  (1, 1, 1, 1),
  (2, 1, 2, 2),
  (3, 2, 3, 1),
  (4, 3, 2, 3);
