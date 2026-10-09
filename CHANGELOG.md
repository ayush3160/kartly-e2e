# Changelog

## Unreleased

- `GET /orders/{id}` shows `pointsEarned`: the loyalty points the order
  earned, from the new `loyalty_ledger` table (migration 0002). (KART-405)
