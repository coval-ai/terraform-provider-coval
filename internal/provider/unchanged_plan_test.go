package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestUnchangedResourcePlan(t *testing.T) {
	t.Parallel()
	for _, r := range []resource.Resource{newAgentResource(), newMetricResource(), newPersonaResource(), newTestCaseResource(), newTestSetResource()} {
		var metadata resource.MetadataResponse
		r.Metadata(t.Context(), resource.MetadataRequest{ProviderTypeName: "coval"}, &metadata)
		t.Run(metadata.TypeName, func(t *testing.T) {
			editable := "description"
			switch metadata.TypeName {
			case "coval_agent":
				editable = "display_name"
			case "coval_persona":
				editable = "name"
			}
			modifier, ok := r.(resource.ResourceWithModifyPlan)
			if !ok {
				t.Fatal("resource has no ModifyPlan")
			}
			var response resource.SchemaResponse
			r.Schema(t.Context(), resource.SchemaRequest{}, &response)
			typ, ok := response.Schema.Type().TerraformType(t.Context()).(tftypes.Object)
			if !ok {
				t.Fatal("resource schema must be an object")
			}
			values := make(map[string]tftypes.Value)
			for name, typ := range typ.AttributeTypes {
				values[name] = tftypes.NewValue(typ, nil)
			}
			state := tfsdk.State{Schema: response.Schema, Raw: tftypes.NewValue(typ, values)}
			if d := state.SetAttribute(t.Context(), path.Root("update_time"), types.StringValue("2026-01-01T00:00:00Z")); d.HasError() {
				t.Fatal(d)
			}
			computed := []path.Path{path.Root("update_time")}
			switch metadata.TypeName {
			case "coval_test_set":
				if d := state.SetAttribute(t.Context(), path.Root("test_case_count"), types.Int64Value(3)); d.HasError() {
					t.Fatal(d)
				}
				computed = append(computed, path.Root("test_case_count"))
			case "coval_persona":
				if d := state.SetAttribute(t.Context(), path.Root("audio_degradation").AtName("preset_version"), types.Int64Value(1)); d.HasError() {
					t.Fatal(d)
				}
				computed = append(computed, path.Root("audio_degradation").AtName("preset_version"))
			}
			for _, scenario := range []string{"unchanged", "edit", "create", "destroy", "known output", "unknown config"} {
				t.Run(scenario, func(t *testing.T) {
					plan := tfsdk.Plan(state)
					config := tfsdk.Config{Schema: response.Schema, Raw: tftypes.NewValue(typ, values)}
					prior := state
					for _, p := range computed {
						var value attr.Value
						if d := state.GetAttribute(t.Context(), p, &value); d.HasError() {
							t.Fatal(d)
						}
						typ := value.Type(t.Context())
						unknown, err := typ.ValueFromTerraform(t.Context(), tftypes.NewValue(typ.TerraformType(t.Context()), tftypes.UnknownValue))
						if err != nil {
							t.Fatal(err)
						}
						if d := plan.SetAttribute(t.Context(), p, unknown); d.HasError() {
							t.Fatal(d)
						}
					}
					switch scenario {
					case "edit":
						if d := plan.SetAttribute(t.Context(), path.Root(editable), types.StringValue("Updated")); d.HasError() {
							t.Fatal(d)
						}
					case "create":
						prior.Raw = tftypes.NewValue(typ, nil)
					case "destroy":
						plan.Raw = tftypes.NewValue(typ, nil)
					case "known output":
						if d := plan.SetAttribute(t.Context(), path.Root("update_time"), types.StringValue("2026-02-01T00:00:00Z")); d.HasError() {
							t.Fatal(d)
						}
					case "unknown config":
						config.Raw = plan.Raw
					}
					result := resource.ModifyPlanResponse{Plan: plan}
					modifier.ModifyPlan(t.Context(), resource.ModifyPlanRequest{Plan: plan, State: prior, Config: config}, &result)
					if result.Diagnostics.HasError() {
						t.Fatal(result.Diagnostics)
					}
					want := plan.Raw
					if scenario == "unchanged" {
						want = state.Raw
					}
					if !result.Plan.Raw.Equal(want) {
						t.Fatalf("plan = %s, want %s", result.Plan.Raw, want)
					}
				})
			}
		})
	}
}

