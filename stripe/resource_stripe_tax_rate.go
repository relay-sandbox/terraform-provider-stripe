package stripe

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	stripe "github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/client"
)

func resourceStripeTaxRate() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceStripeTaxRateCreate,
		ReadContext:   resourceStripeTaxRateRead,
		UpdateContext: resourceStripeTaxRateUpdate,
		DeleteContext: resourceStripeTaxRateDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"active": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"created": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"display_name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"inclusive": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"jurisdiction": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"livemode": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"metadata": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"percentage": {
				Type:     schema.TypeFloat,
				Required: true,
			},
		},
	}
}

func resourceStripeTaxRateCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)
	taxRateDisplayName := d.Get("display_name").(string)
	taxRateInclusive := d.Get("inclusive").(bool)
	taxRatePercentage := d.Get("percentage").(float64)

	params := &stripe.TaxRateParams{
		DisplayName: stripe.String(taxRateDisplayName),
		Inclusive:   stripe.Bool(taxRateInclusive),
		Percentage:  stripe.Float64(taxRatePercentage),
	}
	params.Context = ctx

	if active, ok := d.GetOk("active"); ok {
		params.Active = stripe.Bool(active.(bool))
	}

	if description, ok := d.GetOk("description"); ok {
		params.Description = stripe.String(description.(string))
	}

	if jurisdiction, ok := d.GetOk("jurisdiction"); ok {
		params.Jurisdiction = stripe.String(jurisdiction.(string))
	}

	params.Metadata = expandMetadata(d)

	tax, err := client.TaxRates.New(params)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Create Tax Rate: %s (%f)", tax.ID, tax.Percentage)
	d.SetId(tax.ID)
	d.Set("display_name", tax.DisplayName)
	d.Set("inclusive", tax.Inclusive)
	d.Set("percentage", tax.Percentage)
	d.Set("created", tax.Created)
	d.Set("livemode", tax.Livemode)

	return nil
}

func resourceStripeTaxRateRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)

	params := &stripe.TaxRateParams{}
	params.Context = ctx

	tax, err := client.TaxRates.Get(d.Id(), params)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("active", tax.Active)
	d.Set("created", tax.Created)
	d.Set("description", tax.Description)
	d.Set("display_name", tax.DisplayName)
	d.Set("inclusive", tax.Inclusive)
	d.Set("jurisdiction", tax.Jurisdiction)
	d.Set("livemode", tax.Livemode)
	d.Set("metadata", tax.Metadata)

	return nil
}

func resourceStripeTaxRateUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)

	params := &stripe.TaxRateParams{}
	params.Context = ctx

	if d.HasChange("active") {
		params.Active = stripe.Bool(d.Get("active").(bool))
	}

	if d.HasChange("description") {
		params.Description = stripe.String(d.Get("description").(string))
	}

	if d.HasChange("diplay_name") {
		params.DisplayName = stripe.String(d.Get("display_name").(string))
	}

	if d.HasChange("jurisdiction") {
		params.Jurisdiction = stripe.String(d.Get("jurisdiction").(string))
	}

	if d.HasChange("metadata") {
		params.Metadata = expandMetadata(d)
	}

	if _, err := client.TaxRates.Update(d.Id(), params); err != nil {
		return diag.FromErr(err)
	}

	return resourceStripeTaxRateRead(ctx, d, m)
}

func resourceStripeTaxRateDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return diag.Errorf("[WARNING] Stripe doesn't allow deleting tax rates via the API.  Your state file contains at least one (\"%v\") that needs deletion.  Please remove it manually.", d.Get("display_name"))
}
