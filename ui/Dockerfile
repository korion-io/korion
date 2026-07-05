# Copyright 2026 The Korion Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Frontend image. Build the Vite static bundle, then serve it from nginx.
# The production build defaults API_BASE_URL to '' (same-origin), so in
# cluster the browser calls /api/... on this nginx, which the Helm chart's
# ConfigMap-mounted config proxies to the controller's read API Service.
#
# Node 24 matches the host toolchain that authors package-lock.json. We use
# `npm install`, not `npm ci`: the lock is generated on Windows, which never
# records the linux-x64 optional native subtrees (e.g. @emnapi/*, the
# @tailwindcss/oxide native binding), so strict `npm ci` fails its sync check
# on any linux build. `npm install` is lock-aware but reconciles the
# platform-specific optional deps for the build platform.
FROM node:24-alpine AS builder
WORKDIR /app

COPY package.json package-lock.json ./
RUN npm install --no-audit --no-fund

COPY . .
RUN npm run build

FROM nginx:1.27-alpine
# Baked default config serves the SPA for standalone `docker run`. In cluster
# the Helm chart overrides /etc/nginx/conf.d/default.conf via a ConfigMap
# volume so it can also proxy /api/ to the controller Service.
COPY nginx.default.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80
