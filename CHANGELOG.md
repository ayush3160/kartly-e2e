# Changelog

## Unreleased

- Checkout and quote accept `giftWrap: true`. The quote and the order total
  include the gift-wrap fee (₹49 / $4.99), shown as `giftWrapFee`. pricing-svc
  v2.7 added the `giftWrap` field; it is always sent so pricing can report
  wrap uptake. (KART-402)
