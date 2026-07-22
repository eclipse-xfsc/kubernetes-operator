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
	docker build -t $(IMG) .

docker-push:
	docker push $(IMG)

kind-load:
	kind load docker-image $(IMG) --name $(KIND_CLUSTER)

deploy:
	kubectl apply -k config/default

undeploy:
	kubectl delete -k config/default --ignore-not-found=true

test-setup: 
	brew install libpq
    export PATH="/opt/homebrew/opt/libpq/bin:$PATH"
	brew install minio/stable/mc
