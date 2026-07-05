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

import { useState } from 'react'
import { healthDotColorFor } from '../../api/colors'
import type { Graph, GraphNode } from '../../api/types'
import { useUIStore } from '../../state/useUIStore'

const tabs = ['Overview', 'Metrics', 'Logs', 'Events'] as const
type Tab = (typeof tabs)[number]

interface ServiceDetailsProps {
  graph: Graph
}

function formatMetadataKey(key: string): string {
  const spaced = key.replace(/([a-z0-9])([A-Z])/g, '$1 $2')
  return spaced.charAt(0).toUpperCase() + spaced.slice(1)
}

function OverviewTab({ node }: { node: GraphNode }) {
  const entries = Object.entries(node.metadata ?? {})
  if (entries.length === 0) {
    return <p className="text-sm text-korion-text-muted">No metadata reported for this node.</p>
  }
  return (
    <dl className="flex flex-col gap-2 text-sm">
      {entries.map(([key, value]) => (
        <div key={key} className="flex items-baseline justify-between gap-4">
          <dt className="text-korion-text-muted">{formatMetadataKey(key)}</dt>
          <dd className="truncate text-right text-korion-text">
            {typeof value === 'object' ? JSON.stringify(value) : String(value)}
          </dd>
        </div>
      ))}
    </dl>
  )
}

function PlaceholderTab({ message }: { message: string }) {
  return <p className="text-sm text-korion-text-muted">{message}</p>
}

export function ServiceDetails({ graph }: ServiceDetailsProps) {
  const [activeTab, setActiveTab] = useState<Tab>('Overview')
  const selectedNodeId = useUIStore((s) => s.selectedNodeId)
  const node = graph.nodes.find((n) => n.id === selectedNodeId)

  return (
    <aside className="flex h-full w-80 shrink-0 flex-col border-l border-korion-border bg-korion-panel p-4">
      <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-korion-text-muted">
        Service Details
      </h2>

      {!node ? (
        <p className="text-sm text-korion-text-muted">Select a node to view details.</p>
      ) : (
        <>
          <div className="mb-4 flex items-center gap-2">
            <span
              className="h-2.5 w-2.5 rounded-full"
              style={{ backgroundColor: healthDotColorFor(node.status) }}
              aria-hidden
            />
            <h3 className="truncate text-base font-medium text-korion-text">{node.label}</h3>
          </div>

          <div role="tablist" aria-label="Service detail tabs" className="mb-4 flex gap-1 border-b border-korion-border">
            {tabs.map((tab) => (
              <button
                key={tab}
                type="button"
                role="tab"
                aria-selected={activeTab === tab}
                onClick={() => setActiveTab(tab)}
                className={`px-2 pb-2 text-sm ${
                  activeTab === tab
                    ? 'border-b-2 border-korion-cyan text-korion-cyan'
                    : 'text-korion-text-muted hover:text-korion-text'
                }`}
              >
                {tab}
              </button>
            ))}
          </div>

          <div role="tabpanel">
            {activeTab === 'Overview' && <OverviewTab node={node} />}
            {activeTab === 'Metrics' && (
              <PlaceholderTab message="No live metrics yet — Prometheus discovery lands in Phase 6." />
            )}
            {activeTab === 'Logs' && (
              <PlaceholderTab message="No logs yet — Loki collection is ARIA's responsibility (Phase 9)." />
            )}
            {activeTab === 'Events' && (
              <PlaceholderTab message="No recent events reported for this node." />
            )}
          </div>
        </>
      )}
    </aside>
  )
}
