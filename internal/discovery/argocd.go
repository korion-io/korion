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

// argoCDApplicationGVR is the well-known ArgoCD Application resource. A
// dynamic client is used against it (rather than vendoring the full argo-cd
// Go module) to avoid heavy transitive dependencies and coupling Korion's
// release cadence to ArgoCD's own -- see decisions.md.
var argoCDApplicationGVR = schema.GroupVersionResource{
	Group:    "argoproj.io",
	Version:  "v1alpha1",
	Resource: "applications",
}

const argoCDGroupVersion = "argoproj.io/v1alpha1"

// ArgoCDDiscoverer discovers ArgoCD Applications whose destination namespace
// matches the PlatformMap's target namespace, surfacing sync status, health
// status, and last-sync commit info.
type ArgoCDDiscoverer struct {
	Dynamic   dynamic.Interface
	Discovery k8sdiscovery.DiscoveryInterface
}

func (d *ArgoCDDiscoverer) Name() string { return "ArgoCD" }

func (d *ArgoCDDiscoverer) Discover(ctx context.Context, pm *v1alpha1.PlatformMap) DiscoveryResult {
	result := DiscoveryResult{Source: d.Name()}

	if !GroupVersionAvailable(d.Discovery, argoCDGroupVersion) {
		result.Err = ErrCRDNotInstalled
		return result
	}

	list, err := d.Dynamic.Resource(argoCDApplicationGVR).Namespace(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		result.Err = fmt.Errorf("listing argocd applications: %w", err)
		return result
	}

	for _, item := range list.Items {
		destNamespace, _, _ := unstructured.NestedString(item.Object, "spec", "destination", "namespace")
		if destNamespace != pm.Spec.Namespace {
			continue
		}
		result.Nodes = append(result.Nodes, argoCDApplicationNode(item))
	}

	return result
}

func argoCDApplicationNode(app unstructured.Unstructured) graph.Node {
	syncStatus, _, _ := unstructured.NestedString(app.Object, "status", "sync", "status")
	healthStatus, _, _ := unstructured.NestedString(app.Object, "status", "health", "status")
	revision, _, _ := unstructured.NestedString(app.Object, "status", "sync", "revision")
	lastSyncMessage, _, _ := unstructured.NestedString(app.Object, "status", "operationState", "message")
	reconciledAt, _, _ := unstructured.NestedString(app.Object, "status", "reconciledAt")
	repoURL, _, _ := unstructured.NestedString(app.Object, "spec", "source", "repoURL")
	destNamespace, _, _ := unstructured.NestedString(app.Object, "spec", "destination", "namespace")

	return graph.Node{
		ID:     argoCDApplicationNodeID(app.GetNamespace(), app.GetName()),
		Type:   "argocd-application",
		Label:  app.GetName(),
		Status: argoCDHealthToNodeStatus(healthStatus),
		Metadata: map[string]any{
			"namespace":       app.GetNamespace(),
			"targetNamespace": destNamespace,
			"syncStatus":      syncStatus,
			"healthStatus":    healthStatus,
			"revision":        revision,
			"lastSyncMessage": lastSyncMessage,
			"lastSyncTime":    reconciledAt,
			"repoURL":         repoURL,
			"resourceKind":    "Application",
		},
	}
}

func argoCDApplicationNodeID(namespace, name string) string {
	return fmt.Sprintf("argocd-application/%s/%s", namespace, name)
}

// argoCDHealthToNodeStatus maps ArgoCD's own health vocabulary
// (Healthy/Progressing/Degraded/Suspended/Missing/Unknown) down to Korion's
// coarse three-state Node.Status used for the canvas health dot.
func argoCDHealthToNodeStatus(health string) string {
	switch health {
	case "Healthy":
		return "healthy"
	case "Degraded", "Missing":
		return "degraded"
	default:
		return "unknown"
	}
}
