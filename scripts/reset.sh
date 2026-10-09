#!/usr/bin/env bash
# Puts every datastore back to its seed so scripts/traffic.sh records the
# same flows again. Datastores keep no volumes, so restarting them reseeds.
#
#   scripts/reset.sh compose   # local docker compose stack
#   scripts/reset.sh k8s       # the kartly namespace in the current cluster
set -euo pipefail
case "${1:-compose}" in
  compose)
    cd "$(dirname "$0")/../deploy/compose"
    docker compose rm -sfv postgres mysql mongo redis kafka
    docker compose up -d postgres mysql mongo redis kafka
    echo "waiting for the databases to seed"; sleep 25
    docker compose up -d --force-recreate orders-api
    ;;
  k8s)
    kubectl -n kartly rollout restart deploy/postgres deploy/mysql deploy/mongo deploy/redis deploy/kafka
    kubectl -n kartly rollout status deploy/postgres deploy/mysql deploy/mongo deploy/redis deploy/kafka --timeout=180s
    sleep 20
    kubectl -n kartly rollout restart deploy/orders-api
    kubectl -n kartly rollout status deploy/orders-api --timeout=120s
    ;;
  *) echo "usage: $0 compose|k8s" >&2; exit 2 ;;
esac
