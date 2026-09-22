package provider

import (
	"context"
	"fmt"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/dynamicplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &runTemplateResource{}
	_ resource.ResourceWithConfigure   = &runTemplateResource{}
	_ resource.ResourceWithImportState = &runTemplateResource{}
	_ resource.ResourceWithIdentity    = &runTemplateResource{}
	_ resource.ResourceWithModifyPlan  = &runTemplateResource{}
)

type runTemplateResource struct {
	client *client.Client
}

type runTemplateResourceModel struct {
	ID              types.String  `tfsdk:"id"`
	WorkspaceID     types.String  `tfsdk:"workspace_id"`
	Name            types.String  `tfsdk:"name"`
	DisplayName     types.String  `tfsdk:"display_name"`
	Description     types.String  `tfsdk:"description"`
	AgentIDs        types.Set     `tfsdk:"agent_ids"`
	PersonaIDs      types.Set     `tfsdk:"persona_ids"`
	TestSetIDs      types.Set     `tfsdk:"test_set_ids"`
	MetricIDs       types.Set     `tfsdk:"metric_ids"`
	MutationIDs     types.Set     `tfsdk:"mutation_ids"`
	IterationCount  types.Int64   `tfsdk:"iteration_count"`
	Concurrency     types.Int64   `tfsdk:"concurrency"`
	SubSampleSize   types.Int64   `tfsdk:"sub_sample_size"`
	SubSampleSeed   types.Int64   `tfsdk:"sub_sample_seed"`
	Metadata        types.Dynamic `tfsdk:"metadata"`
	Tags            types.Set     `tfsdk:"tags"`
	CreateTime      types.String  `tfsdk:"create_time"`
	UpdateTime      types.String  `tfsdk:"update_time"`
	CreatedByUserID types.String  `tfsdk:"created_by_user_id"`
}

type runTemplateIdentityModel struct {
	ID          types.String `tfsdk:"id"`
	WorkspaceID types.String `tfsdk:"workspace_id"`
}

func newRunTemplateResource() resource.Resource {
	return &runTemplateResource{}
}

func (r *runTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_run_template"
}

func (r *runTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	requiredIDs := func(description string, length int) schema.SetAttribute {
		return schema.SetAttribute{
			MarkdownDescription: description,
			ElementType:         types.StringType,
			Required:            true,
			Validators: []validator.Set{
				setvalidator.SizeAtLeast(1),
				setvalidator.ValueStringsAre(stringvalidator.LengthBetween(length, length)),
			},
		}
	}
	optionalIDs := func(description string, length int) schema.SetAttribute {
		return schema.SetAttribute{
			MarkdownDescription: description,
			ElementType:         types.StringType,
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			Validators:          []validator.Set{setvalidator.ValueStringsAre(stringvalidator.LengthBetween(length, length))},
		}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a reusable Coval run template.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned run-template ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"workspace_id": workspaceResourceAttribute(),
			"name": schema.StringAttribute{
				MarkdownDescription: "Canonical API resource name.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "Human-readable run-template name.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 200)},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Run-template description.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
			},
			"agent_ids":    requiredIDs("Agent IDs included in the template. Every agent must exist and be accessible.", 22),
			"persona_ids":  requiredIDs("Persona IDs included in the template. Every persona must exist.", 22),
			"test_set_ids": requiredIDs("Test-set IDs included in the template. Every test set must exist.", 8),
			"metric_ids":   optionalIDs("Metric IDs evaluated by the template. Set [] to use the agents' default metrics.", 22),
			"mutation_ids": optionalIDs("Mutation IDs used for A/B testing. Set [] to clear them.", 26),
			"iteration_count": schema.Int64Attribute{
				MarkdownDescription: "Number of times to run each test case.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
				Validators:          []validator.Int64{int64validator.Between(1, 100)},
			},
			"concurrency": schema.Int64Attribute{
				MarkdownDescription: "Maximum number of simulations to run concurrently.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
				Validators:          []validator.Int64{int64validator.Between(1, 50)},
			},
			"sub_sample_size": schema.Int64Attribute{
				MarkdownDescription: "Number of test cases to sample randomly. Use 0 to include every test case.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				Validators:          []validator.Int64{int64validator.AtLeast(0)},
			},
			"sub_sample_seed": schema.Int64Attribute{
				MarkdownDescription: "Random seed for reproducible sub-sampling. Omit it to let Coval choose the sample.",
				Optional:            true,
			},
			"metadata": schema.DynamicAttribute{
				MarkdownDescription: "Arbitrary JSON object containing run-template metadata. Set {} to clear it.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Dynamic{dynamicplanmodifier.UseStateForUnknown()},
			},
			"tags": schema.SetAttribute{
				MarkdownDescription: "Tags associated with the run template. Set [] to clear them.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
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
			"created_by_user_id": schema.StringAttribute{
				MarkdownDescription: "ID of the user who created the run template when available.",
				Computed:            true,
			},
		},
	}
}

