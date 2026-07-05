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
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/korion-io/korion/api/v1alpha1"
)

// stubSecretResolver is a SecretResolver returning a fixed token, used so
// GitHubDiscoverer's tests don't need a fake Kubernetes clientset just to
// resolve one Secret key.
type stubSecretResolver struct{ token string }

func (s stubSecretResolver) Resolve(ctx context.Context, namespace string, ref *corev1.SecretKeySelector) (string, error) {
	return s.token, nil
}

func newGitHubTestServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/gc-ghub/superheros/actions/workflows", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_count": 2,
			"workflows": []map[string]any{
				{"id": 1, "name": "catalog-ci", "path": ".github/workflows/catalog.yml", "state": "active"},
				{"id": 2, "name": "orders-ci", "path": ".github/workflows/orders.yml", "state": "active"},
			},
		})
	})
	mux.HandleFunc("/repos/gc-ghub/superheros/actions/workflows/1/runs", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_count": 1,
			"workflow_runs": []map[string]any{
				{"id": 100, "status": "completed", "conclusion": "success", "html_url": "https://github.com/gc-ghub/superheros/actions/runs/100"},
			},
		})
	})
	mux.HandleFunc("/repos/gc-ghub/superheros/actions/workflows/2/runs", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total_count": 1,
			"workflow_runs": []map[string]any{
				{"id": 101, "status": "completed", "conclusion": "failure", "html_url": "https://github.com/gc-ghub/superheros/actions/runs/101"},
			},
		})
	})
	return httptest.NewServer(mux)
}

func TestGitHubDiscoverer_Discover(t *testing.T) {
	server := newGitHubTestServer()
	defer server.Close()
	baseURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatalf("parsing test server URL: %v", err)
	}

	d := &GitHubDiscoverer{
		Secrets: stubSecretResolver{token: "test-token"},
		BaseURL: baseURL,
	}
	pm := &v1alpha1.PlatformMap{
		Spec: v1alpha1.PlatformMapSpec{
			Namespace:  "superheros",
			Repository: "https://github.com/gc-ghub/superheros",
			Tools: v1alpha1.ToolsConfig{
				GitHub: &v1alpha1.GitHubToolConfig{
					Enabled:        true,
					TokenSecretRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "gh-token"}, Key: "token"},
				},
			},
		},
	}

	result := d.Discover(context.Background(), pm)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Nodes) != 3 {
		t.Fatalf("got %d nodes, want 3 (1 repo + 2 workflows): %+v", len(result.Nodes), result.Nodes)
	}

	byID := make(map[string]int)
	for _, n := range result.Nodes {
		byID[n.ID]++
	}
	if byID["github-repository/gc-ghub/superheros"] != 1 {
		t.Errorf("missing github-repository node")
	}

	for _, n := range result.Nodes {
		switch n.ID {
		case "github-actions-workflow/gc-ghub/superheros/1":
			if n.Status != "healthy" {
				t.Errorf("catalog-ci status = %q, want healthy", n.Status)
			}
			if n.Metadata["lastRunConclusion"] != "success" {
				t.Errorf("catalog-ci lastRunConclusion = %v, want success", n.Metadata["lastRunConclusion"])
			}
		case "github-actions-workflow/gc-ghub/superheros/2":
			if n.Status != "degraded" {
				t.Errorf("orders-ci status = %q, want degraded", n.Status)
			}
		}
	}
}

func TestGitHubDiscoverer_Discover_Disabled(t *testing.T) {
	d := &GitHubDiscoverer{Secrets: stubSecretResolver{token: "unused"}}
	pm := &v1alpha1.PlatformMap{Spec: v1alpha1.PlatformMapSpec{Namespace: "superheros"}}

	result := d.Discover(context.Background(), pm)

	if result.Err != nil {
		t.Fatalf("unexpected error when github tool disabled: %v", result.Err)
	}
	if len(result.Nodes) != 0 {
		t.Errorf("expected no nodes when github tool disabled, got %+v", result.Nodes)
	}
}

func TestGitHubDiscoverer_Discover_MissingTokenSecretRef(t *testing.T) {
	d := &GitHubDiscoverer{Secrets: stubSecretResolver{token: "unused"}}
	pm := &v1alpha1.PlatformMap{
		Spec: v1alpha1.PlatformMapSpec{
			Namespace:  "superheros",
			Repository: "https://github.com/gc-ghub/superheros",
			Tools: v1alpha1.ToolsConfig{
				GitHub: &v1alpha1.GitHubToolConfig{Enabled: true},
			},
		},
	}

	result := d.Discover(context.Background(), pm)

	if result.Err == nil {
		t.Fatal("expected an error when tokenSecretRef is unset")
	}
}
