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

# Release - create and push a new version tag
# Usage: make release VERSION=0.2.0
.PHONY: release
release:
ifndef VERSION
	$(error VERSION is required. Usage: make release VERSION=0.2.0)
endif
	@echo "Creating release v$(VERSION)..."
	git tag -a v$(VERSION) -m "Release v$(VERSION)"
	git push origin v$(VERSION)
	@echo "Release v$(VERSION) created and pushed. Check GitHub Actions for build status."

# List existing tags
.PHONY: tags
tags:
	git tag -l "v*" --sort=-v:refname
