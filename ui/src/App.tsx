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

import type { ReactNode } from 'react'
import { DeploymentTimeline } from './components/DeploymentTimeline'
import { PolicyPanel } from './components/PolicyPanel'
import { ServiceDetails } from './components/ServiceDetails'
import { Sidebar } from './components/Sidebar'
import { TopologyCanvas } from './components/TopologyCanvas'
import { usePlatformMap } from './hooks/usePlatformMap'

// Phase 4 renders against the mock fixture; Phase 5 makes this namespace/name
// pair a real route param once the canvas is wired to the live controller.
const NAMESPACE = 'superheros'
const NAME = 'superheros-platform'

function CenteredMessage({ children }: { children: ReactNode }) {
  return (
    <div className="flex h-full items-center justify-center text-sm text-korion-text-muted">
      {children}
    </div>
  )
}

function App() {
  const { data, isLoading, isError, error } = usePlatformMap(NAMESPACE, NAME)

  return (
    <div className="flex h-screen overflow-hidden bg-korion-bg text-korion-text">
      <Sidebar />

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex shrink-0 items-center justify-between border-b border-korion-border px-6 py-4">
          <div>
            <h1 className="text-lg font-semibold">Architecture Flow</h1>
            <p className="text-xs text-korion-text-muted">
              Live view of your DevOps workflow and infrastructure
            </p>
          </div>
          {data && (
            <span className="text-xs text-korion-text-muted">
              {data.cluster} · {data.namespace}
            </span>
          )}
        </header>

        <div className="flex min-h-0 flex-1">
          <main className="min-w-0 flex-1">
            {isLoading && <CenteredMessage>Loading topology…</CenteredMessage>}
            {isError && (
              <CenteredMessage>
                Failed to load PlatformMap: {error instanceof Error ? error.message : 'unknown error'}
              </CenteredMessage>
            )}
            {data && <TopologyCanvas graph={data.topology} />}
          </main>
          {data && <ServiceDetails graph={data.topology} />}
        </div>

        <div className="grid h-64 shrink-0 grid-cols-2 border-t border-korion-border">
          {data && (
            <>
              <DeploymentTimeline events={data.deploymentEvents} />
              <PolicyPanel summary={data.policySummary} />
            </>
          )}
        </div>
      </div>
    </div>
  )
}

export default App
