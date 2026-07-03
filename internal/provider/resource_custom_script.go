package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &customScriptResource{}
	_ resource.ResourceWithConfigure   = &customScriptResource{}
	_ resource.ResourceWithImportState = &customScriptResource{}
)

func NewCustomScriptResource() resource.Resource { return &customScriptResource{} }

type customScriptResource struct {
	client *client.Client
}

type customScriptModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	ExecutionFrequency     types.String `tfsdk:"execution_frequency"`
	Script                 types.String `tfsdk:"script"`
	RemediationScript      types.String `tfsdk:"remediation_script"`
	Active                 types.Bool   `tfsdk:"active"`
	Restart                types.Bool   `tfsdk:"restart"`
	ShowInSelfService      types.Bool   `tfsdk:"show_in_self_service"`
	SelfServiceCategoryID  types.String `tfsdk:"self_service_category_id"`
	SelfServiceRecommended types.Bool   `tfsdk:"self_service_recommended"`
}

func (r *customScriptResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_script"
}

func (r *customScriptResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Iru Custom Script library item.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the custom script.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the custom script.",
			},
			"execution_frequency": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "How often the script runs. One of `once`, `every_15_min`, " +
					"`every_day`, `no_enforcement`.",
				Validators: []validator.String{
					stringvalidator.OneOf("once", "every_15_min", "every_day", "no_enforcement"),
				},
			},
			"script": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The script body.",
			},
			"remediation_script": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Optional remediation script body. The API returns an " +
					"empty string when unset; leaving this unconfigured adopts that value.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the script is active. Defaults to the server value.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"restart": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether to restart after running.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"show_in_self_service": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the script is offered in Self Service.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"self_service_category_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Self Service category ID (from the " +
					"`iru_self_service_categories` data source). Write-only in the API: it is " +
					"accepted on write but not returned, so the configured value is preserved in state.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"self_service_recommended": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Whether the script is marked recommended in Self Service. " +
					"Write-only in the API (see `self_service_category_id`).",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *customScriptResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("expected *client.Client, got %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *customScriptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customScriptModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.CreateCustomScript(ctx, modelToCustomScript(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating custom script", err.Error())
		return
	}

	// Map server response back, but preserve the configured script and write-only
	// self-service values (the API trims trailing whitespace from script and never
	// returns the self-service fields).
	applyCustomScriptResponse(&plan, out, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customScriptResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customScriptModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetCustomScript(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading custom script", err.Error())
		return
	}

	applyCustomScriptResponse(&state, out, state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *customScriptResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan customScriptModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpdateCustomScript(ctx, plan.ID.ValueString(), modelToCustomScript(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating custom script", err.Error())
		return
	}

	applyCustomScriptResponse(&plan, out, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customScriptResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customScriptModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteCustomScript(ctx, state.ID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting custom script", err.Error())
	}
}

func (r *customScriptResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// modelToCustomScript builds the API request from the plan, sending only the fields
// the caller set so server defaults aren't clobbered.
func modelToCustomScript(m customScriptModel) client.CustomScript {
	return client.CustomScript{
		Name:                   m.Name.ValueString(),
		ExecutionFrequency:     m.ExecutionFrequency.ValueString(),
		Script:                 m.Script.ValueString(),
		RemediationScript:      m.RemediationScript.ValueString(),
		ShowInSelfService:      m.ShowInSelfService.ValueBool(),
		SelfServiceCategoryID:  m.SelfServiceCategoryID.ValueString(),
		SelfServiceRecommended: m.SelfServiceRecommended.ValueBool(),
		Active:                 m.Active.ValueBool(),
		Restart:                m.Restart.ValueBool(),
	}
}

// applyCustomScriptResponse maps the server response into dst, while carrying forward
// values the API does not faithfully round-trip: the configured `script` (the API
// trims trailing whitespace) and the write-only self-service fields (never returned).
// prior holds the values to preserve (the plan on write, the state on read).
func applyCustomScriptResponse(dst *customScriptModel, out *client.CustomScript, prior customScriptModel) {
	dst.ID = types.StringValue(out.ID)
	dst.Name = types.StringValue(out.Name)
	dst.ExecutionFrequency = types.StringValue(out.ExecutionFrequency)
	dst.Active = types.BoolValue(out.Active)
	dst.Restart = types.BoolValue(out.Restart)
	dst.ShowInSelfService = types.BoolValue(out.ShowInSelfService)

	// remediation_script is Optional+Computed: adopt the server value (the API returns
	// "" when unset, never null), so an unconfigured field settles on "" and is stable.
	dst.RemediationScript = types.StringValue(out.RemediationScript)

	// script: preserve the configured body unless the server's differs by more than
	// trailing whitespace (which the API strips).
	if trimTrailing(prior.Script.ValueString()) == trimTrailing(out.Script) {
		dst.Script = prior.Script
	} else {
		dst.Script = types.StringValue(out.Script)
	}

	// self-service fields are write-only (never returned by GET) but Computed, so
	// carry the prior (configured/state) value forward. When the config left them
	// unset the prior is unknown/null on create — resolve to a concrete zero value so
	// the post-apply state is known and stable.
	if prior.SelfServiceCategoryID.IsUnknown() || prior.SelfServiceCategoryID.IsNull() {
		dst.SelfServiceCategoryID = types.StringValue("")
	} else {
		dst.SelfServiceCategoryID = prior.SelfServiceCategoryID
	}
	if prior.SelfServiceRecommended.IsUnknown() || prior.SelfServiceRecommended.IsNull() {
		dst.SelfServiceRecommended = types.BoolValue(false)
	} else {
		dst.SelfServiceRecommended = prior.SelfServiceRecommended
	}
}

func trimTrailing(s string) string {
	return strings.TrimRight(s, "\n\r\t ")
}
