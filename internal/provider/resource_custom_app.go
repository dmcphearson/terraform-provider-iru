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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &customAppResource{}
	_ resource.ResourceWithConfigure   = &customAppResource{}
	_ resource.ResourceWithImportState = &customAppResource{}
	_ resource.ResourceWithIdentity    = &customAppResource{}
	_ resource.ResourceWithModifyPlan  = &customAppResource{}
)

func NewCustomAppResource() resource.Resource { return &customAppResource{} }

type customAppResource struct {
	client *client.Client
}

type customAppModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	FileKey                types.String `tfsdk:"file_key"`
	Source                 types.String `tfsdk:"source"`
	SourceSHA256           types.String `tfsdk:"source_sha256"`
	InstallType            types.String `tfsdk:"install_type"`
	InstallEnforcement     types.String `tfsdk:"install_enforcement"`
	UnzipLocation          types.String `tfsdk:"unzip_location"`
	AuditScript            types.String `tfsdk:"audit_script"`
	PreinstallScript       types.String `tfsdk:"preinstall_script"`
	PostinstallScript      types.String `tfsdk:"postinstall_script"`
	Active                 types.Bool   `tfsdk:"active"`
	Restart                types.Bool   `tfsdk:"restart"`
	ShowInSelfService      types.Bool   `tfsdk:"show_in_self_service"`
	SelfServiceCategoryID  types.String `tfsdk:"self_service_category_id"`
	SelfServiceRecommended types.Bool   `tfsdk:"self_service_recommended"`
	FileURL                types.String `tfsdk:"file_url"`
	FileSize               types.Int64  `tfsdk:"file_size"`
	FileUpdated            types.String `tfsdk:"file_updated"`
	SHA256                 types.String `tfsdk:"sha256"`
}

func (r *customAppResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_app"
}

func (r *customAppResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// File-derived metadata (file_url, file_updated, sha256) is recomputed by the
	// server from the current binary. It must NOT use UseStateForUnknown: when
	// file_key changes (a new binary), the plan would otherwise predict the stale
	// value while the API returns the new one -> "inconsistent result after apply".
	// file_url is additionally a short-lived presigned URL that changes every refresh.
	// So these settle to "known after apply" whenever the resource changes.
	computedStr := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: desc,
		}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Iru Custom App library item. Set `source` to a local " +
			"installer and the provider uploads it, re-uploading only when the file's SHA256 " +
			"changes. Alternatively set `file_key` to reference a binary uploaded out of band; " +
			"changing it rolls out that binary.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the custom app.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the custom app.",
			},
			"file_key": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Storage key of an installer uploaded out of band. Exactly one of " +
					"`file_key` or `source` is required; with `source` this is set by the provider.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:    []validator.String{stringvalidator.ExactlyOneOf(path.MatchRoot("source"))},
			},
			"source": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Local path to the installer (`.pkg`, `.zip` or `.dmg`) to upload.",
			},
			"source_sha256": schema.StringAttribute{
				Computed: true,
				MarkdownDescription: "SHA256 of the file at `source`, computed at plan time. A change " +
					"triggers a new upload.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"install_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Installer type. One of `package`, `zip`, `image`.",
				Validators:          []validator.String{stringvalidator.OneOf("package", "zip", "image")},
			},
			"install_enforcement": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "Enforcement mode. One of `install_once`, `continuously_enforce`, " +
					"`no_enforcement`.",
				Validators: []validator.String{stringvalidator.OneOf("install_once", "continuously_enforce", "no_enforcement")},
			},
			"unzip_location": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Destination path for `zip` installs.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"audit_script":       optionalComputedScript("Audit script body."),
			"preinstall_script":  optionalComputedScript("Pre-install script body."),
			"postinstall_script": optionalComputedScript("Post-install script body."),
			"active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the app is active.",
				Default:             booldefault.StaticBool(true),
			},
			"restart": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether to restart after install.",
				Default:             booldefault.StaticBool(false),
			},
			"show_in_self_service": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the app is offered in Self Service.",
				Default:             booldefault.StaticBool(false),
			},
			"self_service_category_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Self Service category ID (from `iru_self_service_categories`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"self_service_recommended": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the app is marked recommended in Self Service.",
				Default:             booldefault.StaticBool(false),
			},
			"file_url":     computedStr("Short-lived download URL for the current binary."),
			"file_updated": computedStr("Timestamp the binary was last updated."),
			"sha256":       computedStr("SHA256 of the current binary, as reported by the API."),
			"file_size": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Size in bytes of the current binary.",
			},
		},
	}
}

