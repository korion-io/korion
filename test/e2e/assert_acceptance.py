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
"""Assert the SuperHeros v0.1 acceptance criteria against a PlatformMap.

Reads the PlatformMap JSON (as returned by the controller's read API,
`GET /api/v1/platformmaps/{ns}/{name}`, or `kubectl get platformmap -o json`)
from a file path argument or stdin, then walks the CLAUDE.md v0.1 acceptance
checklist over `.status.topology` and `.status.conditions`.

Pure logic with no cluster dependency, so it is unit-testable on its own
(see assert_acceptance_test.py) and reused unchanged by run-acceptance.sh.
Exit code is 0 only if every non-skipped check passes.
"""

from __future__ import annotations

import json
import sys
from dataclasses import dataclass


# Node IDs are the stable keys the discovery engines emit (see
# internal/discovery/*.go). Cluster-scoped reports have an empty namespace
# segment, hence the doubled slash.
EXPECTED_DEPLOYMENTS = [
    "frontend",
    "catalog-v1",
    "catalog-v2",
    "catalog-v3",
    "inventory",
    "orders",
    "payment",
]
EXPECTED_SERVICES = ["frontend", "catalog", "inventory", "orders", "payment"]
NS = "superheros"

PASS, FAIL, SKIP = "PASS", "FAIL", "SKIP"


@dataclass
class Check:
    status: str
    title: str
    detail: str


def _node_index(topology: dict) -> dict[str, dict]:
    return {n.get("id"): n for n in topology.get("nodes") or []}


def _conditions(pm: dict) -> dict[str, dict]:
    conds = (pm.get("status") or {}).get("conditions") or []
    return {c.get("type"): c for c in conds}


def _types(nodes: dict[str, dict]) -> dict[str, int]:
    counts: dict[str, int] = {}
    for n in nodes.values():
        counts[n.get("type", "")] = counts.get(n.get("type", ""), 0) + 1
    return counts


