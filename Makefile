BINARY_NAME=kubernetes-operator
MODULE=github.com/eclipse-xfsc/kubernetes-operator
VERSION?=0.1.0

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: generate
generate: generate-goa generate-k8s

.PHONY: generate-goa
generate-goa:
	go generate ./cmd/apigen

.PHONY: generate-k8s
generate-k8s:
	controller-gen object paths="./api/..."

.PHONY: build
build:
	go build -ldflags "-X $(MODULE)/internal/runtimeinfo.OperatorVersion=$(VERSION)" -o bin/$(BINARY_NAME) ./cmd/operator

.PHONY: run
run:
	go run ./cmd/operator --api-bind-address=:8088 --metrics-bind-address=:8080

.PHONY: manifests
manifests:
	controller-gen \
		rbac:roleName=manager-role \
		crd \
		paths="./..." \
		output:crd:artifacts:config=config/crd

.PHONY: install-tools
install-tools:
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
	go install goa.design/goa/v3/cmd/goa@latest

.PHONY: all
all: tidy generate manifests build