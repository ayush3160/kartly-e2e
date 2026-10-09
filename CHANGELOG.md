# Changelog

## Unreleased

- **Breaking for the support console:** `GET /orders/{id}` no longer embeds
  `notes`. Staff read them from the new `GET /orders/{id}/notes`. The order
  detail no longer waits on Mongo, which was most of its p99. The support
  console switched in support-ui#212. (KART-430)
