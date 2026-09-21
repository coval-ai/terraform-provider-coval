package provider

import (
	"context"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &testSetResource{}
	_ resource.ResourceWithConfigure   = &testSetResource{}
	_ resource.ResourceWithImportState = &testSetResource{}
	_ resource.ResourceWithIdentity    = &testSetResource{}
	_ resource.ResourceWithModifyPlan  = &testSetResource{}
)

type testSetResource struct {
	client *client.Client
}

type testSetResourceModel struct {
	ID              types.String  `tfsdk:"id"`
	Name            types.String  `tfsdk:"name"`
	Slug            types.String  `tfsdk:"slug"`
	DisplayName     types.String  `tfsdk:"display_name"`
	Description     types.String  `tfsdk:"description"`
	TestSetType     types.String  `tfsdk:"test_set_type"`
	TestSetMetadata types.Dynamic `tfsdk:"test_set_metadata"`
	Parameters      types.Dynamic `tfsdk:"parameters"`
	TestCaseCount   types.Int64   `tfsdk:"test_case_count"`
	Tags            types.Set     `tfsdk:"tags"`
	CreateTime      types.String  `tfsdk:"create_time"`
	UpdateTime      types.String  `tfsdk:"update_time"`
}

type testSetIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

func newTestSetResource() resource.Resource {
	return &testSetResource{}
}

func (r *testSetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_test_set"
}

func (r *testSetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coval test set.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned test-set ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Canonical API resource name.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"slug": schema.StringAttribute{
				MarkdownDescription: "URL-friendly identifier. Coval derives it from display_name when omitted.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 100),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z0-9_-]+$`),
						"must contain only lowercase letters, numbers, dashes, and underscores",
					),
				},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Human-readable test-set name.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 100)},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Test-set description.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"test_set_type": schema.StringAttribute{
				MarkdownDescription: "Test-set type, such as DEFAULT, SCENARIO, TRANSCRIPT, or WORKFLOW.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:          []validator.String{stringvalidator.LengthAtMost(50)},
			},
			"test_set_metadata": schema.DynamicAttribute{
				MarkdownDescription: "Arbitrary JSON object containing additional test-set configuration. Set {} to clear it.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Dynamic{dynamicplanmodifier.UseStateForUnknown()},
			},
			"parameters": schema.DynamicAttribute{
				MarkdownDescription: "JSON object mapping parameter names to value arrays. Set {} to clear it.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Dynamic{dynamicplanmodifier.UseStateForUnknown()},
			},
			"test_case_count": schema.Int64Attribute{
				MarkdownDescription: "Number of active test cases in the test set.",
				Computed:            true,
			},
			"tags": schema.SetAttribute{
				MarkdownDescription: "Tags associated with the test set. Set [] to clear them.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
				Validators:          testSetTagValidators(),
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

func testSetTagValidators() []validator.Set {
	return []validator.Set{
		setvalidator.SizeAtMost(20),
		setvalidator.ValueStringsAre(
			stringvalidator.LengthAtMost(200),
			stringvalidator.RegexMatches(regexp.MustCompile(`\S`), "must contain at least one non-whitespace character"),
		),
	}
}

func (r *testSetResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	useStateForUnchangedPlan(ctx, req, resp,
		path.Root("test_case_count"),
		path.Root("update_time"),
	)
}

func (r *testSetResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}

func (r *testSetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *testSetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan testSetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, diagnostics := createTestSetInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateTestSet(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coval test set", err.Error())
		return
	}
	state, diagnostics := testSetResourceState(ctx, created, &plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, testSetIdentityModel{ID: state.ID})...)
}

func (r *testSetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state testSetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote, err := r.client.GetTestSet(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coval test set", err.Error())
		return
	}
	refreshed, diagnostics := testSetResourceState(ctx, remote, &state)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, testSetIdentityModel{ID: refreshed.ID})...)
}

func (r *testSetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan testSetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, diagnostics := updateTestSetInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.UpdateTestSet(ctx, plan.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update Coval test set", err.Error())
		return
	}
	state, diagnostics := testSetResourceState(ctx, updated, &plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, testSetIdentityModel{ID: state.ID})...)
}

func (r *testSetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state testSetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteTestSet(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Coval test set", err.Error())
	}
}

func (r *testSetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("id"), req, resp)
}

