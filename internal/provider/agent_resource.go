package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/dynamicplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &agentResource{}
	_ resource.ResourceWithConfigure   = &agentResource{}
	_ resource.ResourceWithImportState = &agentResource{}
	_ resource.ResourceWithIdentity    = &agentResource{}
)

var agentModelTypes = []string{
	"MODEL_TYPE_VOICE",
	"MODEL_TYPE_OUTBOUND_VOICE",
	"MODEL_TYPE_CHAT",
	"MODEL_TYPE_CHAT_A2A",
	"MODEL_TYPE_CHAT_WEBSOCKET",
	"MODEL_TYPE_SMS",
	"MODEL_TYPE_WEBSOCKET",
	"MODEL_TYPE_LIVEKIT",
	"MODEL_TYPE_DAILY",
	"MODEL_TYPE_OPENAI_REALTIME",
	"MODEL_TYPE_GEMINI_REALTIME",
	"MODEL_TYPE_GROK_REALTIME",
	"MODEL_TYPE_DEEPGRAM_REALTIME",
}

type agentResource struct {
	client *client.Client
}

type agentResourceModel struct {
	ID               types.String  `tfsdk:"id"`
	CustomerAgentID  types.String  `tfsdk:"customer_agent_id"`
	DisplayName      types.String  `tfsdk:"display_name"`
	ModelType        types.String  `tfsdk:"model_type"`
	PhoneNumber      types.String  `tfsdk:"phone_number"`
	Endpoint         types.String  `tfsdk:"endpoint"`
	Prompt           types.String  `tfsdk:"prompt"`
	Language         types.String  `tfsdk:"language"`
	Attributes       types.Dynamic `tfsdk:"attributes"`
	Metadata         types.Dynamic `tfsdk:"metadata"`
	Workflows        types.Dynamic `tfsdk:"workflows"`
	MetricIDs        types.Set     `tfsdk:"metric_ids"`
	TestSetIDs       types.Set     `tfsdk:"test_set_ids"`
	KnowledgeBaseIDs types.Set     `tfsdk:"knowledge_base_ids"`
	Tags             types.Set     `tfsdk:"tags"`
	CreateTime       types.String  `tfsdk:"create_time"`
	UpdateTime       types.String  `tfsdk:"update_time"`
}

type agentIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

func newAgentResource() resource.Resource {
	return &agentResource{}
}

func (r *agentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent"
}

func (r *agentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coval agent configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned agent ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"customer_agent_id": schema.StringAttribute{
				MarkdownDescription: "Customer-defined external identifier. Coval defaults it to id when omitted.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 200)},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Human-readable agent name.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 200)},
			},
			"model_type": schema.StringAttribute{
				MarkdownDescription: "Agent simulator type.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.OneOf(agentModelTypes...)},
			},
			"phone_number": schema.StringAttribute{
				MarkdownDescription: "Phone number in E.164 format or SIP address for voice and SMS agents.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.LengthAtMost(200)},
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Webhook endpoint URL used by agent types that require one.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.LengthAtMost(200)},
			},
			"prompt": schema.StringAttribute{
				MarkdownDescription: "Agent instructions or system prompt.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.LengthAtMost(50000)},
			},
			"language": schema.StringAttribute{
				MarkdownDescription: "Primary language for the agent.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.LengthAtMost(200)},
			},
			"attributes": schema.DynamicAttribute{
				MarkdownDescription: "Free-form JSON object containing agent attributes. Set null to clear it.",
				Optional:            true,
			},
			"metadata": schema.DynamicAttribute{
				MarkdownDescription: "Simulator-specific JSON configuration. The required shape depends on model_type. Set {} to clear it.",
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.Dynamic{dynamicplanmodifier.UseStateForUnknown()},
			},
			"workflows": schema.DynamicAttribute{
				MarkdownDescription: "Workflow JSON configuration. Set {} to clear it.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Dynamic{dynamicplanmodifier.UseStateForUnknown()},
			},
			"metric_ids": schema.SetAttribute{
				MarkdownDescription: "Metric IDs associated with the agent. Set [] to clear them.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
				Validators:          []validator.Set{setvalidator.ValueStringsAre(stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9]{22}$`), "must be a 22-character Coval resource ID"))},
			},
			"test_set_ids": schema.SetAttribute{
				MarkdownDescription: "Test-set IDs associated with the agent. Set [] to clear them.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
				Validators:          []validator.Set{setvalidator.ValueStringsAre(stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9]{8}$`), "must be an 8-character Coval test-set ID"))},
			},
			"knowledge_base_ids": schema.SetAttribute{
				MarkdownDescription: "Knowledge-base entry IDs associated with the agent.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"tags": schema.SetAttribute{
				MarkdownDescription: "Tags associated with the agent. Set [] to clear them.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
				Validators:          agentTagValidators(),
			},
			"create_time": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 creation timestamp.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"update_time": schema.StringAttribute{
				MarkdownDescription: "RFC 3339 timestamp of the latest update.",
				Computed:            true,
			},
		},
	}
}

