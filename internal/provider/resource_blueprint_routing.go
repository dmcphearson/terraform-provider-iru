package provider

import (
	"context"
	"fmt"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = &blueprintRoutingResource{}
	_ resource.ResourceWithConfigure   = &blueprintRoutingResource{}
	_ resource.ResourceWithImportState = &blueprintRoutingResource{}
	_ resource.ResourceWithIdentity    = &blueprintRoutingResource{}
)

// blueprintRoutingID is the fixed ID of the tenant's one routing setting.
const blueprintRoutingID = "blueprint_routing"

func NewBlueprintRoutingResource() resource.Resource { return &blueprintRoutingResource{} }

// blueprintRoutingResource adopts the tenant's Blueprint Routing setting. It
// always exists, so it is never created, and destroy leaves it as it is.
type blueprintRoutingResource struct {
	client *client.Client
}

type blueprintRoutingModel struct {
	ID             types.String `tfsdk:"id"`
	EnrollmentCode types.Object `tfsdk:"enrollment_code"`
}

func (r *blueprintRoutingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprint_routing"
}

func (r *blueprintRoutingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the tenant's Blueprint Routing enrollment code, used for Manual " +
			"and Android Work Profile enrollment.\n\n" +
			"There is one routing setting per tenant and it always exists, so this resource " +
			"**cannot create it**: import it with the ID `" + blueprintRoutingID + "`. **Destroy only " +
			"removes it from Terraform state**; the setting in Iru is left as it is. Routing rules " +
			"are not exposed by the Iru API and are managed in the Iru UI.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Always `" + blueprintRoutingID + "`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enrollment_code": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Routing enrollment code. Attributes left out keep their current value.",
				PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
				Attributes: map[string]schema.Attribute{
					"code": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "The enrollment code.",
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"is_active": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "Whether the enrollment code is active.",
						PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
					},
				},
			},
		},
	}
}

func (r *blueprintRoutingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *blueprintRoutingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("iru_blueprint_routing must be imported",
		"Every tenant has one Blueprint Routing setting. Import it instead, with an import block or "+
			"`terraform import iru_blueprint_routing.<name> "+blueprintRoutingID+"`, so the current "+
			"enrollment code is never overwritten by accident.")
}

func (r *blueprintRoutingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	out, err := r.client.GetBlueprintRouting(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading blueprint routing", err.Error())
		return
	}
	m, diags := blueprintRoutingFrom(out)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
	resp.Diagnostics.Append(setIDIdentity(ctx, resp.Identity, m.ID)...)
}

func (r *blueprintRoutingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state blueprintRoutingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	patch, diags := blueprintRoutingPatch(ctx, state, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var out *client.BlueprintRouting
	var err error
	if len(patch) > 0 {
		out, err = r.client.UpdateBlueprintRouting(ctx, patch)
	} else {
		out, err = r.client.GetBlueprintRouting(ctx)
	}
	if err != nil {
		resp.Diagnostics.AddError("Error updating blueprint routing", err.Error())
		return
	}
	m, diags := blueprintRoutingFrom(out)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
	resp.Diagnostics.Append(setIDIdentity(ctx, resp.Identity, m.ID)...)
}

// Delete only removes the setting from state. It makes no API call.
func (r *blueprintRoutingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("Blueprint routing removed from Terraform state only",
		"The routing setting and its enrollment code are unchanged in Iru.")
}

func (r *blueprintRoutingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("id"), req, resp)
}

func (r *blueprintRoutingResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = idIdentitySchema()
}

func blueprintRoutingFrom(b *client.BlueprintRouting) (blueprintRoutingModel, diag.Diagnostics) {
	code, diags := types.ObjectValue(enrollmentCodeAttrTypes, map[string]attr.Value{
		"code":      types.StringValue(b.EnrollmentCode.Code),
		"is_active": types.BoolValue(b.EnrollmentCode.IsActive),
	})
	return blueprintRoutingModel{ID: types.StringValue(blueprintRoutingID), EnrollmentCode: code}, diags
}

// blueprintRoutingPatch returns the enrollment_code keys that changed between
// state and plan. Unknown and null plan values are left alone.
func blueprintRoutingPatch(ctx context.Context, state, plan blueprintRoutingModel) (map[string]any, diag.Diagnostics) {
	patch := map[string]any{}
	if plan.EnrollmentCode.IsNull() || plan.EnrollmentCode.IsUnknown() {
		return patch, nil
	}
	var p, s enrollmentCodeModel
	opts := basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true, UnhandledUnknownAsEmpty: true}
	diags := plan.EnrollmentCode.As(ctx, &p, opts)
	diags.Append(state.EnrollmentCode.As(ctx, &s, opts)...)
	if !p.Code.IsUnknown() && !p.Code.IsNull() && !p.Code.Equal(s.Code) {
		patch["code"] = p.Code.ValueString()
	}
	if !p.IsActive.IsUnknown() && !p.IsActive.IsNull() && !p.IsActive.Equal(s.IsActive) {
		patch["is_active"] = p.IsActive.ValueBool()
	}
	return patch, diags
}
