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
"""Tests for assert_acceptance.evaluate over a synthetic PlatformMap.

Runnable either under pytest (`pytest test/e2e/assert_acceptance_test.py`) or
standalone (`python test/e2e/assert_acceptance_test.py`) with no third-party
dependency -- it mirrors the exact node shapes the Go discovery engines emit,
so it guards the acceptance-assertion logic independently of a live cluster.
"""

from assert_acceptance import evaluate, FAIL, PASS, SKIP


def _good_platformmap() -> dict:
    """A PlatformMap whose topology matches the e2e fixtures exactly."""
    nodes = []
    for d in ("frontend", "catalog-v1", "catalog-v2", "catalog-v3",
              "inventory", "orders", "payment"):
        md = {"namespace": "superheros", "image": f"img/{d}:v1",
              "replicasReady": 1, "replicasTotal": 1, "resourceKind": "Deployment"}
        nodes.append({"id": f"deployment/superheros/{d}", "type": "k8s-deployment",
                      "label": d, "status": "healthy", "metadata": md})
    for s in ("frontend", "catalog", "inventory", "orders", "payment"):
        md = {"namespace": "superheros", "resourceKind": "Service"}
        if s == "catalog":
            md["istioTrafficWeights"] = {"v1": 20, "v2": 30, "v3": 50}
        nodes.append({"id": f"service/superheros/{s}", "type": "k8s-service",
                      "label": s, "status": "healthy", "metadata": md})
    # kyverno enrichment onto payment
    for n in nodes:
        if n["id"] == "deployment/superheros/payment":
            n["metadata"].update({"policyViolations": 2, "policyViolationsFail": 2,
                                   "policyViolationsWarn": 0})
    nodes += [
        {"id": "argocd-application/argocd/superheros", "type": "argocd-application",
         "label": "superheros", "status": "healthy",
         "metadata": {"syncStatus": "Synced", "healthStatus": "Healthy",
                      "revision": "9f4c1a2e7b3d5680a1c2f3e4", "lastSyncTime": "2026-07-06T05:58:12Z"}},
        {"id": "istio-virtualservice/superheros/catalog", "type": "istio-virtualservice",
         "label": "catalog", "status": "healthy", "metadata": {}},
        {"id": "istio-destinationrule/superheros/catalog", "type": "istio-destinationrule",
         "label": "catalog", "status": "healthy", "metadata": {}},
        {"id": "kyverno-policyreport/superheros/superheros-namespace-report",
         "type": "kyverno-policyreport", "label": "superheros-namespace-report",
         "status": "degraded", "metadata": {"fail": 2}},
    ]
    edges = [{"from": f"service/superheros/{s}", "to": f"deployment/superheros/{d}",
              "type": "routes-to"}
             for s, d in [("frontend", "frontend"), ("catalog", "catalog-v1"),
                          ("catalog", "catalog-v2"), ("catalog", "catalog-v3"),
                          ("inventory", "inventory"), ("orders", "orders"),
                          ("payment", "payment")]]
    conds = [{"type": f"{s}Detected", "status": "True", "message": "ok"}
             for s in ("K8s", "ArgoCD", "Istio", "Kyverno")]
    return {"status": {"lastDiscoveryTime": "2026-07-06T05:58:20Z",
                       "topology": {"nodes": nodes, "edges": edges},
                       "conditions": conds}}


def test_good_platformmap_passes_all_and_skips_github():
    checks = evaluate(_good_platformmap())
    fails = [c for c in checks if c.status == FAIL]
    skips = [c for c in checks if c.status == SKIP]
    assert not fails, f"unexpected failures: {[(c.title, c.detail) for c in fails]}"
    # GitHub is the one expected SKIP in a hermetic run.
    assert len(skips) == 1 and "GitHub" in skips[0].title


def test_unreconciled_platformmap_fails_fast():
    checks = evaluate({"status": {}})
    assert len(checks) == 1 and checks[0].status == FAIL


def test_missing_istio_weights_fails():
    pm = _good_platformmap()
    for n in pm["status"]["topology"]["nodes"]:
        if n["id"] == "service/superheros/catalog":
            n["metadata"].pop("istioTrafficWeights")
    titles = {c.title: c.status for c in evaluate(pm)}
    assert titles["Istio canary weights on catalog"] == FAIL


def test_missing_kyverno_violations_fails():
    pm = _good_platformmap()
    for n in pm["status"]["topology"]["nodes"]:
        if n["id"] == "deployment/superheros/payment":
            n["metadata"].pop("policyViolations")
    titles = {c.title: c.status for c in evaluate(pm)}
    assert titles["Kyverno violation badge on payment"] == FAIL


def test_argocd_condition_false_fails():
    pm = _good_platformmap()
    for c in pm["status"]["conditions"]:
        if c["type"] == "ArgoCDDetected":
            c["status"] = "False"
    titles = {c.title: c.status for c in evaluate(pm)}
    assert titles["ArgoCDDetected condition True"] == FAIL


def _run_standalone() -> int:
    tests = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    failed = 0
    for t in tests:
        try:
            t()
            print(f"  ok   {t.__name__}")
        except AssertionError as e:
            failed += 1
            print(f"  FAIL {t.__name__}: {e}")
    print(f"\n{len(tests) - failed}/{len(tests)} passed")
    return 1 if failed else 0


if __name__ == "__main__":
    import sys
    sys.exit(_run_standalone())
