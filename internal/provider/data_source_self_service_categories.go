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
	_ datasource.DataSource              = &selfServiceCategoriesDataSource{}
	_ datasource.DataSourceWithConfigure = &selfServiceCategoriesDataSource{}
)

func NewSelfServiceCategoriesDataSource() datasource.DataSource {
	return &selfServiceCategoriesDataSource{}
}

type selfServiceCategoriesDataSource struct {
	client *client.Client
}

type selfServiceCategoriesModel struct {
	Name    types.String               `tfsdk:"name"`
	Results []selfServiceCategoryModel `tfsdk:"results"`
}

type selfServiceCategoryModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (d *selfServiceCategoriesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_self_service_categories"
}

func (d *selfServiceCategoriesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up Iru Self Service categories, optionally filtered by name. " +
			"Use the returned `id` for `self_service_category_id` on scripts and apps.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "If set, only categories whose name exactly matches are returned.",
			},
			"results": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Matching Self Service categories.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true, MarkdownDescription: "Category ID."},
						"name": schema.StringAttribute{Computed: true, MarkdownDescription: "Category name."},
					},
				},
			},
		},
	}
}

func (d *selfServiceCategoriesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *selfServiceCategoriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config selfServiceCategoriesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cats, err := d.client.ListSelfServiceCategories(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading self service categories", err.Error())
		return
	}

	filter := config.Name.ValueString()
	config.Results = make([]selfServiceCategoryModel, 0, len(cats))
	for _, c := range cats {
		if filter != "" && c.Name != filter {
			continue
		}
		config.Results = append(config.Results, selfServiceCategoryModel{
			ID:   types.StringValue(c.ID),
			Name: types.StringValue(c.Name),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
