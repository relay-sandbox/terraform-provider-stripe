package stripe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	stripe "github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/client"

	"log"
)

func resourceStripeWebhookEndpoint() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceStripeWebhookEndpointCreate,
		ReadContext:   resourceStripeWebhookEndpointRead,
		UpdateContext: resourceStripeWebhookEndpointUpdate,
		DeleteContext: resourceStripeWebhookEndpointDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"url": {
				Type:     schema.TypeString,
				Required: true,
			},
			"enabled_events": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
			},
			"connect": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"secret": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceStripeWebhookEndpointCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)
	url := d.Get("url").(string)

	params := &stripe.WebhookEndpointParams{
		URL:           stripe.String(url),
		EnabledEvents: expandStringList(d, "enabled_events"),
	}
	params.Context = ctx

	if connect, ok := d.GetOk("connect"); ok {
		params.Connect = stripe.Bool(connect.(bool))
	}

	webhookEndpoint, err := client.WebhookEndpoints.New(params)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Create webbook endpoint: %s", url)
	d.SetId(webhookEndpoint.ID)
	d.Set("secret", webhookEndpoint.Secret)

	return nil
}

func resourceStripeWebhookEndpointRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)

	params := &stripe.WebhookEndpointParams{}
	params.Context = ctx

	webhookEndpoint, err := client.WebhookEndpoints.Get(d.Id(), params)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("url", webhookEndpoint.URL)
	d.Set("enabled_events", webhookEndpoint.EnabledEvents)
	d.Set("connect", webhookEndpoint.Application != "")

	return nil
}

func resourceStripeWebhookEndpointUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)

	params := &stripe.WebhookEndpointParams{}
	params.Context = ctx

	if d.HasChange("url") {
		params.URL = stripe.String(d.Get("url").(string))
	}

	if d.HasChange("enabled_events") {
		params.EnabledEvents = expandStringList(d, "enabled_events")
	}

	if d.HasChange("connect") {
		params.Connect = stripe.Bool(d.Get("connect").(bool))
	}

	if _, err := client.WebhookEndpoints.Update(d.Id(), params); err != nil {
		return diag.FromErr(err)
	}

	return resourceStripeWebhookEndpointRead(ctx, d, m)
}

func resourceStripeWebhookEndpointDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)

	params := &stripe.WebhookEndpointParams{}
	params.Context = ctx

	if _, err := client.WebhookEndpoints.Del(d.Id(), params); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
