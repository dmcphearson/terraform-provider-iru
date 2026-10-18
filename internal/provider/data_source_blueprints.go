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
	_ datasource.DataSource              = &blueprintsDataSource{}
	_ datasource.DataSourceWithConfigure = &blueprintsDataSource{}
)

func NewBlueprintsDataSource() datasource.DataSource {
	return &blueprintsDataSource{}
}

type blueprintsDataSource struct {
	client *client.Client
}

type blueprintsModel struct {
	Name    types.String            `tfsdk:"name"`
	Results []blueprintSummaryModel `tfsdk:"results"`
}

type blueprintSummaryModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	Description    types.String `tfsdk:"description"`
	ComputersCount types.Int64  `tfsdk:"computers_count"`
}

func (d *blueprintsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprints"
}

func (d *blueprintsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up Iru blueprints, optionally filtered by name. Use the returned `id` for `blueprint_id` on assignments.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "If set, only blueprints whose name exactly matches are returned.",
			},
			"results": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Matching blueprints.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":              schema.StringAttribute{Computed: true, MarkdownDescription: "Blueprint ID."},
						"name":            schema.StringAttribute{Computed: true, MarkdownDescription: "Blueprint name."},
						"type":            schema.StringAttribute{Computed: true, MarkdownDescription: "Blueprint type, `classic` or `map`."},
						"description":     schema.StringAttribute{Computed: true, MarkdownDescription: "Blueprint description."},
						"computers_count": schema.Int64Attribute{Computed: true, MarkdownDescription: "Number of devices on the blueprint."},
					},
				},
			},
		},
	}
}

func (d *blueprintsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *blueprintsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config blueprintsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	items, err := d.client.ListBlueprints(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading blueprints", err.Error())
		return
	}

	filter := config.Name.ValueString()
	config.Results = make([]blueprintSummaryModel, 0, len(items))
	for _, c := range items {
		if filter != "" && c.Name != filter {
			continue
		}
		config.Results = append(config.Results, blueprintSummaryModel{
			ID:             types.StringValue(c.ID),
			Name:           types.StringValue(c.Name),
			Type:           types.StringValue(c.Type),
			Description:    types.StringValue(c.Description),
			ComputersCount: types.Int64Value(c.ComputersCount),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
