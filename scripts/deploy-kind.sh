#!/usr/bin/env bash
# Builds the images, loads them into a kind cluster and deploys Kartly into
# the kartly namespace: the setup to record main on.
#
#   KIND_CLUSTER=kartly scripts/deploy-kind.sh
set -euo pipefail
cluster=${KIND_CLUSTER:-kartly}
tag=${TAG:-main}
cd "$(dirname "$0")/.."
kind get clusters | grep -qx "$cluster" || kind create cluster --name "$cluster"
docker build --target orders-api -t "kartly/orders-api:$tag" .
docker build --target stubs -t "kartly/stubs:$tag" .
kind load docker-image --name "$cluster" "kartly/orders-api:$tag" "kartly/stubs:$tag"
kubectl apply -f deploy/k8s/
kubectl -n kartly rollout status deploy --timeout=300s
echo "port-forward with: kubectl -n kartly port-forward svc/orders-api 8080:8080"
