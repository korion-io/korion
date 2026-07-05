/**
 * Copyright 2026 The Korion Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Mirrors internal/graph/types.go, frozen as of Phase 3
// (internal/graph/testdata/sample-topology.json is the committed contract
// fixture). Adding a field here is fine; renaming/removing one is a breaking
// change to that fixture and to this file together.

/** A coarse health indicator driving a node's health-dot color. */
export type NodeStatus = 'healthy' | 'degraded' | 'down' | 'unknown'

/** A single entity in the discovered platform topology. */
export interface GraphNode {
  /** Stable, globally unique key (e.g. "deployment/ns/name"). */
  id: string
  /** Entity kind (e.g. "k8s-deployment", "argocd-application"), used to pick
   * a NodeType component and a brand color. */
  type: string
  label: string
  status: NodeStatus | string
  /** Node border color. Stamped by the Go builder's Merge from the single
   * BrandColorFor lookup table -- never invented client-side except as a
   * fallback for mock data that predates a real discovery engine. */
  brandColor?: string
  /** Source-specific detail (replica counts, image tag, sync status, etc)
   * surfaced in ServiceDetails. */
  metadata?: Record<string, unknown>
}

/** A directed relationship between two GraphNodes, referenced by ID. */
export interface GraphEdge {
  from: string
  to: string
  type: string
  label?: string
}

/** The full discovered topology -- PlatformMap.status.topology's shape. */
export interface Graph {
  nodes: GraphNode[]
  edges: GraphEdge[]
}

/** A single Deployment Timeline event (ArgoCD sync / CI run history).
 * Not yet part of the frozen graph contract -- ArgoCD/GitHub discovery lands
 * in Phase 6. Mocked in the UI ahead of that so the panel layout can be
 * built and reviewed now. */
export interface DeploymentEvent {
  id: string
  timestamp: string
  title: string
  description: string
  status: 'success' | 'failed' | 'in-progress'
  source: 'github' | 'github-actions' | 'docker' | 'argocd' | 'k8s'
}

export type PolicyResult = 'pass' | 'warn' | 'fail'

/** A single Kyverno/PolicyReport violation. Mocked ahead of Phase 6. */
export interface PolicyViolation {
  id: string
  policy: string
  resource: string
  result: PolicyResult
  message: string
  timestamp: string
}

export interface PolicySummary {
  total: number
  passed: number
  warnings: number
  failed: number
  violations: PolicyViolation[]
}

/** The full mock/real payload the canvas and side panels render from. */
export interface PlatformMapView {
  name: string
  namespace: string
  cluster: string
  topology: Graph
  deploymentEvents: DeploymentEvent[]
  policySummary: PolicySummary
}
