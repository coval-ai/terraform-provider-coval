package provider

import (
	"context"
	"fmt"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/dynamicplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &testCaseResource{}
	_ resource.ResourceWithConfigure      = &testCaseResource{}
	_ resource.ResourceWithImportState    = &testCaseResource{}
	_ resource.ResourceWithIdentity       = &testCaseResource{}
	_ resource.ResourceWithModifyPlan     = &testCaseResource{}
	_ resource.ResourceWithValidateConfig = &testCaseResource{}
)

type testCaseResource struct {
	client *client.Client
}

type testCaseResourceModel struct {
	ID                 types.String  `tfsdk:"id"`
	Name               types.String  `tfsdk:"name"`
	TestSetID          types.String  `tfsdk:"test_set_id"`
	InputString        types.String  `tfsdk:"input_str"`
	ExpectedBehaviors  types.List    `tfsdk:"expected_behaviors"`
	ExpectedOutputJSON types.Dynamic `tfsdk:"expected_output_json"`
	Description        types.String  `tfsdk:"description"`
	InputType          types.String  `tfsdk:"input_type"`
	ScriptTurns        types.Dynamic `tfsdk:"script_turns"`
	SimulationMetadata types.Dynamic `tfsdk:"simulation_metadata_input"`
	MetricInput        types.Dynamic `tfsdk:"metric_input"`
	UserNotes          types.String  `tfsdk:"user_notes"`
	CreateTime         types.String  `tfsdk:"create_time"`
	UpdateTime         types.String  `tfsdk:"update_time"`
}

type testCaseIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

func newTestCaseResource() resource.Resource {
	return &testCaseResource{}
}

func (r *testCaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_test_case"
}

func (r *testCaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coval test case within a test set.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned test-case ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Canonical API resource name.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"test_set_id": schema.StringAttribute{
				MarkdownDescription: "ID of the Coval test set containing this test case.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(8, 8)},
			},
			"input_str": schema.StringAttribute{
				MarkdownDescription: "Scenario, transcript, or other input presented by this test case.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"expected_behaviors": schema.ListAttribute{
				MarkdownDescription: "Ordered behaviors expected from the agent. Set [] to clear them.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"expected_output_json": schema.DynamicAttribute{
				MarkdownDescription: "Arbitrary JSON object containing the expected structured output. Set {} to clear it.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Dynamic{dynamicplanmodifier.UseStateForUnknown()},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Human-readable test-case description. Set an empty string to clear it.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"input_type": schema.StringAttribute{
				MarkdownDescription: "Input type. SCRIPT requires a non-empty script_turns value; changing to another type clears script_turns.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:          []validator.String{stringvalidator.LengthAtMost(200)},
			},
			"script_turns": schema.DynamicAttribute{
				MarkdownDescription: "Non-empty ordered JSON array required for SCRIPT test cases. Each turn is a string, a text object, a DTMF object, or a skip object. Omit it for other input types.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Dynamic{dynamicplanmodifier.UseStateForUnknown()},
			},
			"simulation_metadata_input": schema.DynamicAttribute{
				MarkdownDescription: "Arbitrary JSON object containing simulation metadata. The legacy nested script_turns key is not supported; use the top-level script_turns attribute instead. Set {} to clear it.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Dynamic{dynamicplanmodifier.UseStateForUnknown()},
			},
			"metric_input": schema.DynamicAttribute{
				MarkdownDescription: "Arbitrary JSON object containing metric input data. Set {} to clear it.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Dynamic{dynamicplanmodifier.UseStateForUnknown()},
			},
			"user_notes": schema.StringAttribute{
				MarkdownDescription: "User-provided notes about the test case. Set an empty string to clear them.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
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

func (r *testCaseResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}

func (r *testCaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *testCaseResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config testCaseResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateTestCaseScriptConfig(config)...)
}

func validateTestCaseScriptConfig(config testCaseResourceModel) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if !config.SimulationMetadata.IsNull() && !config.SimulationMetadata.IsUnknown() && !config.SimulationMetadata.IsUnderlyingValueUnknown() {
		hasLegacyScriptTurns, err := dynamicJSONObjectHasKey(config.SimulationMetadata, "script_turns")
		if err != nil {
			diagnostics.AddAttributeError(path.Root("simulation_metadata_input"), "Invalid simulation metadata", err.Error())
		} else if hasLegacyScriptTurns {
			diagnostics.AddAttributeError(
				path.Root("simulation_metadata_input"),
				"Legacy nested script turns are not supported",
				"Move simulation_metadata_input.script_turns to the top-level script_turns attribute.",
			)
		}
	}
	if config.InputType.IsUnknown() || config.ScriptTurns.IsUnknown() || config.ScriptTurns.IsUnderlyingValueUnknown() {
		return diagnostics
	}

	hasScriptTurns := !config.ScriptTurns.IsNull()
	if config.InputType.IsNull() || config.InputType.ValueString() != "SCRIPT" {
		if hasScriptTurns {
			diagnostics.AddAttributeError(
				path.Root("script_turns"),
				"Invalid script turns",
				"script_turns must be omitted unless input_type is SCRIPT.",
			)
		}
		return diagnostics
	}

	if !hasScriptTurns {
		diagnostics.AddAttributeError(
			path.Root("script_turns"),
			"Missing script turns",
			"script_turns must be a non-empty array when input_type is SCRIPT.",
		)
		return diagnostics
	}
	length, err := dynamicJSONArrayLength(config.ScriptTurns)
	if err != nil {
		diagnostics.AddAttributeError(path.Root("script_turns"), "Invalid script turns", err.Error())
		return diagnostics
	}
	if length == 0 {
		diagnostics.AddAttributeError(
			path.Root("script_turns"),
			"Empty script turns",
			"script_turns must be a non-empty array when input_type is SCRIPT.",
		)
	}
	return diagnostics
}

