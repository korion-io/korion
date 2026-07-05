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
	"strconv"
	"strings"
	"testing"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/korion-io/korion/api/v1alpha1"
)

// promQueryResponse builds a minimal Prometheus HTTP API instant-query
// response for a vector result, one sample per (service, value) pair.
func promQueryResponse(values map[string]float64) []byte {
	results := make([]map[string]any, 0, len(values))
	for service, v := range values {
		results = append(results, map[string]any{
			"metric": map[string]string{"service": service},
			"value":  []any{0, strconv.FormatFloat(v, 'f', -1, 64)},
		})
	}
	body, _ := json.Marshal(map[string]any{
		"status": "success",
		"data": map[string]any{
			"resultType": "vector",
			"result":     results,
		},
	})
	return body
}

func newPrometheusTestServer(byQuerySubstring map[string]map[string]float64) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/query", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if query == "" {
			_ = r.ParseForm()
			query = r.FormValue("query")
		}
		for substr, values := range byQuerySubstring {
			if strings.Contains(query, substr) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(promQueryResponse(values))
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(promQueryResponse(nil))
	})
	return httptest.NewServer(mux)
}

func newTestPrometheusNewAPI(server *httptest.Server) func(address string) (promv1.API, error) {
	return func(address string) (promv1.API, error) {
		client, err := promapi.NewClient(promapi.Config{Address: server.URL})
		if err != nil {
			return nil, err
		}
		return promv1.NewAPI(client), nil
	}
}

func TestPrometheusDiscoverer_Discover(t *testing.T) {
	server := newPrometheusTestServer(map[string]map[string]float64{
		"http_requests_total":                  {"catalog": 2.5, "orders": 0},
		"http_request_duration_seconds_bucket": {"catalog": 120, "orders": 45},
		"container_cpu_usage_seconds_total":    {"catalog": 250, "orders": 80},
		"container_memory_working_set_bytes":   {"catalog": 512, "orders": 256},
	})
	defer server.Close()

	d := &PrometheusDiscoverer{NewAPI: newTestPrometheusNewAPI(server)}
	pm := &v1alpha1.PlatformMap{
		Spec: v1alpha1.PlatformMapSpec{
			Namespace: "superheros",
			Tools: v1alpha1.ToolsConfig{
				Prometheus: &v1alpha1.ToolConfig{Enabled: true},
			},
		},
	}

	result := d.Discover(context.Background(), pm)

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if len(result.Nodes) != 2 {
		t.Fatalf("got %d nodes, want 2 (catalog, orders): %+v", len(result.Nodes), result.Nodes)
	}

	for _, n := range result.Nodes {
		switch n.ID {
		case serviceNodeID("superheros", "catalog"):
			if n.Metadata["errorRatePct"] != 2.5 {
				t.Errorf("catalog errorRatePct = %v, want 2.5", n.Metadata["errorRatePct"])
			}
			if n.Metadata["latencyP99Ms"] != 120.0 {
				t.Errorf("catalog latencyP99Ms = %v, want 120", n.Metadata["latencyP99Ms"])
			}
			if n.Metadata["cpuUsageM"] != 250.0 {
				t.Errorf("catalog cpuUsageM = %v, want 250", n.Metadata["cpuUsageM"])
			}
			if n.Metadata["memoryUsageMi"] != 512.0 {
				t.Errorf("catalog memoryUsageMi = %v, want 512", n.Metadata["memoryUsageMi"])
			}
		case serviceNodeID("superheros", "orders"):
			if n.Metadata["errorRatePct"] != 0.0 {
				t.Errorf("orders errorRatePct = %v, want 0", n.Metadata["errorRatePct"])
			}
		default:
			t.Errorf("unexpected node ID %q", n.ID)
		}
	}
}

func TestPrometheusDiscoverer_Discover_Disabled(t *testing.T) {
	d := &PrometheusDiscoverer{}
	pm := &v1alpha1.PlatformMap{Spec: v1alpha1.PlatformMapSpec{Namespace: "superheros"}}

	result := d.Discover(context.Background(), pm)

	if result.Err != nil {
		t.Fatalf("unexpected error when prometheus tool disabled: %v", result.Err)
	}
	if len(result.Nodes) != 0 {
		t.Errorf("expected no nodes when prometheus tool disabled, got %+v", result.Nodes)
	}
}

func TestPrometheusDiscoverer_Discover_Unreachable(t *testing.T) {
	d := &PrometheusDiscoverer{}
	pm := &v1alpha1.PlatformMap{
		Spec: v1alpha1.PlatformMapSpec{
			Namespace: "superheros",
			Tools: v1alpha1.ToolsConfig{
				Prometheus: &v1alpha1.ToolConfig{Enabled: true, URL: "http://127.0.0.1:0"},
			},
		},
	}

	result := d.Discover(context.Background(), pm)

	if result.Err == nil {
		t.Fatal("expected an error when prometheus is unreachable")
	}
}
