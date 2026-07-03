// Package provider implements the Terraform provider for Iru (Kandji) endpoint
// management, built on the Terraform Plugin Framework (protocol 6.0).
package provider

import (
	"context"
	"os"
	"strings"

	"github.com/dmcphearson/terraform-provider-iru/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure iruProvider satisfies the provider.Provider interface.
var _ provider.Provider = &iruProvider{}

type iruProvider struct {
	version string
}

// New returns a provider constructor bound to a build version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &iruProvider{version: version}
	}
}

type iruProviderModel struct {
	APIURL   types.String `tfsdk:"api_url"`
	APIToken types.String `tfsdk:"api_token"`
}

func (p *iruProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "iru"
	resp.Version = p.version
}

func (p *iruProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Iru (Kandji) endpoint management resources as code. " +
			"Authenticates to a single tenant with a bearer API token.",
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Full tenant API URL, e.g. `https://acme.api.kandji.io`. " +
					"May also be set via the `IRU_API_URL` environment variable.",
			},
			"api_token": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				MarkdownDescription: "Iru API bearer token. May also be set via the " +
					"`IRU_API_TOKEN` environment variable (preferred; keep tokens out of config).",
			},
		},
	}
}

func (p *iruProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config iruProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Config value wins; fall back to environment.
	apiURL := os.Getenv("IRU_API_URL")
	if !config.APIURL.IsNull() && config.APIURL.ValueString() != "" {
		apiURL = config.APIURL.ValueString()
	}
	apiToken := os.Getenv("IRU_API_TOKEN")
	if !config.APIToken.IsNull() && config.APIToken.ValueString() != "" {
		apiToken = config.APIToken.ValueString()
	}

	if apiURL == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_url"),
			"Missing Iru API URL",
			"Set the api_url provider attribute or the IRU_API_URL environment variable "+
				"to your tenant URL, e.g. https://acme.api.kandji.io.",
		)
	} else if !strings.HasPrefix(apiURL, "https://") {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_url"),
			"Invalid Iru API URL",
			"api_url must be an https:// URL, got: "+apiURL,
		)
	}
	if apiToken == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Missing Iru API token",
			"Set the api_token provider attribute or the IRU_API_TOKEN environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	c := client.New(apiURL, apiToken, p.version)
	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *iruProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCustomScriptResource,
		NewTagResource,
		NewCustomProfileResource,
		NewCustomAppResource,
		NewBlueprintResource,
		NewBlueprintAssignmentResource,
	}
}

func (p *iruProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSelfServiceCategoriesDataSource,
	}
}
