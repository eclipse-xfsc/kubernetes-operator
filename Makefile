IMG ?= node-654e3bca7fbeeed18f81d7c7.ps-xaas.io/common-services/kubernetes-operator:dev

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
	docker buildx build --platform linux/amd64 -t $(IMG) -f deployment/docker/Dockerfile .

docker-push:
	docker push $(IMG)
