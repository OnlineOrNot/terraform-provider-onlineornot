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

# Generate schemas from OpenAPI spec (Step 1 + 2)
.PHONY: generate-schemas
generate-schemas:
	@echo "Fetching OpenAPI spec..."
	curl -sSL -o openapi.json https://raw.githubusercontent.com/OnlineOrNot/api-schemas/main/openapi.json
	@echo "Generating provider code specification from OpenAPI..."
	go run github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi generate \
		--config generator_config.yml \
		--output provider_code_spec.json \
		openapi.json
	@echo "Generating framework code from specification..."
	go run github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework generate resources \
		--input provider_code_spec.json \
		--output internal/provider

# Generate documentation (Step 3 + 4)
.PHONY: docs
docs:
	terraform fmt -recursive ./examples/
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate
	go run ./tools/enrich-docs

# Generate everything (schemas + docs)
.PHONY: generate
generate: generate-schemas docs
	@echo "Generation complete!"

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
