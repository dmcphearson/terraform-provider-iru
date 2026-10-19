package provider

import (
	"context"
	"fmt"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/list"
	listschema "github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// List resources back `terraform query`: each lists the existing objects of one
// managed resource type so they can be imported in bulk, with generated config.

var (
	_ list.ListResource              = &iruListResource{}
	_ list.ListResourceWithConfigure = &iruListResource{}
)

// listEntry is one listed object: its display name, its identity model, and a
// loader for its full resource model, called only when Terraform asks for it.
type listEntry struct {
	displayName string
	identity    any
	load        func(ctx context.Context) (any, error)
}

type iruListResource struct {
	client *client.Client
	// typeName is the managed resource name without the provider prefix.
	typeName string
	// filterAttr is the optional config attribute that narrows the listing.
	filterAttr string
	filterDesc string
	entries    func(ctx context.Context, c *client.Client, filter string) ([]listEntry, error)
}

func (r *iruListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.typeName
}

func (r *iruListResource) ListResourceConfigSchema(ctx context.Context, req list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = listschema.Schema{
		MarkdownDescription: fmt.Sprintf("Lists existing `iru_%s` objects for `terraform query`.", r.typeName),
		Attributes: map[string]listschema.Attribute{
			r.filterAttr: listschema.StringAttribute{
				Optional:            true,
				MarkdownDescription: r.filterDesc,
			},
		},
	}
}

func (r *iruListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, got %T.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *iruListResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	var filter types.String
	if diags := req.Config.GetAttribute(ctx, path.Root(r.filterAttr), &filter); diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}
	entries, err := r.entries(ctx, r.client, filter.ValueString())
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError(fmt.Sprintf("Error listing iru_%s", r.typeName), err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}
	stream.Results = func(push func(list.ListResult) bool) {
		for i, e := range entries {
			if req.Limit > 0 && int64(i) >= req.Limit {
				return
			}
			result := req.NewListResult(ctx)
			result.DisplayName = e.displayName
			result.Diagnostics.Append(result.Identity.Set(ctx, e.identity)...)
			if req.IncludeResource {
				m, err := e.load(ctx)
				if err != nil {
					result.Diagnostics.AddError(fmt.Sprintf("Error reading iru_%s %q", r.typeName, e.displayName), err.Error())
				} else {
					result.Diagnostics.Append(result.Resource.Set(ctx, m)...)
				}
			}
			if !push(result) {
				return
			}
		}
	}
}

func nameFilterDesc(kind string) string {
	return fmt.Sprintf("Only list %s with exactly this name.", kind)
}

// byID builds entries for a resource identified by a single id, keeping the
// items whose name matches filter when one is set.
func byID[T any](items []T, filter string, name, id func(T) string, load func(ctx context.Context, id string) (any, error)) []listEntry {
	var out []listEntry
	for _, it := range items {
		if filter != "" && name(it) != filter {
			continue
		}
		itemID := id(it)
		out = append(out, listEntry{
			displayName: name(it),
			identity:    idIdentityModel{ID: types.StringValue(itemID)},
			load:        func(ctx context.Context) (any, error) { return load(ctx, itemID) },
		})
	}
	return out
}

func NewTagListResource() list.ListResource {
	return &iruListResource{
		typeName: "tag", filterAttr: "name", filterDesc: nameFilterDesc("tags"),
		entries: func(ctx context.Context, c *client.Client, filter string) ([]listEntry, error) {
			tags, err := c.ListTags(ctx)
			if err != nil {
				return nil, err
			}
			// Tags have no GET-by-id and the list item is the whole object.
			var out []listEntry
			for _, t := range tags {
				if filter != "" && t.Name != filter {
					continue
				}
				m := &tagModel{ID: types.StringValue(t.ID), Name: types.StringValue(t.Name)}
				out = append(out, listEntry{
					displayName: t.Name,
					identity:    idIdentityModel{ID: m.ID},
					load:        func(context.Context) (any, error) { return m, nil },
				})
			}
			return out, nil
		},
	}
}

