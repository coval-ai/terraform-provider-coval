package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/dynamicplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &alertResource{}
	_ resource.ResourceWithConfigure   = &alertResource{}
	_ resource.ResourceWithImportState = &alertResource{}
	_ resource.ResourceWithIdentity    = &alertResource{}
	_ resource.ResourceWithModifyPlan  = &alertResource{}
)

type alertResource struct{ client *client.Client }

type alertResourceModel struct {
	ID                    types.String  `tfsdk:"id"`
	WorkspaceID           types.String  `tfsdk:"workspace_id"`
	Name                  types.String  `tfsdk:"name"`
	Description           types.String  `tfsdk:"description"`
	Status                types.String  `tfsdk:"status"`
	EvaluationType        types.String  `tfsdk:"evaluation_type"`
	ConversationSource    types.String  `tfsdk:"conversation_source"`
	MatchMode             types.String  `tfsdk:"match_mode"`
	CooldownSeconds       types.Int64   `tfsdk:"cooldown_seconds"`
	CustomMessageTemplate types.String  `tfsdk:"custom_message_template"`
	AgentIDs              types.Set     `tfsdk:"agent_ids"`
	RequiredTags          types.Set     `tfsdk:"required_tags"`
	ScheduledRunIDs       types.Set     `tfsdk:"scheduled_run_ids"`
	Conditions            types.Dynamic `tfsdk:"conditions"`
	Channels              types.Dynamic `tfsdk:"channels"`
	TriggerCount          types.Int64   `tfsdk:"trigger_count"`
	LastTriggeredAt       types.String  `tfsdk:"last_triggered_at"`
	CreateTime            types.String  `tfsdk:"create_time"`
	UpdateTime            types.String  `tfsdk:"update_time"`
}

type alertIdentityModel struct {
	ID          types.String `tfsdk:"id"`
	WorkspaceID types.String `tfsdk:"workspace_id"`
}

func newAlertResource() resource.Resource { return &alertResource{} }

func (r *alertResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert"
}

func (r *alertResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	optionalSet := func(description string) schema.SetAttribute {
		return schema.SetAttribute{
			MarkdownDescription: description,
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
			Default:             setdefault.StaticValue(types.SetValueMust(types.StringType, nil)),
		}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coval alert definition.",
		Attributes: map[string]schema.Attribute{
			"id":                      schema.StringAttribute{MarkdownDescription: "Server-assigned alert ULID.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"workspace_id":            workspaceResourceAttribute(),
			"name":                    schema.StringAttribute{MarkdownDescription: "Human-readable alert name.", Required: true, Validators: []validator.String{stringvalidator.LengthBetween(1, 200)}},
			"description":             schema.StringAttribute{MarkdownDescription: "Alert description.", Optional: true, Computed: true, Default: stringdefault.StaticString(""), Validators: []validator.String{stringvalidator.LengthAtMost(2000)}},
			"status":                  schema.StringAttribute{MarkdownDescription: "Alert status.", Computed: true},
			"evaluation_type":         schema.StringAttribute{MarkdownDescription: "When the alert evaluates.", Optional: true, Computed: true, Default: stringdefault.StaticString("ON_RUN_COMPLETE"), Validators: []validator.String{stringvalidator.OneOf("ON_RUN_COMPLETE")}},
			"conversation_source":     schema.StringAttribute{MarkdownDescription: "Conversation source to which the alert applies.", Optional: true, Computed: true, Default: stringdefault.StaticString("ALL"), Validators: []validator.String{stringvalidator.OneOf("ALL", "UPLOADED", "SIMULATED")}},
			"match_mode":              schema.StringAttribute{MarkdownDescription: "How multiple conditions are combined.", Optional: true, Computed: true, Default: stringdefault.StaticString("ALL"), Validators: []validator.String{stringvalidator.OneOf("ALL", "ANY")}},
			"cooldown_seconds":        schema.Int64Attribute{MarkdownDescription: "Minimum seconds between triggers.", Optional: true, Computed: true, Default: int64default.StaticInt64(0), Validators: []validator.Int64{int64validator.Between(0, 86400)}},
			"custom_message_template": schema.StringAttribute{MarkdownDescription: "Optional custom notification message template.", Optional: true, Validators: []validator.String{stringvalidator.LengthAtMost(5000)}},
			"agent_ids":               optionalSet("Agent IDs to which the alert is restricted. Set [] to remove the restriction."),
			"required_tags":           optionalSet("Tags that matching runs must contain. Set [] to remove the restriction."),
			"scheduled_run_ids":       optionalSet("Scheduled-run IDs to which the alert is restricted. Set [] to remove the restriction."),
			"conditions": schema.DynamicAttribute{
				MarkdownDescription: "Non-empty list of public API condition objects. Supported aggregations are SINGLE, RUN_AVERAGE, RUN_FRACTION, JOB_SUCCESS, and BASELINE_DEVIATION.",
				Required:            true,
			},
			"channels": schema.DynamicAttribute{
				MarkdownDescription: "List of public API notification channel objects. Channel config may contain credentials and is stored as sensitive Terraform state. Set [] for evaluation-only alerts.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.Dynamic{dynamicplanmodifier.UseStateForUnknown()},
			},
			"trigger_count":     schema.Int64Attribute{MarkdownDescription: "Number of times the alert has triggered.", Computed: true},
			"last_triggered_at": schema.StringAttribute{MarkdownDescription: "RFC 3339 timestamp of the latest trigger when available.", Computed: true},
			"create_time":       schema.StringAttribute{MarkdownDescription: "RFC 3339 creation timestamp.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"update_time":       schema.StringAttribute{MarkdownDescription: "RFC 3339 timestamp of the latest update.", Computed: true},
		},
	}
}

func (r *alertResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	useStateForUnchangedPlan(ctx, req, resp, path.Root("update_time"))
}

func (r *alertResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"id": identityschema.StringAttribute{RequiredForImport: true}, "workspace_id": identityschema.StringAttribute{OptionalForImport: true},
	}}
}

