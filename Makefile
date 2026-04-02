MODULE := github.com/opcua-kube-gateway/opcua-kube-gateway
IMG ?= ghcr.io/opcua-kube-gateway/opcua-kube-gateway:latest
CONTROLLER_GEN ?= $(shell which controller-gen 2>/dev/null)
GOLANGCI_LINT ?= $(shell which golangci-lint 2>/dev/null)

.PHONY: build test lint generate docker-build docker-push fmt vet tidy

build:
	go build -o bin/operator ./cmd/operator

test:
	go test -race -coverprofile=coverage.out ./...
	@echo "Coverage:"
	@go tool cover -func=coverage.out | tail -1

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./api/..."
	$(CONTROLLER_GEN) crd paths="./api/..." output:crd:artifacts:config=charts/opcua-kube-gateway/crds

docker-build:
	docker build -t $(IMG) .

docker-push:
	docker push $(IMG)

controller-gen:
ifeq (, $(CONTROLLER_GEN))
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
	$(eval CONTROLLER_GEN := $(shell go env GOPATH)/bin/controller-gen)
endif

golangci-lint:
ifeq (, $(GOLANGCI_LINT))
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(eval GOLANGCI_LINT := $(shell go env GOPATH)/bin/golangci-lint)
endif

help:
	@echo "Targets:"
	@echo "  build        - Build the operator binary"
	@echo "  test         - Run tests with race detection and coverage"
	@echo "  lint         - Run golangci-lint"
	@echo "  fmt          - Format Go code"
	@echo "  vet          - Run go vet"
	@echo "  tidy         - Run go mod tidy"
	@echo "  generate     - Generate CRD manifests and deepcopy"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-push  - Push Docker image"
