default: build

# Build the provider
.PHONY: build
build:
	go build -v ./...

# Run unit tests
.PHONY: test
test:
	go test -v ./...

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

# Generate documentation
.PHONY: docs
docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate

# Install the provider locally for development
.PHONY: install
install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/onlineornot/onlineornot/0.0.1/darwin_arm64
	cp terraform-provider-onlineornot ~/.terraform.d/plugins/registry.terraform.io/onlineornot/onlineornot/0.0.1/darwin_arm64/terraform-provider-onlineornot_v0.0.1

# Clean up
.PHONY: clean
clean:
	rm -f terraform-provider-onlineornot
