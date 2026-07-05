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

import { useMemo } from 'react'
import { Background, Controls, ReactFlow, type Edge } from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { categoryForNodeType } from '../../api/nodeCategory'
import type { Graph } from '../../api/types'
import { useUIStore } from '../../state/useUIStore'
import { nodeTypes, type ToolFlowNode } from '../NodeTypes/ToolNode'
import { layoutGraph } from './layout'

interface TopologyCanvasProps {
  graph: Graph
}

export function TopologyCanvas({ graph }: TopologyCanvasProps) {
  const selectedNodeId = useUIStore((s) => s.selectedNodeId)
  const activeFilter = useUIStore((s) => s.activeFilter)
  const selectNode = useUIStore((s) => s.selectNode)

  const positions = useMemo(() => layoutGraph(graph), [graph])

  const visibleNodeIds = useMemo(() => {
    if (activeFilter === 'all') return new Set(graph.nodes.map((n) => n.id))
    return new Set(
      graph.nodes.filter((n) => categoryForNodeType(n.type) === activeFilter).map((n) => n.id),
    )
  }, [graph.nodes, activeFilter])

  const flowNodes: ToolFlowNode[] = useMemo(
    () =>
      graph.nodes
        .filter((n) => visibleNodeIds.has(n.id))
        .map((n) => ({
          id: n.id,
          type: 'tool',
          position: positions[n.id] ?? { x: 0, y: 0 },
          data: { node: n },
          selected: n.id === selectedNodeId,
        })),
    [graph.nodes, visibleNodeIds, positions, selectedNodeId],
  )

  const flowEdges: Edge[] = useMemo(
    () =>
      graph.edges
        .filter((e) => visibleNodeIds.has(e.from) && visibleNodeIds.has(e.to))
        .map((e) => ({
          id: `${e.from}->${e.to}-${e.type}`,
          source: e.from,
          target: e.to,
          label: e.label,
          style: { stroke: '#1b2536' },
          labelStyle: { fill: '#8b95a7', fontSize: 11 },
        })),
    [graph.edges, visibleNodeIds],
  )

  return (
    <div data-testid="topology-canvas" className="h-full w-full bg-korion-bg">
      <ReactFlow
        nodes={flowNodes}
        edges={flowEdges}
        nodeTypes={nodeTypes}
        onNodeClick={(_, node) => selectNode(node.id)}
        onPaneClick={() => selectNode(null)}
        fitView
        proOptions={{ hideAttribution: true }}
      >
        <Background color="#1b2536" gap={24} />
        <Controls className="!bg-korion-panel !fill-korion-text !text-korion-text" />
      </ReactFlow>
    </div>
  )
}
