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

// Package graph defines the discovery-agnostic topology shape written to
// PlatformMap.status.topology and consumed by the frontend. It never imports
// internal/discovery -- discoverers depend on graph, not the reverse.
package graph

// Node is a single entity in the discovered platform topology (a Kubernetes
// Deployment, Service, ArgoCD Application, etc).
//
// This shape is frozen as of Phase 3: internal/graph/testdata/
// sample-topology.json is the committed contract fixture every discovery
// engine's output and the frontend's hand-written TS types must match.
// Adding a field is fine (frontend/Go both tolerate unknown-to-them
// additions); renaming or removing one is a breaking change to that fixture.
type Node struct {
	// ID is a stable, globally unique key (e.g. "deployment/ns/name").
	ID string `json:"id"`

	// Type identifies the kind of entity (e.g. "k8s-deployment",
	// "k8s-service", "argocd-application"), used by the frontend to pick a
	// NodeType component, and by BrandColorFor to pick a border color.
	Type string `json:"type"`

	Label string `json:"label"`

	// Status is a coarse health indicator (e.g. "healthy", "degraded",
	// "unknown") driving the node's health-dot color.
	Status string `json:"status"`

	// BrandColor is the node border color for the frontend canvas. It is
	// never set by a Discoverer directly -- Merge stamps it from the single
	// BrandColorFor lookup table, keyed off Type, so brand colors live in
	// exactly one place (see colors.go) rather than being duplicated across
	// every discovery engine.
	BrandColor string `json:"brandColor,omitempty"`

	// Metadata carries source-specific detail (replica counts, image tag,
	// sync status, etc) surfaced in the frontend's ServiceDetails panel.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Edge is a directed relationship between two Nodes, referenced by ID.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
	// +optional
	Label string `json:"label,omitempty"`
}

// Graph is the full discovered topology -- the shape written to
// PlatformMap.status.topology and rendered by the frontend's React Flow
// canvas.
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}
