package provider

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/onlineornot/terraform-provider-onlineornot/internal/client"
)

var _ resource.ResourceWithImportState = &TokenResource{}

type TokenResource struct{ client *client.Client }
type tokenModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Grants       types.Set    `tfsdk:"grants"`
	ExpiresAt    types.String `tfsdk:"expires_at"`
	NeverExpires types.Bool   `tfsdk:"never_expires"`
	ExpiresAfter types.String `tfsdk:"expires_after"`
	Token        types.String `tfsdk:"token"`
}
type tokenGrantModel struct {
	Scope      types.String `tfsdk:"scope"`
	Permission types.String `tfsdk:"permission"`
}

var tokenGrantType = types.ObjectType{AttrTypes: map[string]attr.Type{"scope": types.StringType, "permission": types.StringType}}

func NewTokenResource() resource.Resource { return &TokenResource{} }
func (r *TokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token"
}

// Hand-written schema: the API has asymmetric create/read fields and no update.
func (r *TokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an API token. Configuration changes replace the token. The secret is only returned on creation and is stored in Terraform state.",
		Attributes: map[string]schema.Attribute{
			"id":            schema.StringAttribute{Computed: true, Description: "Token ID."},
			"name":          schema.StringAttribute{Required: true, Description: "Token name. Changes replace the token.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"expires_at":    schema.StringAttribute{Optional: true, Description: "Expiration in UTC RFC3339 format, with at most millisecond precision. Omit for the API default of 365 days from creation. To disable expiration, set never_expires = true. Changes replace the token.", Validators: []validator.String{tokenExpiryValidator{}}, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"never_expires": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Explicitly create a token without expiration. Cannot be combined with expires_at. Changes replace the token.", PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()}},
			"expires_after": schema.StringAttribute{Computed: true, Description: "Actual expiration returned by the API, including the default expiration. Null for a non-expiring token."},
			"token":         schema.StringAttribute{Computed: true, Sensitive: true, Description: "Secret returned only on creation, preserved on refresh. Unavailable (null) after import. Stored in state; protect state and plan files."},
			"grants": schema.SetNestedAttribute{Required: true, Description: "Unordered scope/permission pairs. EDIT implies READ. Changes replace the token.", PlanModifiers: []planmodifier.Set{setplanmodifier.RequiresReplace()}, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"scope":      schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("UPTIME_CHECKS", "STATUS_PAGES", "HEARTBEAT_CHECKS", "MAINTENANCE_WINDOWS", "PEOPLE", "INTEGRATIONS", "API_TOKENS", "WEBHOOKS")}},
				"permission": schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.OneOf("READ", "EDIT")}},
			}}},
		},
	}
}

// The API validates UTC datetimes and persists JavaScript Date milliseconds.
type tokenExpiryValidator struct{}

func (tokenExpiryValidator) Description(context.Context) string {
	return "must be a UTC RFC3339 timestamp with at most millisecond precision"
}
func (v tokenExpiryValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v tokenExpiryValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || len(value) == 0 || value[len(value)-1] != 'Z' || parsed.Nanosecond()%1000000 != 0 {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid expiration", v.Description(ctx))
	}
}
func (r *TokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	var ok bool
	r.client, ok = req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected resource configure type", fmt.Sprintf("Expected *client.Client, got %T", req.ProviderData))
	}
}

var _ resource.ResourceWithValidateConfig = &TokenResource{}

func (r *TokenResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data tokenModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if data.NeverExpires.ValueBool() && !data.ExpiresAt.IsNull() && !data.ExpiresAt.IsUnknown() {
		resp.Diagnostics.AddAttributeError(path.Root("expires_at"), "Conflicting expiration settings", "Do not set expires_at when never_expires is true.")
	}
}

// Never relay raw API errors here: upstream errors could echo the secret.
func tokenError(err error) string {
	if err == nil {
		return ""
	}
	return "The token API request failed. Check provider credentials, API_TOKENS permissions, requested grants, and API availability. Response details are withheld to protect token secrets."
}
func (r *TokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data tokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	var grants []tokenGrantModel
	resp.Diagnostics.Append(data.Grants.ElementsAs(ctx, &grants, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input := &client.CreateTokenRequest{Name: data.Name.ValueString(), Grants: make([]client.TokenGrant, 0, len(grants))}
	for _, g := range grants {
		input.Grants = append(input.Grants, client.TokenGrant{Scope: g.Scope.ValueString(), Permission: g.Permission.ValueString()})
	}
	if !data.ExpiresAt.IsNull() {
		expiry := data.ExpiresAt.ValueString()
		expiryPtr := &expiry
		input.ExpiresAt = &expiryPtr
	}
	if data.NeverExpires.ValueBool() {
		var noExpiry *string
		input.ExpiresAt = &noExpiry
	}
	token, err := r.client.CreateToken(input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating token", tokenError(err))
		return
	}
	data.ID = types.StringValue(token.ID)
	data.Token = types.StringValue(token.Token)
	data.ExpiresAfter = types.StringPointerValue(token.ExpiresAfter)
	// Create does not return grants. Keep the configured grants and the user's
	// equivalent timestamp representation (API normalizes to milliseconds).
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r *TokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data tokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	token, err := r.client.GetToken(data.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading token", tokenError(err))
		return
	}
	// Import initializes only id. Name is required and always populated after
	// creation/refresh, so its absence identifies the initial imported read.
	initialImport := data.Name.IsNull()
	data.ID = types.StringValue(token.ID)
	data.Name = types.StringValue(token.Name)
	grants := make([]tokenGrantModel, 0, len(token.Grants))
	for _, g := range token.Grants {
		grants = append(grants, tokenGrantModel{Scope: types.StringValue(g.Scope), Permission: types.StringValue(g.Permission)})
	}
	set, diags := types.SetValueFrom(ctx, tokenGrantType, grants)
	resp.Diagnostics.Append(diags...)
	data.Grants = set
	data.ExpiresAfter = types.StringPointerValue(token.ExpiresAfter)
	// Keep omitted expires_at as an input meaning "use the API default". The
	// actual date is tracked separately to avoid a changing time-based default.
	// On import there is no creation intent to preserve: recover the existing
	// expiration so matching configuration does not replace the credential.
	if initialImport || !data.ExpiresAt.IsNull() {
		if token.ExpiresAfter == nil {
			data.ExpiresAt = types.StringNull()
		} else if !sameTokenExpiry(data.ExpiresAt.ValueString(), *token.ExpiresAfter) {
			data.ExpiresAt = types.StringValue(*token.ExpiresAfter)
		}
	}
	data.NeverExpires = types.BoolValue(token.ExpiresAfter == nil)
	// GET never returns the secret. Preserve it from state; import remains null.
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func sameTokenExpiry(a, b string) bool {
	ta, ea := time.Parse(time.RFC3339Nano, a)
	tb, eb := time.Parse(time.RFC3339Nano, b)
	return ea == nil && eb == nil && ta.Equal(tb)
}
func (r *TokenResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Token update unsupported", "The API has no update endpoint; configuration changes must replace the token.")
}
func (r *TokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data tokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteToken(data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting token", tokenError(err))
	}
}
func (r *TokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if !regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString(req.ID) || req.ID == "verify" || req.ID == "permissions" {
		resp.Diagnostics.AddError("Invalid token ID", "Import requires an alphanumeric token ID, not the token secret or a URL.")
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
