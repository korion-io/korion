#!/usr/bin/env bash
#
# Copyright 2026 The Korion Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Korion v0.1 acceptance harness (Phase 8). Stands up a Kind cluster, installs
# Korion via Helm, applies the SuperHeros fixtures (workloads + ArgoCD/Istio/
# Kyverno CRs), then polls the read API and asserts the CLAUDE.md v0.1
# acceptance criteria via assert_acceptance.py.
#
# This is the automatable form of Phase 8's acceptance walk-through: it is
# hermetic (GitHub + Prometheus discovery are disabled in the fixture
# PlatformMap, since they need a real token / endpoint -- see ACCEPTANCE.md for
# the live-cluster versions of those two criteria).
#
# Usage:
#   test/e2e/run-acceptance.sh [--keep] [--skip-build] [--cluster NAME] [--image IMG]
#
#   --keep         leave the Kind cluster running after the run (for debugging)
#   --skip-build   reuse an already-built/loaded controller image
#   --cluster      Kind cluster name (default: korion-e2e)
#   --image        controller image ref (default: korion/korion:e2e)

set -euo pipefail

CLUSTER="korion-e2e"
IMAGE="korion/korion:e2e"
KEEP=0
SKIP_BUILD=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --keep) KEEP=1 ;;
    --skip-build) SKIP_BUILD=1 ;;
    --cluster) CLUSTER="$2"; shift ;;
    --image) IMAGE="$2"; shift ;;
    *) echo "unknown flag: $1" >&2; exit 2 ;;
  esac
  shift
done

E2E_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$E2E_DIR/../.." && pwd)"
FIXTURES="$E2E_DIR/fixtures"
IMAGE_REPO="${IMAGE%:*}"
IMAGE_TAG="${IMAGE##*:}"

PF_PID=""
WORKDIR="$(mktemp -d)"

log() { printf '\n\033[1;36m==> %s\033[0m\n' "$*"; }

cleanup() {
  local rc=$?
  [[ -n "$PF_PID" ]] && kill "$PF_PID" >/dev/null 2>&1 || true
  if [[ "$KEEP" -eq 0 ]]; then
    log "Tearing down Kind cluster '$CLUSTER'"
    kind delete cluster --name "$CLUSTER" >/dev/null 2>&1 || true
  else
    log "Leaving cluster '$CLUSTER' running (--keep). Delete with: kind delete cluster --name $CLUSTER"
  fi
  rm -rf "$WORKDIR"
  exit $rc
}
trap cleanup EXIT

for tool in kind kubectl helm docker python; do
  command -v "$tool" >/dev/null 2>&1 || { echo "required tool not found: $tool" >&2; exit 1; }
done
PY="$(command -v python)"

# 1. Cluster ----------------------------------------------------------------
if kind get clusters 2>/dev/null | grep -qx "$CLUSTER"; then
  log "Reusing existing Kind cluster '$CLUSTER'"
else
  log "Creating Kind cluster '$CLUSTER'"
  kind create cluster --name "$CLUSTER" --wait 120s
fi

# 2. Controller image -------------------------------------------------------
if [[ "$SKIP_BUILD" -eq 0 ]]; then
  log "Building controller image '$IMAGE'"
  docker build -t "$IMAGE" "$ROOT"
fi
log "Loading '$IMAGE' into Kind"
kind load docker-image "$IMAGE" --name "$CLUSTER"

# 3. Optional-tool CRDs (ArgoCD/Istio/Kyverno) ------------------------------
log "Installing optional-tool CRDs"
kubectl apply -f "$FIXTURES/00-crds.yaml"
kubectl wait --for=condition=Established --timeout=60s \
  crd/applications.argoproj.io \
  crd/virtualservices.networking.istio.io \
  crd/destinationrules.networking.istio.io \
  crd/policyreports.wgpolicyk8s.io \
  crd/clusterpolicyreports.wgpolicyk8s.io

# 4. Install Korion via Helm (UI off -- this run asserts the read API) ------
log "Installing Korion via Helm"
helm upgrade --install korion "$ROOT/helm/korion" \
  --namespace korion-system --create-namespace \
  --set controller.image.repository="$IMAGE_REPO" \
  --set controller.image.tag="$IMAGE_TAG" \
  --set controller.image.pullPolicy=Never \
  --set ui.enabled=false \
  --wait --timeout 180s

# 5. SuperHeros fixtures ----------------------------------------------------
log "Applying SuperHeros workloads + ArgoCD/Istio/Kyverno fixtures"
kubectl apply -f "$FIXTURES/10-workloads.yaml"
kubectl apply -f "$FIXTURES/20-argocd.yaml"
kubectl apply -f "$FIXTURES/30-istio.yaml"
kubectl apply -f "$FIXTURES/40-kyverno.yaml"
kubectl -n superheros wait --for=condition=Available --timeout=120s deployment --all

# 6. Trigger discovery ------------------------------------------------------
log "Applying the PlatformMap"
kubectl apply -f "$FIXTURES/50-platformmap.yaml"

# 7. Port-forward the controller read API -----------------------------------
SVC="$(kubectl -n korion-system get svc -o name | grep controller | head -1)"
[[ -n "$SVC" ]] || { echo "controller Service not found" >&2; exit 1; }
log "Port-forwarding $SVC :8082"
kubectl -n korion-system port-forward "$SVC" 8082:8082 >/dev/null 2>&1 &
PF_PID=$!
sleep 3

# 8. Poll the read API until the first reconcile completes (<=60s budget) ---
API="http://127.0.0.1:8082/api/v1/platformmaps/superheros/superheros-platform"
OUT="$WORKDIR/platformmap.json"
log "Polling read API for a completed discovery (60s acceptance budget)"
START=$(date +%s)
DEADLINE=$((START + 90))
while :; do
  if curl -fsS "$API" -o "$OUT" 2>/dev/null && grep -q '"lastDiscoveryTime"' "$OUT"; then
    ELAPSED=$(( $(date +%s) - START ))
    echo "  discovery completed in ~${ELAPSED}s"
    break
  fi
  if [[ $(date +%s) -ge $DEADLINE ]]; then
    echo "timed out waiting for discovery; last response:" >&2
    cat "$OUT" >&2 || true
    exit 1
  fi
  sleep 3
done

# 9. Assert the v0.1 acceptance checklist -----------------------------------
"$PY" "$E2E_DIR/assert_acceptance.py" "$OUT"
