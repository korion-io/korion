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

import type { PolicySummary } from '../../api/types'

interface PolicyPanelProps {
  summary: PolicySummary
}

const resultColor: Record<string, string> = {
  pass: 'var(--color-health-healthy)',
  warn: 'var(--color-health-degraded)',
  fail: 'var(--color-health-down)',
}

const resultLabel: Record<string, string> = {
  pass: 'Passed',
  warn: 'Warning',
  fail: 'Failed',
}

export function PolicyPanel({ summary }: PolicyPanelProps) {
  return (
    <section className="flex h-full flex-col border-l border-korion-border bg-korion-panel p-4">
      <h2 className="mb-1 text-sm font-semibold text-korion-text">Policy & Security</h2>
      <p className="mb-3 text-xs text-korion-text-muted">Kyverno policy reports and security insights</p>

      <div className="mb-4 grid grid-cols-4 gap-2 text-center">
        <div className="rounded-md bg-korion-bg p-2">
          <div className="text-xs text-korion-text-muted">Policies</div>
          <div className="text-lg font-semibold text-korion-text">{summary.total}</div>
        </div>
        <div className="rounded-md bg-korion-bg p-2">
          <div className="text-xs text-korion-text-muted">Passed</div>
          <div className="text-lg font-semibold" style={{ color: resultColor.pass }}>
            {summary.passed}
          </div>
        </div>
        <div className="rounded-md bg-korion-bg p-2">
          <div className="text-xs text-korion-text-muted">Warning</div>
          <div className="text-lg font-semibold" style={{ color: resultColor.warn }}>
            {summary.warnings}
          </div>
        </div>
        <div className="rounded-md bg-korion-bg p-2">
          <div className="text-xs text-korion-text-muted">Failed</div>
          <div className="text-lg font-semibold" style={{ color: resultColor.fail }}>
            {summary.failed}
          </div>
        </div>
      </div>

      <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-korion-text-muted">
        Recent Violations
      </h3>
      <ul className="flex flex-col gap-2 overflow-y-auto">
        {summary.violations.map((violation) => (
          <li key={violation.id} className="text-sm">
            <div className="flex items-center justify-between gap-2">
              <span className="truncate text-korion-text">{violation.policy}</span>
              <span
                className="shrink-0 rounded px-1.5 py-0.5 text-xs font-medium"
                style={{
                  color: resultColor[violation.result],
                  backgroundColor: 'var(--color-korion-bg)',
                }}
              >
                {resultLabel[violation.result]}
              </span>
            </div>
            <div className="text-xs text-korion-text-muted">
              {violation.resource} · {violation.timestamp}
            </div>
          </li>
        ))}
      </ul>
    </section>
  )
}
