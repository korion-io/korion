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

import { create } from 'zustand'
import type { NodeCategory } from '../api/nodeCategory'

interface UIState {
  selectedNodeId: string | null
  activeFilter: NodeCategory | 'all'
  selectNode: (id: string | null) => void
  setActiveFilter: (filter: NodeCategory | 'all') => void
}

// Cross-panel UI state that isn't server data: which node is selected
// (drives ServiceDetails) and which sidebar filter is active (drives canvas
// visibility). Plain zustand, not Redux, per docs/PLAN.md Phase 4.
export const useUIStore = create<UIState>((set) => ({
  selectedNodeId: null,
  activeFilter: 'all',
  selectNode: (id) => set({ selectedNodeId: id }),
  setActiveFilter: (filter) => set({ activeFilter: filter }),
}))