func (r *testCaseResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var config testCaseResourceModel
	var plan testCaseResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan = testCasePlanForConfig(config, plan)
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
	useStateForUnchangedPlan(ctx, req, resp, path.Root("update_time"))
}

func testCasePlanForConfig(config testCaseResourceModel, plan testCaseResourceModel) testCaseResourceModel {
	if config.ScriptTurns.IsNull() && !plan.InputType.IsNull() && !plan.InputType.IsUnknown() && plan.InputType.ValueString() != "SCRIPT" {
		plan.ScriptTurns = types.DynamicNull()
	}
	return plan
}

func (r *testCaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan testCaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, diagnostics := createTestCaseInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreateTestCase(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coval test case", err.Error())
		return
	}
	state, diagnostics := testCaseResourceState(ctx, created, &plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, testCaseIdentityModel{ID: state.ID})...)
}

func (r *testCaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state testCaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote, err := r.client.GetTestCase(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coval test case", err.Error())
		return
	}
	refreshed, diagnostics := testCaseResourceState(ctx, remote, &state)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, testCaseIdentityModel{ID: refreshed.ID})...)
}

func (r *testCaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan testCaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, diagnostics := updateTestCaseInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.UpdateTestCase(ctx, plan.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update Coval test case", err.Error())
		return
	}
	state, diagnostics := testCaseResourceState(ctx, updated, &plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, testCaseIdentityModel{ID: state.ID})...)
}

func (r *testCaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state testCaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteTestCase(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Coval test case", err.Error())
	}
}

func (r *testCaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("id"), req, resp)
}

func createTestCaseInput(ctx context.Context, plan testCaseResourceModel) (client.CreateTestCaseInput, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	expectedBehaviors, behaviorDiagnostics := stringList(ctx, plan.ExpectedBehaviors)
	diagnostics.Append(behaviorDiagnostics...)
	expectedOutput, err := dynamicJSONObject(plan.ExpectedOutputJSON)
	if err != nil {
		diagnostics.AddAttributeError(path.Root("expected_output_json"), "Invalid expected output", err.Error())
	}
	scriptTurns, err := dynamicJSONArray(plan.ScriptTurns)
	if err != nil {
		diagnostics.AddAttributeError(path.Root("script_turns"), "Invalid script turns", err.Error())
	}
	simulationMetadata, err := dynamicJSONObject(plan.SimulationMetadata)
	if err != nil {
		diagnostics.AddAttributeError(path.Root("simulation_metadata_input"), "Invalid simulation metadata", err.Error())
	} else {
		hasLegacyScriptTurns, keyErr := dynamicJSONObjectHasKey(plan.SimulationMetadata, "script_turns")
		if keyErr != nil {
			diagnostics.AddAttributeError(path.Root("simulation_metadata_input"), "Invalid simulation metadata", keyErr.Error())
		} else if hasLegacyScriptTurns {
			diagnostics.AddAttributeError(
				path.Root("simulation_metadata_input"),
				"Legacy nested script turns are not supported",
				"Move simulation_metadata_input.script_turns to the top-level script_turns attribute.",
			)
		}
	}
	metricInput, err := dynamicJSONObject(plan.MetricInput)
	if err != nil {
		diagnostics.AddAttributeError(path.Root("metric_input"), "Invalid metric input", err.Error())
	}

	return client.CreateTestCaseInput{
		TestSetID:          plan.TestSetID.ValueString(),
		InputString:        plan.InputString.ValueString(),
		ExpectedBehaviors:  expectedBehaviors,
		ExpectedOutputJSON: expectedOutput,
		Description:        stringPointer(plan.Description),
		InputType:          stringPointer(plan.InputType),
		ScriptTurns:        scriptTurns,
		SimulationMetadata: simulationMetadata,
		MetricInput:        metricInput,
		UserNotes:          stringPointer(plan.UserNotes),
	}, diagnostics
}

