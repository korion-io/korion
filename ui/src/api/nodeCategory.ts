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

// Groups the frozen Graph's Node.type strings into the coarse categories the
// Sidebar filters by (CLAUDE.md repo layout: "Sidebar/ # Left nav: All,
// GitHub, ArgoCD, Istio, etc."). Kept separate from colors.ts's exact-type
// brand-color table since a filter category and a border color are
// different concerns that happen to both key off Node.type.
export type NodeCategory =
  | 'github'
  | 'docker'
  | 'argocd'
  | 'k8s'
  | 'istio'
  | 'kyverno'
  | 'prometheus'
  | 'other'

export const sidebarFilters: { id: NodeCategory | 'all'; label: string }[] = [
  { id: 'all', label: 'All' },
  { id: 'github', label: 'GitHub' },
  { id: 'docker', label: 'Docker' },
  { id: 'argocd', label: 'ArgoCD' },
  { id: 'k8s', label: 'Kubernetes' },
  { id: 'istio', label: 'Istio' },
  { id: 'kyverno', label: 'Kyverno' },
  { id: 'prometheus', label: 'Prometheus' },
]

export function categoryForNodeType(nodeType: string): NodeCategory {
  if (nodeType.startsWith('github')) return 'github'
  if (nodeType.startsWith('docker')) return 'docker'
  if (nodeType.startsWith('argocd')) return 'argocd'
  if (nodeType.startsWith('k8s')) return 'k8s'
  if (nodeType.startsWith('istio')) return 'istio'
  if (nodeType.startsWith('kyverno')) return 'kyverno'
  if (nodeType.startsWith('prometheus') || nodeType.startsWith('grafana')) return 'prometheus'
  return 'other'
}
