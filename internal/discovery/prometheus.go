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
	"time"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/korion-io/korion/api/v1alpha1"
	"github.com/korion-io/korion/internal/graph"
)

// Instant PromQL query templates, one per metric, each grouped by a
// "service" label so a single query yields every service's value at once.
// These are deliberately simple defaults (standard Prometheus Operator /
// kube-state-metrics metric names) meant to be tuned against real-world
// SuperHeros metrics in Phase 8, not treated as final.
const (
	errorRateQueryTemplate   = `sum(rate(http_requests_total{namespace=%q,status=~"5.."}[5m])) by (service) / sum(rate(http_requests_total{namespace=%q}[5m])) by (service) * 100`
	latencyP99QueryTemplate  = `histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket{namespace=%q}[5m])) by (service, le)) * 1000`
	cpuUsageQueryTemplate    = `sum(rate(container_cpu_usage_seconds_total{namespace=%q, container!="", container!="POD"}[5m])) by (service) * 1000`
	memoryUsageQueryTemplate = `sum(container_memory_working_set_bytes{namespace=%q, container!="", container!="POD"}) by (service) / (1024*1024)`
)

// PrometheusDiscoverer discovers per-service error rate, p99 latency, CPU,
// and memory usage from Prometheus, enriching each matching k8s-service
// node's Metadata rather than creating standalone nodes -- these are
// point-in-time measurements of an existing entity, not a new topology
// node.
//
// Unlike ArgoCD/Istio/Kyverno (which query the one shared in-cluster
// dynamic client regardless of which PlatformMap triggered discovery),
// Prometheus is an HTTP endpoint that can differ per PlatformMap via
// spec.tools.prometheus.url. NewAPI is therefore called fresh on every
// Discover, not held as a single fixed client.
type PrometheusDiscoverer struct {
	// NewAPI builds a Prometheus API client for the given address. Defaults
	// to NewPrometheusAPI; tests override it to point at an httptest.Server.
	NewAPI func(address string) (promv1.API, error)
}

func (d *PrometheusDiscoverer) Name() string { return "Prometheus" }

func (d *PrometheusDiscoverer) Discover(ctx context.Context, pm *v1alpha1.PlatformMap) DiscoveryResult {
	result := DiscoveryResult{Source: d.Name()}

	if pm.Spec.Tools.Prometheus == nil || !pm.Spec.Tools.Prometheus.Enabled {
		return result
	}

	newAPI := d.NewAPI
	if newAPI == nil {
		newAPI = NewPrometheusAPI
	}
	api, err := newAPI(prometheusAddress(pm))
	if err != nil {
		result.Err = fmt.Errorf("creating prometheus client: %w", err)
		return result
	}

	ns := pm.Spec.Namespace
	queries := []struct{ key, query string }{
		{"errorRatePct", fmt.Sprintf(errorRateQueryTemplate, ns, ns)},
		{"latencyP99Ms", fmt.Sprintf(latencyP99QueryTemplate, ns)},
		{"cpuUsageM", fmt.Sprintf(cpuUsageQueryTemplate, ns)},
		{"memoryUsageMi", fmt.Sprintf(memoryUsageQueryTemplate, ns)},
	}

	byService := make(map[string]map[string]any)
	var firstErr error
	for _, q := range queries {
		values, err := queryByService(ctx, api, q.query)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("querying prometheus for %s: %w", q.key, err)
			}
			continue
		}
		for service, v := range values {
			if byService[service] == nil {
				byService[service] = make(map[string]any)
			}
			byService[service][q.key] = v
		}
	}
	if firstErr != nil {
		result.Err = firstErr
	}

	for service, metrics := range byService {
		result.Nodes = append(result.Nodes, graph.Node{
			ID:       serviceNodeID(ns, service),
			Metadata: metrics,
		})
	}

	return result
}

// queryByService runs an instant PromQL query and returns its result vector
// keyed by the "service" label -- every query template above groups "by
// (service)" so this extraction is uniform across all four metrics.
func queryByService(ctx context.Context, api promv1.API, query string) (map[string]float64, error) {
	val, _, err := api.Query(ctx, query, time.Now())
	if err != nil {
		return nil, err
	}
	vector, ok := val.(model.Vector)
	if !ok {
		return nil, fmt.Errorf("unexpected prometheus result type %T for query %q", val, query)
	}

	out := make(map[string]float64, len(vector))
	for _, sample := range vector {
		service := string(sample.Metric["service"])
		if service == "" {
			continue
		}
		out[service] = float64(sample.Value)
	}
	return out, nil
}

// NewPrometheusAPI is the default PrometheusDiscoverer.NewAPI implementation:
// a real HTTP-backed client against the given address.
func NewPrometheusAPI(address string) (promv1.API, error) {
	client, err := promapi.NewClient(promapi.Config{Address: address})
	if err != nil {
		return nil, err
	}
	return promv1.NewAPI(client), nil
}

// prometheusAddress returns spec.tools.prometheus.url if set, otherwise a
// same-namespace "prometheus" Service as a best-effort default -- a
// simplification documented for tuning against the real SuperHeros cluster
// in Phase 8, since Prometheus's actual Service name/namespace varies by
// install method (kube-prometheus-stack, bare Prometheus Operator, etc).
func prometheusAddress(pm *v1alpha1.PlatformMap) string {
	if pm.Spec.Tools.Prometheus.URL != "" {
		return pm.Spec.Tools.Prometheus.URL
	}
	return fmt.Sprintf("http://prometheus.%s.svc.cluster.local:9090", pm.Spec.Namespace)
}
