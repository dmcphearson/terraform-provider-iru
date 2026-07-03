package provider

import (
	"context"
	"fmt"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = &blueprintResource{}
	_ resource.ResourceWithConfigure   = &blueprintResource{}
	_ resource.ResourceWithImportState = &blueprintResource{}
)

func NewBlueprintResource() resource.Resource { return &blueprintResource{} }

type blueprintResource struct {
	client *client.Client
}

type blueprintModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Icon           types.String `tfsdk:"icon"`
	Color          types.String `tfsdk:"color"`
	Description    types.String `tfsdk:"description"`
	Type           types.String `tfsdk:"type"`
	SourceID       types.String `tfsdk:"source_id"`
	ComputersCount types.Int64  `tfsdk:"computers_count"`
	EnrollmentCode types.Object `tfsdk:"enrollment_code"`
}

type enrollmentCodeModel struct {
	Code     types.String `tfsdk:"code"`
	IsActive types.Bool   `tfsdk:"is_active"`
}

// enrollmentCodeAttrTypes describes the enrollment_code object for conversions.
var enrollmentCodeAttrTypes = map[string]attr.Type{
	"code":      types.StringType,
	"is_active": types.BoolType,
}

func (r *blueprintResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprint"
}

func (r *blueprintResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Iru Blueprint. A blueprint groups library items and " +
			"assignment rules. `type` is `classic` or `map`; map blueprints are seeded from a " +
			"source blueprint at creation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the blueprint.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the blueprint.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Description of the blueprint.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"icon": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Icon identifier (e.g. `ss-files`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"color": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Color identifier (e.g. `aqua-500`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Blueprint type: `classic` or `map`. Changing forces replacement.",
				Validators:          []validator.String{stringvalidator.OneOf("classic", "map")},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_id": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "For map blueprints: the source blueprint ID to seed from at " +
					"creation. Used only on create; changing forces replacement.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"computers_count": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of computers assigned to the blueprint.",
			},
			"enrollment_code": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Enrollment code settings.",
				Attributes: map[string]schema.Attribute{
					"code": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "The enrollment code. Server-generated if unset.",
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"is_active": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Whether the enrollment code is active.",
					},
				},
			},
		},
	}
}

func (r *blueprintResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *blueprintResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan blueprintModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bp, diags := blueprintFromModel(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	in := client.BlueprintCreate{Blueprint: bp}
	if plan.Type.ValueString() == "map" && plan.SourceID.ValueString() != "" {
		in.SourceType = "blueprint"
		in.SourceID = plan.SourceID.ValueString()
	}

	out, err := r.client.CreateBlueprint(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating blueprint", err.Error())
		return
	}
	resp.Diagnostics.Append(applyBlueprint(ctx, &plan, out)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *blueprintResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state blueprintModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetBlueprint(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading blueprint", err.Error())
		return
	}
	resp.Diagnostics.Append(applyBlueprint(ctx, &state, out)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *blueprintResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan blueprintModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	bp, diags := blueprintFromModel(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.UpdateBlueprint(ctx, plan.ID.ValueString(), bp)
	if err != nil {
		resp.Diagnostics.AddError("Error updating blueprint", err.Error())
		return
	}
	resp.Diagnostics.Append(applyBlueprint(ctx, &plan, out)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *blueprintResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state blueprintModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteBlueprint(ctx, state.ID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting blueprint", err.Error())
	}
}

func (r *blueprintResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// blueprintFromModel builds the API payload from the plan/state model.
func blueprintFromModel(ctx context.Context, m blueprintModel) (client.Blueprint, diag.Diagnostics) {
	var diags diag.Diagnostics
	b := client.Blueprint{
		Name:        m.Name.ValueString(),
		Description: m.Description.ValueString(),
		Icon:        m.Icon.ValueString(),
		Color:       m.Color.ValueString(),
		Type:        m.Type.ValueString(),
	}
	if !m.EnrollmentCode.IsNull() && !m.EnrollmentCode.IsUnknown() {
		var ec enrollmentCodeModel
		d := m.EnrollmentCode.As(ctx, &ec, basetypes.ObjectAsOptions{})
		diags = append(diags, d...)
		b.EnrollmentCode = &client.EnrollmentCode{
			Code:     ec.Code.ValueString(),
			IsActive: ec.IsActive.ValueBool(),
		}
	}
	return b, diags
}

// applyBlueprint maps the API response into the model.
func applyBlueprint(ctx context.Context, dst *blueprintModel, out *client.Blueprint) diag.Diagnostics {
	var diags diag.Diagnostics
	dst.ID = types.StringValue(out.ID)
	dst.Name = types.StringValue(out.Name)
	dst.Description = types.StringValue(out.Description)
	dst.Icon = types.StringValue(out.Icon)
	dst.Color = types.StringValue(out.Color)
	dst.Type = types.StringValue(out.Type)
	dst.ComputersCount = types.Int64Value(out.ComputersCount)

	ecObj := types.ObjectNull(enrollmentCodeAttrTypes)
	if out.EnrollmentCode != nil {
		obj, d := types.ObjectValue(enrollmentCodeAttrTypes, map[string]attr.Value{
			"code":      types.StringValue(out.EnrollmentCode.Code),
			"is_active": types.BoolValue(out.EnrollmentCode.IsActive),
		})
		diags = append(diags, d...)
		ecObj = obj
	}
	dst.EnrollmentCode = ecObj
	return diags
}
