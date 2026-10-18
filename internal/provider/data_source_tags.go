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
	_ datasource.DataSource              = &tagsDataSource{}
	_ datasource.DataSourceWithConfigure = &tagsDataSource{}
)

func NewTagsDataSource() datasource.DataSource {
	return &tagsDataSource{}
}

type tagsDataSource struct {
	client *client.Client
}

type tagsModel struct {
	Name    types.String      `tfsdk:"name"`
	Results []tagSummaryModel `tfsdk:"results"`
}

type tagSummaryModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (d *tagsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tags"
}

func (d *tagsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up Iru tags, optionally filtered by name.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "If set, only tags whose name exactly matches are returned.",
			},
			"results": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Matching tags.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true, MarkdownDescription: "Tag ID."},
						"name": schema.StringAttribute{Computed: true, MarkdownDescription: "Tag name."},
					},
				},
			},
		},
	}
}

func (d *tagsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *tagsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config tagsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	items, err := d.client.ListTags(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading tags", err.Error())
		return
	}

	filter := config.Name.ValueString()
	config.Results = make([]tagSummaryModel, 0, len(items))
	for _, c := range items {
		if filter != "" && c.Name != filter {
			continue
		}
		config.Results = append(config.Results, tagSummaryModel{
			ID:   types.StringValue(c.ID),
			Name: types.StringValue(c.Name),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
