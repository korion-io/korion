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

import { afterEach, describe, expect, it, vi } from 'vitest'
import { getPlatformMap, PlatformMapNotFoundError } from './client'

function jsonResponse(body: unknown, init?: ResponseInit): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
}

describe('getPlatformMap', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('transforms a reconciled PlatformMap resource into a PlatformMapView', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({
        metadata: { name: 'demo-platform', namespace: 'demo' },
        status: {
          topology: {
            nodes: [{ id: 'deployment/demo/frontend', type: 'k8s-deployment', label: 'frontend', status: 'healthy' }],
            edges: [],
          },
          conditions: [{ type: 'K8sDetected', status: 'True' }],
          lastDiscoveryTime: '2026-07-06T12:00:00Z',
        },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    const view = await getPlatformMap('demo', 'demo-platform')

    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/platformmaps/demo/demo-platform'),
    )
    expect(view.name).toBe('demo-platform')
    expect(view.namespace).toBe('demo')
    expect(view.topology.nodes).toHaveLength(1)
    expect(view.conditions).toEqual([{ type: 'K8sDetected', status: 'True' }])
    expect(view.lastDiscoveryTime).toBe('2026-07-06T12:00:00Z')
    expect(view.deploymentEvents).toEqual([])
    expect(view.policySummary).toEqual({ total: 0, passed: 0, warnings: 0, failed: 0, violations: [] })
  })

  it('defaults topology/conditions/lastDiscoveryTime when status is absent (not yet reconciled)', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(jsonResponse({ metadata: { name: 'demo-platform', namespace: 'demo' } })),
    )

    const view = await getPlatformMap('demo', 'demo-platform')

    expect(view.topology).toEqual({ nodes: [], edges: [] })
    expect(view.conditions).toEqual([])
    expect(view.lastDiscoveryTime).toBeNull()
  })

  it('throws PlatformMapNotFoundError on a 404', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('not found', { status: 404 })))

    await expect(getPlatformMap('demo', 'missing')).rejects.toThrow(PlatformMapNotFoundError)
  })

  it('throws a generic Error on a non-404 failure', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response('boom', { status: 500, statusText: 'Internal Server Error' })),
    )

    await expect(getPlatformMap('demo', 'demo-platform')).rejects.toThrow(/500/)
  })
})
