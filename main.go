package main

import (
	"github.com/franckverrot/terraform-provider-stripe/stripe"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: stripe.Provider,
	})
}
