package stripe

import (
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_token": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("STRIPE_API_TOKEN", nil),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"stripe_coupon":           resourceStripeCoupon(),
			"stripe_plan":             resourceStripePlan(),
			"stripe_price":            resourceStripePrice(),
			"stripe_product":          resourceStripeProduct(),
			"stripe_tax_rate":         resourceStripeTaxRate(),
			"stripe_webhook_endpoint": resourceStripeWebhookEndpoint(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		APIToken: d.Get("api_token").(string),
	}

	log.Println("[INFO] Initializing Stripe client")
	return config.Client()
}
