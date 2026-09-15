package provider

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &CovalProvider{}

// CovalProvider configures access to the Coval public API.
type CovalProvider struct {
	version string
}

type covalProviderModel struct {
	APIKey     types.String `tfsdk:"api_key"`
	APIBaseURL types.String `tfsdk:"api_base_url"`
}

type providerConfiguration struct {
	APIKey     string
	APIBaseURL string
}

func (p *CovalProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "coval"
	resp.Version = p.version
}

func (p *CovalProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Coval configuration resources through the public API.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Coval API key. May also be set with the `COVAL_API_KEY` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"api_base_url": schema.StringAttribute{
				MarkdownDescription: "Base URL for the Coval API. May also be set with `COVAL_API_BASE_URL`. Defaults to `https://api.coval.dev/v1`.",
				Optional:            true,
			},
		},
	}
}

func (p *CovalProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data covalProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown Coval API Key",
			"The provider cannot create a Coval API client because api_key is unknown. Set api_key to a known value or use COVAL_API_KEY.",
		)
	}
	if data.APIBaseURL.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_base_url"),
			"Unknown Coval API Base URL",
			"The provider cannot create a Coval API client because api_base_url is unknown. Set api_base_url to a known value, use COVAL_API_BASE_URL, or omit it to use the default.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	configuration, err := resolveProviderConfiguration(data)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Coval API Key",
			"Set api_key in the provider configuration or set the COVAL_API_KEY environment variable.",
		)
		return
	}

	apiClient, err := client.New(configuration.APIKey, configuration.APIBaseURL)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_base_url"),
			"Invalid Coval API Base URL",
			err.Error(),
		)
		return
	}

	resp.DataSourceData = apiClient
	resp.ResourceData = apiClient
}

func resolveProviderConfiguration(data covalProviderModel) (providerConfiguration, error) {
	apiKey := configuredValue(data.APIKey, "COVAL_API_KEY", "")
	if strings.TrimSpace(apiKey) == "" {
		return providerConfiguration{}, errors.New("the Coval API key must not be empty")
	}
	return providerConfiguration{
		APIKey:     apiKey,
		APIBaseURL: configuredValue(data.APIBaseURL, "COVAL_API_BASE_URL", client.DefaultBaseURL),
	}, nil
}

func configuredValue(value types.String, environmentVariable string, fallback string) string {
	if !value.IsNull() {
		return value.ValueString()
	}
	if environmentValue := os.Getenv(environmentVariable); environmentValue != "" {
		return environmentValue
	}
	return fallback
}

func (p *CovalProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		newTestSetResource,
		newTestCaseResource,
	}
}

func (p *CovalProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		newTestSetDataSource,
		newTestSetsDataSource,
		newTestCaseDataSource,
		newTestCasesDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CovalProvider{version: version}
	}
}
