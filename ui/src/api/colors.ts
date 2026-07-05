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

// Mirrors internal/graph/colors.go's brandColors table (CLAUDE.md rule #7,
// "Tool brand colors for nodes"). The Go builder stamps GraphNode.brandColor
// server-side, so in the real (Phase 5+) API this table is only a fallback;
// it's kept here (not invented ad hoc) so mock data built ahead of a real
// discovery engine still gets the correct color, and so this file itself is
// a second, independent check that the two tables agree.
const brandColors: Record<string, string> = {
  'k8s-deployment': '#326CE5',
  'k8s-service': '#326CE5',

  'argocd-application': '#EF7B4D',
  'istio-virtualservice': '#466BB0',
  'istio-destinationrule': '#466BB0',
  'kyverno-policyreport': '#1E40AF',
  'kyverno-clusterpolicyreport': '#1E40AF',
  'github-actions-workflow': '#3B82F6',
  'github-repository': '#333333',
  'docker-image': '#2496ED',
  'prometheus-metric': '#EA580C',
  'grafana-dashboard': '#F46800',
  'loki-log-source': '#10B981',
}

export const defaultBrandColor = '#6B7280'

export function brandColorFor(nodeType: string): string {
  return brandColors[nodeType] ?? defaultBrandColor
}

export const healthDotColors: Record<string, string> = {
  healthy: 'var(--color-health-healthy)',
  degraded: 'var(--color-health-degraded)',
  down: 'var(--color-health-down)',
  unknown: 'var(--color-health-unknown)',
}

export function healthDotColorFor(status: string): string {
  return healthDotColors[status] ?? healthDotColors.unknown
}
