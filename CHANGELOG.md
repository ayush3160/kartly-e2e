# Changelog

## Unreleased

- Adding to the cart checks live warehouse stock (inventory `Get`) and
  answers 409 `insufficient_stock` when the quantity is more than we have.
  The catalog's `inStock` flag can lag by an hour. (KART-388)
