# API changelog

Changes to the public storefront API, newest first.

## Unreleased

- `GET /products` and `GET /products/{id}`: each product gains `url`, its
  share link on the storefront (`https://kartly.in/p/{id}`). Additive; no
  field changed or removed. (KART-448)
