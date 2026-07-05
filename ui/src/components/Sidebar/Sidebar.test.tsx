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

import { fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import { useUIStore } from '../../state/useUIStore'
import { Sidebar } from './Sidebar'

afterEach(() => {
  useUIStore.setState({ activeFilter: 'all', selectedNodeId: null })
})

describe('Sidebar', () => {
  it('marks "All" as the active filter by default', () => {
    render(<Sidebar />)
    expect(screen.getByRole('button', { name: 'All' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('updates the shared UI store filter when a filter is clicked', () => {
    render(<Sidebar />)

    fireEvent.click(screen.getByRole('button', { name: 'ArgoCD' }))

    expect(useUIStore.getState().activeFilter).toBe('argocd')
    expect(screen.getByRole('button', { name: 'ArgoCD' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'All' })).toHaveAttribute('aria-pressed', 'false')
  })
})
