package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &inHouseAppResource{}
	_ resource.ResourceWithConfigure   = &inHouseAppResource{}
	_ resource.ResourceWithImportState = &inHouseAppResource{}
	_ resource.ResourceWithIdentity    = &inHouseAppResource{}
	_ resource.ResourceWithModifyPlan  = &inHouseAppResource{}
)

func NewInHouseAppResource() resource.Resource { return &inHouseAppResource{} }

type inHouseAppResource struct {
	client *client.Client
}

type inHouseAppModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Source           types.String `tfsdk:"source"`
	SourceSHA256     types.String `tfsdk:"source_sha256"`
	Active           types.Bool   `tfsdk:"active"`
	RunsOnIPhone     types.Bool   `tfsdk:"runs_on_iphone"`
	RunsOnIPad       types.Bool   `tfsdk:"runs_on_ipad"`
	RunsOnTV         types.Bool   `tfsdk:"runs_on_tv"`
	FileKey          types.String `tfsdk:"file_key"`
	SHA256           types.String `tfsdk:"sha256"`
	FileSize         types.Int64  `tfsdk:"file_size"`
	FileUpdated      types.String `tfsdk:"file_updated"`
	AppIdentifier    types.String `tfsdk:"app_identifier"`
	AppName          types.String `tfsdk:"app_name"`
	AppVersion       types.String `tfsdk:"app_version"`
	MinimumOSVersion types.String `tfsdk:"minimum_os_version"`
}

// fileDerivedStrings are the computed attributes that change only when a new .ipa is
// uploaded. They keep their state value until ModifyPlan sees a new source hash.
var fileDerivedStrings = []string{"file_key", "sha256", "file_updated", "app_identifier", "app_name", "app_version", "minimum_os_version"}

func (r *inHouseAppResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_in_house_app"
}

func (r *inHouseAppResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	keep := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	fileStr := func(desc string) schema.StringAttribute {
		return schema.StringAttribute{Computed: true, MarkdownDescription: desc, PlanModifiers: keep}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an Iru In-House App (an `.ipa` for iPhone, iPad or Apple TV). " +
			"The provider uploads the file at `source` and waits for Iru to validate it. A new " +
			"upload happens only when the file's SHA256 changes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier of the in-house app.",
				PlanModifiers:       keep,
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the library item.",
			},
			"source": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Local path to the `.ipa` file to upload.",
			},
			"source_sha256": schema.StringAttribute{
				Computed: true,
				MarkdownDescription: "SHA256 of the file at `source`, computed at plan time. A change " +
					"triggers a new upload.",
				PlanModifiers: keep,
			},
			"active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the app is active.",
				Default:             booldefault.StaticBool(true),
			},
			"runs_on_iphone": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Install on iPhone.",
				Default:             booldefault.StaticBool(true),
			},
			"runs_on_ipad": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Install on iPad.",
				Default:             booldefault.StaticBool(true),
			},
			"runs_on_tv": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Install on Apple TV.",
				Default:             booldefault.StaticBool(false),
			},
			"file_key":           fileStr("Storage key of the uploaded `.ipa`."),
			"sha256":             fileStr("SHA256 of the uploaded `.ipa`, as reported by the API."),
			"file_updated":       fileStr("Timestamp the `.ipa` was last uploaded."),
			"app_identifier":     fileStr("Bundle identifier read from the `.ipa`."),
			"app_name":           fileStr("App name read from the `.ipa`."),
			"app_version":        fileStr("App version read from the `.ipa`."),
			"minimum_os_version": fileStr("Minimum OS version read from the `.ipa`."),
			"file_size": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Size in bytes of the uploaded `.ipa`.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *inHouseAppResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan hashes the local file so a changed .ipa shows up in the plan, and marks
// the file-derived attributes unknown when it will be re-uploaded.
func (r *inHouseAppResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return // destroy
	}
	var source types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("source"), &source)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if source.IsUnknown() {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("source_sha256"), types.StringUnknown())...)
		markFileDerivedUnknown(ctx, resp)
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
	var state inHouseAppModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if needsUpload(state, sum) {
		markFileDerivedUnknown(ctx, resp)
	}
}

func markFileDerivedUnknown(ctx context.Context, resp *resource.ModifyPlanResponse) {
	for _, a := range fileDerivedStrings {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root(a), types.StringUnknown())...)
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("file_size"), types.Int64Unknown())...)
}

