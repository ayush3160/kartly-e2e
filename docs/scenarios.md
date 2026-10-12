# Scenario branches

Every branch is one pull request against `main`, written the way a teammate
would write it. Recording happens once on `main`; each branch is replayed by
CI and then triaged by a fresh agent session given only the
`keploy cloud triage` command.

"Expected" is what a reviewer who knows the intent would conclude. Paired
branches carry nearly the same diff with opposite intent, to measure whether
the agent reads intent or only the diff.

| Branch | Change | Expected group(s) | Verdict |
|---|---|---|---|
| `e2e/feat-gift-wrap` | Checkout sends `giftWrap` to pricing; response shows `giftWrapFee` | CALL_CHANGED | intended |
| `e2e/fix-coupon-rounding` | "Fix rounding" sends the coupon value in cents | CALL_CHANGED + RESPONSE_CHANGED | regression |
| `e2e/feat-hide-unpaid-orders` | Order lists add `status <> 'payment_failed'` (ticket, doc) | CALL_CHANGED (query shape) | intended |
| `e2e/refactor-order-filters` | Same filter, slipped into a refactor | CALL_CHANGED (query shape) | regression |
| `e2e/refactor-tenant-scoping` | Order lookup drops `tenant_id` | CALL_CHANGED (bind) | regression |
| `e2e/feat-reserve-stock-at-cart` | Adding to cart calls `inventory.Get` | CALL_ADDED (gRPC) | intended |
| `e2e/fix-session-cache` | Session read from Redis twice per request | CALL_ADDED, repeated | regression |
| `e2e/fix-retry-on-timeout` | Retry authorizes payment twice | CALL_ADDED, repeated (gRPC) | regression |
| `e2e/feat-order-notes-api-split` | Order detail stops reading Mongo notes | CALL_REMOVED | intended |
| `e2e/chore-auth-middleware-cleanup` | Order detail stops calling `/perms` | CALL_REMOVED (shared HTTP) | regression |
| `e2e/perf-cache-product-search` | Product search cached in Redis | CALL_ADDED + removed | intended |
| `e2e/perf-fix-n-plus-1` | Order items loaded in one `IN (…)` query | CALL_CHANGED, same response | intended |
| `e2e/feat-loyalty-points` | Order detail reads loyalty points; response gains `pointsEarned` | CALL_ADDED + RESPONSE_CHANGED | intended |
| `e2e/fix-timezone` | Order dates rendered in IST instead of UTC | RESPONSE_CHANGED | regression |
| `e2e/fix-currency-format` | DTO renames `amount` to `amt` | RESPONSE_CHANGED | regression |
| `e2e/feat-express-shipping` | Rates request gains `serviceLevel` | CALL_CHANGED (HTTP/2) | intended |
| `e2e/feat-order-events-v2` | Publishes to `order-events-v2` | CALL_ADDED + removed (Kafka) | intended |
| `e2e/chore-json-omitempty` | Response structs omit empty fields | RESPONSE_CHANGED (many tests) | regression |
| `e2e/chore-request-ids` | Random request id on outgoing pricing calls | VALUES_DRIFTED | not-code |
| `e2e/chore-config-cleanup` | A removed env var crashes the app at boot | APP_NO_RESPONSE | regression |
| `e2e/chore-logging-only` | Log lines only | no failures | control |
| `e2e/release-sprint-42` | gift wrap + session cache + timezone | three groups | mixed |
| `e2e/feat-half-star-ratings` | Ratings rounded to the nearest half star (ticket, API changelog) | RESPONSE_CHANGED (`~ average`) | intended |
| `e2e/refactor-coupon-errors` | "No behaviour change" refactor returns coupon errors by pointer, so `errors.As` misses them: 422 becomes 500 and checkout keeps its idempotency key | RESPONSE_CHANGED (422 → 500) + CALL_REMOVED (Redis `del`) | regression |
| `e2e/perf-account-one-query` | "One query" for the account joins only the default address | CALL_CHANGED (MySQL, `+ table addresses`) | regression |
| `e2e/feat-product-share-links` | Products gain a `url` field (additive) | no failures: Keploy accepts added fields | control |
| `e2e/release-sprint-43` | order notes split + share links + coupon errors + account query | four groups (the additive change never fails) | mixed |
| `e2e/release-sprint-44` | order notes split + half-star ratings + coupon errors + account query | five groups: two intended, three regressions (two from one bug) | mixed |

### `e2e/release-sprint-44`, the mixed release

Run `ed0c6d9a-9eb8-4b8a-94b5-579ce6bd25c6`: 10 of 118 tests fail in five groups.

| Group | Type | Tests | Endpoints | Cause | Verdict |
|---|---|---|---|---|---|
| MySQL `SELECT accounts` changed: `+ table addresses, ~ columns, ~ order` | CALL_CHANGED | 2 | GET /me, PUT /me/address | perf-account-one-query drops non-default addresses | regression |
| `Mongo find order_notes` no longer made | CALL_REMOVED | 2 | GET /orders/{id} | feat-order-notes-api-split (KART-430) | intended |
| `Redis del idem:checkout:{id}:{id}` no longer made | CALL_REMOVED | 1 | POST /checkout | refactor-coupon-errors: the coupon error path no longer releases the key | regression |
| `200 → 200 \| ~ average` | RESPONSE_CHANGED | 4 | GET /products/{id} | feat-half-star-ratings (KART-448) | intended |
| `422 → 500 \| ~ error, ~ message` | RESPONSE_CHANGED | 1 | POST /checkout/quote | refactor-coupon-errors | regression |

The agent should connect the last group and the Redis group to one commit,
and should not read the account query as intended just because its commit
says "perf": a performance change must not change the response.

## Scoring a run

For each branch record: what was downloaded (and its size), the groups and
types, whether each view shows the right evidence, the agent's explanation,
its verdicts, the tokens it used, and whether it asked before recording a
verdict and never edited a test or mock.
