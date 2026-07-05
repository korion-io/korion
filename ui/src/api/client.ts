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

import { mockPlatformMap } from './fixtures/mockPlatformMap'
import type { PlatformMapView } from './types'

/**
 * Fetches a PlatformMap view for the topology canvas and side panels.
 *
 * Phase 4: returns the static mock fixture. Phase 5 swaps this for a real
 * `fetch` against the Go controller's read API
 * (`GET /api/v1/platformmaps/{namespace}/{name}`) -- callers (usePlatformMap)
 * don't change, since the interface (namespace/name in, PlatformMapView out)
 * is the same either way.
 */
export async function getPlatformMap(
  namespace: string,
  name: string,
): Promise<PlatformMapView> {
  if (namespace !== mockPlatformMap.namespace || name !== mockPlatformMap.name) {
    throw new Error(`PlatformMap ${namespace}/${name} not found`)
  }
  return mockPlatformMap
}
