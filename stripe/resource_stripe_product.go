package stripe

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/client"

	"log"
)

func expandAttributes(d *schema.ResourceData) []*string {
	return expandStringList(d, "attributes")
}

func resourceStripeProduct() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceStripeProductCreate,
		ReadContext:   resourceStripeProductRead,
		UpdateContext: resourceStripeProductUpdate,
		DeleteContext: resourceStripeProductDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"product_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"active": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"attributes": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"metadata": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"statement_descriptor": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"unit_label": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceStripeProductCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)
	productName := d.Get("name").(string)
	productType := d.Get("type").(string)
	productStatementDescriptor := d.Get("statement_descriptor").(string)
	productUnitLabel := d.Get("unit_label").(string)

	var stripeProductType stripe.ProductType

	switch productType {
	case "good":
		stripeProductType = stripe.ProductTypeGood
	case "service":
		stripeProductType = stripe.ProductTypeService
	default:
		return diag.Errorf("unknown type: %s", productType)
	}

	params := &stripe.ProductParams{
		Name: stripe.String(productName),
		Type: stripe.String(string(stripeProductType)),
	}
	params.Context = ctx

	if productID, ok := d.GetOk("product_id"); ok {
		params.ID = stripe.String(productID.(string))
	}

	if active, ok := d.GetOk("active"); ok {
		params.Active = stripe.Bool(active.(bool))
	}

	params.Attributes = expandAttributes(d)

	params.Metadata = expandMetadata(d)

	if productStatementDescriptor != "" {
		params.StatementDescriptor = stripe.String(productStatementDescriptor)
	}

	if productUnitLabel != "" {
		params.UnitLabel = stripe.String(productUnitLabel)
	}

	product, err := client.Products.New(params)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Created Stripe product: %s", productName)
	d.SetId(product.ID)

	return resourceStripeProductRead(ctx, d, m)
}

func resourceStripeProductRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)

	params := &stripe.ProductParams{}
	params.Context = ctx

	product, err := client.Products.Get(d.Id(), params)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("product_id", product.ID)
	d.Set("name", product.Name)
	d.Set("type", product.Type)
	d.Set("active", product.Active)
	d.Set("attributes", product.Attributes)
	d.Set("metadata", product.Metadata)
	d.Set("statement_descriptor", product.StatementDescriptor)
	d.Set("unit_label", product.UnitLabel)

	return nil
}

func resourceStripeProductUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)

	params := &stripe.ProductParams{}
	params.Context = ctx

	if d.HasChange("name") {
		params.Name = stripe.String(d.Get("name").(string))
	}

	if d.HasChange("type") {
		params.Type = stripe.String(d.Get("type").(string))
	}

	if d.HasChange("active") {
		params.Active = stripe.Bool(d.Get("active").(bool))
	}

	if d.HasChange("attributes") {
		params.Attributes = expandAttributes(d)
	}

	if d.HasChange("metadata") {
		params.Metadata = expandMetadata(d)
	}

	if d.HasChange("statement_descriptor") {
		params.StatementDescriptor = stripe.String(d.Get("statement_descriptor").(string))
	}

	if d.HasChange("unit_label") {
		params.UnitLabel = stripe.String(d.Get("unit_label").(string))
	}

	_, err := client.Products.Update(d.Id(), params)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceStripeProductRead(ctx, d, m)
}

func resourceStripeProductDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)

	params := &stripe.ProductParams{}
	params.Context = ctx

	if _, err := client.Products.Del(d.Id(), params); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
