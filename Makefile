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
LOCALBIN ?= $(shell pwd)/bin

CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ADDLICENSE ?= $(LOCALBIN)/addlicense

.PHONY: help
help: ## Show this help.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

##@ Code generation

.PHONY: generate
generate: controller-gen ## Generate DeepCopy methods for API types.
	"$(CONTROLLER_GEN)" object:headerFile="hack/boilerplate.go.txt" paths="./api/..."

.PHONY: manifests
manifests: controller-gen ## Generate CRD manifests and RBAC from kubebuilder markers.
	"$(CONTROLLER_GEN)" rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

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

##@ Tooling

.PHONY: controller-gen
controller-gen: ## Install controller-gen if not already present.
	@test -s "$(CONTROLLER_GEN)" || GOBIN="$(LOCALBIN)" go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest

.PHONY: addlicense
addlicense: ## Install addlicense if not already present.
	@test -s "$(ADDLICENSE)" || GOBIN="$(LOCALBIN)" go install github.com/google/addlicense@latest
