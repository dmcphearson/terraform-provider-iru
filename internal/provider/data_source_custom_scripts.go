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
	_ datasource.DataSource              = &customScriptsDataSource{}
	_ datasource.DataSourceWithConfigure = &customScriptsDataSource{}
)

func NewCustomScriptsDataSource() datasource.DataSource {
	return &customScriptsDataSource{}
}

type customScriptsDataSource struct {
	client *client.Client
}

type customScriptsModel struct {
	Name    types.String               `tfsdk:"name"`
	Results []customScriptSummaryModel `tfsdk:"results"`
}

type customScriptSummaryModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	ExecutionFrequency types.String `tfsdk:"execution_frequency"`
	Active             types.Bool   `tfsdk:"active"`
}

func (d *customScriptsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_scripts"
}

func (d *customScriptsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up Iru custom scripts, optionally filtered by name. Use the returned `id` for `library_item_id` on assignments.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "If set, only custom scripts whose name exactly matches are returned.",
			},
			"results": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Matching custom scripts.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                  schema.StringAttribute{Computed: true, MarkdownDescription: "Library item ID."},
						"name":                schema.StringAttribute{Computed: true, MarkdownDescription: "Script name."},
						"execution_frequency": schema.StringAttribute{Computed: true, MarkdownDescription: "How often the script runs."},
						"active":              schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the script is active."},
					},
				},
			},
		},
	}
}

func (d *customScriptsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *customScriptsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config customScriptsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	items, err := d.client.ListCustomScripts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading custom scripts", err.Error())
		return
	}

	filter := config.Name.ValueString()
	config.Results = make([]customScriptSummaryModel, 0, len(items))
	for _, c := range items {
		if filter != "" && c.Name != filter {
			continue
		}
		config.Results = append(config.Results, customScriptSummaryModel{
			ID:                 types.StringValue(c.ID),
			Name:               types.StringValue(c.Name),
			ExecutionFrequency: types.StringValue(c.ExecutionFrequency),
			Active:             types.BoolValue(c.Active),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