func agentTagValidators() []validator.Set {
	return []validator.Set{
		setvalidator.SizeAtMost(20),
		setvalidator.ValueStringsAre(
			stringvalidator.LengthBetween(1, 200),
			stringvalidator.RegexMatches(regexp.MustCompile(`\S`), "must contain at least one non-whitespace character"),
		),
	}
}

func (r *agentResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"id": identityschema.StringAttribute{RequiredForImport: true},
	}}
}

func (r *agentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *agentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan agentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input, diagnostics := createAgentInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreateAgent(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coval agent", err.Error())
		return
	}
	state, diagnostics := agentResourceState(ctx, created, &plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, agentIdentityModel{ID: state.ID})...)
}

func (r *agentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state agentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := r.client.GetAgent(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coval agent", err.Error())
		return
	}
	refreshed, diagnostics := agentResourceState(ctx, remote, &state)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, agentIdentityModel{ID: refreshed.ID})...)
}

func (r *agentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan agentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input, diagnostics := updateAgentInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.UpdateAgent(ctx, plan.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update Coval agent", err.Error())
		return
	}
	state, diagnostics := agentResourceState(ctx, updated, &plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, agentIdentityModel{ID: state.ID})...)
}

func (r *agentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state agentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteAgent(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Coval agent", err.Error())
	}
}

func (r *agentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("id"), req, resp)
}

func createAgentInput(ctx context.Context, plan agentResourceModel) (client.CreateAgentInput, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	attributes := agentDynamicObject(plan.Attributes, "attributes", &diagnostics)
	metadata := agentDynamicObject(plan.Metadata, "metadata", &diagnostics)
	workflows := agentDynamicObject(plan.Workflows, "workflows", &diagnostics)
	metricIDs, metricDiagnostics := stringSet(ctx, plan.MetricIDs)
	diagnostics.Append(metricDiagnostics...)
	testSetIDs, testSetDiagnostics := stringSet(ctx, plan.TestSetIDs)
	diagnostics.Append(testSetDiagnostics...)
	tags, tagDiagnostics := stringSet(ctx, plan.Tags)
	diagnostics.Append(tagDiagnostics...)

	return client.CreateAgentInput{
		DisplayName:     plan.DisplayName.ValueString(),
		ModelType:       plan.ModelType.ValueString(),
		PhoneNumber:     stringPointer(plan.PhoneNumber),
		Endpoint:        stringPointer(plan.Endpoint),
		Prompt:          stringPointer(plan.Prompt),
		CustomerAgentID: stringPointer(plan.CustomerAgentID),
		Language:        stringPointer(plan.Language),
		Attributes:      attributes,
		Metadata:        metadata,
		Workflows:       workflows,
		MetricIDs:       metricIDs,
		TestSetIDs:      testSetIDs,
		Tags:            tags,
	}, diagnostics
}

func updateAgentInput(ctx context.Context, plan agentResourceModel) (client.UpdateAgentInput, diag.Diagnostics) {
	createInput, diagnostics := createAgentInput(ctx, plan)
	return client.UpdateAgentInput{
		DisplayName:     createInput.DisplayName,
		ModelType:       createInput.ModelType,
		PhoneNumber:     createInput.PhoneNumber,
		Endpoint:        createInput.Endpoint,
		Prompt:          createInput.Prompt,
		CustomerAgentID: plan.CustomerAgentID.ValueString(),
		Language:        createInput.Language,
		Attributes:      rawJSONOrNull(createInput.Attributes),
		Metadata:        rawJSONOrObject(createInput.Metadata),
		Workflows:       rawJSONOrObject(createInput.Workflows),
		MetricIDs:       stringSliceOrEmpty(createInput.MetricIDs),
		TestSetIDs:      stringSliceOrEmpty(createInput.TestSetIDs),
		Tags:            stringSliceOrEmpty(createInput.Tags),
	}, diagnostics
}