func NewCustomScriptListResource() list.ListResource {
	return &iruListResource{
		typeName: "custom_script", filterAttr: "name", filterDesc: nameFilterDesc("custom scripts"),
		entries: func(ctx context.Context, c *client.Client, filter string) ([]listEntry, error) {
			items, err := c.ListCustomScripts(ctx)
			if err != nil {
				return nil, err
			}
			return byID(items, filter,
				func(s client.CustomScript) string { return s.Name },
				func(s client.CustomScript) string { return s.ID },
				func(ctx context.Context, id string) (any, error) {
					out, err := c.GetCustomScript(ctx, id)
					if err != nil {
						return nil, err
					}
					var m customScriptModel
					applyCustomScriptResponse(&m, out, customScriptModel{})
					return &m, nil
				}), nil
		},
	}
}

func NewCustomProfileListResource() list.ListResource {
	return &iruListResource{
		typeName: "custom_profile", filterAttr: "name", filterDesc: nameFilterDesc("custom profiles"),
		entries: func(ctx context.Context, c *client.Client, filter string) ([]listEntry, error) {
			items, err := c.ListCustomProfiles(ctx)
			if err != nil {
				return nil, err
			}
			return byID(items, filter,
				func(p client.CustomProfile) string { return p.Name },
				func(p client.CustomProfile) string { return p.ID },
				func(ctx context.Context, id string) (any, error) {
					out, err := c.GetCustomProfile(ctx, id)
					if err != nil {
						return nil, err
					}
					var m customProfileModel
					applyProfileResponse(&m, out)
					// Generated config needs profile_file; use the profile Iru holds.
					m.ProfileFile = types.StringValue(out.Profile)
					return &m, nil
				}), nil
		},
	}
}

func NewCustomAppListResource() list.ListResource {
	return &iruListResource{
		typeName: "custom_app", filterAttr: "name", filterDesc: nameFilterDesc("custom apps"),
		entries: func(ctx context.Context, c *client.Client, filter string) ([]listEntry, error) {
			items, err := c.ListCustomApps(ctx)
			if err != nil {
				return nil, err
			}
			return byID(items, filter,
				func(a client.CustomApp) string { return a.Name },
				func(a client.CustomApp) string { return a.ID },
				func(ctx context.Context, id string) (any, error) {
					out, err := c.GetCustomApp(ctx, id)
					if err != nil {
						return nil, err
					}
					var m customAppModel
					applyAppResponse(&m, out, customAppModel{})
					return &m, nil
				}), nil
		},
	}
}

func NewInHouseAppListResource() list.ListResource {
	return &iruListResource{
		typeName: "in_house_app", filterAttr: "name", filterDesc: nameFilterDesc("in-house apps"),
		entries: func(ctx context.Context, c *client.Client, filter string) ([]listEntry, error) {
			items, err := c.ListInHouseApps(ctx)
			if err != nil {
				return nil, err
			}
			return byID(items, filter,
				func(a client.InHouseApp) string { return a.Name },
				func(a client.InHouseApp) string { return a.ID },
				func(ctx context.Context, id string) (any, error) {
					out, err := c.GetInHouseApp(ctx, id)
					if err != nil {
						return nil, err
					}
					var m inHouseAppModel
					applyInHouseAppResponse(&m, out)
					return &m, nil
				}), nil
		},
	}
}

func NewBlueprintListResource() list.ListResource {
	return &iruListResource{
		typeName: "blueprint", filterAttr: "name", filterDesc: nameFilterDesc("blueprints"),
		entries: func(ctx context.Context, c *client.Client, filter string) ([]listEntry, error) {
			items, err := c.ListBlueprints(ctx)
			if err != nil {
				return nil, err
			}
			return byID(items, filter,
				func(b client.Blueprint) string { return b.Name },
				func(b client.Blueprint) string { return b.ID },
				func(ctx context.Context, id string) (any, error) {
					out, err := c.GetBlueprint(ctx, id)
					if err != nil {
						return nil, err
					}
					var m blueprintModel
					if diags := applyBlueprint(&m, out); diags.HasError() {
						return nil, fmt.Errorf("%v", diags)
					}
					return &m, nil
				}), nil
		},
	}
}

