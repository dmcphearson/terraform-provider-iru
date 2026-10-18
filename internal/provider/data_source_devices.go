package provider

import (
	"context"
	"fmt"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &devicesDataSource{}
	_ datasource.DataSourceWithConfigure = &devicesDataSource{}
	_ datasource.DataSource              = &deviceDataSource{}
	_ datasource.DataSourceWithConfigure = &deviceDataSource{}
	_ datasource.DataSource              = &deviceDetailsDataSource{}
	_ datasource.DataSourceWithConfigure = &deviceDetailsDataSource{}
)

// deviceFields are the attributes shared by iru_device and iru_devices results.
type deviceFields struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Model           types.String `tfsdk:"model"`
	SerialNumber    types.String `tfsdk:"serial_number"`
	Platform        types.String `tfsdk:"platform"`
	OSVersion       types.String `tfsdk:"os_version"`
	LastCheckIn     types.String `tfsdk:"last_check_in"`
	UserID          types.String `tfsdk:"user_id"`
	UserEmail       types.String `tfsdk:"user_email"`
	UserName        types.String `tfsdk:"user_name"`
	AssetTag        types.String `tfsdk:"asset_tag"`
	BlueprintID     types.String `tfsdk:"blueprint_id"`
	BlueprintName   types.String `tfsdk:"blueprint_name"`
	MDMEnabled      types.Bool   `tfsdk:"mdm_enabled"`
	AgentInstalled  types.Bool   `tfsdk:"agent_installed"`
	AgentVersion    types.String `tfsdk:"agent_version"`
	IsMissing       types.Bool   `tfsdk:"is_missing"`
	IsRemoved       types.Bool   `tfsdk:"is_removed"`
	FirstEnrollment types.String `tfsdk:"first_enrollment"`
	LastEnrollment  types.String `tfsdk:"last_enrollment"`
	Tags            []string     `tfsdk:"tags"`
}

func deviceFieldsFrom(d client.Device) deviceFields {
	f := deviceFields{
		ID:              types.StringValue(d.ID),
		Name:            types.StringValue(d.Name),
		Model:           types.StringValue(d.Model),
		SerialNumber:    types.StringValue(d.SerialNumber),
		Platform:        types.StringValue(d.Platform),
		OSVersion:       types.StringValue(d.OSVersion),
		LastCheckIn:     types.StringValue(d.LastCheckIn),
		UserID:          types.StringNull(),
		UserEmail:       types.StringNull(),
		UserName:        types.StringNull(),
		AssetTag:        types.StringValue(d.AssetTag),
		BlueprintID:     types.StringValue(d.BlueprintID),
		BlueprintName:   types.StringValue(d.BlueprintName),
		MDMEnabled:      types.BoolValue(d.MDMEnabled),
		AgentInstalled:  types.BoolValue(d.AgentInstalled),
		AgentVersion:    types.StringValue(d.AgentVersion),
		IsMissing:       types.BoolValue(d.IsMissing),
		IsRemoved:       types.BoolValue(d.IsRemoved),
		FirstEnrollment: types.StringValue(d.FirstEnrollment),
		LastEnrollment:  types.StringValue(d.LastEnrollment),
		Tags:            d.Tags,
	}
	if f.Tags == nil {
		f.Tags = []string{}
	}
	if d.User != nil {
		f.UserID = types.StringValue(d.User.ID)
		f.UserEmail = types.StringValue(d.User.Email)
		f.UserName = types.StringValue(d.User.Name)
	}
	return f
}

// deviceAttributes returns the computed device attributes, leaving out the ones
// named in skip so a data source can declare them itself.
func deviceAttributes(skip ...string) map[string]schema.Attribute {
	str := func(desc string) schema.Attribute {
		return schema.StringAttribute{Computed: true, MarkdownDescription: desc}
	}
	boolean := func(desc string) schema.Attribute {
		return schema.BoolAttribute{Computed: true, MarkdownDescription: desc}
	}
	attrs := map[string]schema.Attribute{
		"id":               str("Device ID."),
		"name":             str("Device name."),
		"model":            str("Hardware model."),
		"serial_number":    str("Serial number."),
		"platform":         str("`Mac`, `iPhone`, `iPad` or `AppleTV`."),
		"os_version":       str("OS version."),
		"last_check_in":    str("Last check-in time."),
		"user_id":          str("Assigned user's ID; null when unassigned."),
		"user_email":       str("Assigned user's email; null when unassigned."),
		"user_name":        str("Assigned user's name; null when unassigned."),
		"asset_tag":        str("Asset tag."),
		"blueprint_id":     str("Blueprint ID."),
		"blueprint_name":   str("Blueprint name."),
		"mdm_enabled":      boolean("Whether MDM is enabled."),
		"agent_installed":  boolean("Whether the Iru agent is installed."),
		"agent_version":    str("Iru agent version."),
		"is_missing":       boolean("Whether the device is marked missing."),
		"is_removed":       boolean("Whether the device has been removed."),
		"first_enrollment": str("First enrollment time."),
		"last_enrollment":  str("Last enrollment time."),
		"tags": schema.ListAttribute{
			Computed: true, ElementType: types.StringType, MarkdownDescription: "Tag names.",
		},
	}
	for _, k := range skip {
		delete(attrs, k)
	}
	return attrs
}

func configureClient(providerData any, diags interface {
	AddError(string, string)
}) *client.Client {
	if providerData == nil {
		return nil
	}
	c, ok := providerData.(*client.Client)
	if !ok {
		diags.AddError("Unexpected provider data", fmt.Sprintf("expected *client.Client, got %T", providerData))
		return nil
	}
	return c
}

