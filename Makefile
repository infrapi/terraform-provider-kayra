# terraform-provider-kayra: make help lists the targets.

BINARY := terraform-provider-kayra
KAYRA ?= ../kayra

.DEFAULT_GOAL := help
.PHONY: help all contracts generate fmt lint security test testacc build install snapshot clean

help: ## list the targets
	@awk 'BEGIN {FS = ":.*## "} /^[a-z-]+:.*## / {printf "  %-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: generate fmt lint security test build ## the gate before handing work off

contracts: ## copy Kayra's contracts from $(KAYRA)/api, then generate
	cp $(KAYRA)/api/system-v1.yaml $(KAYRA)/api/catalog-v1.yaml $(KAYRA)/api/rbac-v1.yaml $(KAYRA)/api/events-v1.yaml internal/contracts/
	$(MAKE) generate

generate: ## the schemas from the contracts, then docs/ from the schemas and examples/
	go generate ./...

fmt: ## format the code
	gofmt -w .

lint: ## vet and lint (golangci-lint)
	go vet ./...
	golangci-lint run

security: ## gosec
	gosec -quiet ./...

test: ## the unit tests
	go test ./...

testacc: ## the acceptance tests, against KAYRA_URL with KAYRA_TOKEN, tofu on the PATH
	TF_ACC=1 TF_ACC_PROVIDER_NAMESPACE=infrapi TF_ACC_TERRAFORM_PATH=$$(command -v tofu || command -v terraform) \
		go test -v -count=1 -timeout 10m ./internal/provider/ -run TestAcc

build: ## bin/terraform-provider-kayra
	CGO_ENABLED=0 go build -trimpath -o bin/$(BINARY) .

install: ## the provider in ~/go/bin, for dev_overrides
	go install .

snapshot: ## every release artifact, locally, unsigned and unpublished (goreleaser)
	goreleaser release --snapshot --clean --skip=sign

clean: ## remove bin/ and dist/
	rm -rf bin dist
