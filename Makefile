BUILD_PATH ?= $(shell pwd)/build
# Image URL to use all building/pushing image targets
IMG ?= bk-micro-gateway-operator:development

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
GOBIN=$(LOCALBIN)
$(LOCALBIN):
	mkdir -p $(LOCALBIN)


## Tool Binaries
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
MOCKGEN = $(LOCALBIN)/mockgen
GINKGO = $(LOCALBIN)/ginkgo
KUSTOMIZE = $(LOCALBIN)/kustomize
GOLINES = $(LOCALBIN)/golines
GOFUMPT = $(LOCALBIN)/gofumpt
CONTROLLER_GEN = $(LOCALBIN)/controller-gen
CONTROLLER_GEN_OLD = $(LOCALBIN)/controller-gen-old
SETUP_ENVTEST = $(LOCALBIN)/setup-envtest

## Tool Versions
GOLANGCI_LINT_VERSION ?= v2.1.5
ENVTEST_K8S_VERSION = 1.22  # ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
CONTROLLER_TOOLS_VERSION ?= v0.18.0
MOCKGEN_VERSION ?= v1.6.0
GINKGO_VERSION ?= v2.3.1
KUSTOMIZE_VERSION ?= 3.8.7


# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all


init: $(LOCALBIN)
	pip install pre-commit
	pre-commit install
	# for golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LOCALBIN) $(GOLANGCI_LINT_VERSION)
	# for make mock
	GOBIN=$(LOCALBIN) go install github.com/golang/mock/mockgen@$(MOCKGEN_VERSION)
	# for ginkgo
	GOBIN=$(LOCALBIN) go install github.com/onsi/ginkgo/v2/ginkgo@$(GINKGO_VERSION)
	# for gofumpt
	GOBIN=$(LOCALBIN) go install mvdan.cc/gofumpt@latest
	# for golines
	GOBIN=$(LOCALBIN) go install github.com/segmentio/golines@latest
	# for envtest
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
	# for controller-gen-old
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.6.0  && \
	    mv $(LOCALBIN)/controller-gen  $(LOCALBIN)/controller-gen-old
	# for controller-gen
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)
	# for kustomize
	curl -Ss https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh --output install_kustomize.sh \
	    && bash install_kustomize.sh $(KUSTOMIZE_VERSION)  $(LOCALBIN); rm install_kustomize.sh;



all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests

$(BUILD_PATH):
	mkdir -p $(BUILD_PATH)

manifests: ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(CONTROLLER_GEN_OLD) crd:crdVersions=v1beta1 paths="./..." output:crd:artifacts:config=config/crd/v1beta1

.PHONY: generate
generate: ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: gofumpt
gofumpt: ## Run go fmt against code.
	$(GOLINES) ./ -m 120 -w --base-formatter gofmt --no-reformat-tags
	$(GOFUMPT) -l -w .

.PHONY: vet
vet: fmt ## Run go vet against code.
	go vet ./...

lint: vet
	$(GOLANGCI_LINT) run  ./...

.PHONY: test
test: ## Run tests.
	KUBEBUILDER_ASSETS="$(shell setup-envtest use $(ENVTEST_K8S_VERSION) -p path)" \
	$(GINKGO) --skip-package=vendor,tests/integration -ldflags="-s=false" -gcflags="-l" --cover --coverprofile cover.out ./...

##@ Build
build-common: $(BUILD_PATH) generate manifests fmt vet

.PHONY: build
build: ## Build manager binary.
	go build -ldflags "-X github.com/TencentBlueKing/blueking-apigateway-operator/pkg/version.Version=`git describe --tags --abbrev=0`  \
		-X github.com/TencentBlueKing/blueking-apigateway-operator/pkg/version.Commit=`git rev-parse HEAD` \
		-X github.com/TencentBlueKing/blueking-apigateway-operator/pkg/version.BuildTime=`date +%Y-%m-%d_%I:%M:%S` \
		-X 'github.com/TencentBlueKing/blueking-apigateway-operator/pkg/version.GoVersion=`go version`'" \
		-o $(BUILD_PATH)/micro-gateway-operator .

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	docker build -t ${IMG} .

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests   ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && kustomize edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

integration: docker-build
	cd tests/integration && docker compose down && docker compose up -d && ginkgo -ldflags="-s=false" -gcflags="-l";docker compose down


