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

const istioGroupVersion = "networking.istio.io/v1beta1"

var (
	istioVirtualServiceGVR = schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "virtualservices",
	}
	istioDestinationRuleGVR = schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "destinationrules",
	}
)

// IstioDiscoverer discovers Istio VirtualServices and DestinationRules in the
// PlatformMap's target namespace. It emits a node per VirtualService/
// DestinationRule, and additionally enriches the k8s-service node each
// VirtualService routes to with the traffic-weight split across subsets
// (e.g. catalog v1/v2/v3 canary weights) -- an enrichment node sharing the
// service's stable ID, joined by graph.Merge rather than duplicated.
type IstioDiscoverer struct {
	Dynamic   dynamic.Interface
	Discovery k8sdiscovery.DiscoveryInterface
}

func (d *IstioDiscoverer) Name() string { return "Istio" }

func (d *IstioDiscoverer) Discover(ctx context.Context, pm *v1alpha1.PlatformMap) DiscoveryResult {
	result := DiscoveryResult{Source: d.Name()}
	ns := pm.Spec.Namespace

	if !GroupVersionAvailable(d.Discovery, istioGroupVersion) {
		result.Err = ErrCRDNotInstalled
		return result
	}

	vsList, err := d.Dynamic.Resource(istioVirtualServiceGVR).Namespace(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		result.Err = fmt.Errorf("listing istio virtualservices in %q: %w", ns, err)
		return result
	}

	weightsByHost := make(map[string]map[string]int64)
	for _, item := range vsList.Items {
		result.Nodes = append(result.Nodes, virtualServiceNode(item))
		collectTrafficWeights(item, weightsByHost)
	}
	for host, weights := range weightsByHost {
		result.Nodes = append(result.Nodes, graph.Node{
			ID:       serviceNodeID(ns, host),
			Metadata: map[string]any{"istioTrafficWeights": weights},
		})
	}

	drList, err := d.Dynamic.Resource(istioDestinationRuleGVR).Namespace(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		result.Err = fmt.Errorf("listing istio destinationrules in %q: %w", ns, err)
		return result
	}
	for _, item := range drList.Items {
		result.Nodes = append(result.Nodes, destinationRuleNode(item))
	}

	return result
}

// collectTrafficWeights extracts spec.http[].route[].{destination.host,
// destination.subset, weight} from a VirtualService and accumulates a
// per-host, per-subset weight map -- the shape the canvas needs to show
// catalog v1/v2/v3 canary traffic splits on the underlying Service node.
func collectTrafficWeights(vs unstructured.Unstructured, weightsByHost map[string]map[string]int64) {
	httpRoutes, found, _ := unstructured.NestedSlice(vs.Object, "spec", "http")
	if !found {
		return
	}
	for _, hr := range httpRoutes {
		hrMap, ok := hr.(map[string]any)
		if !ok {
			continue
		}
		routes, found, _ := unstructured.NestedSlice(hrMap, "route")
		if !found {
			continue
		}
		for _, r := range routes {
			rMap, ok := r.(map[string]any)
			if !ok {
				continue
			}
			host, _, _ := unstructured.NestedString(rMap, "destination", "host")
			subset, _, _ := unstructured.NestedString(rMap, "destination", "subset")
			weight, _, _ := unstructured.NestedInt64(rMap, "weight")
			if host == "" || subset == "" {
				continue
			}
			if weightsByHost[host] == nil {
				weightsByHost[host] = make(map[string]int64)
			}
			weightsByHost[host][subset] = weight
		}
	}
}

func virtualServiceNode(vs unstructured.Unstructured) graph.Node {
	hosts, _, _ := unstructured.NestedStringSlice(vs.Object, "spec", "hosts")

	return graph.Node{
		ID:     istioVirtualServiceNodeID(vs.GetNamespace(), vs.GetName()),
		Type:   "istio-virtualservice",
		Label:  vs.GetName(),
		Status: "healthy",
		Metadata: map[string]any{
			"namespace":    vs.GetNamespace(),
			"hosts":        hosts,
			"resourceKind": "VirtualService",
		},
	}
}

func destinationRuleNode(dr unstructured.Unstructured) graph.Node {
	host, _, _ := unstructured.NestedString(dr.Object, "spec", "host")
	subsets, _, _ := unstructured.NestedSlice(dr.Object, "spec", "subsets")

	subsetNames := make([]string, 0, len(subsets))
	for _, s := range subsets {
		if sMap, ok := s.(map[string]any); ok {
			if name, _, _ := unstructured.NestedString(sMap, "name"); name != "" {
				subsetNames = append(subsetNames, name)
			}
		}
	}

	return graph.Node{
		ID:     istioDestinationRuleNodeID(dr.GetNamespace(), dr.GetName()),
		Type:   "istio-destinationrule",
		Label:  dr.GetName(),
		Status: "healthy",
		Metadata: map[string]any{
			"namespace":    dr.GetNamespace(),
			"host":         host,
			"subsets":      subsetNames,
			"resourceKind": "DestinationRule",
		},
	}
}

func istioVirtualServiceNodeID(namespace, name string) string {
	return fmt.Sprintf("istio-virtualservice/%s/%s", namespace, name)
}

func istioDestinationRuleNodeID(namespace, name string) string {
	return fmt.Sprintf("istio-destinationrule/%s/%s", namespace, name)
}
