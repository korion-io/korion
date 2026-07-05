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

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	korionv1alpha1 "github.com/korion-io/korion/api/v1alpha1"
	"github.com/korion-io/korion/internal/discovery"
	"github.com/korion-io/korion/internal/graph"
)

// stubDiscoverer is a Discoverer test double that returns a fixed result,
// so controller tests exercise the reconcile/merge/status-write logic
// without depending on a real (or fake) Kubernetes API for Deployments and
// Services -- that's internal/discovery's own test responsibility.
type stubDiscoverer struct {
	name   string
	result discovery.DiscoveryResult
}

func (s *stubDiscoverer) Name() string { return s.name }
func (s *stubDiscoverer) Discover(_ context.Context, _ *korionv1alpha1.PlatformMap) discovery.DiscoveryResult {
	return s.result
}

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := korionv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("adding scheme: %v", err)
	}
	return scheme
}

func TestReconcile_MergesDiscoveryResultsIntoStatus(t *testing.T) {
	scheme := newScheme(t)
	pm := &korionv1alpha1.PlatformMap{
		ObjectMeta: metav1.ObjectMeta{Name: "superheros-platform", Namespace: "superheros"},
		Spec: korionv1alpha1.PlatformMapSpec{
			Namespace:    "superheros",
			AutoDiscover: true,
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pm).WithStatusSubresource(pm).Build()

	r := &PlatformMapReconciler{
		Client: c,
		Scheme: scheme,
		Discoverers: []discovery.Discoverer{
			&stubDiscoverer{name: "k8s", result: discovery.DiscoveryResult{
				Source: "k8s",
				Nodes:  []graph.Node{{ID: "deployment/superheros/catalog", Type: "k8s-deployment", Status: "healthy"}},
			}},
		},
	}

	res, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "superheros", Name: "superheros-platform"}})
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	if res.RequeueAfter <= 0 {
		t.Errorf("expected a positive RequeueAfter, got %v", res.RequeueAfter)
	}

	var got korionv1alpha1.PlatformMap
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "superheros", Name: "superheros-platform"}, &got); err != nil {
		t.Fatalf("fetching updated PlatformMap: %v", err)
	}

	if got.Status.Topology == nil {
		t.Fatal("status.topology was not set")
	}
	var topology graph.Graph
	if err := json.Unmarshal(got.Status.Topology.Raw, &topology); err != nil {
		t.Fatalf("unmarshaling topology: %v", err)
	}
	if len(topology.Nodes) != 1 || topology.Nodes[0].ID != "deployment/superheros/catalog" {
		t.Errorf("unexpected topology: %+v", topology)
	}

	if got.Status.LastDiscoveryTime == nil {
		t.Error("status.lastDiscoveryTime was not set")
	}

	foundCondition := false
	for _, c := range got.Status.Conditions {
		if c.Type == "k8sDetected" {
			foundCondition = true
			if c.Status != metav1.ConditionTrue {
				t.Errorf("k8sDetected condition status = %v, want True", c.Status)
			}
		}
	}
	if !foundCondition {
		t.Errorf("expected a k8sDetected condition, got %+v", got.Status.Conditions)
	}
}

func TestReconcile_SourceErrorDoesNotBlockOtherSources(t *testing.T) {
	scheme := newScheme(t)
	pm := &korionv1alpha1.PlatformMap{
		ObjectMeta: metav1.ObjectMeta{Name: "pm", Namespace: "ns"},
		Spec:       korionv1alpha1.PlatformMapSpec{Namespace: "ns", AutoDiscover: true},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pm).WithStatusSubresource(pm).Build()

	r := &PlatformMapReconciler{
		Client: c,
		Scheme: scheme,
		Discoverers: []discovery.Discoverer{
			&stubDiscoverer{name: "argocd", result: discovery.DiscoveryResult{Source: "argocd", Err: errors.New("CRD not installed")}},
			&stubDiscoverer{name: "k8s", result: discovery.DiscoveryResult{Source: "k8s", Nodes: []graph.Node{{ID: "a"}}}},
		},
	}

	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "ns", Name: "pm"}}); err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	var got korionv1alpha1.PlatformMap
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "ns", Name: "pm"}, &got); err != nil {
		t.Fatalf("fetching updated PlatformMap: %v", err)
	}

	var topology graph.Graph
	if err := json.Unmarshal(got.Status.Topology.Raw, &topology); err != nil {
		t.Fatalf("unmarshaling topology: %v", err)
	}
	if len(topology.Nodes) != 1 || topology.Nodes[0].ID != "a" {
		t.Errorf("expected the k8s source's node despite argocd erroring, got %+v", topology)
	}

	var argoCondition *metav1.Condition
	for i := range got.Status.Conditions {
		if got.Status.Conditions[i].Type == "argocdDetected" {
			argoCondition = &got.Status.Conditions[i]
		}
	}
	if argoCondition == nil {
		t.Fatal("expected an argocdDetected condition")
	}
	if argoCondition.Status != metav1.ConditionFalse {
		t.Errorf("argocdDetected status = %v, want False", argoCondition.Status)
	}
}

func TestReconcile_AutoDiscoverDisabled_SkipsDiscovery(t *testing.T) {
	scheme := newScheme(t)
	pm := &korionv1alpha1.PlatformMap{
		ObjectMeta: metav1.ObjectMeta{Name: "pm", Namespace: "ns"},
		Spec:       korionv1alpha1.PlatformMapSpec{Namespace: "ns", AutoDiscover: false},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pm).WithStatusSubresource(pm).Build()

	called := false
	r := &PlatformMapReconciler{
		Client: c,
		Scheme: scheme,
		Discoverers: []discovery.Discoverer{
			&stubDiscoverer{name: "k8s", result: discovery.DiscoveryResult{Source: "k8s"}},
		},
	}
	_ = called

	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "ns", Name: "pm"}}); err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	var got korionv1alpha1.PlatformMap
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "ns", Name: "pm"}, &got); err != nil {
		t.Fatalf("fetching PlatformMap: %v", err)
	}
	if got.Status.Topology != nil {
		t.Errorf("expected status.topology to remain unset when autoDiscover is false, got %+v", got.Status.Topology)
	}
}

func TestReconcile_NotFound_ReturnsNoError(t *testing.T) {
	scheme := newScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := &PlatformMapReconciler{Client: c, Scheme: scheme}

	res, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "ns", Name: "missing"}})
	if err != nil {
		t.Fatalf("expected no error for a not-found PlatformMap, got %v", err)
	}
	if res.RequeueAfter != 0 {
		t.Errorf("expected no requeue for a not-found PlatformMap, got %v", res.RequeueAfter)
	}
}
