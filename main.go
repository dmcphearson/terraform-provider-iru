package main

import (
	"context"
	"flag"
	"log"

	"github.com/dmcphearson/terraform-provider-iru/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// version is set at build time via -ldflags. Defaults to "dev".
var version = "dev"

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/dmcphearson/iru",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err.Error())
	}
}
