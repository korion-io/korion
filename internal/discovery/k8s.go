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

// Package discovery implements the engines that inspect a cluster (and,
// later, ArgoCD/Istio/Kyverno/GitHub/Prometheus) on behalf of a PlatformMap
// and report their findings as generic graph.Nodes/Edges.
package discovery

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/korion-io/korion/api/v1alpha1"
	"github.com/korion-io/korion/internal/graph"
)

// DiscoveryResult is the shared contract every discovery engine returns. Err
// is non-fatal: a partial result (possibly with zero Nodes/Edges) is still
// merged into the topology graph rather than failing the whole reconcile --
// this is how Korion tolerates optional tools (ArgoCD, Istio, Kyverno) being
// absent from a cluster.
type DiscoveryResult struct {
	Source string
	Nodes  []graph.Node
	Edges  []graph.Edge
	Err    error
}

// Discoverer is implemented by every discovery engine (K8s, and later
// ArgoCD, Istio, Kyverno, GitHub, Prometheus).
type Discoverer interface {
	Name() string
	Discover(ctx context.Context, pm *v1alpha1.PlatformMap) DiscoveryResult
}

// K8sDiscoverer discovers plain Kubernetes Deployments and Services in the
// PlatformMap's target namespace. It's the baseline discovery source: it
// requires no optional CRDs, so it's what makes the vertical slice (CRD ->
// controller -> status -> read API -> canvas) demoable against a bare Kind
// cluster before any of the other five engines exist.
type K8sDiscoverer struct {
	Clientset kubernetes.Interface
}

func (d *K8sDiscoverer) Name() string { return "K8s" }

func (d *K8sDiscoverer) Discover(ctx context.Context, pm *v1alpha1.PlatformMap) DiscoveryResult {
	ns := pm.Spec.Namespace
	result := DiscoveryResult{Source: d.Name()}

	deployments, err := d.Clientset.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		result.Err = fmt.Errorf("listing deployments in %q: %w", ns, err)
		return result
	}
	for _, dep := range deployments.Items {
		result.Nodes = append(result.Nodes, deploymentNode(dep))
	}

	services, err := d.Clientset.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		result.Err = fmt.Errorf("listing services in %q: %w", ns, err)
		return result
	}
	for _, svc := range services.Items {
		result.Nodes = append(result.Nodes, serviceNode(svc))
		for _, dep := range deployments.Items {
			if selectorMatchesLabels(svc.Spec.Selector, dep.Spec.Template.Labels) {
				result.Edges = append(result.Edges, graph.Edge{
					From: serviceNodeID(svc.Namespace, svc.Name),
					To:   deploymentNodeID(dep.Namespace, dep.Name),
					Type: "routes-to",
				})
			}
		}
	}

	return result
}

func deploymentNodeID(namespace, name string) string {
	return fmt.Sprintf("deployment/%s/%s", namespace, name)
}

func serviceNodeID(namespace, name string) string {
	return fmt.Sprintf("service/%s/%s", namespace, name)
}

func deploymentNode(dep appsv1.Deployment) graph.Node {
	desiredReplicas := int32(1)
	if dep.Spec.Replicas != nil {
		desiredReplicas = *dep.Spec.Replicas
	}
	status := "healthy"
	if dep.Status.ReadyReplicas < desiredReplicas {
		status = "degraded"
	}

	var image string
	if len(dep.Spec.Template.Spec.Containers) > 0 {
		image = dep.Spec.Template.Spec.Containers[0].Image
	}

	return graph.Node{
		ID:     deploymentNodeID(dep.Namespace, dep.Name),
		Type:   "k8s-deployment",
		Label:  dep.Name,
		Status: status,
		Metadata: map[string]any{
			"namespace":     dep.Namespace,
			"replicasReady": dep.Status.ReadyReplicas,
			"replicasTotal": dep.Status.Replicas,
			"image":         image,
			"generation":    dep.Generation,
			"resourceKind":  "Deployment",
		},
	}
}

func serviceNode(svc corev1.Service) graph.Node {
	return graph.Node{
		ID:     serviceNodeID(svc.Namespace, svc.Name),
		Type:   "k8s-service",
		Label:  svc.Name,
		Status: "healthy",
		Metadata: map[string]any{
			"namespace":    svc.Namespace,
			"clusterIP":    svc.Spec.ClusterIP,
			"type":         string(svc.Spec.Type),
			"resourceKind": "Service",
		},
	}
}

// selectorMatchesLabels reports whether every key/value in selector is
// present in labels. An empty selector matches nothing -- an unselective
// Service isn't meaningfully "routing to" every Deployment in the
// namespace.
func selectorMatchesLabels(selector, labels map[string]string) bool {
	if len(selector) == 0 {
		return false
	}
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}
