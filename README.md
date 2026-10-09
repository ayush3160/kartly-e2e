# Kartly e2e: a test bed for Keploy cloud triage

Kartly is a realistic e-commerce **orders service** and a set of branches that
each carry one change a developer would ship in a normal sprint. It exists to
test, end to end, how `keploy cloud replay` failures are downloaded, grouped
and explained by `keploy cloud triage`, and whether a coding agent can tell an
intended change from a regression.

Plan and scoring: <https://claude.ai/artifact/8wfzYRQc3oZpPogUvbnjo7>

## The service

`orders-api` (Go) owns account, catalog, cart, checkout, orders, returns,
admin and webhooks. It calls every protocol the triage engine reports on:

| Dependency | Protocol | Used for |
|---|---|---|
| catalog-svc, pricing-svc, auth-svc, flags-svc | HTTP/JSON | products, quotes, tokens and `/perms`, feature flags |
| payments, inventory | gRPC | authorize/capture/refund, reserve/release stock |
| shipping-partner | HTTP/2 (h2c) | rates, labels |
| Postgres | SQL | orders, items, coupons, returns, outbox, webhooks, audit |
| MySQL | SQL (text protocol) | legacy accounts and addresses |
| Mongo | documents | carts, order notes, reviews |
| Redis | cache | product cache, sessions, idempotency keys, rate limits |
| Kafka | events | `order-events`, `email-requests` |

The downstream services are deterministic stubs (`cmd/stubs`), so the same
request always gets the same answer.

## Recording main (once, locally)

1. Deploy to a cluster connected to Keploy (kind works):
   `KIND_CLUSTER=kartly scripts/deploy-kind.sh`
2. Start recording `kartly.orders-api` with `keploy k8s`.
3. Drive the traffic: `kubectl -n kartly port-forward svc/orders-api 8080:8080`,
   then `BASE_URL=http://localhost:8080 scripts/traffic.sh`. It sends 140
   requests in 8 test sets and checks each status code. To record a set on its
   own, pass its name: `scripts/traffic.sh checkout`.
4. Stop recording. To record again, reseed first: `scripts/reset.sh k8s`.

Recording never runs in CI.

## Replaying a branch (CI)

`.github/workflows/cloud-replay.yml` runs on every pull request: it builds
the PR's `orders-api` image and runs

```
keploy cloud replay --app kartly.orders-api --cluster $KEPLOY_CLUSTER --image <PR image>
```

Set the `KEPLOY_API_KEY` secret and the `KEPLOY_CLUSTER` variable in the repo.
When the replay fails, start a fresh coding-agent session on the branch and
give it only the `keploy cloud triage --app kartly.orders-api --run <id>`
command the job prints.

## Scenario branches

Each `e2e/*` branch is one pull request against `main`. See
[docs/scenarios.md](docs/scenarios.md) for what each changes, the group the
triage should produce, and the verdict a reviewer would give.

## Local run

```
cd deploy/compose && docker compose up -d --build
BASE_URL=http://localhost:8080 scripts/traffic.sh
```
