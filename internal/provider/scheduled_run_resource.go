package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &scheduledRunResource{}
	_ resource.ResourceWithConfigure   = &scheduledRunResource{}
	_ resource.ResourceWithImportState = &scheduledRunResource{}
	_ resource.ResourceWithIdentity    = &scheduledRunResource{}
	_ resource.ResourceWithModifyPlan  = &scheduledRunResource{}
)

type scheduledRunResource struct{ client *client.Client }

type scheduledRunResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	WorkspaceID        types.String `tfsdk:"workspace_id"`
	Name               types.String `tfsdk:"name"`
	DisplayName        types.String `tfsdk:"display_name"`
	RunTemplateID      types.String `tfsdk:"run_template_id"`
	ScheduleExpression types.String `tfsdk:"schedule_expression"`
	ScheduleTimezone   types.String `tfsdk:"schedule_timezone"`
	Enabled            types.Bool   `tfsdk:"enabled"`
	LastRunAt          types.String `tfsdk:"last_run_at"`
	LastRunID          types.String `tfsdk:"last_run_id"`
	CreateTime         types.String `tfsdk:"create_time"`
	UpdateTime         types.String `tfsdk:"update_time"`
}

type scheduledRunIdentityModel struct {
	ID          types.String `tfsdk:"id"`
	WorkspaceID types.String `tfsdk:"workspace_id"`
}

func newScheduledRunResource() resource.Resource { return &scheduledRunResource{} }

func (r *scheduledRunResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduled_run"
}

func (r *scheduledRunResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a recurring Coval scheduled run.",
		Attributes: map[string]schema.Attribute{
			"id":                  schema.StringAttribute{MarkdownDescription: "Server-assigned scheduled-run ID.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"workspace_id":        workspaceResourceAttribute(),
			"name":                schema.StringAttribute{MarkdownDescription: "Canonical API resource name.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"display_name":        schema.StringAttribute{MarkdownDescription: "Human-readable schedule name.", Required: true, Validators: []validator.String{stringvalidator.LengthBetween(1, 200)}},
			"run_template_id":     schema.StringAttribute{MarkdownDescription: "Run-template ID executed by this schedule.", Required: true, Validators: []validator.String{stringvalidator.LengthBetween(22, 22)}},
			"schedule_expression": schema.StringAttribute{MarkdownDescription: "Rate or cron schedule expression, such as rate(1 day) or cron(0 9 ? * MON-FRI *).", Required: true, Validators: []validator.String{stringvalidator.LengthAtMost(200), stringvalidator.RegexMatches(regexp.MustCompile(`^(rate|cron)\(.+\)$`), "must use rate(...) or cron(...) format")}},
			"schedule_timezone":   schema.StringAttribute{MarkdownDescription: "IANA timezone used by cron expressions.", Optional: true, Computed: true, Default: stringdefault.StaticString("UTC"), Validators: []validator.String{stringvalidator.LengthAtMost(100)}},
			"enabled":             schema.BoolAttribute{MarkdownDescription: "Whether the schedule is active.", Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"last_run_at":         schema.StringAttribute{MarkdownDescription: "RFC 3339 timestamp of the latest produced run when available.", Computed: true},
			"last_run_id":         schema.StringAttribute{MarkdownDescription: "ID of the latest produced run when available.", Computed: true},
			"create_time":         schema.StringAttribute{MarkdownDescription: "RFC 3339 creation timestamp.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"update_time":         schema.StringAttribute{MarkdownDescription: "RFC 3339 timestamp of the latest update.", Computed: true},
		},
	}
}

func (r *scheduledRunResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	useStateForUnchangedPlan(ctx, req, resp, path.Root("update_time"))
}
func (r *scheduledRunResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{Attributes: map[string]identityschema.Attribute{"id": identityschema.StringAttribute{RequiredForImport: true}, "workspace_id": identityschema.StringAttribute{OptionalForImport: true}}}
}
func (r *scheduledRunResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *scheduledRunResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scheduledRunResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := clientForWorkspace(r.client, plan.WorkspaceID).CreateScheduledRun(ctx, scheduledRunInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coval scheduled run", err.Error())
		return
	}
	state := scheduledRunState(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, scheduledRunIdentityModel{ID: state.ID, WorkspaceID: state.WorkspaceID})...)
}
func (r *scheduledRunResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scheduledRunResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := clientForWorkspace(r.client, state.WorkspaceID).GetScheduledRun(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coval scheduled run", err.Error())
		return
	}
	refreshed := scheduledRunState(remote, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, scheduledRunIdentityModel{ID: refreshed.ID, WorkspaceID: refreshed.WorkspaceID})...)
}
func (r *scheduledRunResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan scheduledRunResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := clientForWorkspace(r.client, plan.WorkspaceID).UpdateScheduledRun(ctx, plan.ID.ValueString(), scheduledRunInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Unable to update Coval scheduled run", err.Error())
		return
	}
	state := scheduledRunState(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, scheduledRunIdentityModel{ID: state.ID, WorkspaceID: state.WorkspaceID})...)
}
func (r *scheduledRunResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scheduledRunResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := clientForWorkspace(r.client, state.WorkspaceID).DeleteScheduledRun(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Coval scheduled run", err.Error())
	}
}
func (r *scheduledRunResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	importStateWithWorkspaceIdentity(ctx, req, resp)
}

func scheduledRunInput(plan scheduledRunResourceModel) client.CreateScheduledRunInput {
	return client.CreateScheduledRunInput{DisplayName: plan.DisplayName.ValueString(), RunTemplateID: plan.RunTemplateID.ValueString(), ScheduleExpression: plan.ScheduleExpression.ValueString(), ScheduleTimezone: plan.ScheduleTimezone.ValueString(), Enabled: plan.Enabled.ValueBool()}
}
func scheduledRunState(remote client.ScheduledRun, prior *scheduledRunResourceModel) scheduledRunResourceModel {
	state := scheduledRunResourceModel{ID: types.StringValue(remote.ID), WorkspaceID: types.StringNull(), Name: types.StringValue(remote.Name), DisplayName: types.StringValue(remote.DisplayName), RunTemplateID: types.StringValue(remote.RunTemplateID), ScheduleExpression: types.StringValue(remote.ScheduleExpression), ScheduleTimezone: types.StringValue(remote.ScheduleTimezone), Enabled: types.BoolValue(remote.Enabled), LastRunAt: nullableString(remote.LastRunAt), LastRunID: nullableString(remote.LastRunID), CreateTime: types.StringValue(remote.CreateTime), UpdateTime: nullableString(remote.UpdateTime)}
	if prior != nil {
		state.WorkspaceID = prior.WorkspaceID
	}
	return state
}
