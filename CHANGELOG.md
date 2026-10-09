# Changelog

## Unreleased

- Order lists (`GET /orders`, `GET /me/orders`, `GET /admin/orders`) no
  longer include orders whose payment failed. They were never charged, and
  customers kept opening tickets about "orders" they never placed. A failed
  payment is still visible on `GET /orders/{id}`. (KART-399)
