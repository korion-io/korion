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
)

// LLMProvider configures which large language model backs ARIA's reasoning.
type LLMProvider struct {
	// Provider selects the LLM backend.
	// +kubebuilder:validation:Enum=anthropic;openai;ollama
	Provider string `json:"provider"`

	// Model is the provider-specific model identifier.
	Model string `json:"model"`

	// APIKeySecretRef references the Secret key holding the provider API key.
	// +optional
	APIKeySecretRef *corev1.SecretKeySelector `json:"apiKeySecretRef,omitempty"`
}

// AlertEnrichmentFeature enriches Alertmanager webhooks with system-specific
// SRE analysis posted to Slack.
type AlertEnrichmentFeature struct {
	Enabled bool `json:"enabled"`

	// SlackWebhookSecretRef references the Secret key holding the Slack
	// incoming webhook URL.
	// +optional
	SlackWebhookSecretRef *corev1.SecretKeySelector `json:"slackWebhookSecretRef,omitempty"`
}

// HealthAdvisorFeature runs a scheduled platform health scan.
type HealthAdvisorFeature struct {
	Enabled bool `json:"enabled"`

	// Schedule is a cron expression controlling how often the health
	// advisor runs.
	// +optional
	Schedule string `json:"schedule,omitempty"`
}

// SREDiagnosticsFeature enables manual or triggered incident investigation.
type SREDiagnosticsFeature struct {
	Enabled bool `json:"enabled"`
}

// CanaryDecisionFeature enables promote/hold/rollback recommendations for
// versioned canary services.
type CanaryDecisionFeature struct {
	Enabled bool `json:"enabled"`
}

// AgentFeatures toggles each of ARIA's capabilities independently.
type AgentFeatures struct {
	// +optional
	AlertEnrichment *AlertEnrichmentFeature `json:"alertEnrichment,omitempty"`

	// +optional
	HealthAdvisor *HealthAdvisorFeature `json:"healthAdvisor,omitempty"`

	// +optional
	SREDiagnostics *SREDiagnosticsFeature `json:"sreDiagnostics,omitempty"`

	// +optional
	CanaryDecision *CanaryDecisionFeature `json:"canaryDecision,omitempty"`
}

// Runbook maps a trigger condition to an ARIA action.
type Runbook struct {
	// Name identifies the runbook.
	Name string `json:"name"`

	// Trigger is a condition expression evaluated against platform state
	// (e.g. "podRestarts > 3").
	Trigger string `json:"trigger"`

	// Action is the ARIA action to take when Trigger evaluates true.
	Action string `json:"action"`
}

// PlatformAgentSpec defines ARIA's configuration: which PlatformMap it
// reasons over, how autonomously it may act, which LLM backs it, and which
// capabilities and runbooks are enabled.
type PlatformAgentSpec struct {
	// PlatformMap is the name of the PlatformMap this agent reasons over, in
	// the same namespace.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	PlatformMap string `json:"platformMap"`

	// AutonomyLevel controls how much ARIA acts without human approval.
	// suggest: recommends only. approve: acts after human approval. auto:
	// acts autonomously, subject to the confidence-score gate.
	// +kubebuilder:validation:Enum=suggest;approve;auto
	// +kubebuilder:default=suggest
	AutonomyLevel string `json:"autonomyLevel,omitempty"`

	// LLMProvider selects and configures the backing LLM.
	LLMProvider LLMProvider `json:"llmProvider"`

	// Features toggles ARIA's individual capabilities.
	// +optional
	Features AgentFeatures `json:"features,omitempty"`

	// Runbooks maps trigger conditions to ARIA actions.
	// +optional
	Runbooks []Runbook `json:"runbooks,omitempty"`
}

// PlatformAgentStatus reports ARIA's reconciliation health. The reconciler
// for this type is inert until ARIA itself exists to be reconciled against.
type PlatformAgentStatus struct {
	// Conditions report ARIA's operational health (e.g. PlatformMapFound,
	// ARIAReachable), keyed by Condition.Type.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// LastReconcileTime is when the controller last reconciled this
	// PlatformAgent.
	// +optional
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="PlatformMap",type=string,JSONPath=`.spec.platformMap`
// +kubebuilder:printcolumn:name="Autonomy",type=string,JSONPath=`.spec.autonomyLevel`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// PlatformAgent is ARIA: the AI intelligence layer that reasons over a
// referenced PlatformMap's live topology to produce system-specific SRE
// analysis, health reports, and canary decisions.
type PlatformAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlatformAgentSpec   `json:"spec,omitempty"`
	Status PlatformAgentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PlatformAgentList contains a list of PlatformAgent.
type PlatformAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PlatformAgent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PlatformAgent{}, &PlatformAgentList{})
}