// NewBlueprintAssignmentListResource lists every library item assigned to each
// blueprint. Node IDs on map blueprints can't be read back, so results carry
// blueprint_id and library_item_id only.
func NewBlueprintAssignmentListResource() list.ListResource {
	return &iruListResource{
		typeName: "blueprint_assignment", filterAttr: "blueprint_id",
		filterDesc: "Only list assignments on the blueprint with this ID.",
		entries: func(ctx context.Context, c *client.Client, filter string) ([]listEntry, error) {
			blueprints, err := c.ListBlueprints(ctx)
			if err != nil {
				return nil, err
			}
			var out []listEntry
			for _, b := range blueprints {
				if filter != "" && b.ID != filter {
					continue
				}
				items, err := c.ListBlueprintLibraryItems(ctx, b.ID)
				if err != nil {
					return nil, err
				}
				for _, it := range items {
					m := &blueprintAssignmentModel{
						BlueprintID:      types.StringValue(b.ID),
						LibraryItemID:    types.StringValue(it.ID),
						AssignmentNodeID: types.StringNull(),
					}
					m.ID = types.StringValue(assignmentID(*m))
					out = append(out, listEntry{
						displayName: b.Name + " / " + it.Name,
						identity: blueprintAssignmentIdentityModel{
							BlueprintID: m.BlueprintID, LibraryItemID: m.LibraryItemID, AssignmentNodeID: m.AssignmentNodeID,
						},
						load: func(context.Context) (any, error) { return m, nil },
					})
				}
			}
			return out, nil
		},
	}
}

// NewDeviceListResource lists enrolled devices. The list endpoint returns the
// same fields as GET, so results are built without a request per device.
func NewDeviceListResource() list.ListResource {
	return &iruListResource{
		typeName: "device", filterAttr: "blueprint_id",
		filterDesc: "Only list devices on the blueprint with this ID.",
		entries: func(ctx context.Context, c *client.Client, filter string) ([]listEntry, error) {
			devices, err := c.ListDevices(ctx, client.DeviceFilter{BlueprintID: filter})
			if err != nil {
				return nil, err
			}
			out := make([]listEntry, 0, len(devices))
			for i := range devices {
				d := devices[i]
				name := d.Name
				if d.SerialNumber != "" {
					name += " (" + d.SerialNumber + ")"
				}
				out = append(out, listEntry{
					displayName: name,
					identity:    idIdentityModel{ID: types.StringValue(d.ID)},
					load: func(ctx context.Context) (any, error) {
						var m deviceModel
						if diags := applyDevice(ctx, &m, &d); diags.HasError() {
							return nil, fmt.Errorf("%v", diags)
						}
						return &m, nil
					},
				})
			}
			return out, nil
		},
	}
}

// NewDeviceNoteListResource lists device notes. Without device_id it reads the
// notes of every device, one request per device.
func NewDeviceNoteListResource() list.ListResource {
	return &iruListResource{
		typeName: "device_note", filterAttr: "device_id",
		filterDesc: "Only list notes on the device with this ID. Recommended on large tenants.",
		entries: func(ctx context.Context, c *client.Client, filter string) ([]listEntry, error) {
			devices := []client.Device{{ID: filter}}
			if filter == "" {
				var err error
				if devices, err = c.ListDevices(ctx, client.DeviceFilter{}); err != nil {
					return nil, err
				}
			}
			var out []listEntry
			for _, d := range devices {
				notes, err := c.ListDeviceNotes(ctx, d.ID)
				if err != nil {
					return nil, err
				}
				for i := range notes {
					m := &deviceNoteModel{DeviceID: types.StringValue(d.ID), Content: types.StringNull()}
					applyDeviceNote(m, &notes[i])
					name := noteText(notes[i].Content)
					if r := []rune(name); len(r) > 40 {
						name = string(r[:40]) + "..."
					}
					if d.Name != "" {
						name = d.Name + ": " + name
					}
					out = append(out, listEntry{
						displayName: name,
						identity:    deviceNoteIdentityModel{DeviceID: m.DeviceID, NoteID: m.NoteID},
						load:        func(context.Context) (any, error) { return m, nil },
					})
				}
			}
			return out, nil
		},
	}
}

// NewBlueprintRoutingListResource lists the tenant's one routing setting.
func NewBlueprintRoutingListResource() list.ListResource {
	return &iruListResource{
		typeName: "blueprint_routing", filterAttr: "id",
		filterDesc: "Unused: every tenant has exactly one routing setting.",
		entries: func(ctx context.Context, c *client.Client, _ string) ([]listEntry, error) {
			out, err := c.GetBlueprintRouting(ctx)
			if err != nil {
				return nil, err
			}
			m, diags := blueprintRoutingFrom(out)
			if diags.HasError() {
				return nil, fmt.Errorf("%v", diags)
			}
			return []listEntry{{
				displayName: "Blueprint Routing",
				identity:    idIdentityModel{ID: m.ID},
				load:        func(context.Context) (any, error) { return &m, nil },
			}}, nil
		},
	}
}