def evaluate(pm: dict) -> list[Check]:
    """Return the ordered acceptance checklist for a PlatformMap object."""
    checks: list[Check] = []
    status = pm.get("status") or {}
    topology = status.get("topology") or {}
    nodes = _node_index(topology)
    edges = topology.get("edges") or []
    conds = _conditions(pm)

    # Precondition: the controller actually reconciled.
    if not status.get("lastDiscoveryTime"):
        checks.append(Check(FAIL, "Controller reconciled the PlatformMap",
                             "status.lastDiscoveryTime is unset -- no discovery ran"))
        return checks
    checks.append(Check(PASS, "Controller reconciled the PlatformMap",
                        f"lastDiscoveryTime={status['lastDiscoveryTime']}, "
                        f"{len(nodes)} nodes / {len(edges)} edges"))

    # 1. All six microservices present as nodes (catalog spans v1/v2/v3).
    missing_dep = [d for d in EXPECTED_DEPLOYMENTS
                   if f"deployment/{NS}/{d}" not in nodes]
    missing_svc = [s for s in EXPECTED_SERVICES
                   if f"service/{NS}/{s}" not in nodes]
    if not missing_dep and not missing_svc:
        checks.append(Check(PASS, "All 6 microservices on the canvas",
                            f"{len(EXPECTED_DEPLOYMENTS)} deployments + "
                            f"{len(EXPECTED_SERVICES)} services "
                            "(frontend, catalog[v1/v2/v3], inventory, orders, payment)"))
    else:
        checks.append(Check(FAIL, "All 6 microservices on the canvas",
                            f"missing deployments={missing_dep} services={missing_svc}"))

    # routes-to edges: frontend, inventory, orders, payment (1 each) + catalog (3).
    routes = [e for e in edges if e.get("type") == "routes-to"]
    if len(routes) >= 7:
        checks.append(Check(PASS, "Service->Deployment routing edges",
                            f"{len(routes)} routes-to edges (catalog Service fans out to 3 versions)"))
    else:
        checks.append(Check(FAIL, "Service->Deployment routing edges",
                            f"expected >=7 routes-to edges, found {len(routes)}"))

    # 2. ArgoCD node with sync status.
    argo = nodes.get(f"argocd-application/argocd/{NS}")
    if argo and (argo.get("metadata") or {}).get("syncStatus") == "Synced":
        md = argo["metadata"]
        checks.append(Check(PASS, "ArgoCD node shows sync status",
                            f"sync={md.get('syncStatus')} health={md.get('healthStatus')} "
                            f"node.status={argo.get('status')}"))
    else:
        checks.append(Check(FAIL, "ArgoCD node shows sync status",
                            f"argocd application node missing or not Synced: {argo}"))

    # 3. GitHub Actions node -- needs a real token, skipped in hermetic runs.
    gh_types = _types(nodes)
    if gh_types.get("github-actions-workflow", 0) > 0:
        checks.append(Check(PASS, "GitHub Actions node shows last run",
                            f"{gh_types['github-actions-workflow']} workflow node(s)"))
    else:
        checks.append(Check(SKIP, "GitHub Actions node shows last run",
                            "GitHub discovery disabled (needs API token) -- see ACCEPTANCE.md"))

    # 4. Istio traffic weights on catalog v1/v2/v3.
    catalog = nodes.get(f"service/{NS}/catalog") or {}
    weights = (catalog.get("metadata") or {}).get("istioTrafficWeights") or {}
    if {k: weights.get(k) for k in ("v1", "v2", "v3")} == {"v1": 20, "v2": 30, "v3": 50}:
        checks.append(Check(PASS, "Istio canary weights on catalog",
                            f"v1/v2/v3 = {weights.get('v1')}/{weights.get('v2')}/{weights.get('v3')}"))
    else:
        checks.append(Check(FAIL, "Istio canary weights on catalog",
                            f"expected v1/v2/v3=20/30/50, found {weights}"))
    # VirtualService / DestinationRule nodes themselves.
    if (f"istio-virtualservice/{NS}/catalog" in nodes
            and f"istio-destinationrule/{NS}/catalog" in nodes):
        checks.append(Check(PASS, "Istio VirtualService + DestinationRule nodes",
                            "catalog VirtualService and DestinationRule discovered"))
    else:
        checks.append(Check(FAIL, "Istio VirtualService + DestinationRule nodes",
                            "catalog VirtualService/DestinationRule node(s) missing"))

    # 5. Kyverno violation badge on the affected service.
    payment = nodes.get(f"deployment/{NS}/payment") or {}
    violations = (payment.get("metadata") or {}).get("policyViolations")
    if violations == 2:
        checks.append(Check(PASS, "Kyverno violation badge on payment",
                            f"policyViolations={violations} folded onto the payment Deployment node"))
    else:
        checks.append(Check(FAIL, "Kyverno violation badge on payment",
                            f"expected policyViolations=2 on payment, found {violations}"))
    report_types = _types(nodes)
    if report_types.get("kyverno-policyreport", 0) >= 1:
        checks.append(Check(PASS, "Kyverno PolicyReport node",
                            f"{report_types['kyverno-policyreport']} PolicyReport node(s)"))
    else:
        checks.append(Check(FAIL, "Kyverno PolicyReport node",
                            "no kyverno-policyreport node discovered"))

    # 6. ServiceDetails data present on a service node (metrics need Prometheus).
    fe = nodes.get(f"deployment/{NS}/frontend") or {}
    fe_md = fe.get("metadata") or {}
    if fe_md.get("image") and "replicasReady" in fe_md:
        checks.append(Check(PASS, "ServiceDetails data present on node click",
                            f"frontend image={fe_md.get('image')} "
                            f"ready={fe_md.get('replicasReady')}/{fe_md.get('replicasTotal')} "
                            "(live Prometheus metrics require a real endpoint -- see ACCEPTANCE.md)"))
    else:
        checks.append(Check(FAIL, "ServiceDetails data present on node click",
                            f"frontend node missing image/replica metadata: {fe_md}"))

    # 7. Deployment Timeline source data (ArgoCD sync history fields).
    argo_md = (argo or {}).get("metadata") or {}
    if argo_md.get("revision") and argo_md.get("lastSyncTime"):
        checks.append(Check(PASS, "Deployment Timeline source data",
                            f"ArgoCD revision={argo_md['revision'][:12]}... "
                            f"lastSyncTime={argo_md['lastSyncTime']}"))
    else:
        checks.append(Check(FAIL, "Deployment Timeline source data",
                            "ArgoCD node lacks revision/lastSyncTime for the timeline"))

    # Per-source discovery conditions (the four hermetically reproducible ones).
    for src in ("K8s", "ArgoCD", "Istio", "Kyverno"):
        c = conds.get(f"{src}Detected")
        if c and c.get("status") == "True":
            checks.append(Check(PASS, f"{src}Detected condition True",
                                c.get("message", "")))
        else:
            checks.append(Check(FAIL, f"{src}Detected condition True",
                                f"condition missing or not True: {c}"))

    return checks


def main(argv: list[str]) -> int:
    raw = open(argv[1]).read() if len(argv) > 1 else sys.stdin.read()
    pm = json.loads(raw)
    checks = evaluate(pm)

    icon = {PASS: "OK", FAIL: "XX", SKIP: "--"}
    print("\nKorion v0.1 acceptance -- SuperHeros platform\n" + "=" * 60)
    for c in checks:
        print(f"  [{icon[c.status]}] {c.status:4} {c.title}")
        if c.detail:
            print(f"           {c.detail}")
    n_fail = sum(1 for c in checks if c.status == FAIL)
    n_pass = sum(1 for c in checks if c.status == PASS)
    n_skip = sum(1 for c in checks if c.status == SKIP)
    print("=" * 60)
    print(f"  {n_pass} passed, {n_fail} failed, {n_skip} skipped\n")
    return 1 if n_fail else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
