# NOTE: kind version is specified at aqua.yaml
E2ETEST_K8S_VERSION  := 1.30.4
CERT_MANAGER_VERSION := 1.15.0

PROJECT_DIR := $(CURDIR)/../
BIN_DIR := $(PROJECT_DIR)/bin

CURL := curl -sSLf
KUBECTL := $(BIN_DIR)/kubectl

KIND_CLUSTER_NAME := cat-gate
KIND_CONFIG := kind-config.yaml
export KUBECONFIG

.PHONY: help
help:
	@echo "Choose one of the following target"
	@echo
	@echo "setup      Setup tools"
	@echo "start      Start kind cluster and install accurate"
	@echo "test       Run e2e tests"
	@echo "logs       Save logs as logs.tar.gz"
	@echo "stop       Stop the kind cluster"

.PHONY: setup
setup:
	mkdir -p $(BIN_DIR)
	$(CURL) -o $(BIN_DIR)/kubectl https://storage.googleapis.com/kubernetes-release/release/v$(E2ETEST_K8S_VERSION)/bin/linux/amd64/kubectl && chmod a+x $(BIN_DIR)/kubectl
	# TODO: specify kind version
	GOBIN=$(BIN_DIR) go install sigs.k8s.io/kind@latest

.PHONY: start
start:
	kind create cluster --name=$(KIND_CLUSTER_NAME) --config=$(KIND_CONFIG) --image=kindest/node:v$(E2ETEST_K8S_VERSION) --wait 1m
	$(MAKE) -C ../ docker-build
	kind load docker-image cat-gate:dev --name=$(KIND_CLUSTER_NAME)
	$(KUBECTL) apply -f https://github.com/jetstack/cert-manager/releases/download/v$(CERT_MANAGER_VERSION)/cert-manager.yaml
	$(KUBECTL) -n cert-manager wait --for=condition=available --timeout=180s --all deployments
	$(KUBECTL) apply -k ../config/default
	$(KUBECTL) -n cat-gate-system wait --for=condition=available --timeout=180s --all deployments

.PHONY: test
test:
	env RUN_E2E=1 \
		go test -v -race . -ginkgo.v -ginkgo.fail-fast

.PHONY: logs
logs:
	rm -rf logs.tar.gz logs
	kind export logs --name=$(KIND_CLUSTER_NAME) ./logs
	tar czf logs.tar.gz logs
	rm -rf logs

.PHONY: stop
stop:
	kind delete cluster --name=$(KIND_CLUSTER_NAME)
	-docker image rm cat-gate:dev
	-docker image prune -f