func TestMetricPlanWithPartialRuntimeConfig(t *testing.T) {
	t.Parallel()
	var response resource.SchemaResponse
	newMetricResource().Schema(t.Context(), resource.SchemaRequest{}, &response)
	typ, ok := response.Schema.Type().TerraformType(t.Context()).(tftypes.Object)
	if !ok {
		t.Fatal("resource schema must be an object")
	}
	values := make(map[string]tftypes.Value)
	for name, typ := range typ.AttributeTypes {
		values[name] = tftypes.NewValue(typ, nil)
	}
	state := tfsdk.State{Schema: response.Schema, Raw: tftypes.NewValue(typ, values)}
	for name, value := range map[string]attr.Value{
		"id":          types.StringValue("example-metric"),
		"metric_name": types.StringValue("Resolution"),
		"description": types.StringValue("Whether the issue was resolved"),
		"metric_type": types.StringValue("METRIC_LLM_BINARY"),
		"created_by":  types.StringValue("creator@example.com"),
		"update_time": types.StringValue("2026-01-01T00:00:00Z"),
		"runtime_config": types.ObjectValueMust(metricRuntimeConfigAttributeTypes, map[string]attr.Value{
			"model_version": types.StringValue("example-model"), "thinking_enabled": types.BoolValue(true),
		}),
		"current_version": types.ObjectValueMust(currentMetricVersionAttributeTypes, map[string]attr.Value{
			"ulid": types.StringValue("example-version"), "version_number": types.Int64Value(1), "change_type": types.StringValue("CREATE"),
		}),
	} {
		if d := state.SetAttribute(t.Context(), path.Root(name), value); d.HasError() {
			t.Fatal(d)
		}
	}
	for _, scenario := range []string{"unchanged", "edit", "unknown model", "reset during update"} {
		t.Run(scenario, func(t *testing.T) {
			config := tfsdk.Plan(state)
			for name, attribute := range response.Schema.Attributes {
				if attribute.IsComputed() && !attribute.IsOptional() {
					if d := config.SetAttribute(t.Context(), path.Root(name), nilValue(t, attribute.GetType())); d.HasError() {
						t.Fatal(d)
					}
				}
			}
			runtime := map[string]attr.Value{"model_version": types.StringValue("example-model"), "thinking_enabled": types.BoolNull()}
			if scenario == "unknown model" {
				runtime["model_version"] = types.StringUnknown()
			}
			if scenario == "reset during update" {
				runtime["model_version"] = types.StringNull()
			}
			if d := config.SetAttribute(t.Context(), path.Root("runtime_config"), types.ObjectValueMust(metricRuntimeConfigAttributeTypes, runtime)); d.HasError() {
				t.Fatal(d)
			}
			proposed := tfsdk.Plan(state)
			if d := proposed.SetAttribute(t.Context(), path.Root("runtime_config"), types.ObjectValueMust(metricRuntimeConfigAttributeTypes, runtime)); d.HasError() {
				t.Fatal(d)
			}
			if scenario == "edit" || scenario == "reset during update" {
				for _, plan := range []*tfsdk.Plan{&config, &proposed} {
					if d := plan.SetAttribute(t.Context(), path.Root("description"), types.StringValue("Updated")); d.HasError() {
						t.Fatal(d)
					}
				}
			}
			encode := func(value tftypes.Value) *tfprotov6.DynamicValue {
				encoded, err := tfprotov6.NewDynamicValue(typ, value)
				if err != nil {
					t.Fatal(err)
				}
				return &encoded
			}
			server := providerserver.NewProtocol6(New("test")())()
			result, err := server.PlanResourceChange(t.Context(), &tfprotov6.PlanResourceChangeRequest{
				TypeName: "coval_metric", PriorState: encode(state.Raw), Config: encode(config.Raw), ProposedNewState: encode(proposed.Raw),
			})
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range result.Diagnostics {
				if d.Severity == tfprotov6.DiagnosticSeverityError {
					t.Fatalf("%s: %s", d.Summary, d.Detail)
				}
			}
			planned, err := result.PlannedState.Unmarshal(typ)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "unchanged" {
				if !planned.Equal(state.Raw) {
					t.Fatalf("unchanged metric plans an update: %s", planned)
				}
				return
			}
			plan := tfsdk.Plan{Schema: response.Schema, Raw: planned}
			for _, name := range []string{"update_time", "current_version", "evaluation"} {
				var value attr.Value
				if d := plan.GetAttribute(t.Context(), path.Root(name), &value); d.HasError() {
					t.Fatal(d)
				}
				if !value.IsUnknown() {
					t.Errorf("%s = %s, want unknown", name, value)
				}
			}
			if scenario == "reset during update" {
				var runtime types.Object
				if d := plan.GetAttribute(t.Context(), path.Root("runtime_config"), &runtime); d.HasError() {
					t.Fatal(d)
				}
				for name, value := range runtime.Attributes() {
					if !value.IsUnknown() {
						t.Errorf("%s = %s, want unknown during reset", name, value)
					}
				}
			}
		})
	}
}

func nilValue(t *testing.T, typ attr.Type) attr.Value {
	t.Helper()
	value, err := typ.ValueFromTerraform(t.Context(), tftypes.NewValue(typ.TerraformType(t.Context()), nil))
	if err != nil {
		t.Fatal(err)
	}
	return value
}
