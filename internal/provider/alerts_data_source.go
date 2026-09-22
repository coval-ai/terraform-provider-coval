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
	_ datasource.DataSource              = &alertsDataSource{}
	_ datasource.DataSourceWithConfigure = &alertsDataSource{}
)

type alertsDataSource struct{ client *client.Client }

type alertsDataSourceModel struct {
	WorkspaceID        types.String        `tfsdk:"workspace_id"`
	ConversationSource types.String        `tfsdk:"conversation_source"`
	Alerts             []alertSummaryModel `tfsdk:"alerts"`
}

type alertSummaryModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Description           types.String `tfsdk:"description"`
	Status                types.String `tfsdk:"status"`
	EvaluationType        types.String `tfsdk:"evaluation_type"`
	ConversationSource    types.String `tfsdk:"conversation_source"`
	MatchMode             types.String `tfsdk:"match_mode"`
	CooldownSeconds       types.Int64  `tfsdk:"cooldown_seconds"`
	CustomMessageTemplate types.String `tfsdk:"custom_message_template"`
	AgentIDs              types.Set    `tfsdk:"agent_ids"`
	RequiredTags          types.Set    `tfsdk:"required_tags"`
	ScheduledRunIDs       types.Set    `tfsdk:"scheduled_run_ids"`
	TriggerCount          types.Int64  `tfsdk:"trigger_count"`
	LastTriggeredAt       types.String `tfsdk:"last_triggered_at"`
	CreateTime            types.String `tfsdk:"create_time"`
	UpdateTime            types.String `tfsdk:"update_time"`
}

func newAlertsDataSource() datasource.DataSource { return &alertsDataSource{} }

func (d *alertsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alerts"
}

func (d *alertsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Coval alert summaries visible to the configured API key.",
		Attributes: map[string]schema.Attribute{
			"workspace_id":        workspaceDataSourceAttribute(),
			"conversation_source": schema.StringAttribute{MarkdownDescription: "Optional conversation-source filter.", Optional: true, Validators: []validator.String{stringvalidator.OneOf("ALL", "UPLOADED", "SIMULATED")}},
			"alerts": schema.ListNestedAttribute{
				MarkdownDescription: "All matching alert summaries. Use the singular coval_alert data source to retrieve conditions and sensitive channel configuration.", Computed: true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{Computed: true}, "name": schema.StringAttribute{Computed: true}, "description": schema.StringAttribute{Computed: true},
					"status": schema.StringAttribute{Computed: true}, "evaluation_type": schema.StringAttribute{Computed: true}, "conversation_source": schema.StringAttribute{Computed: true},
					"match_mode": schema.StringAttribute{Computed: true}, "cooldown_seconds": schema.Int64Attribute{Computed: true}, "custom_message_template": schema.StringAttribute{Computed: true},
					"agent_ids": schema.SetAttribute{ElementType: types.StringType, Computed: true}, "required_tags": schema.SetAttribute{ElementType: types.StringType, Computed: true},
					"scheduled_run_ids": schema.SetAttribute{ElementType: types.StringType, Computed: true}, "trigger_count": schema.Int64Attribute{Computed: true},
					"last_triggered_at": schema.StringAttribute{Computed: true}, "create_time": schema.StringAttribute{Computed: true}, "update_time": schema.StringAttribute{Computed: true},
				}},
			},
		},
	}
}

func (d *alertsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *alertsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config alertsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	options := client.ListAlertsOptions{ConversationSource: config.ConversationSource.ValueString(), PageSize: 100}
	all, err := listAllAlerts(ctx, clientForWorkspace(d.client, config.WorkspaceID), options)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coval alerts", err.Error())
		return
	}
	config.Alerts = make([]alertSummaryModel, len(all))
	for index, remote := range all {
		state, diagnostics := alertState(ctx, remote, nil)
		resp.Diagnostics.Append(diagnostics...)
		config.Alerts[index] = alertSummaryModel{
			ID: state.ID, Name: state.Name, Description: state.Description, Status: state.Status, EvaluationType: state.EvaluationType,
			ConversationSource: state.ConversationSource, MatchMode: state.MatchMode, CooldownSeconds: state.CooldownSeconds, CustomMessageTemplate: state.CustomMessageTemplate,
			AgentIDs: state.AgentIDs, RequiredTags: state.RequiredTags, ScheduledRunIDs: state.ScheduledRunIDs, TriggerCount: state.TriggerCount,
			LastTriggeredAt: state.LastTriggeredAt, CreateTime: state.CreateTime, UpdateTime: state.UpdateTime,
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func listAllAlerts(ctx context.Context, apiClient *client.Client, options client.ListAlertsOptions) ([]client.Alert, error) {
	var all []client.Alert
	seenTokens := map[string]struct{}{}
	for {
		page, err := apiClient.ListAlerts(ctx, options)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Alerts...)
		if page.NextPageToken == nil || *page.NextPageToken == "" {
			return all, nil
		}
		if _, duplicate := seenTokens[*page.NextPageToken]; duplicate {
			return nil, fmt.Errorf("coval returned repeated pagination token %q", *page.NextPageToken)
		}
		seenTokens[*page.NextPageToken] = struct{}{}
		options.PageToken = *page.NextPageToken
	}
}
