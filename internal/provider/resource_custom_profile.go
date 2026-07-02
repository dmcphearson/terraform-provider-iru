package provider

import (
	"context"
	"fmt"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &customProfileResource{}
	_ resource.ResourceWithConfigure   = &customProfileResource{}
	_ resource.ResourceWithImportState = &customProfileResource{}
)

func NewCustomProfileResource() resource.Resource { return &customProfileResource{} }

type customProfileResource struct {
	client *client.Client
}

type customProfileModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	ProfileFile types.String `tfsdk:"profile_file"`
	Active      types.Bool   `tfsdk:"active"`
	RunsOnMac   types.Bool   `tfsdk:"runs_on_mac"`
	RunsOnIPhone types.Bool  `tfsdk:"runs_on_iphone"`
	RunsOnIPad  types.Bool   `tfsdk:"runs_on_ipad"`
	RunsOnTV    types.Bool   `tfsdk:"runs_on_tv"`
}

func (r *customProfileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_profile"
}

func (r *customProfileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Iru Custom Profile (.mobileconfig) library item.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the custom profile.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the custom profile.",
			},
			"profile_file": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "The .mobileconfig XML content (typically `file(\"...\")`). " +
					"The API does not return this field on read, so it is tracked from configuration only.",
			},
			"active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the profile is active.",
			},
			"runs_on_mac": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Install on macOS devices.",
			},
			"runs_on_iphone": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Install on iPhone devices.",
			},
			"runs_on_ipad": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Install on iPad devices.",
			},
			"runs_on_tv": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Install on Apple TV devices.",
			},
		},
	}
}

func (r *customProfileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *customProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customProfileModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.CreateCustomProfile(ctx, profileInput(plan, true))
	if err != nil {
		resp.Diagnostics.AddError("Error creating custom profile", err.Error())
		return
	}
	applyProfileResponse(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customProfileModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetCustomProfile(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading custom profile", err.Error())
		return
	}
	// profile_file is not returned by the API; keep the configured value as-is.
	applyProfileResponse(&state, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *customProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state customProfileModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Only re-upload the file when the configured content changed.
	includeFile := plan.ProfileFile.ValueString() != state.ProfileFile.ValueString()
	out, err := r.client.UpdateCustomProfile(ctx, plan.ID.ValueString(), profileInput(plan, includeFile))
	if err != nil {
		resp.Diagnostics.AddError("Error updating custom profile", err.Error())
		return
	}
	applyProfileResponse(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customProfileModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteCustomProfile(ctx, state.ID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting custom profile", err.Error())
	}
}

func (r *customProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func profileInput(m customProfileModel, includeFile bool) client.CustomProfileInput {
	return client.CustomProfileInput{
		Name:         m.Name.ValueString(),
		ProfileXML:   m.ProfileFile.ValueString(),
		Active:       m.Active.ValueBool(),
		RunsOnMac:    m.RunsOnMac.ValueBool(),
		RunsOnIPhone: m.RunsOnIPhone.ValueBool(),
		RunsOnIPad:   m.RunsOnIPad.ValueBool(),
		RunsOnTV:     m.RunsOnTV.ValueBool(),
		IncludeFile:  includeFile,
	}
}

func applyProfileResponse(dst *customProfileModel, out *client.CustomProfile) {
	dst.ID = types.StringValue(out.ID)
	dst.Name = types.StringValue(out.Name)
	dst.Active = types.BoolValue(out.Active)
	dst.RunsOnMac = types.BoolValue(out.RunsOnMac)
	dst.RunsOnIPhone = types.BoolValue(out.RunsOnIPhone)
	dst.RunsOnIPad = types.BoolValue(out.RunsOnIPad)
	dst.RunsOnTV = types.BoolValue(out.RunsOnTV)
	// dst.ProfileFile intentionally left untouched (not returned by API).
}
