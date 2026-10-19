package provider

import (
	"context"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &blueprintLibraryItemsDataSource{}
	_ datasource.DataSourceWithConfigure = &blueprintLibraryItemsDataSource{}
)

func NewBlueprintLibraryItemsDataSource() datasource.DataSource {
	return &blueprintLibraryItemsDataSource{}
}

type blueprintLibraryItemsDataSource struct{ client *client.Client }

type blueprintLibraryItemsModel struct {
	BlueprintID types.String              `tfsdk:"blueprint_id"`
	Results     []blueprintLibraryItemRef `tfsdk:"results"`
}

type blueprintLibraryItemRef struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (d *blueprintLibraryItemsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprint_library_items"
}

func (d *blueprintLibraryItemsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the library items assigned to a blueprint, including items assigned " +
			"outside Terraform. The API returns only each item's ID and name.",
		Attributes: map[string]schema.Attribute{
			"blueprint_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Blueprint ID.",
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"results": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Assigned library items.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true, MarkdownDescription: "Library item ID."},
						"name": schema.StringAttribute{Computed: true, MarkdownDescription: "Library item name."},
					},
				},
			},
		},
	}
}

func (d *blueprintLibraryItemsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if c := configureClient(req.ProviderData, &resp.Diagnostics); c != nil {
		d.client = c
	}
}

func (d *blueprintLibraryItemsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config blueprintLibraryItemsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, err := d.client.ListBlueprintLibraryItems(ctx, config.BlueprintID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing blueprint library items", err.Error())
		return
	}
	config.Results = make([]blueprintLibraryItemRef, 0, len(items))
	for _, it := range items {
		config.Results = append(config.Results, blueprintLibraryItemRef{ID: types.StringValue(it.ID), Name: types.StringValue(it.Name)})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
