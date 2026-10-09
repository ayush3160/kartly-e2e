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

## Scoring a run

For each branch record: what was downloaded (and its size), the groups and
types, whether each view shows the right evidence, the agent's explanation,
its verdicts, the tokens it used, and whether it asked before recording a
verdict and never edited a test or mock.
