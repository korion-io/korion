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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	discoveryfake "k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/korion-io/korion/api/v1alpha1"
)

func newArgoApplication(ns, name, destNamespace, syncStatus, healthStatus, revision string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"metadata": map[string]any{
			"name":      name,
			"namespace": ns,
		},
		"spec": map[string]any{
			"destination": map[string]any{"namespace": destNamespace},
			"source":      map[string]any{"repoURL": "https://github.com/gc-ghub/superheros"},
		},
		"status": map[string]any{
			"sync":   map[string]any{"status": syncStatus, "revision": revision},
			"health": map[string]any{"status": healthStatus},
		},
	}}
}

// discoveryClientWithGroupVersions returns a fake discovery client that
// reports the given group/versions as available -- used so ArgoCD/Istio/
// Kyverno engine tests can exercise both "CRD installed" and "CRD absent"
// paths without a real API server.
func discoveryClientWithGroupVersions(groupVersions ...string) *discoveryfake.FakeDiscovery {
	clientset := kubefake.NewClientset()
	fd := clientset.Discovery().(*discoveryfake.FakeDiscovery)
	resources := make([]*metav1.APIResourceList, 0, len(groupVersions))
	for _, gv := range groupVersions {
		resources = append(resources, &metav1.APIResourceList{GroupVersion: gv})
	}
	fd.Resources = resources
	return fd
}

func TestArgoCDDiscoverer_Discover(t *testing.T) {
	ns := "superheros"
	scheme := runtime.NewScheme()
	gvrListKind := map[schema.GroupVersionResource]string{
		argoCDApplicationGVR: "ApplicationList",
	}
	dynClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrListKind,
		newArgoApplication("argocd", "catalog", ns, "Synced", "Healthy", "abc123"),
		newArgoApplication("argocd", "orders", ns, "OutOfSync", "Degraded", "def456"),
		newArgoApplication("argocd", "other-platform", "other-namespace", "Synced", "Healthy", "zzz"),
	)

	d := &ArgoCDDiscoverer{
		Dynamic:   dynClient,
		Discovery: discoveryClientWithGroupVersions(argoCDGroupVersion),
	}
	pm := &v1alpha1.PlatformMap{Spec: v1alpha1.PlatformMapSpec{Namespace: ns}}

	result := d.Discover(context.Background(), pm)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Nodes) != 2 {
		t.Fatalf("got %d nodes, want 2 (scoped to destination namespace %q): %+v", len(result.Nodes), ns, result.Nodes)
	}

	for _, n := range result.Nodes {
		if n.Type != "argocd-application" {
			t.Errorf("node %q Type = %q, want argocd-application", n.ID, n.Type)
		}
		switch n.ID {
		case "argocd-application/argocd/catalog":
			if n.Status != "healthy" {
				t.Errorf("catalog Status = %q, want healthy", n.Status)
			}
			if n.Metadata["syncStatus"] != "Synced" {
				t.Errorf("catalog syncStatus = %v, want Synced", n.Metadata["syncStatus"])
			}
			if n.Metadata["revision"] != "abc123" {
				t.Errorf("catalog revision = %v, want abc123", n.Metadata["revision"])
			}
		case "argocd-application/argocd/orders":
			if n.Status != "degraded" {
				t.Errorf("orders Status = %q, want degraded", n.Status)
			}
		}
	}
}

func TestArgoCDDiscoverer_Discover_CRDNotInstalled(t *testing.T) {
	d := &ArgoCDDiscoverer{
		Dynamic:   dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()),
		Discovery: discoveryClientWithGroupVersions(), // no group versions registered
	}
	pm := &v1alpha1.PlatformMap{Spec: v1alpha1.PlatformMapSpec{Namespace: "superheros"}}

	result := d.Discover(context.Background(), pm)

	if !errors.Is(result.Err, ErrCRDNotInstalled) {
		t.Fatalf("expected ErrCRDNotInstalled, got %v", result.Err)
	}
	if len(result.Nodes) != 0 {
		t.Errorf("expected no nodes when CRD absent, got %+v", result.Nodes)
	}
}
