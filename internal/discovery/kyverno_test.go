/*
Copyright 2026 The Korion Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package discovery

import (
	"context"
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	"github.com/korion-io/korion/api/v1alpha1"
)

func newPolicyReport(ns, name string, pass, fail, warn, errCount int64, results []map[string]any) *unstructured.Unstructured {
	resultsAny := make([]any, len(results))
	for i, r := range results {
		resultsAny[i] = r
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "wgpolicyk8s.io/v1alpha2",
		"kind":       "PolicyReport",
		"metadata":   map[string]any{"name": name, "namespace": ns},
		"summary": map[string]any{
			"pass": pass, "fail": fail, "warn": warn, "error": errCount, "skip": int64(0),
		},
		"results": resultsAny,
	}}
}

func policyResult(outcome, resourceKind, resourceNamespace, resourceName string) map[string]any {
	return map[string]any{
		"policy": "require-labels",
		"rule":   "check-team-label",
		"result": outcome,
		"resources": []any{
			map[string]any{"apiVersion": "apps/v1", "kind": resourceKind, "namespace": resourceNamespace, "name": resourceName},
		},
	}
}

func TestKyvernoDiscoverer_Discover(t *testing.T) {
	ns := "superheros"
	scheme := runtime.NewScheme()
	gvrListKind := map[schema.GroupVersionResource]string{
		policyReportGVR:        "PolicyReportList",
		clusterPolicyReportGVR: "ClusterPolicyReportList",
	}
	pr := newPolicyReport(ns, "catalog-report", 5, 2, 1, 0, []map[string]any{
		policyResult("fail", "Deployment", ns, "catalog"),
		policyResult("fail", "Deployment", ns, "catalog"),
		policyResult("warn", "Deployment", ns, "orders"),
		policyResult("pass", "Deployment", ns, "orders"),
	})

	dynClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrListKind, pr)

	d := &KyvernoDiscoverer{
		Dynamic:   dynClient,
		Discovery: discoveryClientWithGroupVersions(kyvernoGroupVersion),
	}
	pm := &v1alpha1.PlatformMap{Spec: v1alpha1.PlatformMapSpec{Namespace: ns}}

	result := d.Discover(context.Background(), pm)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	nodesByID := make(map[string]int)
	for _, n := range result.Nodes {
		nodesByID[n.ID]++
	}

	if nodesByID["kyverno-policyreport/superheros/catalog-report"] != 1 {
		t.Fatalf("expected exactly one report node, got %d", nodesByID["kyverno-policyreport/superheros/catalog-report"])
	}

	for _, n := range result.Nodes {
		switch n.ID {
		case deploymentNodeID(ns, "catalog"):
			if n.Metadata["policyViolations"] != int64(2) {
				t.Errorf("catalog policyViolations = %v, want 2", n.Metadata["policyViolations"])
			}
		case deploymentNodeID(ns, "orders"):
			if n.Metadata["policyViolationsWarn"] != int64(1) {
				t.Errorf("orders policyViolationsWarn = %v, want 1", n.Metadata["policyViolationsWarn"])
			}
			if n.Metadata["policyViolations"] != int64(0) {
				t.Errorf("orders policyViolations (fail+error) = %v, want 0", n.Metadata["policyViolations"])
			}
		}
	}
}

func TestKyvernoDiscoverer_Discover_CRDNotInstalled(t *testing.T) {
	d := &KyvernoDiscoverer{
		Dynamic:   dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		Discovery: discoveryClientWithGroupVersions(),
	}
	pm := &v1alpha1.PlatformMap{Spec: v1alpha1.PlatformMapSpec{Namespace: "superheros"}}

	result := d.Discover(context.Background(), pm)

	if !errors.Is(result.Err, ErrCRDNotInstalled) {
		t.Fatalf("expected ErrCRDNotInstalled, got %v", result.Err)
	}
}
