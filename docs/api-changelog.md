# API changelog

Changes to the public storefront API, newest first.

## Unreleased

- `GET /products/{id}`: `rating.average` is rounded to the nearest half star
  (4.3 → 4.5, 4.6 → 4.5) instead of truncated to one decimal, to match how
  the storefront draws it. Same field and type; values change. (KART-448)
