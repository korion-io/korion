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

import type { Graph } from '../../api/types'

export interface LayoutPosition {
  x: number
  y: number
}

const COLUMN_WIDTH = 240
const ROW_HEIGHT = 110

/**
 * Assigns each node a column (BFS depth from a root with no incoming edges)
 * and a row (order within that column), giving a left-to-right layered
 * layout that approximates the mockup's pipeline flow without pulling in a
 * full graph-layout dependency (e.g. dagre) for Phase 4's mock data.
 */
export function layoutGraph(graph: Graph): Record<string, LayoutPosition> {
  const incomingCount = new Map<string, number>()
  graph.nodes.forEach((n) => incomingCount.set(n.id, 0))
  graph.edges.forEach((e) => {
    if (incomingCount.has(e.to)) {
      incomingCount.set(e.to, (incomingCount.get(e.to) ?? 0) + 1)
    }
  })

  const adjacency = new Map<string, string[]>()
  graph.edges.forEach((e) => {
    adjacency.set(e.from, [...(adjacency.get(e.from) ?? []), e.to])
  })

  const depth = new Map<string, number>()
  const queue: string[] = []
  graph.nodes.forEach((n) => {
    if ((incomingCount.get(n.id) ?? 0) === 0) {
      depth.set(n.id, 0)
      queue.push(n.id)
    }
  })

  let iterations = 0
  const maxIterations = graph.nodes.length * 4
  while (queue.length > 0 && iterations < maxIterations) {
    iterations += 1
    const current = queue.shift()!
    const currentDepth = depth.get(current) ?? 0
    for (const next of adjacency.get(current) ?? []) {
      if ((depth.get(next) ?? -1) < currentDepth + 1) {
        depth.set(next, currentDepth + 1)
        queue.push(next)
      }
    }
  }

  graph.nodes.forEach((n) => {
    if (!depth.has(n.id)) depth.set(n.id, 0)
  })

  const rowsUsedPerColumn = new Map<number, number>()
  const positions: Record<string, LayoutPosition> = {}
  graph.nodes.forEach((n) => {
    const column = depth.get(n.id) ?? 0
    const row = rowsUsedPerColumn.get(column) ?? 0
    rowsUsedPerColumn.set(column, row + 1)
    positions[n.id] = { x: column * COLUMN_WIDTH, y: row * ROW_HEIGHT }
  })

  return positions
}
