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

import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import type { Graph } from '../types'
import uiCopy from './sample-topology.json'

// Guards against ui/src/api/fixtures/sample-topology.json drifting from the
// Go-owned frozen contract fixture (internal/graph/testdata/sample-topology.json,
// see docs/PLAN.md Phase 3 and internal/graph/contract_test.go). This is the
// frontend half of the same "lightweight contract test, not codegen" approach.
describe('sample-topology.json frozen contract', () => {
  it('matches internal/graph/testdata/sample-topology.json byte-for-structure', () => {
    const goFixturePath = resolve(
      __dirname,
      '../../../../internal/graph/testdata/sample-topology.json',
    )
    const goFixture = JSON.parse(readFileSync(goFixturePath, 'utf-8'))
    expect(uiCopy).toEqual(goFixture)
  })

  it('parses as a valid Graph', () => {
    const graph: Graph = uiCopy as Graph
    expect(graph.nodes.length).toBeGreaterThan(0)
    expect(graph.nodes[0]).toHaveProperty('id')
    expect(graph.nodes[0]).toHaveProperty('brandColor')
    expect(graph.edges[0]).toHaveProperty('from')
    expect(graph.edges[0]).toHaveProperty('to')
  })
})
