package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/provider"
)

// Run "go generate" to:
// 1. Generate provider code specification from OpenAPI
// 2. Generate framework code from specification
// 3. Format example terraform files
// 4. Generate documentation

// Step 1: Fetch OpenAPI spec and convert to Provider Code Specification
//go:generate curl -sSL -o openapi.json https://raw.githubusercontent.com/OnlineOrNot/api-schemas/main/openapi.json
//go:generate go run github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi generate --config generator_config.yml --output provider_code_spec.json openapi.json

// Step 2: Generate framework code (resources) from the specification
//go:generate go run github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework generate resources --input provider_code_spec.json --output internal/provider

// Step 3: Format example terraform files
//go:generate terraform fmt -recursive ./examples/

// Step 4: Generate documentation
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate -provider-name terraform-provider-onlineornot

var (
	// these will be set by the goreleaser configuration
	// to appropriate values for the compiled binary.
	version string = "dev"

	// goreleaser can pass other information to the main package, such as the specific commit
	// https://goreleaser.com/cookbooks/using-main.version/
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/onlineornot/onlineornot",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