func optionalComputedScript(desc string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		MarkdownDescription: desc,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

func (r *customAppResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *customAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customAppModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := appModelToClient(plan)
	if !plan.Source.IsNull() {
		fileKey, ok := r.upload(ctx, plan, &resp.Diagnostics)
		if !ok {
			return
		}
		in.FileKey = fileKey
	}
	out, err := r.client.CreateCustomApp(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating custom app", err.Error())
		return
	}
	applyAppResponse(&plan, out, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(setIDIdentity(ctx, resp.Identity, plan.ID)...)
}

func (r *customAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customAppModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetCustomApp(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading custom app", err.Error())
		return
	}
	applyAppResponse(&state, out, state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(setIDIdentity(ctx, resp.Identity, state.ID)...)
}

func (r *customAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state customAppModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := appModelToClient(plan)
	if !plan.Source.IsNull() {
		// file_key is sent only when replacing the binary.
		in.FileKey = ""
		if needsUpload(state.SourceSHA256, state.SHA256, plan.SourceSHA256.ValueString()) {
			fileKey, ok := r.upload(ctx, plan, &resp.Diagnostics)
			if !ok {
				return
			}
			in.FileKey = fileKey
		}
	}
	out, err := r.client.UpdateCustomApp(ctx, plan.ID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error updating custom app", err.Error())
		return
	}
	applyAppResponse(&plan, out, plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(setIDIdentity(ctx, resp.Identity, plan.ID)...)
}

func (r *customAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customAppModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteCustomApp(ctx, state.ID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting custom app", err.Error())
	}
}

// ModifyPlan hashes the file at source so a changed installer shows in the plan, and
// marks the binary-derived attributes unknown when it will be re-uploaded.
func (r *customAppResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return // destroy
	}
	var source types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("source"), &source)...)
	if resp.Diagnostics.HasError() {
		return
	}
	switch {
	case source.IsNull():
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("source_sha256"), types.StringNull())...)
		return
	case source.IsUnknown():
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("source_sha256"), types.StringUnknown())...)
		markCustomAppBinaryUnknown(ctx, resp)
		return
	}
	sum, err := fileSHA256(source.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("source"), "Cannot read source file", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("source_sha256"), sum)...)
	if req.State.Raw.IsNull() {
		return // create: computed attributes are already unknown
	}
	var state customAppModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if needsUpload(state.SourceSHA256, state.SHA256, sum) {
		markCustomAppBinaryUnknown(ctx, resp)
	}
}

func markCustomAppBinaryUnknown(ctx context.Context, resp *resource.ModifyPlanResponse) {
	for _, a := range []string{"file_key", "sha256", "file_url", "file_updated"} {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root(a), types.StringUnknown())...)
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("file_size"), types.Int64Unknown())...)
}

// upload sends the installer after checking it still matches the planned hash.
func (r *customAppResource) upload(ctx context.Context, plan customAppModel, diags *diag.Diagnostics) (string, bool) {
	if !checkSourceUnchanged(plan.Source, plan.SourceSHA256, diags) {
		return "", false
	}
	fileKey, err := r.client.UploadCustomApp(ctx, plan.Source.ValueString())
	if err != nil {
		diags.AddError("Error uploading custom app", err.Error())
		return "", false
	}
	return fileKey, true
}

func (r *customAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("id"), req, resp)
}

func (r *customAppResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = idIdentitySchema()
}

func appModelToClient(m customAppModel) client.CustomApp {
	return client.CustomApp{
		Name:                   m.Name.ValueString(),
		FileKey:                m.FileKey.ValueString(),
		InstallType:            m.InstallType.ValueString(),
		InstallEnforcement:     m.InstallEnforcement.ValueString(),
		UnzipLocation:          m.UnzipLocation.ValueString(),
		AuditScript:            m.AuditScript.ValueString(),
		PreinstallScript:       m.PreinstallScript.ValueString(),
		PostinstallScript:      m.PostinstallScript.ValueString(),
		Active:                 m.Active.ValueBool(),
		Restart:                m.Restart.ValueBool(),
		ShowInSelfService:      m.ShowInSelfService.ValueBool(),
		SelfServiceCategoryID:  m.SelfServiceCategoryID.ValueString(),
		SelfServiceRecommended: m.SelfServiceRecommended.ValueBool(),
	}
}

// applyAppResponse maps the server response into dst. The API trims trailing
// whitespace from the script bodies, so for each configured script we keep the prior
// (configured/state) value when it differs only by trailing whitespace; otherwise we
// adopt the server value. prior holds the plan on write and the state on read.
func applyAppResponse(dst *customAppModel, out *client.CustomApp, prior customAppModel) {
	dst.ID = types.StringValue(out.ID)
	dst.Name = types.StringValue(out.Name)
	dst.FileKey = types.StringValue(out.FileKey)
	dst.InstallType = types.StringValue(out.InstallType)
	dst.InstallEnforcement = types.StringValue(out.InstallEnforcement)
	dst.UnzipLocation = types.StringValue(out.UnzipLocation)
	dst.AuditScript = preserveScript(prior.AuditScript, out.AuditScript)
	dst.PreinstallScript = preserveScript(prior.PreinstallScript, out.PreinstallScript)
	dst.PostinstallScript = preserveScript(prior.PostinstallScript, out.PostinstallScript)
	dst.Active = types.BoolValue(out.Active)
	dst.Restart = types.BoolValue(out.Restart)
	dst.ShowInSelfService = types.BoolValue(out.ShowInSelfService)
	dst.SelfServiceCategoryID = types.StringValue(out.SelfServiceCategoryID)
	dst.SelfServiceRecommended = types.BoolValue(out.SelfServiceRecommended)
	dst.FileURL = types.StringValue(out.FileURL)
	dst.FileSize = types.Int64Value(out.FileSize)
	dst.FileUpdated = types.StringValue(out.FileUpdated)
	dst.SHA256 = types.StringValue(out.SHA256)
}

// preserveScript keeps the configured script body when it differs from the server's
// only by trailing whitespace (which the API strips); otherwise it adopts the server
// value. When prior is unset (null/unknown), the server value ("" when unset) is used
// so the field settles on a known, stable value.
func preserveScript(prior types.String, server string) types.String {
	if prior.IsNull() || prior.IsUnknown() {
		return types.StringValue(server)
	}
	if trimTrailing(prior.ValueString()) == trimTrailing(server) {
		return prior
	}
	return types.StringValue(server)
}
