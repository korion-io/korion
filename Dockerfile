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

# Controller manager image. Multi-stage: build the static Go binary, then
# ship it on distroless/static:nonroot so the running image has no shell,
# no package manager, and a non-root default user.
FROM golang:1.26 AS builder
WORKDIR /workspace

# Cache module downloads before copying the rest of the source.
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/

# CGO disabled + GOOS/GOARCH pinned so the result is a fully static binary
# that runs on distroless. TARGETARCH is provided by BuildKit for multi-arch
# builds and defaults to amd64.
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -a -ldflags="-s -w" -o manager ./cmd/manager

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager /manager
USER 65532:65532
ENTRYPOINT ["/manager"]
