IMG ?= ghcr.io/eclipse-xfsc/resource-operator:dev
KIND_CLUSTER ?= kind

.PHONY: test build docker-build docker-push kind-load deploy undeploy manifests run

test:
	go test ./...

build:
	go build -o bin/manager ./cmd/manager

run:
	go run ./cmd/manager

manifests:
	@echo "CRDs are hand-written in config/crd for this MVP. Use controller-gen in a real repo."

docker-build:
	docker build -t $(IMG) -f deployment/docker/Dockerfile .

docker-push:
	docker push $(IMG)
