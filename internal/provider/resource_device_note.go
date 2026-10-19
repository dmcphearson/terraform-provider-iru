package provider

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &deviceNoteResource{}
	_ resource.ResourceWithConfigure   = &deviceNoteResource{}
	_ resource.ResourceWithImportState = &deviceNoteResource{}
	_ resource.ResourceWithIdentity    = &deviceNoteResource{}
)

func NewDeviceNoteResource() resource.Resource { return &deviceNoteResource{} }

// deviceNoteResource manages one note on a device. It touches only that note,
// never the device itself.
type deviceNoteResource struct {
	client *client.Client
}

type deviceNoteModel struct {
	ID        types.String `tfsdk:"id"`
	DeviceID  types.String `tfsdk:"device_id"`
	NoteID    types.String `tfsdk:"note_id"`
	Content   types.String `tfsdk:"content"`
	Author    types.String `tfsdk:"author"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

type deviceNoteIdentityModel struct {
	DeviceID types.String `tfsdk:"device_id"`
	NoteID   types.String `tfsdk:"note_id"`
}

func (r *deviceNoteResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_note"
}

func (r *deviceNoteResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	keep := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages one note on an Iru device. It changes only that note; the " +
			"device itself is never modified, and destroy deletes only the note.\n\n" +
			"Notes written in the Iru UI are stored as HTML. A note whose text matches `content` " +
			"apart from HTML markup does not show as drift.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier `device_id:note_id`.",
				PlanModifiers:       keep,
			},
			"device_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the device the note is on. Changing it moves the note: a new note is created and the old one deleted.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"note_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Note ID.",
				PlanModifiers:       keep,
			},
			"content": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Note text.",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"author": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Who wrote the note.",
				PlanModifiers:       keep,
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the note was created.",
				PlanModifiers:       keep,
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the note was last changed.",
			},
		},
	}
}

func (r *deviceNoteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *deviceNoteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deviceNoteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.CreateDeviceNote(ctx, plan.DeviceID.ValueString(), plan.Content.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating device note", err.Error())
		return
	}
	applyDeviceNote(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(setDeviceNoteIdentity(ctx, resp.Identity, plan)...)
}

func (r *deviceNoteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deviceNoteModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.GetDeviceNote(ctx, state.DeviceID.ValueString(), state.NoteID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading device note", err.Error())
		return
	}
	applyDeviceNote(&state, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(setDeviceNoteIdentity(ctx, resp.Identity, state)...)
}

func (r *deviceNoteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan deviceNoteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	out, err := r.client.UpdateDeviceNote(ctx, plan.DeviceID.ValueString(), plan.NoteID.ValueString(), plan.Content.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating device note", err.Error())
		return
	}
	applyDeviceNote(&plan, out)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	resp.Diagnostics.Append(setDeviceNoteIdentity(ctx, resp.Identity, plan)...)
}

func (r *deviceNoteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state deviceNoteModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteDeviceNote(ctx, state.DeviceID.ValueString(), state.NoteID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting device note", err.Error())
	}
}

// ImportState accepts "device_id:note_id", or an import block identity with the
// same attributes.
func (r *deviceNoteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var ident deviceNoteIdentityModel
	if req.ID != "" {
		parts := strings.Split(req.ID, ":")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			resp.Diagnostics.AddError("Invalid import ID", "expected device_id:note_id, got: "+req.ID)
			return
		}
		ident = deviceNoteIdentityModel{DeviceID: types.StringValue(parts[0]), NoteID: types.StringValue(parts[1])}
	} else {
		resp.Diagnostics.Append(req.Identity.Get(ctx, &ident)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("device_id"), ident.DeviceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("note_id"), ident.NoteID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), ident.DeviceID.ValueString()+":"+ident.NoteID.ValueString())...)
}

func (r *deviceNoteResource) IdentitySchema(ctx context.Context, req resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"device_id": identityschema.StringAttribute{
				RequiredForImport: true,
				Description:       "ID of the device the note is on.",
			},
			"note_id": identityschema.StringAttribute{
				RequiredForImport: true,
				Description:       "Note ID.",
			},
		},
	}
}

func setDeviceNoteIdentity(ctx context.Context, identity *tfsdk.ResourceIdentity, m deviceNoteModel) diag.Diagnostics {
	if identity == nil {
		return nil
	}
	return identity.Set(ctx, deviceNoteIdentityModel{DeviceID: m.DeviceID, NoteID: m.NoteID})
}

// applyDeviceNote copies an API note into the model. Content already in the
// model is kept when it matches the API's text apart from HTML markup.
func applyDeviceNote(m *deviceNoteModel, n *client.DeviceNote) {
	if n.ID != "" {
		m.NoteID = types.StringValue(n.ID)
	}
	m.ID = types.StringValue(m.DeviceID.ValueString() + ":" + m.NoteID.ValueString())
	if m.Content.IsNull() || m.Content.IsUnknown() || noteText(m.Content.ValueString()) != noteText(n.Content) {
		m.Content = types.StringValue(n.Content)
	}
	m.Author = types.StringValue(n.Author)
	m.CreatedAt = types.StringValue(n.CreatedAt)
	m.UpdatedAt = types.StringValue(n.UpdatedAt)
}

var htmlTag = regexp.MustCompile(`<[^>]*>`)

// noteText reduces a note to its visible text: tags dropped, entities decoded
// and whitespace collapsed.
func noteText(s string) string {
	s = html.UnescapeString(htmlTag.ReplaceAllString(s, " "))
	return strings.Join(strings.Fields(strings.ReplaceAll(s, " ", " ")), " ")
}
