#!/usr/bin/env bash
# Drives the recording traffic for Kartly's orders-api: about 140 requests in
# 8 test sets. Run it once, against freshly seeded data, while `keploy k8s`
# records main:
#
#   BASE_URL=http://localhost:8080 scripts/traffic.sh            # every set
#   BASE_URL=http://localhost:8080 scripts/traffic.sh checkout   # one set
#
# Sets that change data (cart, checkout, orders, returns, admin, webhooks)
# depend on the seed, so record them once after scripts/reset.sh.
set -uo pipefail

BASE_URL=${BASE_URL:-http://localhost:8080}
PAUSE=${PAUSE:-0.3}
pass=0
fail=0

# call <name> <expected status> <method> <path> [curl args...]
call() {
  local name=$1 want=$2 method=$3 path=$4
  shift 4
  local got
  got=$(curl -s -o /tmp/kartly-last.json -w '%{http_code}' -X "$method" "$BASE_URL$path" \
    -H 'Content-Type: application/json' "$@")
  if [[ $got == "$want" ]]; then
    pass=$((pass + 1))
    printf '  ok   %-44s %s %s -> %s\n' "$name" "$method" "$path" "$got"
  else
    fail=$((fail + 1))
    printf '  FAIL %-44s %s %s -> %s (want %s) %s\n' "$name" "$method" "$path" "$got" "$want" "$(head -c 200 /tmp/kartly-last.json)"
  fi
  sleep "$PAUSE"
}

U101=(-H 'Authorization: Bearer tok-u101')
U102=(-H 'Authorization: Bearer tok-u102')
U103=(-H 'Authorization: Bearer tok-u103')
U104=(-H 'Authorization: Bearer tok-u104')
SUPPORT=(-H 'Authorization: Bearer tok-support')
ADMIN=(-H 'Authorization: Bearer tok-admin')
EXPIRED=(-H 'Authorization: Bearer tok-expired')

set_auth_account() {
  echo "== auth-account"
  call me-gold-customer            200 GET /me "${U101[@]}"
  call me-standard-customer        200 GET /me "${U102[@]}"
  call me-second-address           200 GET /me "${U104[@]}"
  call me-no-token                 401 GET /me
  call me-expired-token            401 GET /me "${EXPIRED[@]}"
  call me-unknown-token            401 GET /me -H 'Authorization: Bearer tok-nobody'
  call address-update              200 PUT /me/address "${U101[@]}" -d '{"id":2,"line1":"44 Residency Road, Flat 3B","city":"Bengaluru","pincode":"560025","default":true}'
  call address-bad-pincode         400 PUT /me/address "${U102[@]}" -d '{"id":3,"line1":"9 Linking Road","city":"Mumbai","pincode":"40005"}'
  call address-missing-city        400 PUT /me/address "${U103[@]}" -d '{"id":4,"line1":"221 Park Street","city":"","pincode":"700016"}'
  call address-missing-id          400 PUT /me/address "${U104[@]}" -d '{"line1":"7 Anna Salai","city":"Chennai","pincode":"600002"}'
  call address-unknown-field       400 PUT /me/address "${U104[@]}" -d '{"id":5,"line1":"7 Anna Salai","city":"Chennai","pincode":"600002","landmark":"near metro"}'
  call my-orders-with-history      200 GET /me/orders "${U101[@]}"
  call my-orders-u102              200 GET /me/orders "${U102[@]}"
  call my-orders-u104              200 GET /me/orders "${U104[@]}"
  call my-orders-staff-empty       200 GET /me/orders "${SUPPORT[@]}"
}

set_catalog_browse() {
  echo "== catalog-browse"
  call products-first-page         200 GET '/products'
  call products-page-2-size-5      200 GET '/products?page=2&size=5'
  call products-apparel            200 GET '/products?category=apparel'
  call products-electronics        200 GET '/products?category=electronics'
  call products-books-small-page   200 GET '/products?category=books&size=2'
  call products-search-tee         200 GET '/products?q=tee'
  call products-search-usb         200 GET '/products?q=usb'
  call products-search-no-hits     200 GET '/products?q=submarine'
  call products-page-past-end      200 GET '/products?page=9&size=10'
  call products-size-capped        200 GET '/products?size=500'
  call product-tee-cache-miss      200 GET /products/p-1001
  call product-tee-cache-hit       200 GET /products/p-1001
  call product-earbuds             200 GET /products/p-2001
  call product-book-reviews        200 GET /products/p-4002
  call product-no-reviews          200 GET /products/p-3003
  call product-out-of-stock        200 GET /products/p-1005
  call product-unknown             404 GET /products/p-9999
  call stock-normal                200 GET /products/p-2002/stock
  call stock-low                   200 GET /products/p-2006/stock
  call stock-none                  200 GET /products/p-1005/stock
}

set_cart() {
  echo "== cart"
  call guest-cart-seeded           200 GET /cart -H 'X-Cart-Id: guest-7f3a'
  call guest-cart-empty            200 GET /cart -H 'X-Cart-Id: guest-new1'
  call guest-add-socks             200 POST /cart/items -H 'X-Cart-Id: guest-new1' -d '{"productId":"p-1006","quantity":2}'
  call guest-add-more-socks        200 POST /cart/items -H 'X-Cart-Id: guest-new1' -d '{"productId":"p-1006","quantity":1}'
  call cart-no-identity            400 GET /cart
  call cart-bad-guest-id           400 GET /cart -H 'X-Cart-Id: cart-123'
  call user-cart-empty             200 GET /cart "${U101[@]}"
  call user-add-tee                200 POST /cart/items "${U101[@]}" -d '{"productId":"p-1001","quantity":1}'
  call user-add-charger            200 POST /cart/items "${U101[@]}" -d '{"productId":"p-2002","quantity":1}'
  call user-add-book               200 POST /cart/items "${U101[@]}" -d '{"productId":"p-4001","quantity":1}'
  call add-quantity-zero           422 POST /cart/items "${U101[@]}" -d '{"productId":"p-1001","quantity":0}'
  call add-quantity-eleven         422 POST /cart/items "${U101[@]}" -d '{"productId":"p-1001","quantity":11}'
  call add-over-limit-total        422 POST /cart/items "${U101[@]}" -d '{"productId":"p-1001","quantity":10}'
  call add-out-of-stock            409 POST /cart/items "${U101[@]}" -d '{"productId":"p-1005","quantity":1}'
  call add-unknown-product         404 POST /cart/items "${U101[@]}" -d '{"productId":"p-9999","quantity":1}'
  call update-tee-to-three         200 PUT /cart/items/SKU-TEE-BLK-M "${U101[@]}" -d '{"quantity":3}'
  call update-not-in-cart          404 PUT /cart/items/SKU-MUG-CER "${U101[@]}" -d '{"quantity":1}'
  call remove-book                 200 DELETE /cart/items/SKU-BOOK-GO "${U101[@]}"
  call merge-guest-into-user       200 POST /cart/merge "${U102[@]}" -H 'X-Cart-Id: guest-7f3a'
  call user-cart-after-merge       200 GET /cart "${U102[@]}"
}

set_checkout() {
  echo "== checkout"
  # u-101 checks out the cart the cart set built (tee x3 + charger).
  call quote-no-coupon             200 POST /checkout/quote "${U101[@]}" -d '{"pincode":"560025"}'
  call quote-save10                200 POST /checkout/quote "${U101[@]}" -d '{"coupon":"SAVE10","pincode":"560025"}'
  call quote-flat500               200 POST /checkout/quote "${U101[@]}" -d '{"coupon":"FLAT500","pincode":"560025"}'
  call quote-freeship              200 POST /checkout/quote "${U101[@]}" -d '{"coupon":"FREESHIP","pincode":"110001"}'
  call quote-expired-coupon        422 POST /checkout/quote "${U101[@]}" -d '{"coupon":"EXPIRED20","pincode":"560025"}'
  call quote-unknown-coupon        422 POST /checkout/quote "${U101[@]}" -d '{"coupon":"NOPE","pincode":"560025"}'
  call quote-empty-cart            422 POST /checkout/quote "${SUPPORT[@]}" -d '{"pincode":"560001"}'
  call checkout-no-idempotency     400 POST /checkout "${U101[@]}" -d '{"cardToken":"tok_visa","addressId":2}'
  call checkout-no-card            400 POST /checkout "${U101[@]}" -H 'Idempotency-Key: ck-101-a' -d '{"addressId":2}'
  call checkout-declined           402 POST /checkout "${U101[@]}" -H 'Idempotency-Key: ck-101-decline' -d '{"cardToken":"tok_decline","addressId":2}'
  call checkout-3ds                202 POST /checkout "${U101[@]}" -H 'Idempotency-Key: ck-101-3ds' -d '{"cardToken":"tok_3ds","addressId":2}'
  call checkout-expired-coupon     422 POST /checkout "${U101[@]}" -H 'Idempotency-Key: ck-101-exp' -d '{"coupon":"EXPIRED20","cardToken":"tok_visa","addressId":2}'
  call checkout-save10             201 POST /checkout "${U101[@]}" -H 'Idempotency-Key: ck-101-ok' -d '{"coupon":"SAVE10","cardToken":"tok_visa","addressId":2}'
  call checkout-retry-same-key     200 POST /checkout "${U101[@]}" -H 'Idempotency-Key: ck-101-ok' -d '{"coupon":"SAVE10","cardToken":"tok_visa","addressId":2}'
  call checkout-after-cart-cleared 422 POST /checkout "${U101[@]}" -H 'Idempotency-Key: ck-101-again' -d '{"cardToken":"tok_visa","addressId":2}'
  # u-102 checks out the merged cart with a fixed discount.
  call quote-merged-cart           200 POST /checkout/quote "${U102[@]}" -d '{"coupon":"FLAT500","pincode":"400050"}'
  call checkout-flat500            201 POST /checkout "${U102[@]}" -H 'Idempotency-Key: ck-102-ok' -d '{"coupon":"FLAT500","cardToken":"tok_visa","addressId":3}'
  # u-103 has a seeded cart; free shipping.
  call quote-u103                  200 POST /checkout/quote "${U103[@]}" -d '{"pincode":"700016"}'
  call checkout-freeship           201 POST /checkout "${U103[@]}" -H 'Idempotency-Key: ck-103-ok' -d '{"coupon":"FREESHIP","cardToken":"tok_visa","addressId":4}'
  # u-104 wants 3 of a keyboard with 2 left.
  call add-low-stock-to-u104       200 GET /cart "${U104[@]}"
  call quote-low-stock             200 POST /checkout/quote "${U104[@]}" -d '{"coupon":"FESTIVE25","pincode":"600002"}'
  call checkout-insufficient-stock 409 POST /checkout "${U104[@]}" -H 'Idempotency-Key: ck-104-a' -d '{"coupon":"FESTIVE25","cardToken":"tok_visa","addressId":5}'
  call reduce-to-available         200 PUT /cart/items/SKU-LOW-STOCK "${U104[@]}" -d '{"quantity":2}'
  call checkout-festive25          201 POST /checkout "${U104[@]}" -H 'Idempotency-Key: ck-104-b' -d '{"coupon":"FESTIVE25","cardToken":"tok_visa","addressId":5}'
  call checkout-no-token           401 POST /checkout -H 'Idempotency-Key: ck-anon' -d '{"cardToken":"tok_visa","addressId":1}'
}

set_orders() {
  echo "== orders"
  call list-mine                   200 GET /orders "${U101[@]}"
  call list-mine-paid              200 GET '/orders?status=paid' "${U101[@]}"
  call list-mine-delivered         200 GET '/orders?status=delivered' "${U101[@]}"
  call list-mine-page-2            200 GET '/orders?page=2&size=2' "${U101[@]}"
  call list-bad-status             400 GET '/orders?status=lost' "${U101[@]}"
  call list-u103-failed            200 GET '/orders?status=payment_failed' "${U103[@]}"
  call get-own-paid                200 GET /orders/ord_seed_0001 "${U101[@]}"
  call get-own-with-notes          200 GET /orders/ord_seed_0002 "${U101[@]}"
  call get-own-delivered           200 GET /orders/ord_seed_0003 "${U101[@]}"
  call get-other-customers         403 GET /orders/ord_seed_0006 "${U101[@]}"
  call get-support-sees-notes      200 GET /orders/ord_seed_0006 "${SUPPORT[@]}"
  call get-admin-sees-order        200 GET /orders/ord_seed_0011 "${ADMIN[@]}"
  call get-other-tenant            404 GET /orders/ord_seed_0014 "${U101[@]}"
  call get-unknown                 404 GET /orders/ord_missing "${U101[@]}"
  call get-no-token                401 GET /orders/ord_seed_0001
  call cancel-paid-refunds         200 POST /orders/ord_seed_0010/cancel "${U103[@]}"
  call cancel-placed-no-refund     200 POST /orders/ord_seed_0007/cancel "${U102[@]}"
  call cancel-shipped-refused      409 POST /orders/ord_seed_0011/cancel "${U103[@]}"
  call cancel-already-cancelled    409 POST /orders/ord_seed_0005/cancel "${U101[@]}"
  call cancel-others-order         403 POST /orders/ord_seed_0013/cancel "${U101[@]}"
}

set_returns() {
  echo "== returns"
  call return-full                 201 POST /returns "${U101[@]}" -d '{"orderId":"ord_seed_0003","reason":"size_too_small","pincode":"560025"}'
  call return-get-created          200 GET /returns/ret_seed_0003 "${U101[@]}"
  call return-window-closed        422 POST /returns "${U101[@]}" -d '{"orderId":"ord_seed_0004","reason":"changed_mind","pincode":"560001"}'
  call return-not-delivered        409 POST /returns "${U101[@]}" -d '{"orderId":"ord_seed_0002","reason":"late","pincode":"560001"}'
  call return-cancelled-order      409 POST /returns "${U101[@]}" -d '{"orderId":"ord_seed_0005","reason":"changed_mind","pincode":"560001"}'
  call return-last-day             201 POST /returns "${U104[@]}" -d '{"orderId":"ord_seed_0012","reason":"defective","pincode":"600002"}'
  call return-partial-sku          201 POST /returns "${U102[@]}" -d '{"orderId":"ord_seed_0008","skus":["SKU-BOOK-DDIA"],"reason":"damaged","pincode":"400050"}'
  call return-sku-not-on-order     422 POST /returns "${U104[@]}" -d '{"orderId":"ord_seed_0012","skus":["SKU-MUG-CER"],"reason":"defective","pincode":"600002"}'
  call return-others-order         403 POST /returns "${U101[@]}" -d '{"orderId":"ord_seed_0008","reason":"damaged","pincode":"560001"}'
  call return-unknown-order        404 POST /returns "${U101[@]}" -d '{"orderId":"ord_missing","reason":"damaged","pincode":"560001"}'
  call return-missing-reason       400 POST /returns "${U101[@]}" -d '{"orderId":"ord_seed_0003","pincode":"560001"}'
  call return-get-seeded           200 GET /returns/ret_seed_0008 "${U102[@]}"
  call return-get-unknown          404 GET /returns/ret_missing "${U102[@]}"
  call return-no-token             401 POST /returns -d '{"orderId":"ord_seed_0003","reason":"x"}'
  call return-get-last-day         200 GET /returns/ret_seed_0012 "${U104[@]}"
}

set_admin() {
  echo "== admin"
  call admin-list-all              200 GET /admin/orders "${ADMIN[@]}"
  call admin-list-paid             200 GET '/admin/orders?status=paid' "${ADMIN[@]}"
  call admin-list-shipped          200 GET '/admin/orders?status=shipped' "${ADMIN[@]}"
  call admin-list-by-customer      200 GET '/admin/orders?customer=u-102' "${ADMIN[@]}"
  call admin-list-page-2           200 GET '/admin/orders?page=2&size=4' "${ADMIN[@]}"
  call admin-list-bad-status       400 GET '/admin/orders?status=lost' "${ADMIN[@]}"
  call admin-list-as-customer      403 GET /admin/orders "${U101[@]}"
  call admin-list-as-support       403 GET /admin/orders "${SUPPORT[@]}"
  call admin-ship-paid             200 PATCH /admin/orders/ord_seed_0001 "${ADMIN[@]}" -d '{"status":"shipped"}'
  call admin-deliver-shipped       200 PATCH /admin/orders/ord_seed_0002 "${ADMIN[@]}" -d '{"status":"delivered"}'
  call admin-cancel-paid           200 PATCH /admin/orders/ord_seed_0013 "${ADMIN[@]}" -d '{"status":"cancelled"}'
  call admin-bad-transition        409 PATCH /admin/orders/ord_seed_0003 "${ADMIN[@]}" -d '{"status":"shipped"}'
  call admin-unknown-status        400 PATCH /admin/orders/ord_seed_0006 "${ADMIN[@]}" -d '{"status":"lost"}'
  call admin-unknown-order         409 PATCH /admin/orders/ord_missing "${ADMIN[@]}" -d '{"status":"shipped"}'
  call admin-patch-as-customer     403 PATCH /admin/orders/ord_seed_0006 "${U102[@]}" -d '{"status":"shipped"}'
}

set_webhooks() {
  echo "== webhooks"
  PAY=(-H 'X-Payments-Signature: sig-payments')
  SHIP=(-H 'X-Shipping-Signature: sig-shipping')
  call pay-captured-for-failed     200 POST /webhooks/payments "${PAY[@]}" -d '{"id":"evt_p_001","type":"payment.captured","orderId":"ord_seed_0009"}'
  call pay-captured-duplicate      200 POST /webhooks/payments "${PAY[@]}" -d '{"id":"evt_p_001","type":"payment.captured","orderId":"ord_seed_0009"}'
  call pay-failed-wrong-state      200 POST /webhooks/payments "${PAY[@]}" -d '{"id":"evt_p_002","type":"payment.failed","orderId":"ord_seed_0006"}'
  call pay-unhandled-type          200 POST /webhooks/payments "${PAY[@]}" -d '{"id":"evt_p_003","type":"payment.disputed","orderId":"ord_seed_0006"}'
  call pay-bad-signature           401 POST /webhooks/payments -H 'X-Payments-Signature: forged' -d '{"id":"evt_p_004","type":"payment.captured","orderId":"ord_seed_0006"}'
  call pay-missing-order           400 POST /webhooks/payments "${PAY[@]}" -d '{"id":"evt_p_005","type":"payment.captured"}'
  call ship-dispatched             200 POST /webhooks/shipping "${SHIP[@]}" -d '{"id":"evt_s_001","type":"shipment.dispatched","orderId":"ord_seed_0006"}'
  call ship-delivered              200 POST /webhooks/shipping "${SHIP[@]}" -d '{"id":"evt_s_002","type":"shipment.delivered","orderId":"ord_seed_0006"}'
  call ship-delivered-out-of-order 200 POST /webhooks/shipping "${SHIP[@]}" -d '{"id":"evt_s_003","type":"shipment.delivered","orderId":"ord_seed_0011"}'
  call ship-bad-signature          401 POST /webhooks/shipping -d '{"id":"evt_s_004","type":"shipment.delivered","orderId":"ord_seed_0011"}'
}

echo "waiting for $BASE_URL/health"
for _ in $(seq 1 90); do
  curl -sf "$BASE_URL/health" >/dev/null && break
  sleep 2
done
curl -sf "$BASE_URL/health" >/dev/null || { echo "orders-api is not up at $BASE_URL" >&2; exit 1; }

all_sets=(auth_account catalog_browse cart checkout orders returns admin webhooks)
sets=("$@")
[[ ${#sets[@]} -eq 0 ]] && sets=("${all_sets[@]}")
for s in "${sets[@]}"; do
  "set_${s//-/_}"
done
echo "passed $pass, failed $fail"
[[ $fail -eq 0 ]]