// needsUpload reports whether a file with hash sum differs from what Iru already
// holds. After an import source_sha256 is null, so the API's sha256 is compared too:
// adopting an existing app whose binary matches does not re-upload it.
func needsUpload(state inHouseAppModel, sum string) bool {
	return sum != state.SourceSHA256.ValueString() && sum != state.SHA256.ValueString()
}

func (r *inHouseAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan inHouseAppModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	fileKey, ok := r.upload(ctx, plan, &resp.Diagnostics)
	if !ok {
		return
	}
	in := inHouseAppModelToClient(plan)
	in.FileKey = fileKey
	out, err := r.client.CreateInHouseApp(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating in-house app", err.Error())
		return
	}
	applyInHouseAppResponse(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(setIDIdentity(ctx, resp.Identity, plan.ID)...)
}

func (r *inHouseAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state inHouseAppModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetInHouseApp(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading in-house app", err.Error())
		return
	}
	applyInHouseAppResponse(&state, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(setIDIdentity(ctx, resp.Identity, state.ID)...)
}

func (r *inHouseAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state inHouseAppModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := inHouseAppModelToClient(plan)
	if needsUpload(state, plan.SourceSHA256.ValueString()) {
		fileKey, ok := r.upload(ctx, plan, &resp.Diagnostics)
		if !ok {
			return
		}
		in.FileKey = fileKey
	}
	out, err := r.client.UpdateInHouseApp(ctx, plan.ID.ValueString(), in)
	if err != nil {
		resp.Diagnostics.AddError("Error updating in-house app", err.Error())
		return
	}
	applyInHouseAppResponse(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(setIDIdentity(ctx, resp.Identity, plan.ID)...)
}

func (r *inHouseAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state inHouseAppModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteInHouseApp(ctx, state.ID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting in-house app", err.Error())
	}
}

func (r *inHouseAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("id"), req, resp)
}

func (r *inHouseAppResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = idIdentitySchema()
}

// upload sends the .ipa after checking it still matches the planned hash, so the
// binary that ships is the one the plan showed.
func (r *inHouseAppResource) upload(ctx context.Context, plan inHouseAppModel, diags *diag.Diagnostics) (string, bool) {
	src := plan.Source.ValueString()
	sum, err := fileSHA256(src)
	if err != nil {
		diags.AddAttributeError(path.Root("source"), "Cannot read source file", err.Error())
		return "", false
	}
	if !plan.SourceSHA256.IsUnknown() && sum != plan.SourceSHA256.ValueString() {
		diags.AddAttributeError(path.Root("source"), "Source file changed since plan",
			fmt.Sprintf("%s has SHA256 %s, the plan expected %s. Run plan again.", src, sum, plan.SourceSHA256.ValueString()))
		return "", false
	}
	fileKey, err := r.client.UploadInHouseApp(ctx, src)
	if err != nil {
		diags.AddError("Error uploading in-house app", err.Error())
		return "", false
	}
	return fileKey, true
}

func inHouseAppModelToClient(m inHouseAppModel) client.InHouseApp {
	return client.InHouseApp{
		Name:         m.Name.ValueString(),
		Active:       m.Active.ValueBool(),
		RunsOnIPhone: m.RunsOnIPhone.ValueBool(),
		RunsOnIPad:   m.RunsOnIPad.ValueBool(),
		RunsOnTV:     m.RunsOnTV.ValueBool(),
	}
}

// applyInHouseAppResponse copies API values into m. source and source_sha256 are
// local-only and keep their planned or prior values.
func applyInHouseAppResponse(m *inHouseAppModel, out *client.InHouseApp) {
	m.ID = types.StringValue(out.ID)
	m.Name = types.StringValue(out.Name)
	m.Active = types.BoolValue(out.Active)
	m.RunsOnIPhone = types.BoolValue(out.RunsOnIPhone)
	m.RunsOnIPad = types.BoolValue(out.RunsOnIPad)
	m.RunsOnTV = types.BoolValue(out.RunsOnTV)
	m.FileKey = types.StringValue(out.FileKey)
	m.SHA256 = types.StringValue(out.SHA256)
	m.FileSize = types.Int64Value(out.FileSize)
	m.FileUpdated = types.StringValue(out.FileUpdated)
	m.AppIdentifier = types.StringValue(out.AppIdentifier)
	m.AppName = types.StringValue(out.AppName)
	m.AppVersion = types.StringValue(out.AppVersion)
	m.MinimumOSVersion = types.StringValue(out.MinimumOSVersion)
}

func fileSHA256(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
