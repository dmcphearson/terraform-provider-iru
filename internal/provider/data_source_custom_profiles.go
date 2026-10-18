package provider

import (
	"context"
	"fmt"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &customProfilesDataSource{}
	_ datasource.DataSourceWithConfigure = &customProfilesDataSource{}
)

func NewCustomProfilesDataSource() datasource.DataSource {
	return &customProfilesDataSource{}
}

type customProfilesDataSource struct {
	client *client.Client
}

type customProfilesModel struct {
	Name    types.String                `tfsdk:"name"`
	Results []customProfileSummaryModel `tfsdk:"results"`
}

type customProfileSummaryModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	MDMIdentifier types.String `tfsdk:"mdm_identifier"`
	Active        types.Bool   `tfsdk:"active"`
}

func (d *customProfilesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_profiles"
}

func (d *customProfilesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up Iru custom profiles, optionally filtered by name. Use the returned `id` for `library_item_id` on assignments.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "If set, only custom profiles whose name exactly matches are returned.",
			},
			"results": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Matching custom profiles.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":             schema.StringAttribute{Computed: true, MarkdownDescription: "Library item ID."},
						"name":           schema.StringAttribute{Computed: true, MarkdownDescription: "Profile name."},
						"mdm_identifier": schema.StringAttribute{Computed: true, MarkdownDescription: "MDM payload identifier."},
						"active":         schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the profile is active."},
					},
				},
			},
		},
	}
}

func (d *customProfilesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data",
			fmt.Sprintf("expected *client.Client, got %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *customProfilesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config customProfilesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	items, err := d.client.ListCustomProfiles(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading custom profiles", err.Error())
		return
	}

	filter := config.Name.ValueString()
	config.Results = make([]customProfileSummaryModel, 0, len(items))
	for _, c := range items {
		if filter != "" && c.Name != filter {
			continue
		}
		config.Results = append(config.Results, customProfileSummaryModel{
			ID:            types.StringValue(c.ID),
			Name:          types.StringValue(c.Name),
			MDMIdentifier: types.StringValue(c.MDMIdentifier),
			Active:        types.BoolValue(c.Active),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
