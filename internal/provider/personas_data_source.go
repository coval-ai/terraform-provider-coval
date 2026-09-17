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
	_ datasource.DataSource              = &personasDataSource{}
	_ datasource.DataSourceWithConfigure = &personasDataSource{}
)

type personasDataSource struct {
	client *client.Client
}

type personasDataSourceModel struct {
	Filter     types.String          `tfsdk:"filter"`
	OrderBy    types.String          `tfsdk:"order_by"`
	TagFilters types.Set             `tfsdk:"tag_filters"`
	Personas   []personaSummaryModel `tfsdk:"personas"`
}

type personaSummaryModel struct {
	ID                      types.String  `tfsdk:"id"`
	ResourceName            types.String  `tfsdk:"resource_name"`
	Name                    types.String  `tfsdk:"name"`
	PersonaPrompt           types.String  `tfsdk:"persona_prompt"`
	VoiceName               types.String  `tfsdk:"voice_name"`
	LanguageCode            types.String  `tfsdk:"language_code"`
	SilentMode              types.Bool    `tfsdk:"silent_mode"`
	MultiPhoneConfig        types.Object  `tfsdk:"multi_phone_config"`
	CustomPersonaData       types.String  `tfsdk:"custom_persona_data"`
	Voice                   types.String  `tfsdk:"voice"`
	CustomVoiceID           types.String  `tfsdk:"custom_voice_id"`
	BackgroundSound         types.String  `tfsdk:"background_sound"`
	BackgroundSoundVolume   types.Float64 `tfsdk:"background_sound_volume"`
	VoiceVolume             types.Float64 `tfsdk:"voice_volume"`
	VoiceSpeed              types.Float64 `tfsdk:"voice_speed"`
	WaitSeconds             types.Float64 `tfsdk:"wait_seconds"`
	ConversationInitiation  types.String  `tfsdk:"conversation_initiation"`
	InterruptionRate        types.String  `tfsdk:"interruption_rate"`
	MultiLanguageSTT        types.Bool    `tfsdk:"multi_language_stt"`
	HoldMusicTimeoutSeconds types.Float64 `tfsdk:"hold_music_timeout_seconds"`
	SituateSpeaker          types.String  `tfsdk:"situate_speaker"`
	AudioDegradation        types.Object  `tfsdk:"audio_degradation"`
	Tags                    types.Set     `tfsdk:"tags"`
	CreateTime              types.String  `tfsdk:"create_time"`
	UpdateTime              types.String  `tfsdk:"update_time"`
}

func newPersonasDataSource() datasource.DataSource {
	return &personasDataSource{}
}

func (d *personasDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_personas"
}

func (d *personasDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Coval simulated personas visible to the configured API key.",
		Attributes: map[string]schema.Attribute{
			"filter": schema.StringAttribute{
				MarkdownDescription: "Optional public API filter expression. Quote string values containing spaces.",
				Optional:            true,
			},
			"order_by": schema.StringAttribute{
				MarkdownDescription: "Optional ordering expression using create_time, update_time, or name.",
				Optional:            true,
			},
			"tag_filters": schema.SetAttribute{
				MarkdownDescription: "Optional tags that every returned persona must have.",
				ElementType:         types.StringType,
				Optional:            true,
				Validators:          personaTagValidators(),
			},
			"personas": schema.ListNestedAttribute{
				MarkdownDescription: "All matching personas.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: personaSummaryDataSourceAttributes(),
				},
			},
		},
	}
}

func (d *personasDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *personasDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config personasDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags, tagDiagnostics := stringSet(ctx, config.TagFilters)
	resp.Diagnostics.Append(tagDiagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	options := client.ListPersonasOptions{PageSize: 100}
	if !config.Filter.IsNull() && !config.Filter.IsUnknown() {
		options.Filter = config.Filter.ValueString()
	}
	if !config.OrderBy.IsNull() && !config.OrderBy.IsUnknown() {
		options.OrderBy = config.OrderBy.ValueString()
	}
	if tags != nil {
		options.TagFilters = *tags
	}

	all, err := listAllPersonas(ctx, d.client, options)
	if err != nil {
		resp.Diagnostics.AddError("Unable to list Coval personas", err.Error())
		return
	}

	config.Personas = make([]personaSummaryModel, len(all))
	for index, remote := range all {
		state, diagnostics := personaState(ctx, remote)
		resp.Diagnostics.Append(diagnostics...)
		config.Personas[index] = personaSummaryState(state)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func personaSummaryDataSourceAttributes() map[string]schema.Attribute {
	attributes := personaComputedDataSourceAttributes()
	delete(attributes, "initialization_parameters")
	return attributes
}

func personaSummaryState(state personaResourceModel) personaSummaryModel {
	return personaSummaryModel{
		ID:                      state.ID,
		ResourceName:            state.ResourceName,
		Name:                    state.Name,
		PersonaPrompt:           state.PersonaPrompt,
		VoiceName:               state.VoiceName,
		LanguageCode:            state.LanguageCode,
		SilentMode:              state.SilentMode,
		MultiPhoneConfig:        state.MultiPhoneConfig,
		CustomPersonaData:       state.CustomPersonaData,
		Voice:                   state.Voice,
		CustomVoiceID:           state.CustomVoiceID,
		BackgroundSound:         state.BackgroundSound,
		BackgroundSoundVolume:   state.BackgroundSoundVolume,
		VoiceVolume:             state.VoiceVolume,
		VoiceSpeed:              state.VoiceSpeed,
		WaitSeconds:             state.WaitSeconds,
		ConversationInitiation:  state.ConversationInitiation,
		InterruptionRate:        state.InterruptionRate,
		MultiLanguageSTT:        state.MultiLanguageSTT,
		HoldMusicTimeoutSeconds: state.HoldMusicTimeoutSeconds,
		SituateSpeaker:          state.SituateSpeaker,
		AudioDegradation:        state.AudioDegradation,
		Tags:                    state.Tags,
		CreateTime:              state.CreateTime,
		UpdateTime:              state.UpdateTime,
	}
}

func listAllPersonas(ctx context.Context, apiClient *client.Client, options client.ListPersonasOptions) ([]client.Persona, error) {
	var all []client.Persona
	seenTokens := map[string]struct{}{}
	for {
		page, err := apiClient.ListPersonas(ctx, options)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Personas...)
		if page.NextPageToken == "" {
			return all, nil
		}
		if _, duplicate := seenTokens[page.NextPageToken]; duplicate {
			return nil, fmt.Errorf("coval returned repeated pagination token %q", page.NextPageToken)
		}
		seenTokens[page.NextPageToken] = struct{}{}
		options.PageToken = page.NextPageToken
	}
}
