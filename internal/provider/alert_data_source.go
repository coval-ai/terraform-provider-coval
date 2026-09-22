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
	_ datasource.DataSource              = &alertDataSource{}
	_ datasource.DataSourceWithConfigure = &alertDataSource{}
)

type alertDataSource struct{ client *client.Client }

func newAlertDataSource() datasource.DataSource { return &alertDataSource{} }

func (d *alertDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert"
}

func (d *alertDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a Coval alert by ID.",
		Attributes: map[string]schema.Attribute{
			"workspace_id":            workspaceDataSourceAttribute(),
			"id":                      schema.StringAttribute{MarkdownDescription: "Alert ULID.", Required: true, Validators: []validator.String{stringvalidator.LengthBetween(26, 26)}},
			"name":                    schema.StringAttribute{Computed: true},
			"description":             schema.StringAttribute{Computed: true},
			"status":                  schema.StringAttribute{Computed: true},
			"evaluation_type":         schema.StringAttribute{Computed: true},
			"conversation_source":     schema.StringAttribute{Computed: true},
			"match_mode":              schema.StringAttribute{Computed: true},
			"cooldown_seconds":        schema.Int64Attribute{Computed: true},
			"custom_message_template": schema.StringAttribute{Computed: true},
			"agent_ids":               schema.SetAttribute{ElementType: types.StringType, Computed: true},
			"required_tags":           schema.SetAttribute{ElementType: types.StringType, Computed: true},
			"scheduled_run_ids":       schema.SetAttribute{ElementType: types.StringType, Computed: true},
			"conditions":              schema.DynamicAttribute{MarkdownDescription: "Public API condition objects.", Computed: true},
			"channels":                schema.DynamicAttribute{MarkdownDescription: "Public API notification channel objects.", Computed: true, Sensitive: true},
			"trigger_count":           schema.Int64Attribute{Computed: true},
			"last_triggered_at":       schema.StringAttribute{Computed: true},
			"create_time":             schema.StringAttribute{Computed: true},
			"update_time":             schema.StringAttribute{Computed: true},
		},
	}
}

func (d *alertDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *alertDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config alertResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := clientForWorkspace(d.client, config.WorkspaceID).GetAlert(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coval alert", err.Error())
		return
	}
	state, diagnostics := alertState(ctx, remote, &config)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
