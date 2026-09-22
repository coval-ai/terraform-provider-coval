package provider

import (
	"testing"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestScheduledRunResourceSchemaAndState(t *testing.T) {
	t.Parallel()
	var response resource.SchemaResponse
	newScheduledRunResource().Schema(t.Context(), resource.SchemaRequest{}, &response)
	for _, name := range []string{"display_name", "run_template_id", "schedule_expression"} {
		attribute, ok := response.Schema.Attributes[name].(schema.StringAttribute)
		if !ok || !attribute.Required {
			t.Errorf("%s must be required", name)
		}
	}
	plan := scheduledRunResourceModel{WorkspaceID: types.StringValue("workspace-1"), DisplayName: types.StringValue("Nightly"), RunTemplateID: types.StringValue("abc123def456ghi789jklm"), ScheduleExpression: types.StringValue("rate(1 day)"), ScheduleTimezone: types.StringValue("UTC"), Enabled: types.BoolValue(false)}
	input := scheduledRunInput(plan)
	if input.Enabled || input.ScheduleTimezone != "UTC" {
		t.Fatalf("input = %#v", input)
	}
	state := scheduledRunState(client.ScheduledRun{Name: "scheduled-runs/xyz789uvw456rst123abcd", ID: "xyz789uvw456rst123abcd", DisplayName: "Nightly", RunTemplateID: input.RunTemplateID, ScheduleExpression: input.ScheduleExpression, ScheduleTimezone: input.ScheduleTimezone, Enabled: input.Enabled, CreateTime: "2026-09-22T00:00:00Z"}, &plan)
	if state.WorkspaceID.ValueString() != "workspace-1" || !state.LastRunID.IsNull() || state.Enabled.ValueBool() {
		t.Fatalf("state = %#v", state)
	}
}
