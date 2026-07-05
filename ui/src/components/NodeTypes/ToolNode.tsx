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

import { Handle, Position, type Node, type NodeProps } from '@xyflow/react'
import { defaultBrandColor } from '../../api/colors'
import type { GraphNode } from '../../api/types'
import { HealthDot } from './HealthDot'

export type ToolFlowNode = Node<{ node: GraphNode }, 'tool'>

/**
 * The single custom node component used for every tool type on the canvas.
 * One component keyed off data driven by `node.type`/`node.brandColor`, per
 * docs/PLAN.md Phase 4 ("one custom node component per tool type, border
 * color from the shared brand-color table") -- "per tool type" is achieved
 * through data, not a component per type, so adding a Phase 6 discovery
 * engine's new Node.type needs no new component.
 */
export function ToolNode({ data, selected }: NodeProps<ToolFlowNode>) {
  const { node } = data
  const borderColor = node.brandColor ?? defaultBrandColor

  return (
    <div
      data-testid="tool-node"
      data-node-type={node.type}
      className={`min-w-[180px] rounded-lg border-2 bg-korion-panel px-3 py-2 shadow-lg ${
        selected ? 'ring-2 ring-korion-cyan' : ''
      }`}
      style={{ borderColor }}
    >
      <Handle type="target" position={Position.Top} className="!bg-korion-border" />
      <div className="flex items-center justify-between gap-2">
        <span className="truncate text-sm font-medium text-korion-text">{node.label}</span>
        <HealthDot status={node.status} />
      </div>
      <div className="mt-0.5 truncate text-xs text-korion-text-muted">{node.type}</div>
      <Handle type="source" position={Position.Bottom} className="!bg-korion-border" />
    </div>
  )
}

export const nodeTypes = { tool: ToolNode }
