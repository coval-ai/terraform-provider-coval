package provider

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestResourceStateDiscardsUpdateTime(t *testing.T) {
	t.Parallel()
	for _, r := range []resource.Resource{newAgentResource(), newPersonaResource(), newMetricResource(), newTestSetResource(), newTestCaseResource(), newRunTemplateResource(), newAlertResource(), newScheduledRunResource()} {
		var metadata resource.MetadataResponse
		r.Metadata(t.Context(), resource.MetadataRequest{ProviderTypeName: "coval"}, &metadata)
		t.Run(metadata.TypeName, func(t *testing.T) {
			var schema resource.SchemaResponse
			r.Schema(t.Context(), resource.SchemaRequest{}, &schema)
			prior := make(map[string]any)
			for name := range schema.Schema.Attributes {
				prior[name] = nil
			}
			prior["create_time"] = "2026-01-01T00:00:00Z"
			prior["update_time"] = "2026-01-02T00:00:00Z"
			raw, err := json.Marshal(prior)
			if err != nil {
				t.Fatal(err)
			}
			server := providerserver.NewProtocol6(New("test")())()
			result, err := server.UpgradeResourceState(t.Context(), &tfprotov6.UpgradeResourceStateRequest{TypeName: metadata.TypeName, Version: 0, RawState: &tfprotov6.RawState{JSON: raw}})
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range result.Diagnostics {
				if d.Severity == tfprotov6.DiagnosticSeverityError {
					t.Fatalf("%s: %s", d.Summary, d.Detail)
				}
			}
			typ := schema.Schema.Type().TerraformType(t.Context())
			value, err := result.UpgradedState.Unmarshal(typ)
			if err != nil {
				t.Fatal(err)
			}
			var attributes map[string]tftypes.Value
			if err := value.As(&attributes); err != nil {
				t.Fatal(err)
			}
			if _, ok := attributes["update_time"]; ok {
				t.Fatal("resource state retains update_time")
			}
			var created string
			if err := attributes["create_time"].As(&created); err != nil {
				t.Fatal(err)
			}
			if created != prior["create_time"] {
				t.Fatalf("create_time = %q", created)
			}
		})
	}
}

func TestDataSourceTimestampModels(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		source datasource.DataSource
		model  any
	}{
		{newAgentDataSource(), &agentDataSourceModel{}},
		{newPersonaDataSource(), &personaDataSourceModel{}},
		{newMetricDataSource(), &metricDataSourceModel{}},
		{newTestSetDataSource(), &testSetDataSourceModel{}},
		{newTestCaseDataSource(), &testCaseDataSourceModel{}},
		{newRunTemplateDataSource(), &runTemplateDataSourceModel{}},
		{newAlertDataSource(), &alertDataSourceModel{}},
		{newScheduledRunDataSource(), &scheduledRunDataSourceModel{}},
	} {
		var metadata datasource.MetadataResponse
		test.source.Metadata(t.Context(), datasource.MetadataRequest{ProviderTypeName: "coval"}, &metadata)
		t.Run(metadata.TypeName, func(t *testing.T) {
			var schema datasource.SchemaResponse
			test.source.Schema(t.Context(), datasource.SchemaRequest{}, &schema)
			typ, ok := schema.Schema.Type().TerraformType(t.Context()).(tftypes.Object)
			if !ok {
				t.Fatal("data-source schema must be an object")
			}
			values := make(map[string]tftypes.Value)
			for name, typ := range typ.AttributeTypes {
				values[name] = tftypes.NewValue(typ, nil)
			}
			values["update_time"] = tftypes.NewValue(tftypes.String, "2026-01-02T00:00:00Z")
			state := tfsdk.State{Schema: schema.Schema, Raw: tftypes.NewValue(typ, values)}
			if d := state.Get(t.Context(), test.model); d.HasError() {
				t.Fatal(d)
			}
			if d := state.Set(t.Context(), test.model); d.HasError() {
				t.Fatal(d)
			}
			var updated types.String
			if d := state.GetAttribute(t.Context(), path.Root("update_time"), &updated); d.HasError() {
				t.Fatal(d)
			}
			if updated.ValueString() != "2026-01-02T00:00:00Z" {
				t.Fatalf("update_time = %s", updated)
			}
		})
	}
}
