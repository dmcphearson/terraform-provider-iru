package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &usersDataSource{}
	_ datasource.DataSourceWithConfigure = &usersDataSource{}
	_ datasource.DataSource              = &userDataSource{}
	_ datasource.DataSourceWithConfigure = &userDataSource{}
)

// userFields are the attributes shared by iru_user and iru_users results.
type userFields struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Email           types.String `tfsdk:"email"`
	Active          types.Bool   `tfsdk:"active"`
	Archived        types.Bool   `tfsdk:"archived"`
	Department      types.String `tfsdk:"department"`
	JobTitle        types.String `tfsdk:"job_title"`
	DeviceCount     types.Int64  `tfsdk:"device_count"`
	IntegrationID   types.String `tfsdk:"integration_id"`
	IntegrationName types.String `tfsdk:"integration_name"`
	IntegrationType types.String `tfsdk:"integration_type"`
}

func userFieldsFrom(u client.User) userFields {
	f := userFields{
		ID:              types.StringValue(u.ID),
		Name:            types.StringValue(u.Name),
		Email:           types.StringValue(u.Email),
		Active:          types.BoolValue(u.Active),
		Archived:        types.BoolValue(u.Archived),
		Department:      types.StringValue(u.Department),
		JobTitle:        types.StringValue(u.JobTitle),
		DeviceCount:     types.Int64Value(u.DeviceCount),
		IntegrationID:   types.StringNull(),
		IntegrationName: types.StringNull(),
		IntegrationType: types.StringNull(),
	}
	if u.Integration != nil {
		f.IntegrationID = types.StringValue(u.Integration.UUID)
		f.IntegrationName = types.StringValue(u.Integration.Name)
		f.IntegrationType = types.StringValue(u.Integration.Type)
	}
	return f
}

func userAttributes() map[string]schema.Attribute {
	str := func(desc string) schema.Attribute {
		return schema.StringAttribute{Computed: true, MarkdownDescription: desc}
	}
	return map[string]schema.Attribute{
		"id":               str("User ID."),
		"name":             str("Display name."),
		"email":            str("Email address."),
		"active":           schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the user is active."},
		"archived":         schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the user is archived."},
		"department":       str("Department."),
		"job_title":        str("Job title."),
		"device_count":     schema.Int64Attribute{Computed: true, MarkdownDescription: "Number of devices assigned."},
		"integration_id":   str("Directory integration ID (UUID)."),
		"integration_name": str("Directory integration name."),
		"integration_type": str("Directory integration type, e.g. `office365`."),
	}
}

// iru_users

func NewUsersDataSource() datasource.DataSource { return &usersDataSource{} }

type usersDataSource struct{ client *client.Client }

type usersModel struct {
	Email         types.String `tfsdk:"email"`
	IntegrationID types.String `tfsdk:"integration_id"`
	Archived      types.Bool   `tfsdk:"archived"`
	Results       []userFields `tfsdk:"results"`
}

func (d *usersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (d *usersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up users synced from Iru's directory integrations.",
		Attributes: map[string]schema.Attribute{
			"email":          schema.StringAttribute{Optional: true, MarkdownDescription: "Only the user with this email."},
			"integration_id": schema.StringAttribute{Optional: true, MarkdownDescription: "Only users from this directory integration (UUID)."},
			"archived":       schema.BoolAttribute{Optional: true, MarkdownDescription: "Only archived (`true`) or unarchived (`false`) users."},
			"results": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Matching users.",
				NestedObject:        schema.NestedAttributeObject{Attributes: userAttributes()},
			},
		},
	}
}

func (d *usersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if c := configureClient(req.ProviderData, &resp.Diagnostics); c != nil {
		d.client = c
	}
}

func (d *usersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config usersModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	f := client.UserFilter{Email: config.Email.ValueString(), IntegrationID: config.IntegrationID.ValueString()}
	if !config.Archived.IsNull() {
		v := config.Archived.ValueBool()
		f.Archived = &v
	}
	users, err := d.client.ListUsers(ctx, f)
	if err != nil {
		resp.Diagnostics.AddError("Error reading users", err.Error())
		return
	}
	config.Results = make([]userFields, 0, len(users))
	for _, u := range users {
		config.Results = append(config.Results, userFieldsFrom(u))
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

// iru_user

func NewUserDataSource() datasource.DataSource { return &userDataSource{} }

type userDataSource struct{ client *client.Client }

func (d *userDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *userDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := userAttributes()
	attrs["id"] = schema.StringAttribute{
		Optional: true, Computed: true,
		MarkdownDescription: "User ID. Set this or `email`.",
		Validators:          []validator.String{stringvalidator.ExactlyOneOf(path.MatchRoot("email"))},
	}
	attrs["email"] = schema.StringAttribute{
		Optional: true, Computed: true,
		MarkdownDescription: "Email address, matched case-insensitively. Set this or `id`.",
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up one Iru user by ID or email.",
		Attributes:          attrs,
	}
}

func (d *userDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if c := configureClient(req.ProviderData, &resp.Diagnostics); c != nil {
		d.client = c
	}
}

func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config userFields
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var user *client.User
	if id := config.ID.ValueString(); id != "" {
		out, err := d.client.GetUser(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("Error reading user", err.Error())
			return
		}
		user = out
	} else {
		email := config.Email.ValueString()
		users, err := d.client.ListUsers(ctx, client.UserFilter{Email: email})
		if err != nil {
			resp.Diagnostics.AddError("Error reading user", err.Error())
			return
		}
		for i := range users {
			if strings.EqualFold(users[i].Email, email) {
				user = &users[i]
				break
			}
		}
		if user == nil {
			resp.Diagnostics.AddAttributeError(path.Root("email"), "User not found",
				fmt.Sprintf("No user has email %q.", email))
			return
		}
	}
	state := userFieldsFrom(*user)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
