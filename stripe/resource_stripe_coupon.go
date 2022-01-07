package stripe

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	stripe "github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/client"
)

func resourceStripeCoupon() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceStripeCouponCreate,
		ReadContext:   resourceStripeCouponRead,
		UpdateContext: resourceStripeCouponUpdate,
		DeleteContext: resourceStripeCouponDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"code": {
				Type:     schema.TypeString,
				Required: true, // require it as the default one is more trouble than it's worth
			},
			"amount_off": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"currency": {
				Type:     schema.TypeString, // <- check values
				Optional: true,
				ForceNew: true,
			},
			"duration": {
				Type:     schema.TypeString,
				Required: true, // forever | once | repeating
				ForceNew: true,
			},
			"duration_in_months": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"max_redemptions": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  nil,
				ForceNew: true,
			},
			"metadata": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"percent_off": {
				Type:     schema.TypeFloat,
				Optional: true,
				ForceNew: true,
			},
			"redeem_by": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			// Computed
			"valid": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"created": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"livemode": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"times_redeemed": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func resourceStripeCouponCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)
	couponID := d.Get("code").(string)

	params := &stripe.CouponParams{
		ID: stripe.String(couponID),
	}
	params.Context = ctx

	couponDuration := d.Get("duration").(string)
	validDurations := map[string]bool{
		"repeating": true,
		"once":      true,
		"forever":   true,
	}
	if !(validDurations)[couponDuration] {
		formattedKeys := "( " + strings.Join(getMapKeys(validDurations), " | ") + " )"
		return diag.Errorf("\"%s\" is not a valid value for \"duration\", expected one of %s", couponDuration, formattedKeys)
	}

	if name, ok := d.GetOk("name"); ok {
		params.Name = stripe.String(name.(string))
	}

	if durationInMonths, ok := d.GetOk("duration_in_months"); ok {
		if couponDuration != "repeating" {
			return diag.Errorf("can't set duration in months if event is not repeating")
		}
		params.DurationInMonths = stripe.Int64(int64(durationInMonths.(int)))
	}

	if couponDuration != "" {
		params.Duration = stripe.String(couponDuration)
	}

	if percentOff, ok := d.GetOk("percent_off"); ok {
		params.PercentOff = stripe.Float64(percentOff.(float64))
	}

	if amountOff, ok := d.GetOk("amount_off"); ok {
		params.AmountOff = stripe.Int64(int64(amountOff.(int)))
	}

	if maxRedemptions, ok := d.GetOk("max_redemptions"); ok {
		params.MaxRedemptions = stripe.Int64(int64(maxRedemptions.(int)))
	}

	if currency, ok := d.GetOk("currency"); ok {
		if params.AmountOff == nil {
			return diag.Errorf("can only set currency when using amount_off")
		}
		params.Currency = stripe.String(currency.(string))
	}

	if redeemByStr, ok := d.GetOk("redeem_by"); ok {
		redeemByTime, err := time.Parse(time.RFC3339, redeemByStr.(string))

		if err != nil {
			return diag.Errorf("can't convert time \"%s\" to time.  Please check if it's RFC3339-compliant", redeemByStr)
		}

		params.RedeemBy = stripe.Int64(redeemByTime.Unix())
	}

	params.Metadata = expandMetadata(d)

	coupon, err := client.Coupons.New(params)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Create coupon: %s (%s)", coupon.Name, coupon.ID)
	d.SetId(coupon.ID)
	d.Set("valid", coupon.Valid)
	d.Set("created", coupon.Created)
	d.Set("times_redeemed", coupon.TimesRedeemed)
	d.Set("livemode", coupon.Livemode)
	return nil
}

func resourceStripeCouponRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)

	params := &stripe.CouponParams{}
	params.Context = ctx

	coupon, err := client.Coupons.Get(d.Id(), params)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("code", d.Id())
	d.Set("amount_off", coupon.AmountOff)
	d.Set("currency", coupon.Currency)
	d.Set("duration", coupon.Duration)
	d.Set("duration_in_months", coupon.DurationInMonths)
	d.Set("livemode", coupon.Livemode)
	d.Set("max_redemptions", coupon.MaxRedemptions)
	d.Set("metadata", coupon.Metadata)
	d.Set("name", coupon.Name)
	d.Set("percent_off", coupon.PercentOff)
	d.Set("redeem_by", coupon.RedeemBy)
	d.Set("times_redeemed", coupon.TimesRedeemed)
	d.Set("valid", coupon.Valid)
	d.Set("created", coupon.Valid)
	return nil
}

func resourceStripeCouponUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)

	params := &stripe.CouponParams{}
	params.Context = ctx

	if d.HasChange("metadata") {
		params.Metadata = expandMetadata(d)
	}

	if d.HasChange("name") {
		params.Name = stripe.String(d.Get("name").(string))
	}

	if _, err := client.Coupons.Update(d.Id(), params); err != nil {
		return diag.FromErr(err)
	}

	return resourceStripeCouponRead(ctx, d, m)
}

func resourceStripeCouponDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client.API)

	params := &stripe.CouponParams{}
	params.Context = ctx

	if _, err := client.Coupons.Del(d.Id(), params); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
