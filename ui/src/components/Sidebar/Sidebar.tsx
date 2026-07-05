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

import { sidebarFilters } from '../../api/nodeCategory'
import { useUIStore } from '../../state/useUIStore'

const legend: { label: string; color: string }[] = [
  { label: 'Healthy', color: 'var(--color-health-healthy)' },
  { label: 'Degraded', color: 'var(--color-health-degraded)' },
  { label: 'Down', color: 'var(--color-health-down)' },
  { label: 'Unknown', color: 'var(--color-health-unknown)' },
]

export function Sidebar() {
  const activeFilter = useUIStore((s) => s.activeFilter)
  const setActiveFilter = useUIStore((s) => s.setActiveFilter)

  return (
    <aside className="flex h-full w-56 shrink-0 flex-col border-r border-korion-border bg-korion-panel p-4">
      <div className="mb-6 flex items-center gap-2">
        <span className="h-6 w-6 rounded bg-korion-cyan" aria-hidden />
        <span className="text-lg font-semibold text-korion-text">Korion</span>
      </div>

      <nav aria-label="Topology filters" className="flex flex-col gap-1">
        {sidebarFilters.map((filter) => {
          const isActive = filter.id === activeFilter
          return (
            <button
              key={filter.id}
              type="button"
              onClick={() => setActiveFilter(filter.id)}
              aria-pressed={isActive}
              className={`rounded-md px-3 py-2 text-left text-sm transition-colors ${
                isActive
                  ? 'bg-korion-cyan/10 text-korion-cyan'
                  : 'text-korion-text-muted hover:bg-korion-border/40 hover:text-korion-text'
              }`}
            >
              {filter.label}
            </button>
          )
        })}
      </nav>

      <div className="mt-auto flex flex-col gap-2 border-t border-korion-border pt-4">
        {legend.map((item) => (
          <div key={item.label} className="flex items-center gap-2 text-xs text-korion-text-muted">
            <span
              className="h-2 w-2 rounded-full"
              style={{ backgroundColor: item.color }}
              aria-hidden
            />
            {item.label}
          </div>
        ))}
      </div>
    </aside>
  )
}
