package provider

import (
	"context"
	"reflect"
	"testing"

	frameworkdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func TestWorkspaceScopeIsAvailableAcrossManagedResources(t *testing.T) {
	t.Parallel()

	resources := []struct {
		name    string
		factory func() frameworkresource.Resource
	}{
		{name: "agent", factory: newAgentResource},
		{name: "metric", factory: newMetricResource},
		{name: "persona", factory: newPersonaResource},
		{name: "test_case", factory: newTestCaseResource},
		{name: "test_set", factory: newTestSetResource},
		{name: "alert", factory: newAlertResource},
	}

	for _, test := range resources {
		t.Run(test.name, func(t *testing.T) {
			instance := test.factory()
			var schemaResponse frameworkresource.SchemaResponse
			instance.Schema(context.Background(), frameworkresource.SchemaRequest{}, &schemaResponse)
			attribute, ok := schemaResponse.Schema.Attributes["workspace_id"].(resourceschema.StringAttribute)
			if !ok || !attribute.Optional {
				t.Fatalf("workspace_id must be an optional string: %#v", schemaResponse.Schema.Attributes["workspace_id"])
			}
			if len(attribute.PlanModifiers) != 1 || reflect.TypeOf(attribute.PlanModifiers[0]) != reflect.TypeOf(stringplanmodifier.RequiresReplace()) {
				t.Fatalf("workspace_id must require replacement: %#v", attribute.PlanModifiers)
			}

			identityResource, ok := instance.(frameworkresource.ResourceWithIdentity)
			if !ok {
				t.Fatal("resource does not implement ResourceWithIdentity")
			}
			var identityResponse frameworkresource.IdentitySchemaResponse
			identityResource.IdentitySchema(context.Background(), frameworkresource.IdentitySchemaRequest{}, &identityResponse)
			identity, ok := identityResponse.IdentitySchema.Attributes["workspace_id"].(identityschema.StringAttribute)
			if !ok || !identity.OptionalForImport {
				t.Fatalf("workspace_id must be optional for identity import: %#v", identityResponse.IdentitySchema.Attributes["workspace_id"])
			}
		})
	}
}

func TestWorkspaceScopeIsAvailableAcrossResourceDataSources(t *testing.T) {
	t.Parallel()

	dataSources := []struct {
		name    string
		factory func() frameworkdatasource.DataSource
	}{
		{name: "agent", factory: newAgentDataSource},
		{name: "agents", factory: newAgentsDataSource},
		{name: "metric", factory: newMetricDataSource},
		{name: "metrics", factory: newMetricsDataSource},
		{name: "persona", factory: newPersonaDataSource},
		{name: "personas", factory: newPersonasDataSource},
		{name: "test_case", factory: newTestCaseDataSource},
		{name: "test_cases", factory: newTestCasesDataSource},
		{name: "test_set", factory: newTestSetDataSource},
		{name: "test_sets", factory: newTestSetsDataSource},
		{name: "alert", factory: newAlertDataSource},
		{name: "alerts", factory: newAlertsDataSource},
	}

	for _, test := range dataSources {
		t.Run(test.name, func(t *testing.T) {
			var response frameworkdatasource.SchemaResponse
			test.factory().Schema(context.Background(), frameworkdatasource.SchemaRequest{}, &response)
			attribute, ok := response.Schema.Attributes["workspace_id"].(datasourceschema.StringAttribute)
			if !ok || !attribute.Optional {
				t.Fatalf("workspace_id must be an optional string: %#v", response.Schema.Attributes["workspace_id"])
			}
		})
	}
}
