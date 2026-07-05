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

import type { DeploymentEvent } from '../../api/types'

interface DeploymentTimelineProps {
  events: DeploymentEvent[]
}

const statusGlyph: Record<DeploymentEvent['status'], string> = {
  success: '✓',
  failed: '✕',
  'in-progress': '…',
}

const statusColor: Record<DeploymentEvent['status'], string> = {
  success: 'var(--color-health-healthy)',
  failed: 'var(--color-health-down)',
  'in-progress': 'var(--color-health-degraded)',
}

export function DeploymentTimeline({ events }: DeploymentTimelineProps) {
  return (
    <section className="flex h-full flex-col border-r border-korion-border bg-korion-panel p-4">
      <h2 className="mb-1 text-sm font-semibold text-korion-text">Deployment Timeline</h2>
      <p className="mb-3 text-xs text-korion-text-muted">Real-time deployment events and status</p>

      <ol className="flex flex-col gap-3 overflow-y-auto">
        {events.map((event) => (
          <li key={event.id} className="flex items-start gap-3 text-sm">
            <span className="w-14 shrink-0 text-xs text-korion-text-muted">{event.timestamp}</span>
            <div className="min-w-0 flex-1">
              <p className="truncate text-korion-text">{event.title}</p>
              <p className="truncate text-xs text-korion-text-muted">{event.description}</p>
            </div>
            <span
              className="shrink-0 text-sm font-bold"
              style={{ color: statusColor[event.status] }}
              aria-label={event.status}
            >
              {statusGlyph[event.status]}
            </span>
          </li>
        ))}
      </ol>
    </section>
  )
}
