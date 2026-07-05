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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sdiscovery "k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"

	"github.com/korion-io/korion/api/v1alpha1"
	"github.com/korion-io/korion/internal/graph"
)

// kyvernoGroupVersion is checked via the vendor-neutral wgpolicyk8s.io API
// group (which Kyverno populates), not a Kyverno-specific CRD -- see
// decisions.md. This keeps discovery policy-engine-neutral so a future
// OPA/Gatekeeper install (which also writes PolicyReports) works unmodified.
const kyvernoGroupVersion = "wgpolicyk8s.io/v1alpha2"

var (
	policyReportGVR = schema.GroupVersionResource{
		Group:    "wgpolicyk8s.io",
		Version:  "v1alpha2",
		Resource: "policyreports",
	}
	clusterPolicyReportGVR = schema.GroupVersionResource{
		Group:    "wgpolicyk8s.io",
		Version:  "v1alpha2",
		Resource: "clusterpolicyreports",
	}
)

// KyvernoDiscoverer discovers wgpolicyk8s.io PolicyReports/ClusterPolicyReports,
// emitting a node per report plus enrichment nodes carrying violation counts
// onto the k8s-deployment node each result's resource ref points at (so the
// canvas can show a violation-count badge directly on the affected service).
type KyvernoDiscoverer struct {
	Dynamic   dynamic.Interface
	Discovery k8sdiscovery.DiscoveryInterface
}

func (d *KyvernoDiscoverer) Name() string { return "Kyverno" }

// violationTally accumulates fail/error/warn counts targeting one Deployment
// across every PolicyReport result that references it.
type violationTally struct {
	Fail, Error, Warn int64
}

func (d *KyvernoDiscoverer) Discover(ctx context.Context, pm *v1alpha1.PlatformMap) DiscoveryResult {
	result := DiscoveryResult{Source: d.Name()}
	ns := pm.Spec.Namespace

	if !GroupVersionAvailable(d.Discovery, kyvernoGroupVersion) {
		result.Err = ErrCRDNotInstalled
		return result
	}

	tally := make(map[string]*violationTally)

	prList, err := d.Dynamic.Resource(policyReportGVR).Namespace(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		result.Err = fmt.Errorf("listing kyverno policyreports in %q: %w", ns, err)
		return result
	}
	for _, item := range prList.Items {
		result.Nodes = append(result.Nodes, policyReportNode(item, "kyverno-policyreport"))
		accumulateViolations(item, tally)
	}

	cprList, err := d.Dynamic.Resource(clusterPolicyReportGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		result.Err = fmt.Errorf("listing kyverno clusterpolicyreports: %w", err)
		return result
	}
	for _, item := range cprList.Items {
		result.Nodes = append(result.Nodes, policyReportNode(item, "kyverno-clusterpolicyreport"))
		accumulateViolations(item, tally)
	}

	for nodeID, v := range tally {
		result.Nodes = append(result.Nodes, graph.Node{
			ID: nodeID,
			Metadata: map[string]any{
				"policyViolations":      v.Fail + v.Error,
				"policyViolationsFail":  v.Fail,
				"policyViolationsError": v.Error,
				"policyViolationsWarn":  v.Warn,
			},
		})
	}

	return result
}

func policyReportNode(report unstructured.Unstructured, nodeType string) graph.Node {
	pass, _, _ := unstructured.NestedInt64(report.Object, "summary", "pass")
	fail, _, _ := unstructured.NestedInt64(report.Object, "summary", "fail")
	warn, _, _ := unstructured.NestedInt64(report.Object, "summary", "warn")
	errCount, _, _ := unstructured.NestedInt64(report.Object, "summary", "error")
	skip, _, _ := unstructured.NestedInt64(report.Object, "summary", "skip")

	status := "healthy"
	if fail > 0 || errCount > 0 {
		status = "degraded"
	} else if warn > 0 {
		status = "unknown"
	}

	return graph.Node{
		ID:     fmt.Sprintf("%s/%s/%s", nodeType, report.GetNamespace(), report.GetName()),
		Type:   nodeType,
		Label:  report.GetName(),
		Status: status,
		Metadata: map[string]any{
			"namespace":    report.GetNamespace(),
			"pass":         pass,
			"fail":         fail,
			"warn":         warn,
			"error":        errCount,
			"skip":         skip,
			"resourceKind": report.GetKind(),
		},
	}
}

// accumulateViolations walks a PolicyReport/ClusterPolicyReport's results,
// and for any result whose target resource is a Deployment, tallies its
// fail/error/warn count onto that Deployment's stable node ID -- the join
// key K8sDiscoverer already uses, so graph.Merge enriches the existing node
// rather than creating a duplicate.
func accumulateViolations(report unstructured.Unstructured, tally map[string]*violationTally) {
	results, found, _ := unstructured.NestedSlice(report.Object, "results")
	if !found {
		return
	}
	for _, r := range results {
		rMap, ok := r.(map[string]any)
		if !ok {
			continue
		}
		outcome, _, _ := unstructured.NestedString(rMap, "result")
		if outcome != "fail" && outcome != "error" && outcome != "warn" {
			continue
		}
		resources, found, _ := unstructured.NestedSlice(rMap, "resources")
		if !found {
			continue
		}
		for _, res := range resources {
			resMap, ok := res.(map[string]any)
			if !ok {
				continue
			}
			kind, _, _ := unstructured.NestedString(resMap, "kind")
			if kind != "Deployment" {
				continue
			}
			name, _, _ := unstructured.NestedString(resMap, "name")
			resNamespace, _, _ := unstructured.NestedString(resMap, "namespace")
			if resNamespace == "" {
				resNamespace = report.GetNamespace()
			}
			if name == "" {
				continue
			}

			nodeID := deploymentNodeID(resNamespace, name)
			if tally[nodeID] == nil {
				tally[nodeID] = &violationTally{}
			}
			switch outcome {
			case "fail":
				tally[nodeID].Fail++
			case "error":
				tally[nodeID].Error++
			case "warn":
				tally[nodeID].Warn++
			}
		}
	}
}