func agentDynamicObject(value types.Dynamic, attribute string, diagnostics *diag.Diagnostics) *json.RawMessage {
	result, err := dynamicJSONObject(value)
	if err != nil {
		diagnostics.AddAttributeError(path.Root(attribute), "Invalid agent JSON object", err.Error())
	}
	return result
}

func rawJSONOrNull(value *json.RawMessage) json.RawMessage {
	if value == nil {
		return json.RawMessage(`null`)
	}
	return *value
}

func rawJSONOrObject(value *json.RawMessage) json.RawMessage {
	if value == nil {
		return json.RawMessage(`{}`)
	}
	return *value
}

func stringSliceOrEmpty(value *[]string) []string {
	if value == nil {
		return []string{}
	}
	return *value
}

func agentResourceState(ctx context.Context, remote client.Agent, prior *agentResourceModel) (agentResourceModel, diag.Diagnostics) {
	return agentState(ctx, remote, prior)
}

func agentDataSourceState(ctx context.Context, remote client.Agent) (agentResourceModel, diag.Diagnostics) {
	return agentState(ctx, remote, nil)
}

func agentState(ctx context.Context, remote client.Agent, prior *agentResourceModel) (agentResourceModel, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	attributes, err := nullableAgentObject(remote.Attributes, priorValue(prior, func(model *agentResourceModel) types.Dynamic { return model.Attributes }))
	if err != nil {
		diagnostics.AddError("Unable to decode agent attributes", err.Error())
	}
	metadata, err := agentMetadataState(remote.Metadata, priorValue(prior, func(model *agentResourceModel) types.Dynamic { return model.Metadata }))
	if err != nil {
		diagnostics.AddError("Unable to decode agent metadata", err.Error())
	}
	workflows, err := dynamicFromJSONObjectPreserving(remote.Workflows, priorValue(prior, func(model *agentResourceModel) types.Dynamic { return model.Workflows }))
	if err != nil {
		diagnostics.AddError("Unable to decode agent workflows", err.Error())
	}
	metricIDs, metricDiagnostics := types.SetValueFrom(ctx, types.StringType, remote.MetricIDs)
	diagnostics.Append(metricDiagnostics...)
	testSetIDs, testSetDiagnostics := types.SetValueFrom(ctx, types.StringType, remote.TestSetIDs)
	diagnostics.Append(testSetDiagnostics...)
	knowledgeBaseIDs, knowledgeDiagnostics := types.SetValueFrom(ctx, types.StringType, remote.KnowledgeBaseIDs)
	diagnostics.Append(knowledgeDiagnostics...)
	tags, tagDiagnostics := types.SetValueFrom(ctx, types.StringType, remote.Tags)
	diagnostics.Append(tagDiagnostics...)

	state := agentResourceModel{
		ID:               types.StringValue(remote.ID),
		CustomerAgentID:  types.StringValue(remote.CustomerAgentID),
		DisplayName:      types.StringValue(remote.DisplayName),
		ModelType:        types.StringValue(remote.ModelType),
		PhoneNumber:      nullableString(remote.PhoneNumber),
		Endpoint:         nullableString(remote.Endpoint),
		Prompt:           nullableString(remote.Prompt),
		Language:         nullableString(remote.Language),
		Attributes:       attributes,
		Metadata:         metadata,
		Workflows:        workflows,
		MetricIDs:        metricIDs,
		TestSetIDs:       testSetIDs,
		KnowledgeBaseIDs: knowledgeBaseIDs,
		Tags:             tags,
		CreateTime:       types.StringValue(remote.CreateTime),
		UpdateTime:       nullableString(remote.UpdateTime),
	}
	return state, diagnostics
}

func priorValue(prior *agentResourceModel, selectValue func(*agentResourceModel) types.Dynamic) types.Dynamic {
	if prior == nil {
		return types.DynamicNull()
	}
	return selectValue(prior)
}

func nullableAgentObject(raw json.RawMessage, prior types.Dynamic) (types.Dynamic, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return types.DynamicNull(), nil
	}
	return dynamicFromJSONObjectPreserving(raw, prior)
}

func agentMetadataState(raw json.RawMessage, prior types.Dynamic) (types.Dynamic, error) {
	if !prior.IsNull() && !prior.IsUnknown() && !prior.IsUnderlyingValueUnknown() {
		priorRaw, err := dynamicJSONObject(prior)
		if err == nil && priorRaw != nil && jsonObjectContains(raw, *priorRaw) {
			return prior, nil
		}
	}
	return dynamicFromJSONObject(raw)
}

func nullableString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}
