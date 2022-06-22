package stripe

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	stripe "github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/client"
)

func resourceStripePrice() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceStripePriceCreate,
		ReadContext:   resourceStripePriceRead,
		UpdateContext: resourceStripePriceUpdate,
		DeleteContext: resourceStripePriceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"price_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"active": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"currency": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"metadata": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"nickname": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"product": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"recurring": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"unit_amount": {
				Type:     schema.TypeInt,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},
			"unit_amount_decimal": {
				Type:     schema.TypeFloat,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},
			"billing_scheme": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"created": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"livemode": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"tier": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"up_to": {
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"up_to_inf": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
						"flat_amount": {
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
							Computed: true,
						},
						"flat_amount_decimal": {
							Type:     schema.TypeFloat,
							Optional: true,
							ForceNew: true,
							Computed: true,
						},
						"unit_amount": {
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
							Computed: true,
						},
						"unit_amount_decimal": {
							Type:     schema.TypeFloat,
							Optional: true,
							ForceNew: true,
							Computed: true,
						},
					},
				},
				Optional: true,
				ForceNew: true,
			},
			"tiers_mode": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"tax_behavior": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "unspecified",
				ValidateFunc: validation.StringInSlice([]string{"unspecified", "inclusive", "exclusive"}, false),
			},
		},
		CustomizeDiff: customdiff.All(
			customdiff.ForceNewIfChange("tax_behavior", func(ctx context.Context, old, new, meta interface{}) bool {
				return old != "unspecified"
			}),
		),
	}
}

func expandPriceRecurring(recurring map[string]interface{}) (*stripe.PriceRecurringParams, diag.Diagnostics) {
	params := &stripe.PriceRecurringParams{}
	parsed := expandStringMap(recurring)

	if aggregateUsage, ok := parsed["aggregate_usage"]; ok {
		params.AggregateUsage = stripe.String(aggregateUsage)
	}

	if interval, ok := parsed["interval"]; ok {
		params.Interval = stripe.String(interval)
	}

	if intervalCount, ok := parsed["interval_count"]; ok {
		intervalCountInt, err := strconv.ParseInt(intervalCount, 10, 64)
		if err != nil {
			return nil, diag.Errorf("interval_count must be a string, representing an int (e.g. \"52\")")
		}
		params.IntervalCount = stripe.Int64(intervalCountInt)
	}

	if usageType, ok := parsed["usage_type"]; ok {
		params.UsageType = stripe.String(usageType)
	}

	return params, nil
}

func resourceStripePriceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)
	nickname := d.Get("nickname").(string)
	currency := d.Get("currency").(string)

	params := &stripe.PriceParams{
		Currency: stripe.String(currency),
	}
	params.Context = ctx

	if active, ok := d.GetOk("active"); ok {
		params.Active = stripe.Bool(active.(bool))
	}

	params.Metadata = expandMetadata(d)

	if _, ok := d.GetOk("nickname"); ok {
		params.Nickname = stripe.String(nickname)
	}

	if tiersMode, ok := d.GetOk("tiers_mode"); ok {
		params.TiersMode = stripe.String(tiersMode.(string))
	}

	priceTiers, diags := expandPriceTiers(d)
	if diags.HasError() {
		return diags
	}
	// TODO: Propagate non-error diagnostics
	params.Tiers = priceTiers

	if product, ok := d.GetOk("product"); ok {
		params.Product = stripe.String(product.(string))
	}

	if recurring, ok := d.GetOk("recurring"); ok {
		recurringParams, diags := expandPriceRecurring(recurring.(map[string]interface{}))
		if diags.HasError() {
			return diags
		}
		// TODO: Propagate non-error diagnostics
		params.Recurring = recurringParams
	}

	// TODO: The `GetOkExists` method is deprecated, but there is no other way to
	// support setting prices to 0 when they are typed as integers and floats. Unit
	// amounts should probably be typed as strings and tested for convertability to
	// the desired numeric types, but that will likely break existing Terraform state.
	if unitAmount, ok := d.GetOkExists("unit_amount"); ok {
		params.UnitAmount = stripe.Int64(int64(unitAmount.(int)))
	}

	if unitAmountDecimal, ok := d.GetOkExists("unit_amount_decimal"); ok {
		params.UnitAmountDecimal = stripe.Float64(unitAmountDecimal.(float64))
	}

	if billingScheme, ok := d.GetOk("billing_scheme"); ok {
		params.BillingScheme = stripe.String(billingScheme.(string))
	}

	if taxBehavior, ok := d.GetOk("tax_behavior"); ok {
		params.TaxBehavior = stripe.String(taxBehavior.(string))
	}

	price, err := client.Prices.New(params)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Created Stripe price: %s", nickname)
	d.SetId(price.ID)

	return resourceStripePriceRead(ctx, d, m)
}

func resourceStripePriceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)

	params := &stripe.PriceParams{}
	params.Context = ctx
	params.AddExpand("tiers")

	price, err := client.Prices.Get(d.Id(), params)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("price_id", price.ID)
	d.Set("active", price.Active)
	d.Set("created", price.Created)
	d.Set("currency", price.Currency)
	d.Set("livemode", price.Livemode)
	d.Set("metadata", price.Metadata)
	d.Set("nickname", price.Nickname)
	if price.Product != nil {
		d.Set("product", price.Product.ID)
	}
	d.Set("recurring", price.Active)
	d.Set("unit_amount", price.UnitAmount)
	d.Set("unit_amount_decimal", price.UnitAmountDecimal)
	d.Set("tiers_mode", price.TiersMode)
	d.Set("tier", flattenPriceTiers(price.Tiers))
	d.Set("billing_scheme", price.BillingScheme)
	d.Set("tax_behavior", price.TaxBehavior)

	return nil
}

func flattenPriceTiers(in []*stripe.PriceTier) []map[string]interface{} {
	out := make([]map[string]interface{}, len(in))
	for i, tier := range in {
		out[i] = map[string]interface{}{
			"up_to":               tier.UpTo,
			"up_to_inf":           tier.UpTo == 0,
			"flat_amount":         tier.FlatAmount,
			"flat_amount_decimal": tier.FlatAmountDecimal,
			"unit_amount":         tier.UnitAmount,
			"unit_amount_decimal": tier.UnitAmountDecimal,
		}
	}
	return out
}

func expandPriceTier(d *schema.ResourceData, idx int) (*stripe.PriceTierParams, diag.Diagnostics) {
	params := &stripe.PriceTierParams{}

	upTo, upToOK := d.GetOk(fmt.Sprintf("tier.%d.up_to", idx))
	if upToOK {
		params.UpTo = stripe.Int64(int64(upTo.(int)))
	}

	upToInf, upToInfOK := d.GetOkExists(fmt.Sprintf("tier.%d.up_to_inf", idx))
	if upToInfOK {
		params.UpToInf = stripe.Bool(upToInf.(bool))
	}

	if upToOK && upToInfOK {
		return nil, diag.Errorf("up_to: conflicts with up_to_inf")
	}

	if flatAmount, ok := d.GetOkExists(fmt.Sprintf("tier.%d.flat_amount", idx)); ok {
		params.FlatAmount = stripe.Int64(int64(flatAmount.(int)))
	} else if flatAmountDecimal, ok := d.GetOkExists(fmt.Sprintf("tier.%d.flat_amount_decimal", idx)); ok {
		params.FlatAmountDecimal = stripe.Float64(flatAmountDecimal.(float64))
	}

	if unitAmount, ok := d.GetOkExists(fmt.Sprintf("tier.%d.unit_amount", idx)); ok {
		params.UnitAmount = stripe.Int64(int64(unitAmount.(int)))
	} else if unitAmountDecimal, ok := d.GetOkExists(fmt.Sprintf("tier.%d.unit_amount_decimal", idx)); ok {
		params.UnitAmountDecimal = stripe.Float64(unitAmountDecimal.(float64))
	}

	return params, nil
}

func expandPriceTiers(d *schema.ResourceData) (out []*stripe.PriceTierParams, diags diag.Diagnostics) {
	v, ok := d.GetOk("tier")
	if !ok {
		return
	}

	in := v.([]interface{})
	out = make([]*stripe.PriceTierParams, len(in))
	for i := range in {
		tier, tdgs := expandPriceTier(d, i)
		out[i] = tier
		diags = append(diags, tdgs...)
	}

	return
}

func resourceStripePriceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)

	params := &stripe.PriceParams{}
	params.Context = ctx

	if d.HasChange("active") {
		params.Active = stripe.Bool(d.Get("active").(bool))
	}

	if d.HasChange("metadata") {
		params.Metadata = expandMetadata(d)
	}

	if d.HasChange("nickname") {
		params.Nickname = stripe.String(d.Get("nickname").(string))
	}

	if d.HasChange("tax_behavior") {
		params.TaxBehavior = stripe.String(d.Get("tax_behavior").(string))
	}

	_, err := client.Prices.Update(d.Id(), params)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceStripePriceRead(ctx, d, m)
}

func resourceStripePriceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)

	params := &stripe.PriceParams{
		Active: stripe.Bool(false),
	}
	params.Context = ctx

	if _, err := client.Prices.Update(d.Id(), params); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
