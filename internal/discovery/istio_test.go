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

func newVirtualService(ns, name string, hosts []string, routes []map[string]any) *unstructured.Unstructured {
	httpRoutes := []any{
		map[string]any{"route": routesToAny(routes)},
	}
	hostsAny := make([]any, len(hosts))
	for i, h := range hosts {
		hostsAny[i] = h
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "networking.istio.io/v1beta1",
		"kind":       "VirtualService",
		"metadata":   map[string]any{"name": name, "namespace": ns},
		"spec": map[string]any{
			"hosts": hostsAny,
			"http":  httpRoutes,
		},
	}}
}

func routesToAny(routes []map[string]any) []any {
	out := make([]any, len(routes))
	for i, r := range routes {
		out[i] = r
	}
	return out
}

func routeDestination(host, subset string, weight int64) map[string]any {
	return map[string]any{
		"destination": map[string]any{"host": host, "subset": subset},
		"weight":      weight,
	}
}

func newDestinationRule(ns, name, host string, subsetNames []string) *unstructured.Unstructured {
	subsets := make([]any, len(subsetNames))
	for i, s := range subsetNames {
		subsets[i] = map[string]any{"name": s, "labels": map[string]any{"version": s}}
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "networking.istio.io/v1beta1",
		"kind":       "DestinationRule",
		"metadata":   map[string]any{"name": name, "namespace": ns},
		"spec": map[string]any{
			"host":    host,
			"subsets": subsets,
		},
	}}
}

func TestIstioDiscoverer_Discover(t *testing.T) {
	ns := "superheros"
	scheme := runtime.NewScheme()
	gvrListKind := map[schema.GroupVersionResource]string{
		istioVirtualServiceGVR:  "VirtualServiceList",
		istioDestinationRuleGVR: "DestinationRuleList",
	}
	vs := newVirtualService(ns, "catalog", []string{"catalog"}, []map[string]any{
		routeDestination("catalog", "v1", 20),
		routeDestination("catalog", "v2", 30),
		routeDestination("catalog", "v3", 50),
	})
	dr := newDestinationRule(ns, "catalog", "catalog", []string{"v1", "v2", "v3"})

	dynClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrListKind, vs, dr)

	d := &IstioDiscoverer{
		Dynamic:   dynClient,
		Discovery: discoveryClientWithGroupVersions(istioGroupVersion),
	}
	pm := &v1alpha1.PlatformMap{Spec: v1alpha1.PlatformMapSpec{Namespace: ns}}

	result := d.Discover(context.Background(), pm)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	byID := make(map[string]int)
	for _, n := range result.Nodes {
		byID[n.ID]++
	}

	if byID["istio-virtualservice/superheros/catalog"] != 1 {
		t.Errorf("expected exactly one VirtualService node, got %d", byID["istio-virtualservice/superheros/catalog"])
	}
	if byID["istio-destinationrule/superheros/catalog"] != 1 {
		t.Errorf("expected exactly one DestinationRule node, got %d", byID["istio-destinationrule/superheros/catalog"])
	}

	serviceEnrichmentID := serviceNodeID(ns, "catalog")
	if byID[serviceEnrichmentID] != 1 {
		t.Fatalf("expected one traffic-weight enrichment node for %q, got %d", serviceEnrichmentID, byID[serviceEnrichmentID])
	}

	var enrichment map[string]int64
	for _, n := range result.Nodes {
		if n.ID == serviceEnrichmentID {
			w, ok := n.Metadata["istioTrafficWeights"].(map[string]int64)
			if !ok {
				t.Fatalf("istioTrafficWeights has wrong type: %T", n.Metadata["istioTrafficWeights"])
			}
			enrichment = w
		}
	}
	want := map[string]int64{"v1": 20, "v2": 30, "v3": 50}
	for subset, weight := range want {
		if enrichment[subset] != weight {
			t.Errorf("weight for subset %q = %d, want %d", subset, enrichment[subset], weight)
		}
	}
}

func TestIstioDiscoverer_Discover_CRDNotInstalled(t *testing.T) {
	d := &IstioDiscoverer{
		Dynamic:   dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		Discovery: discoveryClientWithGroupVersions(),
	}
	pm := &v1alpha1.PlatformMap{Spec: v1alpha1.PlatformMapSpec{Namespace: "superheros"}}

	result := d.Discover(context.Background(), pm)

	if !errors.Is(result.Err, ErrCRDNotInstalled) {
		t.Fatalf("expected ErrCRDNotInstalled, got %v", result.Err)
	}
}
