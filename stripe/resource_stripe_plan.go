package stripe

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	stripe "github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/client"
)

func resourceStripePlan() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceStripePlanCreate,
		ReadContext:   resourceStripePlanRead,
		UpdateContext: resourceStripePlanUpdate,
		DeleteContext: resourceStripePlanDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"plan_id": {
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
			"amount": {
				Type:          schema.TypeInt,
				Optional:      true,
				ForceNew:      true,
				Computed:      true,
				ConflictsWith: []string{"amount_decimal"},
			},
			"amount_decimal": {
				Type:          schema.TypeFloat,
				Optional:      true,
				ForceNew:      true,
				Computed:      true,
				ConflictsWith: []string{"amount"},
			},
			"currency": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"interval": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"product": {
				Type:     schema.TypeString,
				Required: true,
			},
			"aggregate_usage": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"billing_scheme": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "per_unit",
			},
			"interval_count": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Default:  1,
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
			"transform_usage": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"divide_by": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},
						"round": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.StringInSlice([]string{"down", "up"}, false),
						},
					},
				},
				MaxItems: 1,
				Optional: true,
				ForceNew: true,
			},
			"trial_period_days": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"usage_type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "licensed",
			},
		},
	}
}

func resourceStripePlanCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)
	planNickname := d.Get("nickname").(string)
	planInterval := d.Get("interval").(string)
	planCurrency := d.Get("currency").(string)
	planProductID := d.Get("product").(string)

	// TODO: check interval
	// TODO: check currency

	params := &stripe.PlanParams{
		Interval:  stripe.String(planInterval),
		ProductID: stripe.String(planProductID),
		Currency:  stripe.String(planCurrency),
	}
	params.Context = ctx

	amount := d.Get("amount").(int)
	amountDecimal := d.Get("amount_decimal").(float64)

	if amountDecimal > 0 {
		params.AmountDecimal = stripe.Float64(float64(amountDecimal))
	} else {
		params.Amount = stripe.Int64(int64(amount))
	}

	if id, ok := d.GetOk("plan_id"); ok {
		params.ID = stripe.String(id.(string))
	}

	if active, ok := d.GetOk("active"); ok {
		params.Active = stripe.Bool(active.(bool))
	}

	if aggregateUsage, ok := d.GetOk("aggregate_usage"); ok {
		params.AggregateUsage = stripe.String(aggregateUsage.(string))
	}

	if billingScheme, ok := d.GetOk("billing_scheme"); ok {
		params.BillingScheme = stripe.String(billingScheme.(string))
		if billingScheme == "tiered" {
			params.Amount = nil
			params.AmountDecimal = nil
		}
	}

	if intervalCount, ok := d.GetOk("interval_count"); ok {
		params.IntervalCount = stripe.Int64(int64(intervalCount.(int)))
	}

	params.Metadata = expandMetadata(d)

	if _, ok := d.GetOk("nickname"); ok {
		params.Nickname = stripe.String(planNickname)
	}

	if tiersMode, ok := d.GetOk("tiers_mode"); ok {
		params.TiersMode = stripe.String(tiersMode.(string))
	}

	tiers, diags := expandPlanTiers(d)
	if diags.HasError() {
		return diags
	}
	// TODO: Propagate non-error diagnostics
	params.Tiers = tiers

	if transformUsage, ok := d.GetOk("transform_usage"); ok {
		params.TransformUsage = expandPlanTransformUsage(transformUsage.([]interface{}))
	}

	if trialPeriodDays, ok := d.GetOk("trial_period_days"); ok {
		params.TrialPeriodDays = stripe.Int64(int64(trialPeriodDays.(int)))
	}

	if usageType, ok := d.GetOk("usage_type"); ok {
		params.UsageType = stripe.String(usageType.(string))
	}

	plan, err := client.Plans.New(params)
	if err != nil {
		return diag.FromErr(err)
	}

	if plan.Nickname != "" {
		log.Printf("[INFO] Create plan: %s (%s)", plan.Nickname, plan.ID)
	} else {
		log.Printf("[INFO] Create anonymous plan: %s", plan.ID)
	}

	d.SetId(plan.ID)

	return nil
}

func resourceStripePlanRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)

	params := &stripe.PlanParams{}
	params.Context = ctx
	params.AddExpand("tiers")

	plan, err := client.Plans.Get(d.Id(), params)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("plan_id", plan.ID)
	d.Set("active", plan.Active)
	d.Set("aggregate_usage", plan.AggregateUsage)
	d.Set("amount", plan.Amount)
	d.Set("amount_decimal", plan.AmountDecimal)
	d.Set("billing_scheme", plan.BillingScheme)
	d.Set("currency", plan.Currency)
	d.Set("interval", plan.Interval)
	d.Set("interval_count", plan.IntervalCount)
	d.Set("metadata", plan.Metadata)
	d.Set("nickname", plan.Nickname)
	d.Set("product", plan.Product)
	d.Set("tiers_mode", plan.TiersMode)
	d.Set("tier", flattenPlanTiers(plan.Tiers))
	d.Set("transform_usage", flattenPlanTransformUsage(plan.TransformUsage))
	d.Set("trial_period_days", plan.TrialPeriodDays)
	d.Set("usage_type", plan.UsageType)

	return nil
}

func flattenPlanTiers(in []*stripe.PlanTier) []map[string]interface{} {
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

func expandPlanTier(d *schema.ResourceData, idx int) (*stripe.PlanTierParams, diag.Diagnostics) {
	params := &stripe.PlanTierParams{}

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

	if flatAmount, ok := d.GetOk(fmt.Sprintf("tier.%d.flat_amount", idx)); ok {
		params.FlatAmount = stripe.Int64(int64(flatAmount.(int)))
	} else if flatAmountDecimal, ok := d.GetOk(fmt.Sprintf("tier.%d.flat_amount_decimal", idx)); ok {
		params.FlatAmountDecimal = stripe.Float64(flatAmountDecimal.(float64))
	}

	if unitAmount, ok := d.GetOk(fmt.Sprintf("tier.%d.unit_amount", idx)); ok {
		params.UnitAmount = stripe.Int64(int64(unitAmount.(int)))
	} else if unitAmountDecimal, ok := d.GetOk(fmt.Sprintf("tier.%d.unit_amount_decimal", idx)); ok {
		params.UnitAmountDecimal = stripe.Float64(unitAmountDecimal.(float64))
	}

	return params, nil
}

func expandPlanTiers(d *schema.ResourceData) (out []*stripe.PlanTierParams, diags diag.Diagnostics) {
	v, ok := d.GetOk("tier")
	if !ok {
		return
	}

	in := v.([]interface{})
	out = make([]*stripe.PlanTierParams, len(in))
	for i := range in {
		tier, tdgs := expandPlanTier(d, i)
		out[i] = tier
		diags = append(diags, tdgs...)
	}

	return
}

func flattenPlanTransformUsage(in *stripe.PlanTransformUsage) []map[string]interface{} {
	n := 1
	if in == nil {
		n = 0
	}
	out := make([]map[string]interface{}, n)

	for i := range out {
		out[i] = map[string]interface{}{
			"divide_by": in.DivideBy,
			"round":     in.Round,
		}
	}
	return out
}

func expandPlanTransformUsage(in []interface{}) *stripe.PlanTransformUsageParams {
	if len(in) == 0 {
		return nil
	}

	transformUsage := in[0].(map[string]interface{})
	out := &stripe.PlanTransformUsageParams{
		DivideBy: stripe.Int64(int64(transformUsage["divide_by"].(int))),
		Round:    stripe.String(transformUsage["round"].(string)),
	}
	return out
}

func resourceStripePlanUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)

	params := &stripe.PlanParams{}
	params.Context = ctx

	if d.HasChange("plan_id") {
		params.ID = stripe.String(d.Get("plan_id").(string))
	}

	if d.HasChange("active") {
		params.Active = stripe.Bool(bool(d.Get("active").(bool)))
	}

	if d.HasChange("metadata") {
		params.Metadata = expandMetadata(d)
	}

	if d.HasChange("nickname") {
		params.Nickname = stripe.String(d.Get("nickname").(string))
	}

	if d.HasChange("trial_period_days") {
		params.TrialPeriodDays = stripe.Int64(int64(d.Get("trial_period_days").(int)))
	}

	if _, err := client.Plans.Update(d.Id(), params); err != nil {
		return diag.FromErr(err)
	}

	return resourceStripePlanRead(ctx, d, m)
}

func resourceStripePlanDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)

	params := &stripe.PlanParams{}
	params.Context = ctx

	if _, err := client.Plans.Del(d.Id(), params); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
