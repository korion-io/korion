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
import { PlatformMapNotFoundError } from './api/client'
import { DeploymentTimeline } from './components/DeploymentTimeline'
import { PolicyPanel } from './components/PolicyPanel'
import { ServiceDetails } from './components/ServiceDetails'
import { Sidebar } from './components/Sidebar'
import { TopologyCanvas } from './components/TopologyCanvas'
import { usePlatformMap } from './hooks/usePlatformMap'

// Which PlatformMap to poll -- defaults to the SuperHeros sample
// (config/samples/platformmap-superheros.yaml) so this points at the
// canonical test case out of the box, but is overridable via env for
// pointing a local dev build at a different cluster/PlatformMap (e.g. Phase
// 2's manual demo/demo-platform) without a code change.
const NAMESPACE = import.meta.env.VITE_PLATFORMMAP_NAMESPACE ?? 'superheros'
const NAME = import.meta.env.VITE_PLATFORMMAP_NAME ?? 'superheros-platform'

function CenteredMessage({ children }: { children: ReactNode }) {
  return (
    <div className="flex h-full items-center justify-center px-8 text-center text-sm text-korion-text-muted">
      {children}
    </div>
  )
}

function App() {
  const { data, isLoading, isError, error } = usePlatformMap(NAMESPACE, NAME)
  const reconciled = data !== undefined && data.lastDiscoveryTime !== null

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
              {data.namespace}/{data.name}
              {data.lastDiscoveryTime &&
                ` · last discovered ${new Date(data.lastDiscoveryTime).toLocaleTimeString()}`}
            </span>
          )}
        </header>

        <div className="flex min-h-0 flex-1">
          <main className="min-w-0 flex-1">
            {isLoading && <CenteredMessage>Loading topology…</CenteredMessage>}
            {isError && error instanceof PlatformMapNotFoundError && (
              <CenteredMessage>
                {error.message}. Apply a PlatformMap to this cluster (see{' '}
                config/samples/platformmap-superheros.yaml) and it will appear here automatically.
              </CenteredMessage>
            )}
            {isError && !(error instanceof PlatformMapNotFoundError) && (
              <CenteredMessage>
                Failed to load PlatformMap: {error instanceof Error ? error.message : 'unknown error'}.
                Is the Korion controller's read API reachable?
              </CenteredMessage>
            )}
            {data && !reconciled && (
              <CenteredMessage>
                PlatformMap {data.namespace}/{data.name} was applied but hasn't completed its first
                discovery reconcile yet -- this refreshes automatically.
              </CenteredMessage>
            )}
            {data && reconciled && <TopologyCanvas graph={data.topology} />}
          </main>
          {data && reconciled && <ServiceDetails graph={data.topology} />}
        </div>

        <div className="grid h-64 shrink-0 grid-cols-2 border-t border-korion-border">
          {data && reconciled && (
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
