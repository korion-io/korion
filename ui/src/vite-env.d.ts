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

/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Base URL of the Go controller's read API (internal/api/server.go).
   * Defaults to http://localhost:8082 in dev, same-origin ('') in production. */
  readonly VITE_API_BASE_URL?: string
  /** PlatformMap namespace/name to poll -- defaults to the SuperHeros sample
   * (config/samples/platformmap-superheros.yaml). */
  readonly VITE_PLATFORMMAP_NAMESPACE?: string
  readonly VITE_PLATFORMMAP_NAME?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