func (r *runTemplateResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	useStateForUnchangedPlan(ctx, req, resp, path.Root("update_time"))
}

func (r *runTemplateResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{
		"id":           identityschema.StringAttribute{RequiredForImport: true},
		"workspace_id": identityschema.StringAttribute{OptionalForImport: true},
	}}
}

func (r *runTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *runTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan runTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, diagnostics := createRunTemplateInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := clientForWorkspace(r.client, plan.WorkspaceID).CreateRunTemplate(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coval run template", err.Error())
		return
	}
	state, diagnostics := runTemplateResourceState(ctx, created, &plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, runTemplateIdentityModel{ID: state.ID, WorkspaceID: state.WorkspaceID})...)
}

func (r *runTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state runTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote, err := clientForWorkspace(r.client, state.WorkspaceID).GetRunTemplate(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coval run template", err.Error())
		return
	}
	refreshed, diagnostics := runTemplateResourceState(ctx, remote, &state)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, runTemplateIdentityModel{ID: refreshed.ID, WorkspaceID: refreshed.WorkspaceID})...)
}

func (r *runTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan runTemplateResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, diagnostics := updateRunTemplateInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := clientForWorkspace(r.client, plan.WorkspaceID).UpdateRunTemplate(ctx, plan.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update Coval run template", err.Error())
		return
	}
	state, diagnostics := runTemplateResourceState(ctx, updated, &plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, runTemplateIdentityModel{ID: state.ID, WorkspaceID: state.WorkspaceID})...)
}

func (r *runTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state runTemplateResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := clientForWorkspace(r.client, state.WorkspaceID).DeleteRunTemplate(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Coval run template", err.Error())
	}
}

func (r *runTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importStateWithWorkspaceIdentity(ctx, req, resp)
}

func createRunTemplateInput(ctx context.Context, plan runTemplateResourceModel) (client.CreateRunTemplateInput, diag.Diagnostics) {
	agentIDs, diagnostics := runTemplateStringSet(ctx, plan.AgentIDs)
	personaIDs, personaDiagnostics := runTemplateStringSet(ctx, plan.PersonaIDs)
	diagnostics.Append(personaDiagnostics...)
	testSetIDs, testSetDiagnostics := runTemplateStringSet(ctx, plan.TestSetIDs)
	diagnostics.Append(testSetDiagnostics...)
	metricIDs, metricDiagnostics := stringSet(ctx, plan.MetricIDs)
	diagnostics.Append(metricDiagnostics...)
	mutationIDs, mutationDiagnostics := stringSet(ctx, plan.MutationIDs)
	diagnostics.Append(mutationDiagnostics...)
	tags, tagDiagnostics := stringSet(ctx, plan.Tags)
	diagnostics.Append(tagDiagnostics...)
	metadata, err := dynamicJSONObject(plan.Metadata)
	if err != nil {
		diagnostics.AddAttributeError(path.Root("metadata"), "Invalid run-template metadata", err.Error())
	}

	return client.CreateRunTemplateInput{
		DisplayName:    plan.DisplayName.ValueString(),
		Description:    stringPointer(plan.Description),
		AgentIDs:       agentIDs,
		PersonaIDs:     personaIDs,
		TestSetIDs:     testSetIDs,
		MetricIDs:      metricIDs,
		MutationIDs:    mutationIDs,
		IterationCount: plan.IterationCount.ValueInt64(),
		Concurrency:    plan.Concurrency.ValueInt64(),
		SubSampleSize:  plan.SubSampleSize.ValueInt64(),
		SubSampleSeed:  intPointer(plan.SubSampleSeed),
		Metadata:       metadata,
		Tags:           tags,
	}, diagnostics
}

