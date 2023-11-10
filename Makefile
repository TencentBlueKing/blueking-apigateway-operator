BUILD_PATH ?= $(shell pwd)/build
# Image URL to use all building/pushing image targets
IMG ?= bk-micro-gateway-operator:development
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.22

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all


init:
	pip install pre-commit
	pre-commit install
	# for golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.46.2
	# for make mock
	go install github.com/golang/mock/mockgen@v1.6.0
	# for ginkgo
	go install github.com/onsi/ginkgo/v2/ginkgo@v2.3.1
	# for gofumpt
	go install mvdan.cc/gofumpt@latest
	# for golines
	go install github.com/segmentio/golines@latest
	# for envtest
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
	# for controller-gen-old
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.6.0  && \
	    mv $(shell go env GOPATH)/bin/controller-gen  $(shell go env GOPATH)/bin/controller-gen-old
	# for controller-gen
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.2
	# for kustomize
	curl -Ss https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh --output install_kustomize.sh \
	    && bash install_kustomize.sh 3.8.7  $(shell go env GOPATH)/bin; rm install_kustomize.sh;



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
	controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	controller-gen-old crd:crdVersions=v1beta1 paths="./..." output:crd:artifacts:config=config/crd/v1beta1

.PHONY: generate
generate: ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: gofumpt
gofumpt: ## Run go fmt against code.
	golines ./ -m 120 -w --base-formatter gofmt --no-reformat-tags
	gofumpt -l -w .

.PHONY: vet
vet: fmt ## Run go vet against code.
	go vet ./...

lint: vet
	golangci-lint run

.PHONY: test
test: ## Run tests.
	KUBEBUILDER_ASSETS="$(shell setup-envtest use $(ENVTEST_K8S_VERSION) -p path)" \
	ginkgo --skip-package=vendor,tests/integration -ldflags="-s=false" -gcflags="-l" --cover --coverprofile cover.out ./...

##@ Build
build-common: $(BUILD_PATH) generate manifests fmt vet

.PHONY: build
build: ## Build manager binary.
	go build -ldflags "-X github.com/TencentBlueKing/blueking-apigateway-operator/pkg/version.Version=${VERSION}  \
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
	kustomize build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	kustomize build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	kustomize build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

integration:
	cd tests/integration && docker-compose down && docker-compose up -d && ginkgo -ldflags="-s=false" -gcflags="-l";docker-compose down


