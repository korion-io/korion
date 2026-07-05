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

import type { Graph, PlatformMapCondition, PlatformMapView } from './types'

// The Go manager's default --api-bind-address (cmd/manager/main.go). Used
// only as a dev-mode fallback when VITE_API_BASE_URL isn't set -- production
// builds default to '' (same-origin, behind the Helm-installed Service).
const DEFAULT_DEV_API_BASE_URL = 'http://localhost:8082'

const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ?? (import.meta.env.DEV ? DEFAULT_DEV_API_BASE_URL : '')

/**
 * Thrown when the read API returns 404 -- the PlatformMap doesn't exist (or
 * hasn't been applied yet / wrong namespace-name), as distinct from a network
 * failure or server error. Callers use this to render a specific "apply your
 * PlatformMap" message instead of a generic failure.
 */
export class PlatformMapNotFoundError extends Error {
  constructor(namespace: string, name: string) {
    super(`PlatformMap ${namespace}/${name} not found`)
    this.name = 'PlatformMapNotFoundError'
  }
}

// Mirrors the JSON shape api/v1alpha1/platformmap_types.go's PlatformMap
// produces, as served by GET /api/v1/platformmaps/{namespace}/{name}
// (internal/api/server.go). Only the fields the UI reads are declared.
interface PlatformMapResource {
  metadata: {
    name: string
    namespace: string
  }
  status?: {
    topology?: Graph
    conditions?: PlatformMapCondition[]
    lastDiscoveryTime?: string
  }
}

function toPlatformMapView(resource: PlatformMapResource): PlatformMapView {
  return {
    name: resource.metadata.name,
    namespace: resource.metadata.namespace,
    topology: resource.status?.topology ?? { nodes: [], edges: [] },
    conditions: resource.status?.conditions ?? [],
    lastDiscoveryTime: resource.status?.lastDiscoveryTime ?? null,
    // Not yet produced by the Go controller: ArgoCD/GitHub discovery (which
    // would drive the Deployment Timeline) and Kyverno discovery (which
    // would drive the Policy panel) land in Phase 6. Empty/zeroed until then
    // rather than fabricated, so the UI honestly reflects what's discovered.
    deploymentEvents: [],
    policySummary: { total: 0, passed: 0, warnings: 0, failed: 0, violations: [] },
  }
}

/**
 * Fetches a PlatformMap view for the topology canvas and side panels from
 * the Go controller's read-only HTTP API (internal/api/server.go).
 */
export async function getPlatformMap(
  namespace: string,
  name: string,
): Promise<PlatformMapView> {
  const res = await fetch(`${API_BASE_URL}/api/v1/platformmaps/${namespace}/${name}`)

  if (res.status === 404) {
    throw new PlatformMapNotFoundError(namespace, name)
  }
  if (!res.ok) {
    throw new Error(
      `GET platformmaps/${namespace}/${name} failed: ${res.status} ${res.statusText}`,
    )
  }

  const resource = (await res.json()) as PlatformMapResource
  return toPlatformMapView(resource)
}