func updateRunTemplateInput(ctx context.Context, plan runTemplateResourceModel) (client.UpdateRunTemplateInput, diag.Diagnostics) {
	createInput, diagnostics := createRunTemplateInput(ctx, plan)
	return client.UpdateRunTemplateInput{
		DisplayName:    createInput.DisplayName,
		Description:    plan.Description.ValueString(),
		AgentIDs:       createInput.AgentIDs,
		PersonaIDs:     createInput.PersonaIDs,
		TestSetIDs:     createInput.TestSetIDs,
		MetricIDs:      valueOrEmpty(createInput.MetricIDs),
		MutationIDs:    valueOrEmpty(createInput.MutationIDs),
		IterationCount: createInput.IterationCount,
		Concurrency:    createInput.Concurrency,
		SubSampleSize:  createInput.SubSampleSize,
		SubSampleSeed:  createInput.SubSampleSeed,
		Metadata:       createInput.Metadata,
		Tags:           valueOrEmpty(createInput.Tags),
	}, diagnostics
}

func valueOrEmpty(value *[]string) []string {
	if value == nil {
		return []string{}
	}
	return *value
}

func runTemplateStringSet(ctx context.Context, value types.Set) ([]string, diag.Diagnostics) {
	values, diagnostics := stringSet(ctx, value)
	if values == nil {
		return []string{}, diagnostics
	}
	return *values, diagnostics
}

func runTemplateResourceState(ctx context.Context, remote client.RunTemplate, prior *runTemplateResourceModel) (runTemplateResourceModel, diag.Diagnostics) {
	return runTemplateState(ctx, remote, prior)
}

func runTemplateDataSourceState(ctx context.Context, remote client.RunTemplate, config *runTemplateResourceModel) (runTemplateResourceModel, diag.Diagnostics) {
	return runTemplateState(ctx, remote, config)
}

func runTemplateState(ctx context.Context, remote client.RunTemplate, prior *runTemplateResourceModel) (runTemplateResourceModel, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	agentIDs, agentDiagnostics := types.SetValueFrom(ctx, types.StringType, remote.AgentIDs)
	diagnostics.Append(agentDiagnostics...)
	personaIDs, personaDiagnostics := types.SetValueFrom(ctx, types.StringType, remote.PersonaIDs)
	diagnostics.Append(personaDiagnostics...)
	testSetIDs, testSetDiagnostics := types.SetValueFrom(ctx, types.StringType, remote.TestSetIDs)
	diagnostics.Append(testSetDiagnostics...)
	metricIDs, metricDiagnostics := types.SetValueFrom(ctx, types.StringType, remote.MetricIDs)
	diagnostics.Append(metricDiagnostics...)
	mutationIDs, mutationDiagnostics := types.SetValueFrom(ctx, types.StringType, remote.MutationIDs)
	diagnostics.Append(mutationDiagnostics...)
	tags, tagDiagnostics := types.SetValueFrom(ctx, types.StringType, remote.Tags)
	diagnostics.Append(tagDiagnostics...)

	metadata := types.DynamicNull()
	var err error
	if prior == nil {
		metadata, err = dynamicFromJSONObject(remote.Metadata)
	} else {
		metadata, err = dynamicFromJSONObjectPreserving(remote.Metadata, prior.Metadata)
	}
	if err != nil {
		diagnostics.AddError("Unable to decode run-template metadata", err.Error())
	}

	state := runTemplateResourceModel{
		ID:              types.StringValue(remote.ID),
		WorkspaceID:     types.StringNull(),
		Name:            types.StringValue(remote.Name),
		DisplayName:     types.StringValue(remote.DisplayName),
		Description:     types.StringValue(remote.Description),
		AgentIDs:        agentIDs,
		PersonaIDs:      personaIDs,
		TestSetIDs:      testSetIDs,
		MetricIDs:       metricIDs,
		MutationIDs:     mutationIDs,
		IterationCount:  types.Int64Value(remote.IterationCount),
		Concurrency:     types.Int64Value(remote.Concurrency),
		SubSampleSize:   types.Int64Value(remote.SubSampleSize),
		SubSampleSeed:   nullableInt(remote.SubSampleSeed),
		Metadata:        metadata,
		Tags:            tags,
		CreateTime:      types.StringValue(remote.CreateTime),
		UpdateTime:      nullableString(remote.UpdateTime),
		CreatedByUserID: nullableString(remote.CreatedByUserID),
	}
	if prior != nil {
		state.WorkspaceID = prior.WorkspaceID
	}
	return state, diagnostics
}
