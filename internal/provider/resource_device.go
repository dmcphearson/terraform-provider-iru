package provider

import (
	"context"
	"fmt"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &deviceResource{}
	_ resource.ResourceWithConfigure   = &deviceResource{}
	_ resource.ResourceWithImportState = &deviceResource{}
	_ resource.ResourceWithIdentity    = &deviceResource{}
)

func NewDeviceResource() resource.Resource { return &deviceResource{} }

// deviceResource adopts an enrolled device. Devices enroll through Iru, so it
// never creates one, and destroy only drops it from state.
type deviceResource struct {
	client *client.Client
}

type deviceModel struct {
	ID            types.String `tfsdk:"id"`
	AssetTag      types.String `tfsdk:"asset_tag"`
	UserID        types.String `tfsdk:"user_id"`
	BlueprintID   types.String `tfsdk:"blueprint_id"`
	Tags          types.Set    `tfsdk:"tags"`
	Name          types.String `tfsdk:"name"`
	SerialNumber  types.String `tfsdk:"serial_number"`
	Platform      types.String `tfsdk:"platform"`
	Model         types.String `tfsdk:"model"`
	OSVersion     types.String `tfsdk:"os_version"`
	UserEmail     types.String `tfsdk:"user_email"`
	UserName      types.String `tfsdk:"user_name"`
	BlueprintName types.String `tfsdk:"blueprint_name"`
}

func (r *deviceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

func (r *deviceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	keep := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	managed := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: desc, PlanModifiers: keep}
	}
	readOnly := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Computed: true, MarkdownDescription: desc, PlanModifiers: keep}
	}
	// user_email, user_name and blueprint_name follow user_id and blueprint_id, so
	// they are recomputed on any change.
	derived := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Computed: true, MarkdownDescription: desc}
	}
	blueprint := managed("Blueprint the device is assigned to. Changing it moves the device, and its library items change with it.")
	blueprint.Validators = []validator.String{stringvalidator.LengthAtLeast(1)}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an enrolled Iru device: its asset tag, assigned user, blueprint " +
			"and tags.\n\n" +
			"Devices enroll through Iru, so this resource **cannot create devices**: bring one under " +
			"Terraform with an import block, `terraform import`, or `terraform query`. **Destroy only " +
			"removes the device from Terraform state**; it never deletes, unenrolls or erases the device.\n\n" +
			"Attributes left out of the configuration are not managed: Terraform keeps the imported " +
			"value and changes made in Iru don't show as drift. An update sends only the attributes " +
			"that changed. The device name can't be set through this API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Device ID.",
				PlanModifiers:       keep,
			},
			"asset_tag":    managed("Asset tag. Set to `\"\"` to clear it."),
			"user_id":      managed("ID of the assigned user (see `iru_user`). Set to `\"\"` to unassign."),
			"blueprint_id": blueprint,
			"tags": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				MarkdownDescription: "Tag names. When set, the device's tags are replaced with exactly " +
					"this set; `[]` clears them.",
				PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
			"name":           readOnly("Device name."),
			"serial_number":  readOnly("Serial number."),
			"platform":       readOnly("`Mac`, `iPhone`, `iPad` or `AppleTV`."),
			"model":          readOnly("Hardware model."),
			"os_version":     derived("OS version."),
			"user_email":     derived("Assigned user's email."),
			"user_name":      derived("Assigned user's name."),
			"blueprint_name": derived("Blueprint name."),
		},
	}
}

func (r *deviceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *deviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError("iru_device cannot create devices",
		"Devices enroll through Iru. Import an existing device instead, with an import block, "+
			"`terraform import iru_device.<name> <device id>`, or `terraform query`.")
}

func (r *deviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deviceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetDevice(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading device", err.Error())
		return
	}
	resp.Diagnostics.Append(applyDevice(ctx, &state, out)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(setIDIdentity(ctx, resp.Identity, state.ID)...)
}

func (r *deviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state deviceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	patch, diags := devicePatch(ctx, state, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()
	if len(patch) > 0 {
		if _, err := r.client.UpdateDevice(ctx, id, patch); err != nil {
			resp.Diagnostics.AddError("Error updating device", err.Error())
			return
		}
	}
	// Re-read so derived fields (user email, blueprint name) are current.
	out, err := r.client.GetDevice(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading device after update", err.Error())
		return
	}
	resp.Diagnostics.Append(applyDevice(ctx, &plan, out)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(setIDIdentity(ctx, resp.Identity, plan.ID)...)
}

// Delete only removes the device from state. It makes no API call.
func (r *deviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("Device removed from Terraform state only",
		"The device is unchanged in Iru. iru_device never deletes, unenrolls or erases devices.")
}

func (r *deviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("id"), req, resp)
}

func (r *deviceResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = idIdentitySchema()
}

// devicePatch returns the PATCH body for what changed between state and plan.
// Unknown plan values are left alone; "" becomes null, which clears the field.
func devicePatch(ctx context.Context, state, plan deviceModel) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	patch := map[string]any{}
	str := func(key string, prior, planned types.String) {
		if planned.IsUnknown() || planned.Equal(prior) {
			return
		}
		if planned.ValueString() == "" {
			patch[key] = nil
			return
		}
		patch[key] = planned.ValueString()
	}
	str("asset_tag", state.AssetTag, plan.AssetTag)
	str("user", state.UserID, plan.UserID)
	str("blueprint_id", state.BlueprintID, plan.BlueprintID)

	if !plan.Tags.IsUnknown() && !plan.Tags.IsNull() && !plan.Tags.Equal(state.Tags) {
		tags := []string{}
		diags.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
		patch["tags"] = tags
	}
	return patch, diags
}

// applyDevice copies an API device into the model.
func applyDevice(ctx context.Context, m *deviceModel, d *client.Device) diag.Diagnostics {
	m.ID = types.StringValue(d.ID)
	m.AssetTag = types.StringValue(d.AssetTag)
	m.BlueprintID = types.StringValue(d.BlueprintID)
	m.BlueprintName = types.StringValue(d.BlueprintName)
	m.Name = types.StringValue(d.Name)
	m.SerialNumber = types.StringValue(d.SerialNumber)
	m.Platform = types.StringValue(d.Platform)
	m.Model = types.StringValue(d.Model)
	m.OSVersion = types.StringValue(d.OSVersion)
	m.UserID, m.UserEmail, m.UserName = types.StringValue(""), types.StringValue(""), types.StringValue("")
	if d.User != nil {
		m.UserID = types.StringValue(d.User.ID)
		m.UserEmail = types.StringValue(d.User.Email)
		m.UserName = types.StringValue(d.User.Name)
	}
	tags := d.Tags
	if tags == nil {
		tags = []string{}
	}
	set, diags := types.SetValueFrom(ctx, types.StringType, tags)
	m.Tags = set
	return diags
}
