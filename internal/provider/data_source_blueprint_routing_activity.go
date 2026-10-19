package provider

import (
	"context"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &blueprintRoutingActivityDataSource{}
	_ datasource.DataSourceWithConfigure = &blueprintRoutingActivityDataSource{}
)

const defaultRoutingActivityLimit = 100

func NewBlueprintRoutingActivityDataSource() datasource.DataSource {
	return &blueprintRoutingActivityDataSource{}
}

type blueprintRoutingActivityDataSource struct{ client *client.Client }

type blueprintRoutingActivityModel struct {
	Limit   types.Int64             `tfsdk:"limit"`
	Results []routingActivityFields `tfsdk:"results"`
}

type routingActivityFields struct {
	ID           types.Int64  `tfsdk:"id"`
	ActivityTime types.String `tfsdk:"activity_time"`
	ActivityType types.String `tfsdk:"activity_type"`
	UserID       types.String `tfsdk:"user_id"`
	UserEmail    types.String `tfsdk:"user_email"`
	DeviceID     types.String `tfsdk:"device_id"`
	Details      types.String `tfsdk:"details"`
}

func (d *blueprintRoutingActivityDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_blueprint_routing_activity"
}

func (d *blueprintRoutingActivityDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	str := func(desc string) schema.Attribute {
		return schema.StringAttribute{Computed: true, MarkdownDescription: desc}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Recent Blueprint Routing activity: enrollments, routing decisions and " +
			"configuration changes, newest first.",
		Attributes: map[string]schema.Attribute{
			"limit": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum number of events to return. Defaults to 100.",
				Validators:          []validator.Int64{int64validator.Between(1, 10000)},
			},
			"results": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Activity events.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":            schema.Int64Attribute{Computed: true, MarkdownDescription: "Event ID."},
						"activity_time": str("When the event happened."),
						"activity_type": str("Event type, e.g. `routing_departure`, `computer_enrollment`, `routing_config_edited`."),
						"user_id":       str("Admin who caused the event; null for device events."),
						"user_email":    str("That admin's email; null for device events."),
						"device_id":     str("Device the event is about; null for configuration events."),
						"details":       str("Event details as JSON; the shape depends on `activity_type`. Decode with `jsondecode`."),
					},
				},
			},
		},
	}
}

func (d *blueprintRoutingActivityDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if c := configureClient(req.ProviderData, &resp.Diagnostics); c != nil {
		d.client = c
	}
}

func (d *blueprintRoutingActivityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config blueprintRoutingActivityModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	limit := int64(defaultRoutingActivityLimit)
	if !config.Limit.IsNull() {
		limit = config.Limit.ValueInt64()
	}
	events, err := d.client.ListBlueprintRoutingActivity(ctx, int(limit))
	if err != nil {
		resp.Diagnostics.AddError("Error listing blueprint routing activity", err.Error())
		return
	}
	config.Results = make([]routingActivityFields, 0, len(events))
	for _, e := range events {
		f := routingActivityFields{
			ID:           types.Int64Value(e.ID),
			ActivityTime: types.StringValue(e.ActivityTime),
			ActivityType: types.StringValue(e.ActivityType),
			UserID:       types.StringNull(),
			UserEmail:    types.StringNull(),
			DeviceID:     types.StringPointerValue(e.DeviceID),
			Details:      types.StringValue("{}"),
		}
		if e.User != nil {
			f.UserID = types.StringValue(e.User.ID)
			f.UserEmail = types.StringValue(e.User.Email)
		}
		if len(e.Details) > 0 && string(e.Details) != "null" {
			f.Details = types.StringValue(string(e.Details))
		}
		config.Results = append(config.Results, f)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
