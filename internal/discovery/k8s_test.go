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
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/korion-io/korion/api/v1alpha1"
	"github.com/korion-io/korion/internal/graph"
)

func int32Ptr(i int32) *int32 { return &i }

func newDeployment(ns, name string, desired, ready int32, labels map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(desired),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Image: "example/" + name + ":v1"}},
				},
			},
		},
		Status: appsv1.DeploymentStatus{ReadyReplicas: ready, Replicas: desired},
	}
}

func newService(ns, name string, selector map[string]string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec:       corev1.ServiceSpec{Selector: selector, ClusterIP: "10.0.0.1"},
	}
}

func TestK8sDiscoverer_Discover(t *testing.T) {
	ns := "superheros"
	labels := map[string]string{"app": "catalog"}

	clientset := fake.NewClientset(
		newDeployment(ns, "catalog", 2, 2, labels),
		newDeployment(ns, "orders", 1, 0, map[string]string{"app": "orders"}),
		newService(ns, "catalog", labels),
		newService(ns, "unrelated-namespace-noise", nil),
	)

	d := &K8sDiscoverer{Clientset: clientset}
	pm := &v1alpha1.PlatformMap{Spec: v1alpha1.PlatformMapSpec{Namespace: ns}}

	result := d.Discover(context.Background(), pm)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Source != "K8s" {
		t.Errorf("Source = %q, want %q", result.Source, "K8s")
	}
	if len(result.Nodes) != 4 {
		t.Fatalf("got %d nodes, want 4: %+v", len(result.Nodes), result.Nodes)
	}

	byID := make(map[string]bool)
	for _, n := range result.Nodes {
		byID[n.ID] = true
	}
	for _, want := range []string{
		"deployment/superheros/catalog",
		"deployment/superheros/orders",
		"service/superheros/catalog",
		"service/superheros/unrelated-namespace-noise",
	} {
		if !byID[want] {
			t.Errorf("missing expected node %q", want)
		}
	}

	if len(result.Edges) != 1 {
		t.Fatalf("got %d edges, want 1 (catalog service -> catalog deployment): %+v", len(result.Edges), result.Edges)
	}
	if result.Edges[0].From != "service/superheros/catalog" || result.Edges[0].To != "deployment/superheros/catalog" {
		t.Errorf("unexpected edge: %+v", result.Edges[0])
	}

	for _, n := range result.Nodes {
		switch n.ID {
		case "deployment/superheros/catalog":
			if n.Status != "healthy" {
				t.Errorf("catalog deployment status = %q, want healthy", n.Status)
			}
		case "deployment/superheros/orders":
			if n.Status != "degraded" {
				t.Errorf("orders deployment status = %q, want degraded", n.Status)
			}
		}
	}
}

func TestK8sDiscoverer_Discover_NamespaceScoped(t *testing.T) {
	clientset := fake.NewClientset(
		newDeployment("superheros", "catalog", 1, 1, nil),
		newDeployment("other-namespace", "leaky", 1, 1, nil),
	)

	d := &K8sDiscoverer{Clientset: clientset}
	pm := &v1alpha1.PlatformMap{Spec: v1alpha1.PlatformMapSpec{Namespace: "superheros"}}

	result := d.Discover(context.Background(), pm)

	if len(result.Nodes) != 1 {
		t.Fatalf("got %d nodes, want 1 (scoped to superheros namespace only): %+v", len(result.Nodes), result.Nodes)
	}
	if result.Nodes[0].ID != "deployment/superheros/catalog" {
		t.Errorf("got node %q, want deployment/superheros/catalog", result.Nodes[0].ID)
	}
}

func TestK8sDiscoverer_Discover_EmptyNamespace(t *testing.T) {
	clientset := fake.NewClientset()
	d := &K8sDiscoverer{Clientset: clientset}
	pm := &v1alpha1.PlatformMap{Spec: v1alpha1.PlatformMapSpec{Namespace: "empty"}}

	result := d.Discover(context.Background(), pm)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Nodes) != 0 || len(result.Edges) != 0 {
		t.Errorf("expected empty result, got %+v", result)
	}
}

// TestK8sDiscoverer_MatchesFrozenTopologyContract runs the real Phase 2
// discovery pipeline (K8sDiscoverer.Discover -> graph.Merge) against a
// fixture cluster shaped to match internal/graph/testdata/sample-topology.json,
// and asserts the resulting JSON is structurally identical to that frozen
// fixture. This is what proves the committed contract isn't just a
// hand-typed literal in internal/graph's own tests -- the actual discovery
// engine plus builder pipeline produces it.
func TestK8sDiscoverer_MatchesFrozenTopologyContract(t *testing.T) {
	ns := "superheros"
	labels := map[string]string{"app": "catalog"}

	dep := newDeployment(ns, "catalog", 2, 2, labels)
	dep.Generation = 1

	svc := newService(ns, "catalog", labels)
	svc.Spec.Type = corev1.ServiceTypeClusterIP

	clientset := fake.NewClientset(dep, svc)
	d := &K8sDiscoverer{Clientset: clientset}
	pm := &v1alpha1.PlatformMap{Spec: v1alpha1.PlatformMapSpec{Namespace: ns}}

	result := d.Discover(context.Background(), pm)
	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}

	merged := graph.Merge(graph.Graph{Nodes: result.Nodes, Edges: result.Edges})

	gotJSON, err := json.Marshal(merged)
	if err != nil {
		t.Fatalf("marshaling merged graph: %v", err)
	}

	fixtureBytes, err := os.ReadFile("../graph/testdata/sample-topology.json")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	var gotGeneric, wantGeneric any
	if err := json.Unmarshal(gotJSON, &gotGeneric); err != nil {
		t.Fatalf("unmarshaling built graph JSON: %v", err)
	}
	if err := json.Unmarshal(fixtureBytes, &wantGeneric); err != nil {
		t.Fatalf("unmarshaling fixture: %v", err)
	}

	if !reflect.DeepEqual(gotGeneric, wantGeneric) {
		t.Errorf("K8sDiscoverer + graph.Merge output does not match frozen fixture ../graph/testdata/sample-topology.json\ngot:  %s\nwant: %s", gotJSON, fixtureBytes)
	}
}

func TestK8sDiscoverer_Discover_ListError(t *testing.T) {
	clientset := fake.NewClientset()
	clientset.PrependReactor("list", "deployments", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("simulated API server failure")
	})

	d := &K8sDiscoverer{Clientset: clientset}
	pm := &v1alpha1.PlatformMap{Spec: v1alpha1.PlatformMapSpec{Namespace: "superheros"}}

	result := d.Discover(context.Background(), pm)

	if result.Err == nil {
		t.Fatal("expected an error, got nil")
	}
	if len(result.Nodes) != 0 {
		t.Errorf("expected no nodes on a failed list, got %+v", result.Nodes)
	}
}
