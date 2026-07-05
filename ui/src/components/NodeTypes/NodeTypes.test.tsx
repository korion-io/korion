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

import { render, screen } from '@testing-library/react'
import { ReactFlow, ReactFlowProvider } from '@xyflow/react'
import { describe, expect, it } from 'vitest'
import type { GraphNode } from '../../api/types'
import { nodeTypes, type ToolFlowNode } from './ToolNode'

function renderNode(node: GraphNode) {
  const flowNode: ToolFlowNode = {
    id: node.id,
    type: 'tool',
    position: { x: 0, y: 0 },
    data: { node },
  }
  return render(
    <ReactFlowProvider>
      <ReactFlow nodes={[flowNode]} edges={[]} nodeTypes={nodeTypes} />
    </ReactFlowProvider>,
  )
}

describe('ToolNode', () => {
  it('uses the node.brandColor as its border color', () => {
    renderNode({
      id: 'deployment/superheros/catalog',
      type: 'k8s-deployment',
      label: 'catalog',
      status: 'healthy',
      brandColor: '#326CE5',
    })

    const el = screen.getByTestId('tool-node')
    expect(el).toHaveStyle({ borderColor: '#326CE5' })
    expect(el).toHaveAttribute('data-node-type', 'k8s-deployment')
  })

  it('falls back to the neutral default color when brandColor is missing', () => {
    renderNode({
      id: 'unknown/thing',
      type: 'unknown-type',
      label: 'mystery',
      status: 'unknown',
    })

    const el = screen.getByTestId('tool-node')
    expect(el).toHaveStyle({ borderColor: '#6B7280' })
  })

  it('renders a health dot colored by status', () => {
    renderNode({
      id: 'deployment/superheros/payment',
      type: 'k8s-deployment',
      label: 'payment',
      status: 'down',
      brandColor: '#326CE5',
    })

    const dot = screen.getByTestId('health-dot')
    expect(dot).toHaveAttribute('data-status', 'down')
  })
})