// iru_devices

func NewDevicesDataSource() datasource.DataSource { return &devicesDataSource{} }

type devicesDataSource struct{ client *client.Client }

type devicesModel struct {
	BlueprintID  types.String   `tfsdk:"blueprint_id"`
	Platform     types.String   `tfsdk:"platform"`
	SerialNumber types.String   `tfsdk:"serial_number"`
	DeviceName   types.String   `tfsdk:"device_name"`
	UserEmail    types.String   `tfsdk:"user_email"`
	AssetTag     types.String   `tfsdk:"asset_tag"`
	Results      []deviceFields `tfsdk:"results"`
}

func (d *devicesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_devices"
}

func (d *devicesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	filter := func(desc string) schema.Attribute {
		return schema.StringAttribute{Optional: true, MarkdownDescription: desc}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up Iru devices. Filters are passed to the API; " +
			"`serial_number`, `device_name` and `user_email` match by substring.",
		Attributes: map[string]schema.Attribute{
			"blueprint_id": filter("Only devices on this blueprint."),
			"platform": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Only devices on this platform: `Mac`, `iPhone`, `iPad` or `AppleTV`.",
				Validators:          []validator.String{stringvalidator.OneOf("Mac", "iPhone", "iPad", "AppleTV")},
			},
			"serial_number": filter("Devices whose serial number contains this value."),
			"device_name":   filter("Devices whose name contains this value."),
			"user_email":    filter("Devices whose assigned user's email contains this value."),
			"asset_tag":     filter("Only devices with this asset tag."),
			"results": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Matching devices.",
				NestedObject:        schema.NestedAttributeObject{Attributes: deviceAttributes()},
			},
		},
	}
}

func (d *devicesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if c := configureClient(req.ProviderData, &resp.Diagnostics); c != nil {
		d.client = c
	}
}

func (d *devicesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config devicesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	devices, err := d.client.ListDevices(ctx, client.DeviceFilter{
		BlueprintID:  config.BlueprintID.ValueString(),
		Platform:     config.Platform.ValueString(),
		SerialNumber: config.SerialNumber.ValueString(),
		DeviceName:   config.DeviceName.ValueString(),
		UserEmail:    config.UserEmail.ValueString(),
		AssetTag:     config.AssetTag.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error reading devices", err.Error())
		return
	}
	config.Results = make([]deviceFields, 0, len(devices))
	for _, dev := range devices {
		config.Results = append(config.Results, deviceFieldsFrom(dev))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// iru_device

func NewDeviceDataSource() datasource.DataSource { return &deviceDataSource{} }

type deviceDataSource struct{ client *client.Client }

func (d *deviceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

func (d *deviceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := deviceAttributes()
	attrs["id"] = schema.StringAttribute{
		Optional: true, Computed: true,
		MarkdownDescription: "Device ID. Set this or `serial_number`.",
		Validators:          []validator.String{stringvalidator.ExactlyOneOf(path.MatchRoot("serial_number"))},
	}
	attrs["serial_number"] = schema.StringAttribute{
		Optional: true, Computed: true,
		MarkdownDescription: "Exact serial number. Set this or `id`.",
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up one Iru device by ID or exact serial number.",
		Attributes:          attrs,
	}
}

func (d *deviceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if c := configureClient(req.ProviderData, &resp.Diagnostics); c != nil {
		d.client = c
	}
}

func (d *deviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config deviceFields
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var dev *client.Device
	if id := config.ID.ValueString(); id != "" {
		out, err := d.client.GetDevice(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Error reading device", err.Error())
			return
		}
		dev = out
	} else {
		serial := config.SerialNumber.ValueString()
		// The API matches serial_number by substring; keep exact matches only.
		devices, err := d.client.ListDevices(ctx, client.DeviceFilter{SerialNumber: serial})
		if err != nil {
			resp.Diagnostics.AddError("Error reading device", err.Error())
			return
		}
		for i := range devices {
			if devices[i].SerialNumber == serial {
				dev = &devices[i]
				break
			}
		}
		if dev == nil {
			resp.Diagnostics.AddAttributeError(path.Root("serial_number"), "Device not found",
				fmt.Sprintf("No device has serial number %q.", serial))
			return
		}
	}
	state := deviceFieldsFrom(*dev)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// iru_device_details

func NewDeviceDetailsDataSource() datasource.DataSource { return &deviceDetailsDataSource{} }

type deviceDetailsDataSource struct{ client *client.Client }

type deviceDetailsModel struct {
	ID   types.String `tfsdk:"id"`
	JSON types.String `tfsdk:"json"`
}

func (d *deviceDetailsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_details"
}

func (d *deviceDetailsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Read a device's full detail record (hardware, MDM, FileVault, " +
			"security and more). Its shape varies by platform and enabled features, so it is " +
			"returned as JSON; read fields with `jsondecode()`.",
		Attributes: map[string]schema.Attribute{
			"id":   schema.StringAttribute{Required: true, MarkdownDescription: "Device ID."},
			"json": schema.StringAttribute{Computed: true, MarkdownDescription: "The detail record as JSON."},
		},
	}
}

func (d *deviceDetailsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if c := configureClient(req.ProviderData, &resp.Diagnostics); c != nil {
		d.client = c
	}
}

func (d *deviceDetailsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config deviceDetailsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	raw, err := d.client.GetDeviceDetails(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading device details", err.Error())
		return
	}
	config.JSON = types.StringValue(string(raw))
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