func (r *alertResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, got %T.", req.ProviderData))
		return
	}
	r.client = apiClient
}

func (r *alertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input, diagnostics := createAlertInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := clientForWorkspace(r.client, plan.WorkspaceID).CreateAlert(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coval alert", err.Error())
		return
	}
	state, diagnostics := alertState(ctx, created, &plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, alertIdentityModel{ID: state.ID, WorkspaceID: state.WorkspaceID})...)
}

func (r *alertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state alertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := clientForWorkspace(r.client, state.WorkspaceID).GetAlert(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coval alert", err.Error())
		return
	}
	refreshed, diagnostics := alertState(ctx, remote, &state)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, alertIdentityModel{ID: refreshed.ID, WorkspaceID: refreshed.WorkspaceID})...)
}

func (r *alertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan alertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input, diagnostics := updateAlertInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := clientForWorkspace(r.client, plan.WorkspaceID).UpdateAlert(ctx, plan.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update Coval alert", err.Error())
		return
	}
	state, diagnostics := alertState(ctx, updated, &plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, alertIdentityModel{ID: state.ID, WorkspaceID: state.WorkspaceID})...)
}

func (r *alertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := clientForWorkspace(r.client, state.WorkspaceID).DeleteAlert(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Coval alert", err.Error())
	}
}

func (r *alertResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importStateWithWorkspaceIdentity(ctx, req, resp)
}

func createAlertInput(ctx context.Context, plan alertResourceModel) (client.CreateAlertInput, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	conditions, err := alertJSONArray(plan.Conditions, true)
	if err != nil {
		diagnostics.AddAttributeError(path.Root("conditions"), "Invalid alert conditions", err.Error())
	}
	channels, err := alertJSONArray(plan.Channels, false)
	if err != nil {
		diagnostics.AddAttributeError(path.Root("channels"), "Invalid alert channels", err.Error())
	}
	agentIDs, agentDiagnostics := stringSet(ctx, plan.AgentIDs)
	diagnostics.Append(agentDiagnostics...)
	requiredTags, tagDiagnostics := stringSet(ctx, plan.RequiredTags)
	diagnostics.Append(tagDiagnostics...)
	scheduledRunIDs, scheduledDiagnostics := stringSet(ctx, plan.ScheduledRunIDs)
	diagnostics.Append(scheduledDiagnostics...)
	input := client.CreateAlertInput{
		Name: plan.Name.ValueString(), Description: stringPointer(plan.Description), EvaluationType: plan.EvaluationType.ValueString(),
		ConversationSource: stringPointer(plan.ConversationSource), MatchMode: stringPointer(plan.MatchMode), CooldownSeconds: intPointer(plan.CooldownSeconds),
		CustomMessageTemplate: stringPointer(plan.CustomMessageTemplate), AgentIDs: agentIDs, RequiredTags: requiredTags, ScheduledRunIDs: scheduledRunIDs,
		Conditions: json.RawMessage(`[]`), Channels: channels,
	}
	if conditions != nil {
		input.Conditions = *conditions
	}
	return input, diagnostics
}

func updateAlertInput(ctx context.Context, plan alertResourceModel) (client.UpdateAlertInput, diag.Diagnostics) {
	created, diagnostics := createAlertInput(ctx, plan)
	return client.UpdateAlertInput{
		Name: created.Name, Description: plan.Description.ValueString(), EvaluationType: created.EvaluationType,
		ConversationSource: valueOrEmptyString(created.ConversationSource), MatchMode: valueOrEmptyString(created.MatchMode), CooldownSeconds: plan.CooldownSeconds.ValueInt64(),
		CustomMessageTemplate: created.CustomMessageTemplate, AgentIDs: alertStringSliceOrEmpty(created.AgentIDs), RequiredTags: alertStringSliceOrEmpty(created.RequiredTags), ScheduledRunIDs: alertStringSliceOrEmpty(created.ScheduledRunIDs),
		Conditions: created.Conditions, Channels: valueOrEmptyJSON(created.Channels),
	}, diagnostics
}

