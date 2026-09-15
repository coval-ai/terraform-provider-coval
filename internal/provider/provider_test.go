package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	frameworkdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	frameworkprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestProviderSchema(t *testing.T) {
	t.Parallel()

	var response frameworkprovider.SchemaResponse
	(&CovalProvider{}).Schema(context.Background(), frameworkprovider.SchemaRequest{}, &response)

	apiKey, ok := response.Schema.Attributes["api_key"].(providerschema.StringAttribute)
	if !ok {
		t.Fatalf("api_key has type %T, want schema.StringAttribute", response.Schema.Attributes["api_key"])
	}
	if !apiKey.Optional {
		t.Error("api_key must be optional")
	}
	if !apiKey.Sensitive {
		t.Error("api_key must be sensitive")
	}

	apiBaseURL, ok := response.Schema.Attributes["api_base_url"].(providerschema.StringAttribute)
	if !ok {
		t.Fatalf("api_base_url has type %T, want schema.StringAttribute", response.Schema.Attributes["api_base_url"])
	}
	if !apiBaseURL.Optional {
		t.Error("api_base_url must be optional")
	}
}

func TestResolveProviderConfiguration(t *testing.T) {
	t.Run("block values take precedence", func(t *testing.T) {
		t.Setenv("COVAL_API_KEY", "environment-key")
		t.Setenv("COVAL_API_BASE_URL", "https://environment.example/v1")

		configuration, err := resolveProviderConfiguration(covalProviderModel{
			APIKey:     types.StringValue("block-key"),
			APIBaseURL: types.StringValue("https://block.example/v1"),
		})
		if err != nil {
			t.Fatalf("resolve configuration: %v", err)
		}
		if configuration.APIKey != "block-key" {
			t.Errorf("APIKey = %q, want block-key", configuration.APIKey)
		}
		if configuration.APIBaseURL != "https://block.example/v1" {
			t.Errorf("APIBaseURL = %q, want block URL", configuration.APIBaseURL)
		}
	})

	t.Run("environment values are fallbacks", func(t *testing.T) {
		t.Setenv("COVAL_API_KEY", "environment-key")
		t.Setenv("COVAL_API_BASE_URL", "https://environment.example/v1")

		configuration, err := resolveProviderConfiguration(covalProviderModel{
			APIKey:     types.StringNull(),
			APIBaseURL: types.StringNull(),
		})
		if err != nil {
			t.Fatalf("resolve configuration: %v", err)
		}
		if configuration.APIKey != "environment-key" {
			t.Errorf("APIKey = %q, want environment-key", configuration.APIKey)
		}
		if configuration.APIBaseURL != "https://environment.example/v1" {
			t.Errorf("APIBaseURL = %q, want environment URL", configuration.APIBaseURL)
		}
	})

	t.Run("base URL has a final default", func(t *testing.T) {
		t.Setenv("COVAL_API_KEY", "environment-key")
		t.Setenv("COVAL_API_BASE_URL", "")

		configuration, err := resolveProviderConfiguration(covalProviderModel{
			APIKey:     types.StringNull(),
			APIBaseURL: types.StringNull(),
		})
		if err != nil {
			t.Fatalf("resolve configuration: %v", err)
		}
		if configuration.APIBaseURL != client.DefaultBaseURL {
			t.Errorf("APIBaseURL = %q, want %q", configuration.APIBaseURL, client.DefaultBaseURL)
		}
	})

	t.Run("missing API key is rejected", func(t *testing.T) {
		t.Setenv("COVAL_API_KEY", "")
		_, err := resolveProviderConfiguration(covalProviderModel{
			APIKey:     types.StringNull(),
			APIBaseURL: types.StringNull(),
		})
		if err == nil {
			t.Fatal("resolve configuration succeeded without an API key")
		}
	})

	t.Run("empty block value still takes precedence", func(t *testing.T) {
		t.Setenv("COVAL_API_KEY", "environment-key")
		_, err := resolveProviderConfiguration(covalProviderModel{
			APIKey:     types.StringValue(""),
			APIBaseURL: types.StringNull(),
		})
		if err == nil {
			t.Fatal("resolve configuration fell back after an explicit empty block value")
		}
	})
}

