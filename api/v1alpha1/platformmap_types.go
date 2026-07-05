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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ToolConfig is the common per-tool auto-discovery toggle shared by
// integrations that need no more than an enabled flag and an optional
// endpoint override.
type ToolConfig struct {
	// Enabled turns on discovery for this tool.
	Enabled bool `json:"enabled"`

	// URL overrides auto-detection of the tool's API endpoint. Left empty,
	// Korion auto-detects the endpoint from well-known Service names in the
	// target namespace.
	// +optional
	URL string `json:"url,omitempty"`
}

// GitHubToolConfig configures GitHub Actions discovery, which additionally
// requires an API token to read workflow run status.
type GitHubToolConfig struct {
	// Enabled turns on discovery for this tool.
	Enabled bool `json:"enabled"`

	// URL overrides auto-detection of the tool's API endpoint.
	// +optional
	URL string `json:"url,omitempty"`

	// TokenSecretRef references the Secret key holding a GitHub API token.
	// +optional
	TokenSecretRef *corev1.SecretKeySelector `json:"tokenSecretRef,omitempty"`
}

// ToolsConfig lists auto-discovery toggles for every supported tool
// integration. Every field is optional; an omitted tool is treated as
// disabled.
type ToolsConfig struct {
	// +optional
	ArgoCD *ToolConfig `json:"argocd,omitempty"`

	// +optional
	Kiali *ToolConfig `json:"kiali,omitempty"`

	// +optional
	Istio *ToolConfig `json:"istio,omitempty"`

	// +optional
	Kyverno *ToolConfig `json:"kyverno,omitempty"`

	// +optional
	Prometheus *ToolConfig `json:"prometheus,omitempty"`

	// +optional
	Loki *ToolConfig `json:"loki,omitempty"`

	// +optional
	GitHub *GitHubToolConfig `json:"github,omitempty"`
}

// PlatformMapSpec defines the desired discovery configuration for a
// platform's DevOps stack.
type PlatformMapSpec struct {
	// Repository is the source Git repository URL for the platform being
	// mapped.
	// +optional
	Repository string `json:"repository,omitempty"`

	// Namespace is the target Kubernetes namespace to discover.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`

	// AutoDiscover enables automatic discovery of the full DevOps stack.
	// +kubebuilder:default=true
	AutoDiscover bool `json:"autoDiscover,omitempty"`

	// Tools configures which discovery integrations are enabled.
	// +optional
	Tools ToolsConfig `json:"tools,omitempty"`

	// RefreshInterval controls how often the controller re-runs discovery.
	// +kubebuilder:default="30s"
	RefreshInterval metav1.Duration `json:"refreshInterval,omitempty"`
}

// PlatformMapStatus reports the last discovered topology and per-source
// discovery health.
type PlatformMapStatus struct {
	// Topology is the discovered platform graph, serialized as opaque JSON.
	// The concrete Node/Edge shape is frozen in internal/graph (see
	// internal/graph/testdata/sample-topology.json) and mirrored by the
	// frontend's hand-written TS types from Phase 4 on; this field stays
	// untyped RawExtension here so the API package never has to import an
	// internal one to describe it.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Topology *runtime.RawExtension `json:"topology,omitempty"`

	// Conditions report discovery health per source (e.g. ArgoCDDetected,
	// IstioDetected), keyed by Condition.Type.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// LastDiscoveryTime is when the controller last completed a full
	// discovery reconcile.
	// +optional
	LastDiscoveryTime *metav1.Time `json:"lastDiscoveryTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.spec.namespace`
// +kubebuilder:printcolumn:name="Discovered",type=date,JSONPath=`.status.lastDiscoveryTime`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// PlatformMap triggers auto-discovery of a platform's full DevOps stack
// (GitHub Actions, ArgoCD, Kubernetes, Istio, Kyverno, Prometheus) and
// renders the result as an interactive topology graph.
type PlatformMap struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlatformMapSpec   `json:"spec,omitempty"`
	Status PlatformMapStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PlatformMapList contains a list of PlatformMap.
type PlatformMapList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PlatformMap `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PlatformMap{}, &PlatformMapList{})
}
