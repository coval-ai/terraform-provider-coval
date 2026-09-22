package provider

import (
	"context"
	"fmt"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &scheduledRunsDataSource{}
	_ datasource.DataSourceWithConfigure = &scheduledRunsDataSource{}
)

type scheduledRunsDataSource struct{ client *client.Client }
type scheduledRunsDataSourceModel struct {
	WorkspaceID   types.String               `tfsdk:"workspace_id"`
	Enabled       types.Bool                 `tfsdk:"enabled"`
	RunTemplateID types.String               `tfsdk:"run_template_id"`
	ScheduledRuns []scheduledRunSummaryModel `tfsdk:"scheduled_runs"`
}
type scheduledRunSummaryModel struct {
	Name               types.String `tfsdk:"name"`
	ID                 types.String `tfsdk:"id"`
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

func newScheduledRunsDataSource() datasource.DataSource { return &scheduledRunsDataSource{} }
func (d *scheduledRunsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduled_runs"
}
func (d *scheduledRunsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Lists Coval scheduled runs visible to the configured API key.", Attributes: map[string]schema.Attribute{
		"workspace_id": workspaceDataSourceAttribute(), "enabled": schema.BoolAttribute{MarkdownDescription: "Optional enabled-state filter.", Optional: true},
		"run_template_id": schema.StringAttribute{MarkdownDescription: "Optional run-template filter.", Optional: true, Validators: []validator.String{stringvalidator.LengthBetween(22, 22)}},
		"scheduled_runs":  schema.ListNestedAttribute{MarkdownDescription: "All matching scheduled runs.", Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: scheduledRunSummaryAttributes()}},
	}}
}
func scheduledRunSummaryAttributes() map[string]schema.Attribute {
	attributes := scheduledRunComputedAttributes()
	attributes["id"] = schema.StringAttribute{Computed: true}
	return attributes
}
func (d *scheduledRunsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *client.Client, got %T.", req.ProviderData))
		return
	}
	d.client = apiClient
}
func (d *scheduledRunsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config scheduledRunsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	options := client.ListScheduledRunsOptions{PageSize: 100, Enabled: boolPointer(config.Enabled), RunTemplateID: config.RunTemplateID.ValueString()}
	all, err := listAllScheduledRuns(ctx, clientForWorkspace(d.client, config.WorkspaceID), options)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coval scheduled runs", err.Error())
		return
	}
	config.ScheduledRuns = make([]scheduledRunSummaryModel, len(all))
	for index, remote := range all {
		state := scheduledRunState(remote, nil)
		config.ScheduledRuns[index] = scheduledRunSummaryModel{Name: state.Name, ID: state.ID, DisplayName: state.DisplayName, RunTemplateID: state.RunTemplateID, ScheduleExpression: state.ScheduleExpression, ScheduleTimezone: state.ScheduleTimezone, Enabled: state.Enabled, LastRunAt: state.LastRunAt, LastRunID: state.LastRunID, CreateTime: state.CreateTime, UpdateTime: state.UpdateTime}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
func listAllScheduledRuns(ctx context.Context, apiClient *client.Client, options client.ListScheduledRunsOptions) ([]client.ScheduledRun, error) {
	var all []client.ScheduledRun
	seen := map[string]struct{}{}
	for {
		page, err := apiClient.ListScheduledRuns(ctx, options)
		if err != nil {
			return nil, err
		}
		all = append(all, page.ScheduledRuns...)
		if page.NextPageToken == nil || *page.NextPageToken == "" {
			return all, nil
		}
		if _, duplicate := seen[*page.NextPageToken]; duplicate {
			return nil, fmt.Errorf("coval returned repeated pagination token %q", *page.NextPageToken)
		}
		seen[*page.NextPageToken] = struct{}{}
		options.PageToken = *page.NextPageToken
	}
}