func TestProviderRegistersResourceSurfaces(t *testing.T) {
	t.Parallel()

	provider := &CovalProvider{}
	resources := provider.Resources(context.Background())
	if len(resources) != 2 {
		t.Fatalf("Resources() returned %d entries, want 2", len(resources))
	}
	wantResources := map[string]bool{"coval_test_set": true, "coval_test_case": true}
	for _, factory := range resources {
		var response frameworkresource.MetadataResponse
		factory().Metadata(context.Background(), frameworkresource.MetadataRequest{ProviderTypeName: "coval"}, &response)
		delete(wantResources, response.TypeName)
	}
	if len(wantResources) != 0 {
		t.Fatalf("Resources() is missing %v", wantResources)
	}

	dataSources := provider.DataSources(context.Background())
	if len(dataSources) != 4 {
		t.Fatalf("DataSources() returned %d entries, want 4", len(dataSources))
	}
	wantDataSources := map[string]bool{
		"coval_test_set":   true,
		"coval_test_sets":  true,
		"coval_test_case":  true,
		"coval_test_cases": true,
	}
	for _, factory := range dataSources {
		var response frameworkdatasource.MetadataResponse
		factory().Metadata(context.Background(), frameworkdatasource.MetadataRequest{ProviderTypeName: "coval"}, &response)
		delete(wantDataSources, response.TypeName)
	}
	if len(wantDataSources) != 0 {
		t.Fatalf("DataSources() is missing %v", wantDataSources)
	}
}

func TestConfigureCreatesSharedClientFromBlockValues(t *testing.T) {
	t.Setenv("COVAL_API_KEY", "environment-key")
	t.Setenv("COVAL_API_BASE_URL", "https://environment.example/v1")

	requestSeen := make(chan *http.Request, 1)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestSeen <- request
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{}`))
	}))
	defer server.Close()

	provider := &CovalProvider{}
	request := newConfigureRequest(t, "block-key", server.URL+"/v1")
	var response frameworkprovider.ConfigureResponse
	provider.Configure(context.Background(), request, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Configure() diagnostics: %v", response.Diagnostics)
	}

	apiClient, ok := response.ResourceData.(*client.Client)
	if !ok {
		t.Fatalf("ResourceData = %T, want *client.Client", response.ResourceData)
	}
	if response.DataSourceData != response.ResourceData {
		t.Fatal("resource and data source provider data do not share a client")
	}
	if err := apiClient.Do(context.Background(), http.MethodGet, "probe", nil, nil); err != nil {
		t.Fatalf("configured client Do(): %v", err)
	}

	seen := <-requestSeen
	if seen.URL.Path != "/v1/probe" {
		t.Errorf("request path = %q, want /v1/probe", seen.URL.Path)
	}
	if got := seen.Header.Get("x-api-key"); got != "block-key" {
		t.Errorf("x-api-key = %q, want block-key", got)
	}
}

func TestConfigureReportsMissingAPIKey(t *testing.T) {
	t.Setenv("COVAL_API_KEY", "")
	provider := &CovalProvider{}
	request := newConfigureRequest(t, nil, nil)
	var response frameworkprovider.ConfigureResponse
	provider.Configure(context.Background(), request, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("Configure() succeeded without an API key")
	}
	if response.ResourceData != nil || response.DataSourceData != nil {
		t.Fatal("Configure() returned provider data after a configuration error")
	}
}

func TestConfigureReportsInvalidBaseURL(t *testing.T) {
	provider := &CovalProvider{}
	request := newConfigureRequest(t, "block-key", "http://api.coval.dev/v1")
	var response frameworkprovider.ConfigureResponse
	provider.Configure(context.Background(), request, &response)
	if !response.Diagnostics.HasError() {
		t.Fatal("Configure() accepted a remote HTTP base URL")
	}
	if response.ResourceData != nil || response.DataSourceData != nil {
		t.Fatal("Configure() returned provider data after a configuration error")
	}
}

func newConfigureRequest(t *testing.T, apiKey any, apiBaseURL any) frameworkprovider.ConfigureRequest {
	t.Helper()
	ctx := context.Background()
	var schemaResponse frameworkprovider.SchemaResponse
	(&CovalProvider{}).Schema(ctx, frameworkprovider.SchemaRequest{}, &schemaResponse)
	terraformType := schemaResponse.Schema.Type().TerraformType(ctx)
	return frameworkprovider.ConfigureRequest{
		Config: tfsdk.Config{
			Raw: tftypes.NewValue(terraformType, map[string]tftypes.Value{
				"api_key":      tftypes.NewValue(tftypes.String, apiKey),
				"api_base_url": tftypes.NewValue(tftypes.String, apiBaseURL),
			}),
			Schema: schemaResponse.Schema,
		},
	}
}
