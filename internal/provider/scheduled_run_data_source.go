package provider

import (
	"context"
	"fmt"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	_ datasource.DataSource              = &scheduledRunDataSource{}
	_ datasource.DataSourceWithConfigure = &scheduledRunDataSource{}
)

type scheduledRunDataSource struct{ client *client.Client }

func newScheduledRunDataSource() datasource.DataSource { return &scheduledRunDataSource{} }
func (d *scheduledRunDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduled_run"
}
func scheduledRunComputedAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name": schema.StringAttribute{Computed: true}, "display_name": schema.StringAttribute{Computed: true}, "run_template_id": schema.StringAttribute{Computed: true},
		"schedule_expression": schema.StringAttribute{Computed: true}, "schedule_timezone": schema.StringAttribute{Computed: true}, "enabled": schema.BoolAttribute{Computed: true},
		"last_run_at": schema.StringAttribute{Computed: true}, "last_run_id": schema.StringAttribute{Computed: true}, "create_time": schema.StringAttribute{Computed: true}, "update_time": schema.StringAttribute{Computed: true},
	}
}
func (d *scheduledRunDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := scheduledRunComputedAttributes()
	attributes["workspace_id"] = workspaceDataSourceAttribute()
	attributes["id"] = schema.StringAttribute{MarkdownDescription: "Scheduled-run ID.", Required: true, Validators: []validator.String{stringvalidator.LengthBetween(22, 22)}}
	resp.Schema = schema.Schema{MarkdownDescription: "Retrieves a Coval scheduled run by ID.", Attributes: attributes}
}
func (d *scheduledRunDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *scheduledRunDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config scheduledRunResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	remote, err := clientForWorkspace(d.client, config.WorkspaceID).GetScheduledRun(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coval scheduled run", err.Error())
		return
	}
	state := scheduledRunState(remote, &config)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
