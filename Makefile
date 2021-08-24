# Image URL to use all building/pushing image targets
IMG ?= {CONTAINER_IMAGE}
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

ifeq ($(IMG),{CONTAINER_IMAGE})
TAG := $(shell ./scripts/get-latest.sh)
IMG := cnwan/cnwan-operator:${TAG}
endif

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Run tests
test: fmt vet
	go test ./... -coverprofile cover.out

# Build manager binary
manager: fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: fmt vet
	go run ./main.go

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}