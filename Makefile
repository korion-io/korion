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

IMG ?= korion/korion:dev
UI_IMG ?= korion/korion-ui:dev
# CURDIR (set by make itself, not a subshell) avoids a `mingw32-make` bug on
# this host where `$(shell pwd)` mis-transcodes the em-dash in this repo's
# path and silently installs tools into a stray sibling directory.
LOCALBIN ?= $(CURDIR)/bin

CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ADDLICENSE ?= $(LOCALBIN)/addlicense

# controller-gen v0.21 errors fatally on any directory matched by a "..."
# glob that has no .go files directly in it (e.g. "./api/..." fails because
# api/ itself is empty, even though api/v1alpha1/ has files) -- list leaf
# package paths explicitly, semicolon-joined, and extend this as new
# packages land (internal/controller in Phase 2, internal/discovery and
# internal/graph in Phase 2/3, etc).
CONTROLLER_GEN_PATHS ?= ./api/v1alpha1/...;./internal/controller/...;./internal/discovery/...;./internal/graph/...

.PHONY: help
help: ## Show this help.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

##@ Code generation

.PHONY: generate
generate: controller-gen ## Generate DeepCopy methods for API types.
	"$(CONTROLLER_GEN)" object:headerFile="hack/boilerplate.go.txt" paths="$(CONTROLLER_GEN_PATHS)"

.PHONY: manifests
manifests: controller-gen ## Generate CRD manifests and RBAC from kubebuilder markers.
	"$(CONTROLLER_GEN)" rbac:roleName=manager-role crd webhook paths="$(CONTROLLER_GEN_PATHS)" output:crd:artifacts:config=config/crd/bases

##@ Testing

.PHONY: test
test: manifests generate ## Run Go tests. Set COVERAGE=true for a coverage report.
ifeq ($(COVERAGE),true)
	go test ./... -coverprofile=cover.out
	go tool cover -func=cover.out
else
	go test ./...
endif

.PHONY: license-check
license-check: addlicense ## Verify every .go/.py/.ts file has an Apache 2.0 header.
	@files="$$(git ls-files '*.go' '*.py' '*.ts' '*.tsx' 2>/dev/null)"; \
	if [ -z "$$files" ]; then echo "no .go/.py/.ts files tracked yet, skipping"; exit 0; fi; \
	"$(ADDLICENSE)" -check -l apache -c "The Korion Authors" \
		-ignore "ui/node_modules/**" -ignore "ui/dist/**" -ignore "aria/.venv/**" \
		$$files

.PHONY: license-fix
license-fix: addlicense ## Add missing Apache 2.0 headers.
	@files="$$(git ls-files '*.go' '*.py' '*.ts' '*.tsx' 2>/dev/null)"; \
	if [ -z "$$files" ]; then echo "no .go/.py/.ts files tracked yet, skipping"; exit 0; fi; \
	"$(ADDLICENSE)" -l apache -c "The Korion Authors" \
		-ignore "ui/node_modules/**" -ignore "ui/dist/**" -ignore "aria/.venv/**" \
		$$files

##@ Build

.PHONY: build
build: generate ## Build the controller manager binary.
	go build -o bin/manager ./cmd/manager

.PHONY: run
run: manifests generate ## Run the controller locally against the current kubeconfig.
	go run ./cmd/manager

.PHONY: docker-build
docker-build: ## Build the controller Docker image.
	docker build -t $(IMG) .

.PHONY: docker-push
docker-push: ## Push the controller Docker image.
	docker push $(IMG)

.PHONY: docker-build-ui
docker-build-ui: ## Build the UI Docker image.
	docker build -t $(UI_IMG) ui

.PHONY: docker-push-ui
docker-push-ui: ## Push the UI Docker image.
	docker push $(UI_IMG)

##@ Deployment

.PHONY: install
install: manifests ## Install CRDs into the current kubeconfig cluster.
	kubectl apply -f config/crd/bases

.PHONY: uninstall
uninstall: manifests ## Uninstall CRDs from the current kubeconfig cluster.
	kubectl delete -f config/crd/bases --ignore-not-found

.PHONY: deploy
deploy: manifests ## Deploy the controller to the cluster via Helm.
	helm upgrade --install korion ./helm/korion -n korion-system --create-namespace --set controller.image=$(IMG)

.PHONY: helm-sync-crds
helm-sync-crds: manifests ## Copy generated CRDs into the Helm chart's crds/ directory.
	mkdir -p helm/korion/crds
	cp config/crd/bases/*.yaml helm/korion/crds/

##@ Acceptance

.PHONY: e2e
e2e: ## Run the Phase 8 SuperHeros v0.1 acceptance harness (Kind + Helm + assert).
	bash test/e2e/run-acceptance.sh

##@ Tooling

.PHONY: controller-gen
controller-gen: ## Install controller-gen if not already present.
	@test -s "$(CONTROLLER_GEN)" || GOBIN="$(LOCALBIN)" go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest

.PHONY: addlicense
addlicense: ## Install addlicense if not already present.
	@test -s "$(ADDLICENSE)" || GOBIN="$(LOCALBIN)" go install github.com/google/addlicense@latest
