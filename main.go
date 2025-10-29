package main

import (
	"context"
	"flag"
	"log"

	"github.com/OSapozhnikov/terraform-provider-atlassian-compass/internal/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	var debugMode bool

	flag.BoolVar(&debugMode, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := &plugin.ServeOpts{
		ProviderFunc: provider.New,
	}

	if debugMode {
		err := plugin.Debug(context.Background(), "registry.terraform.io/temabit/compass", opts)
		if err != nil {
			log.Fatal(err.Error())
		}
		return
	}

	plugin.Serve(opts)
}