func alertJSONArray(value types.Dynamic, requireNonEmpty bool) (*json.RawMessage, error) {
	raw, err := dynamicJSONArray(value)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		if requireNonEmpty {
			return nil, fmt.Errorf("value must be a non-empty list")
		}
		return nil, nil
	}
	var values []json.RawMessage
	if err := json.Unmarshal(*raw, &values); err != nil {
		return nil, fmt.Errorf("decode list: %w", err)
	}
	if requireNonEmpty && len(values) == 0 {
		return nil, fmt.Errorf("value must contain at least one condition")
	}
	for index, value := range values {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(value, &object); err != nil || object == nil {
			return nil, fmt.Errorf("element %d must be an object", index)
		}
	}
	return raw, nil
}

func normalizeAlertArray(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return json.RawMessage(`[]`), nil
	}
	var values []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("decode Coval alert list: %w", err)
	}
	for _, value := range values {
		delete(value, "ulid")
		for key, field := range value {
			if bytes.Equal(bytes.TrimSpace(field), []byte("null")) {
				delete(value, key)
			}
		}
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("encode Coval alert list: %w", err)
	}
	return json.RawMessage(encoded), nil
}

func alertState(ctx context.Context, remote client.Alert, prior *alertResourceModel) (alertResourceModel, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	agentIDs, agentDiagnostics := types.SetValueFrom(ctx, types.StringType, alertStringsOrEmpty(remote.AgentIDs))
	diagnostics.Append(agentDiagnostics...)
	requiredTags, tagDiagnostics := types.SetValueFrom(ctx, types.StringType, alertStringsOrEmpty(remote.RequiredTags))
	diagnostics.Append(tagDiagnostics...)
	scheduledRunIDs, scheduledDiagnostics := types.SetValueFrom(ctx, types.StringType, alertStringsOrEmpty(remote.ScheduledRunIDs))
	diagnostics.Append(scheduledDiagnostics...)
	conditionJSON, err := normalizeAlertArray(remote.Conditions)
	if err != nil {
		diagnostics.AddError("Unable to decode alert conditions", err.Error())
	}
	channelJSON, err := normalizeAlertArray(remote.Channels)
	if err != nil {
		diagnostics.AddError("Unable to decode alert channels", err.Error())
	}
	conditions, err := dynamicFromJSONArray(conditionJSON)
	if prior != nil {
		conditions, err = dynamicFromJSONArrayPreserving(conditionJSON, prior.Conditions)
	}
	if err != nil {
		diagnostics.AddError("Unable to decode alert conditions", err.Error())
	}
	channels, err := dynamicFromJSONArray(channelJSON)
	if prior != nil {
		channels, err = dynamicFromJSONArrayPreserving(channelJSON, prior.Channels)
	}
	if err != nil {
		diagnostics.AddError("Unable to decode alert channels", err.Error())
	}
	state := alertResourceModel{
		ID: types.StringValue(remote.ID), WorkspaceID: types.StringNull(), Name: types.StringValue(remote.Name), Description: types.StringValue(remote.Description),
		Status: types.StringValue(remote.Status), EvaluationType: types.StringValue(remote.EvaluationType), ConversationSource: types.StringValue(remote.ConversationSource), MatchMode: types.StringValue(remote.MatchMode),
		CooldownSeconds: types.Int64Value(remote.CooldownSeconds), CustomMessageTemplate: nullableString(remote.CustomMessageTemplate),
		AgentIDs: agentIDs, RequiredTags: requiredTags, ScheduledRunIDs: scheduledRunIDs, Conditions: conditions, Channels: channels,
		TriggerCount: types.Int64Value(remote.TriggerCount), LastTriggeredAt: nullableString(remote.LastTriggeredAt), CreateTime: types.StringValue(remote.CreateTime), UpdateTime: types.StringValue(remote.UpdateTime),
	}
	if prior != nil {
		state.WorkspaceID = prior.WorkspaceID
	}
	return state, diagnostics
}

func valueOrEmptyString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func valueOrEmptyJSON(value *json.RawMessage) json.RawMessage {
	if value == nil {
		return json.RawMessage(`[]`)
	}
	return *value
}
func alertStringSliceOrEmpty(value *[]string) []string {
	if value == nil {
		return []string{}
	}
	return *value
}

func alertStringsOrEmpty(value []string) []string {
	if value == nil {
		return []string{}
	}
	return value
}