func createTestSetInput(ctx context.Context, plan testSetResourceModel) (client.CreateTestSetInput, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	metadata, err := dynamicJSONObject(plan.TestSetMetadata)
	if err != nil {
		diagnostics.AddAttributeError(path.Root("test_set_metadata"), "Invalid test-set metadata", err.Error())
	}
	parameters, err := dynamicJSONObject(plan.Parameters)
	if err != nil {
		diagnostics.AddAttributeError(path.Root("parameters"), "Invalid test-set parameters", err.Error())
	}
	tags, tagDiagnostics := stringSet(ctx, plan.Tags)
	diagnostics.Append(tagDiagnostics...)

	input := client.CreateTestSetInput{
		DisplayName:     plan.DisplayName.ValueString(),
		Description:     stringPointer(plan.Description),
		TestSetType:     stringPointer(plan.TestSetType),
		TestSetMetadata: metadata,
		Parameters:      parameters,
		Tags:            tags,
	}
	if !plan.Slug.IsNull() && !plan.Slug.IsUnknown() {
		input.Slug = stringPointer(plan.Slug)
	}
	return input, diagnostics
}

func updateTestSetInput(ctx context.Context, plan testSetResourceModel) (client.UpdateTestSetInput, diag.Diagnostics) {
	createInput, diagnostics := createTestSetInput(ctx, plan)
	return client.UpdateTestSetInput{
		DisplayName:     &createInput.DisplayName,
		Slug:            createInput.Slug,
		Description:     createInput.Description,
		TestSetType:     createInput.TestSetType,
		TestSetMetadata: createInput.TestSetMetadata,
		Parameters:      createInput.Parameters,
		Tags:            createInput.Tags,
	}, diagnostics
}

func stringSet(ctx context.Context, value types.Set) (*[]string, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var result []string
	diagnostics := value.ElementsAs(ctx, &result, false)
	return &result, diagnostics
}

func testSetResourceState(ctx context.Context, remote client.TestSet, prior *testSetResourceModel) (testSetResourceModel, diag.Diagnostics) {
	return testSetState(ctx, remote, prior, types.StringValue(""))
}

func testSetDataSourceState(ctx context.Context, remote client.TestSet) (testSetResourceModel, diag.Diagnostics) {
	return testSetState(ctx, remote, nil, types.StringNull())
}

func testSetState(ctx context.Context, remote client.TestSet, prior *testSetResourceModel, descriptionWhenNull types.String) (testSetResourceModel, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	var metadata types.Dynamic
	var parameters types.Dynamic
	var err error
	if prior == nil {
		metadata, err = dynamicFromJSONObject(remote.TestSetMetadata)
	} else {
		metadata, err = dynamicFromJSONObjectPreserving(remote.TestSetMetadata, prior.TestSetMetadata)
	}
	if err != nil {
		diagnostics.AddError("Unable to decode test-set metadata", err.Error())
	}
	if prior == nil {
		parameters, err = dynamicFromJSONObject(remote.Parameters)
	} else {
		parameters, err = dynamicFromJSONObjectPreserving(remote.Parameters, prior.Parameters)
	}
	if err != nil {
		diagnostics.AddError("Unable to decode test-set parameters", err.Error())
	}
	tags, tagDiagnostics := types.SetValueFrom(ctx, types.StringType, remote.Tags)
	diagnostics.Append(tagDiagnostics...)

	state := testSetResourceModel{
		ID:              types.StringValue(remote.ID),
		Name:            types.StringValue(remote.Name),
		Slug:            types.StringValue(remote.Slug),
		DisplayName:     types.StringValue(remote.DisplayName),
		Description:     descriptionWhenNull,
		TestSetType:     types.StringNull(),
		TestSetMetadata: metadata,
		Parameters:      parameters,
		TestCaseCount:   types.Int64Null(),
		Tags:            tags,
		CreateTime:      types.StringValue(remote.CreateTime),
		UpdateTime:      types.StringNull(),
	}
	if remote.Description != nil {
		state.Description = types.StringValue(*remote.Description)
	}
	if remote.TestSetType != nil {
		state.TestSetType = types.StringValue(*remote.TestSetType)
	}
	if remote.TestCaseCount != nil {
		state.TestCaseCount = types.Int64Value(*remote.TestCaseCount)
	}
	if remote.UpdateTime != nil {
		state.UpdateTime = types.StringValue(*remote.UpdateTime)
	}
	return state, diagnostics
}
