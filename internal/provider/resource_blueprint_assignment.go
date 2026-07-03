package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &blueprintAssignmentResource{}
	_ resource.ResourceWithConfigure   = &blueprintAssignmentResource{}
	_ resource.ResourceWithImportState = &blueprintAssignmentResource{}
)

func NewBlueprintAssignmentResource() resource.Resource { return &blueprintAssignmentResource{} }

type blueprintAssignmentResource struct {
	client *client.Client
}

type blueprintAssignmentModel struct {
	ID               types.String `tfsdk:"id"`
	BlueprintID      types.String `tfsdk:"blueprint_id"`
	LibraryItemID    types.String `tfsdk:"library_item_id"`
	AssignmentNodeID types.String `tfsdk:"assignment_node_id"`
}

func (r *blueprintAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprint_assignment"
}

func (r *blueprintAssignmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Assigns a library item to an Iru Blueprint. This is an association " +
			"resource: it manages one (blueprint, library item[, assignment node]) attachment. " +
			"Any attribute change forces a new assignment.\n\n" +
			"**Map blueprint limitation:** the Iru API does not expose a map blueprint's node " +
			"structure, so node IDs cannot be read back. If `assignment_node_id` is omitted, the " +
			"item is placed on the map's default node; you can move it to a specific node in the " +
			"Iru UI and it will not drift (Read only verifies the item is attached to the " +
			"blueprint, not which node). Destroying/recreating the assignment resets it to the " +
			"default node.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Composite identifier `blueprint_id:library_item_id[:assignment_node_id]`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"blueprint_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the blueprint.",
				PlanModifiers:       replace,
			},
			"library_item_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the library item to assign.",
				PlanModifiers:       replace,
			},
			"assignment_node_id": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Assignment node ID. Required for map blueprints that use " +
					"conditional logic; must be omitted for classic blueprints.",
				PlanModifiers: replace,
			},
		},
	}
}

func (r *blueprintAssignmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func assignmentID(m blueprintAssignmentModel) string {
	id := m.BlueprintID.ValueString() + ":" + m.LibraryItemID.ValueString()
	if m.AssignmentNodeID.ValueString() != "" {
		id += ":" + m.AssignmentNodeID.ValueString()
	}
	return id
}

func (r *blueprintAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan blueprintAssignmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.AssignLibraryItem(ctx, plan.BlueprintID.ValueString(),
		plan.LibraryItemID.ValueString(), plan.AssignmentNodeID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error assigning library item to blueprint", err.Error())
		return
	}
	plan.ID = types.StringValue(assignmentID(plan))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *blueprintAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state blueprintAssignmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	assigned, err := r.client.IsLibraryItemAssigned(ctx, state.BlueprintID.ValueString(), state.LibraryItemID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading blueprint assignment", err.Error())
		return
	}
	if !assigned {
		resp.State.RemoveResource(ctx)
		return
	}
	state.ID = types.StringValue(assignmentID(state))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update never runs: every attribute is RequiresReplace.
func (r *blueprintAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unexpected update", "blueprint assignments are replace-only")
}

func (r *blueprintAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state blueprintAssignmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	err := r.client.RemoveLibraryItem(ctx, state.BlueprintID.ValueString(),
		state.LibraryItemID.ValueString(), state.AssignmentNodeID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error removing library item from blueprint", err.Error())
	}
}

// ImportState accepts "blueprint_id:library_item_id[:assignment_node_id]".
func (r *blueprintAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) < 2 || len(parts) > 3 {
		resp.Diagnostics.AddError("Invalid import ID",
			"expected blueprint_id:library_item_id[:assignment_node_id], got: "+req.ID)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("blueprint_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("library_item_id"), parts[1])...)
	if len(parts) == 3 {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("assignment_node_id"), parts[2])...)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
