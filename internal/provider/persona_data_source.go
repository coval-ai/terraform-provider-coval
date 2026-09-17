package provider

import (
	"context"
	"fmt"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &personaDataSource{}
	_ datasource.DataSourceWithConfigure = &personaDataSource{}
)

type personaDataSource struct {
	client *client.Client
}

func newPersonaDataSource() datasource.DataSource {
	return &personaDataSource{}
}

func (d *personaDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_persona"
}

func (d *personaDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := personaComputedDataSourceAttributes()
	attributes["id"] = schema.StringAttribute{
		MarkdownDescription: "Server-assigned persona ID.",
		Required:            true,
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Coval simulated persona by ID.",
		Attributes:          attributes,
	}
}

func (d *personaDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *personaDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config personaResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote, err := d.client.GetPersona(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coval persona", err.Error())
		return
	}
	state, diagnostics := personaState(ctx, remote)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func personaComputedDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":                         schema.StringAttribute{MarkdownDescription: "Server-assigned persona ID.", Computed: true},
		"resource_name":              schema.StringAttribute{MarkdownDescription: "Canonical API resource name.", Computed: true},
		"name":                       schema.StringAttribute{MarkdownDescription: "Human-readable persona name.", Computed: true},
		"persona_prompt":             schema.StringAttribute{MarkdownDescription: "Instructions describing the persona's behavior and personality.", Computed: true},
		"voice_name":                 schema.StringAttribute{MarkdownDescription: "Coval voice name.", Computed: true},
		"language_code":              schema.StringAttribute{MarkdownDescription: "BCP-47 language code for voice synthesis.", Computed: true},
		"silent_mode":                schema.BoolAttribute{MarkdownDescription: "Whether the persona remains silent for the entire simulation.", Computed: true},
		"initialization_parameters":  schema.DynamicAttribute{MarkdownDescription: "Persona-level initialization parameters.", Computed: true},
		"custom_persona_data":        schema.StringAttribute{MarkdownDescription: "Serialized JSON object sent as custom persona data for compatible chat agents.", Computed: true},
		"voice":                      schema.StringAttribute{MarkdownDescription: "Agent voice override for OpenAI Realtime endpoint simulations.", Computed: true},
		"custom_voice_id":            schema.StringAttribute{MarkdownDescription: "Server-issued custom voice reference.", Computed: true},
		"background_sound":           schema.StringAttribute{MarkdownDescription: "Background sound ID.", Computed: true},
		"background_sound_volume":    schema.Float64Attribute{MarkdownDescription: "Background-sound volume.", Computed: true},
		"voice_volume":               schema.Float64Attribute{MarkdownDescription: "Voice gain multiplier.", Computed: true},
		"voice_speed":                schema.Float64Attribute{MarkdownDescription: "Voice-speed multiplier.", Computed: true},
		"wait_seconds":               schema.Float64Attribute{MarkdownDescription: "Response delay in seconds.", Computed: true},
		"conversation_initiation":    schema.StringAttribute{MarkdownDescription: "Who initiates the conversation.", Computed: true},
		"interruption_rate":          schema.StringAttribute{MarkdownDescription: "How often the persona interrupts the agent.", Computed: true},
		"multi_language_stt":         schema.BoolAttribute{MarkdownDescription: "Whether multilingual speech-to-text is enabled.", Computed: true},
		"hold_music_timeout_seconds": schema.Float64Attribute{MarkdownDescription: "Seconds of no speech before disconnecting a hold-music scenario.", Computed: true},
		"situate_speaker":            schema.StringAttribute{MarkdownDescription: "Persona speaker-placement preset.", Computed: true},
		"tags":                       schema.SetAttribute{MarkdownDescription: "Tags associated with the persona.", ElementType: types.StringType, Computed: true},
		"create_time":                schema.StringAttribute{MarkdownDescription: "RFC 3339 creation timestamp.", Computed: true},
		"update_time":                schema.StringAttribute{MarkdownDescription: "RFC 3339 timestamp of the latest update.", Computed: true},
		"multi_phone_config": schema.SingleNestedAttribute{
			MarkdownDescription: "Caller-number selection.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"phone_number_index": schema.Int64Attribute{MarkdownDescription: "One-based caller-number index.", Computed: true},
				"phone_number_name":  schema.StringAttribute{MarkdownDescription: "Display name for the selected caller number.", Computed: true},
			},
		},
		"audio_degradation": schema.SingleNestedAttribute{
			MarkdownDescription: "Channel degradation preset applied to the persona.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"preset":         schema.StringAttribute{MarkdownDescription: "Channel degradation preset.", Computed: true},
				"preset_version": schema.Int64Attribute{MarkdownDescription: "Version of the channel degradation preset.", Computed: true},
			},
		},
	}
}
