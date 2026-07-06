IMG ?= ghcr.io/eclipse-xfsc/resource-operator:dev

.PHONY: test build docker-build manifests run

test:
	go test ./...

build:
	go build -o bin/manager ./cmd/manager

run:
	go run ./cmd/manager

manifests:
	@echo "CRDs are hand-written in config/crd for this MVP. Use controller-gen in a real repo."

docker-build:
	docker build -t $(IMG) .