func updateTestCaseInput(ctx context.Context, plan testCaseResourceModel) (client.UpdateTestCaseInput, diag.Diagnostics) {
	createInput, diagnostics := createTestCaseInput(ctx, plan)
	testSetID := createInput.TestSetID
	inputString := createInput.InputString
	return client.UpdateTestCaseInput{
		TestSetID:          &testSetID,
		InputString:        &inputString,
		ExpectedBehaviors:  createInput.ExpectedBehaviors,
		ExpectedOutputJSON: createInput.ExpectedOutputJSON,
		Description:        createInput.Description,
		InputType:          createInput.InputType,
		ScriptTurns:        createInput.ScriptTurns,
		SimulationMetadata: createInput.SimulationMetadata,
		MetricInput:        createInput.MetricInput,
		UserNotes:          createInput.UserNotes,
	}, diagnostics
}

func stringList(ctx context.Context, value types.List) (*[]string, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var result []string
	diagnostics := value.ElementsAs(ctx, &result, false)
	return &result, diagnostics
}

func testCaseResourceState(ctx context.Context, remote client.TestCase, prior *testCaseResourceModel) (testCaseResourceModel, diag.Diagnostics) {
	var diagnostics diag.Diagnostics

	var expectedOutput types.Dynamic
	var scriptTurns types.Dynamic
	var simulationMetadata types.Dynamic
	var metricInput types.Dynamic
	var err error
	if prior == nil {
		expectedOutput, err = dynamicFromJSONObject(remote.ExpectedOutputJSON)
	} else {
		expectedOutput, err = dynamicFromJSONObjectPreserving(remote.ExpectedOutputJSON, prior.ExpectedOutputJSON)
	}
	if err != nil {
		diagnostics.AddError("Unable to decode test-case expected output", err.Error())
	}
	if prior == nil {
		scriptTurns, err = dynamicFromJSONArray(remote.ScriptTurns)
	} else {
		scriptTurns, err = dynamicFromJSONArrayPreserving(remote.ScriptTurns, prior.ScriptTurns)
	}
	if err != nil {
		diagnostics.AddError("Unable to decode test-case script turns", err.Error())
	}
	simulationMetadataRaw, sanitizeErr := jsonObjectWithoutKey(remote.SimulationMetadata, "script_turns")
	if sanitizeErr != nil {
		diagnostics.AddError("Unable to decode test-case simulation metadata", sanitizeErr.Error())
	} else if prior == nil {
		simulationMetadata, err = dynamicFromJSONObject(simulationMetadataRaw)
	} else {
		simulationMetadata, err = dynamicFromJSONObjectPreserving(simulationMetadataRaw, prior.SimulationMetadata)
	}
	if sanitizeErr == nil && err != nil {
		diagnostics.AddError("Unable to decode test-case simulation metadata", err.Error())
	}
	if prior == nil {
		metricInput, err = dynamicFromJSONObject(remote.MetricInput)
	} else {
		metricInput, err = dynamicFromJSONObjectPreserving(remote.MetricInput, prior.MetricInput)
	}
	if err != nil {
		diagnostics.AddError("Unable to decode test-case metric input", err.Error())
	}

	expectedBehaviors := types.ListNull(types.StringType)
	if remote.ExpectedBehaviors != nil {
		var behaviorDiagnostics diag.Diagnostics
		expectedBehaviors, behaviorDiagnostics = types.ListValueFrom(ctx, types.StringType, *remote.ExpectedBehaviors)
		diagnostics.Append(behaviorDiagnostics...)
	}

	state := testCaseResourceModel{
		ID:                 types.StringValue(remote.ID),
		Name:               types.StringValue(remote.Name),
		TestSetID:          types.StringNull(),
		InputString:        types.StringValue(remote.InputString),
		ExpectedBehaviors:  expectedBehaviors,
		ExpectedOutputJSON: expectedOutput,
		Description:        types.StringNull(),
		InputType:          types.StringNull(),
		ScriptTurns:        scriptTurns,
		SimulationMetadata: simulationMetadata,
		MetricInput:        metricInput,
		UserNotes:          types.StringNull(),
		CreateTime:         types.StringValue(remote.CreateTime),
		UpdateTime:         types.StringNull(),
	}
	if remote.TestSetID != nil {
		state.TestSetID = types.StringValue(*remote.TestSetID)
	}
	if remote.Description != nil {
		state.Description = types.StringValue(*remote.Description)
	}
	if remote.InputType != nil {
		state.InputType = types.StringValue(*remote.InputType)
	}
	if remote.UserNotes != nil {
		state.UserNotes = types.StringValue(*remote.UserNotes)
	}
	if remote.UpdateTime != nil {
		state.UpdateTime = types.StringValue(*remote.UpdateTime)
	}
	return state, diagnostics
}
