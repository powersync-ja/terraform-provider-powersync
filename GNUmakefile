default: build

build:
	go build ./...

install: build
	go install .

# Installs to the local plugin cache so you can use dev_overrides in ~/.terraformrc
# to test without publishing to the registry.
install-dev:
	go build -o ~/.terraform.d/plugins/registry.terraform.io/powersync/powersync/0.1.0/$$(go env GOOS)_$$(go env GOARCH)/terraform-provider-powersync .

generate:
	go generate ./...

lint:
	golangci-lint run ./...

test:
	go test ./... -v -timeout 120s

# Acceptance tests hit the real API — requires PS_ADMIN_TOKEN to be set.
testacc:
	TF_ACC=1 go test ./... -v -run TestAcc -timeout 600s

.PHONY: build install install-dev generate lint test testacc
