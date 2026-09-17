package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                   = &personaResource{}
	_ resource.ResourceWithConfigure      = &personaResource{}
	_ resource.ResourceWithImportState    = &personaResource{}
	_ resource.ResourceWithIdentity       = &personaResource{}
	_ resource.ResourceWithValidateConfig = &personaResource{}

	multiPhoneConfigAttributeTypes = map[string]attr.Type{
		"phone_number_index": types.Int64Type,
		"phone_number_name":  types.StringType,
	}
	audioDegradationAttributeTypes = map[string]attr.Type{
		"preset":         types.StringType,
		"preset_version": types.Int64Type,
	}
)

type personaResource struct {
	client *client.Client
}

type personaResourceModel struct {
	ID                       types.String  `tfsdk:"id"`
	ResourceName             types.String  `tfsdk:"resource_name"`
	Name                     types.String  `tfsdk:"name"`
	PersonaPrompt            types.String  `tfsdk:"persona_prompt"`
	VoiceName                types.String  `tfsdk:"voice_name"`
	LanguageCode             types.String  `tfsdk:"language_code"`
	SilentMode               types.Bool    `tfsdk:"silent_mode"`
	MultiPhoneConfig         types.Object  `tfsdk:"multi_phone_config"`
	InitializationParameters types.Dynamic `tfsdk:"initialization_parameters"`
	CustomPersonaData        types.String  `tfsdk:"custom_persona_data"`
	Voice                    types.String  `tfsdk:"voice"`
	CustomVoiceID            types.String  `tfsdk:"custom_voice_id"`
	BackgroundSound          types.String  `tfsdk:"background_sound"`
	BackgroundSoundVolume    types.Float64 `tfsdk:"background_sound_volume"`
	VoiceVolume              types.Float64 `tfsdk:"voice_volume"`
	VoiceSpeed               types.Float64 `tfsdk:"voice_speed"`
	WaitSeconds              types.Float64 `tfsdk:"wait_seconds"`
	ConversationInitiation   types.String  `tfsdk:"conversation_initiation"`
	InterruptionRate         types.String  `tfsdk:"interruption_rate"`
	MultiLanguageSTT         types.Bool    `tfsdk:"multi_language_stt"`
	HoldMusicTimeoutSeconds  types.Float64 `tfsdk:"hold_music_timeout_seconds"`
	SituateSpeaker           types.String  `tfsdk:"situate_speaker"`
	AudioDegradation         types.Object  `tfsdk:"audio_degradation"`
	Tags                     types.Set     `tfsdk:"tags"`
	CreateTime               types.String  `tfsdk:"create_time"`
	UpdateTime               types.String  `tfsdk:"update_time"`
}

type personaIdentityModel struct {
	ID types.String `tfsdk:"id"`
}

type multiPhoneConfigModel struct {
	PhoneNumberIndex types.Int64  `tfsdk:"phone_number_index"`
	PhoneNumberName  types.String `tfsdk:"phone_number_name"`
}

type audioDegradationModel struct {
	Preset        types.String `tfsdk:"preset"`
	PresetVersion types.Int64  `tfsdk:"preset_version"`
}

func newPersonaResource() resource.Resource {
	return &personaResource{}
}

func (r *personaResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_persona"
}

func (r *personaResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Coval simulated persona.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Server-assigned persona ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"resource_name": schema.StringAttribute{
				MarkdownDescription: "Canonical API resource name.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable persona name.",
				Required:            true,
				Validators:          []validator.String{stringvalidator.LengthBetween(1, 200)},
			},
			"persona_prompt": schema.StringAttribute{
				MarkdownDescription: "Instructions describing the persona's behavior and personality.",
				Optional:            true,
			},
			"voice_name": schema.StringAttribute{
				MarkdownDescription: "Coval voice name. Use the public voices endpoint to discover supported voice and language combinations.",
				Required:            true,
			},
			"language_code": schema.StringAttribute{
				MarkdownDescription: "BCP-47 language code supported by the selected voice.",
				Required:            true,
			},
			"silent_mode": schema.BoolAttribute{
				MarkdownDescription: "Whether the persona remains silent for the entire simulation.",
				Optional:            true,
			},
			"multi_phone_config": schema.SingleNestedAttribute{
				MarkdownDescription: "Caller-number selection. Omit it to let the simulation choose a random available number.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"phone_number_index": schema.Int64Attribute{
						MarkdownDescription: "One-based index of the caller number returned by the public phone-numbers endpoint.",
						Required:            true,
						Validators:          []validator.Int64{int64validator.AtLeast(1)},
					},
					"phone_number_name": schema.StringAttribute{
						MarkdownDescription: "Optional display name for the selected caller number.",
						Optional:            true,
					},
				},
			},
			"initialization_parameters": schema.DynamicAttribute{
				MarkdownDescription: "Arbitrary JSON object containing persona-level initialization parameters. Set {} to replace existing parameters with an empty object.",
				Optional:            true,
			},
			"custom_persona_data": schema.StringAttribute{
				MarkdownDescription: "Serialized JSON object sent as custom persona data for compatible chat agents.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.LengthAtMost(16383)},
			},
			"voice": schema.StringAttribute{
				MarkdownDescription: "Agent voice override for OpenAI Realtime endpoint simulations.",
				Optional:            true,
				Validators: []validator.String{stringvalidator.OneOf(
					"alloy", "ash", "ballad", "coral", "echo", "marin", "sage", "shimmer", "verse",
				)},
			},
			"custom_voice_id": schema.StringAttribute{
				MarkdownDescription: "Server-issued custom voice reference belonging to the configured organization.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"background_sound": schema.StringAttribute{
				MarkdownDescription: "Built-in background sound ID or a custom sound reference in the form custom:<id>.",
				Optional:            true,
				Validators: []validator.String{stringvalidator.RegexMatches(
					regexp.MustCompile(`^(off|office|lounge|crowd|airport|bus|playground|doorbell|train-arrival|portable-air-conditioner|skatepark|small-dog-bark|cafe|ferry-and-announcement|heavy-rain|moderate-wind|newborn-baby-crying|office-with-alarm|street-with-sirens|construction-work|backchanneling|custom:[A-Za-z0-9_-]+)$`),
					"must be a supported built-in sound or custom:<id>",
				)},
			},
			"background_sound_volume": schema.Float64Attribute{
				MarkdownDescription: "Background-sound volume. Must be zero or greater.",
				Optional:            true,
				Validators:          []validator.Float64{float64validator.AtLeast(0)},
			},
			"voice_volume": schema.Float64Attribute{
				MarkdownDescription: "Voice gain multiplier from 0.0 (silent) through 2.0 (double volume).",
				Optional:            true,
				Validators:          []validator.Float64{float64validator.Between(0, 2)},
			},
			"voice_speed": schema.Float64Attribute{
				MarkdownDescription: "Voice-speed multiplier from 0.25 through 2.0.",
				Optional:            true,
				Validators:          []validator.Float64{float64validator.Between(0.25, 2)},
			},
			"wait_seconds": schema.Float64Attribute{
				MarkdownDescription: "Response delay in seconds.",
				Optional:            true,
				Validators:          []validator.Float64{float64validator.Between(0.1, 2)},
			},
			"conversation_initiation": schema.StringAttribute{
				MarkdownDescription: "Who initiates the conversation.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.OneOf("speak_first", "wait_for_user")},
			},
			"interruption_rate": schema.StringAttribute{
				MarkdownDescription: "How often the persona interrupts the agent.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("NONE"),
				Validators:          []validator.String{stringvalidator.OneOf("NONE", "LOW", "MEDIUM", "HIGH")},
			},
			"multi_language_stt": schema.BoolAttribute{
				MarkdownDescription: "Whether multilingual speech-to-text is enabled.",
				Optional:            true,
			},
			"hold_music_timeout_seconds": schema.Float64Attribute{
				MarkdownDescription: "Seconds of no speech before disconnecting a hold-music scenario.",
				Optional:            true,
				Validators:          []validator.Float64{float64validator.Between(5, 300)},
			},
			"situate_speaker": schema.StringAttribute{
				MarkdownDescription: "Persona speaker-placement preset.",
				Optional:            true,
				Validators:          []validator.String{stringvalidator.OneOf("speakerphone-easy", "speakerphone-hard")},
			},
			"audio_degradation": schema.SingleNestedAttribute{
				MarkdownDescription: "Channel degradation preset applied to the persona.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"preset": schema.StringAttribute{
						MarkdownDescription: "Channel degradation preset.",
						Required:            true,
						Validators:          []validator.String{stringvalidator.OneOf("landline", "cell-poor", "cell-handoff")},
					},
					"preset_version": schema.Int64Attribute{
						MarkdownDescription: "Version of the channel degradation preset.",
						Optional:            true,
					},
				},
			},
			"tags": schema.SetAttribute{
				MarkdownDescription: "Tags associated with the persona. Set [] to clear them.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
				Validators:          personaTagValidators(),
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

func personaTagValidators() []validator.Set {
	return []validator.Set{
		setvalidator.SizeAtMost(20),
		setvalidator.ValueStringsAre(
			stringvalidator.LengthAtMost(200),
			stringvalidator.RegexMatches(regexp.MustCompile(`\S`), "must contain at least one non-whitespace character"),
		),
	}
}

func (r *personaResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"id": identityschema.StringAttribute{RequiredForImport: true},
		},
	}
}

func (r *personaResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *personaResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config personaResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.CustomPersonaData.IsNull() || config.CustomPersonaData.IsUnknown() {
		return
	}
	if !isJSONObject(config.CustomPersonaData.ValueString()) {
		resp.Diagnostics.AddAttributeError(
			path.Root("custom_persona_data"),
			"Invalid custom persona data",
			"custom_persona_data must be a serialized JSON object.",
		)
	}
}

func isJSONObject(value string) bool {
	var object map[string]any
	return json.Unmarshal([]byte(value), &object) == nil && object != nil
}

func (r *personaResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan personaResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, diagnostics := createPersonaInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := r.client.CreatePersona(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Coval persona", err.Error())
		return
	}
	state, diagnostics := personaState(ctx, created)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, personaIdentityModel{ID: state.ID})...)
}

func (r *personaResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state personaResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	remote, err := r.client.GetPersona(ctx, state.ID.ValueString())
	if client.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to read Coval persona", err.Error())
		return
	}
	refreshed, diagnostics := personaState(ctx, remote)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &refreshed)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, personaIdentityModel{ID: refreshed.ID})...)
}

func (r *personaResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan personaResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input, diagnostics := updatePersonaInput(ctx, plan)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	updated, err := r.client.UpdatePersona(ctx, plan.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update Coval persona", err.Error())
		return
	}
	state, diagnostics := personaState(ctx, updated)
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Identity.Set(ctx, personaIdentityModel{ID: state.ID})...)
}

func (r *personaResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state personaResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeletePersona(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Unable to delete Coval persona", err.Error())
	}
}

func (r *personaResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughWithIdentity(ctx, path.Root("id"), path.Root("id"), req, resp)
}

func createPersonaInput(ctx context.Context, plan personaResourceModel) (client.CreatePersonaInput, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	multiPhoneConfig, multiPhoneDiagnostics := personaMultiPhoneConfig(ctx, plan.MultiPhoneConfig)
	diagnostics.Append(multiPhoneDiagnostics...)
	initializationParameters, err := dynamicJSONObject(plan.InitializationParameters)
	if err != nil {
		diagnostics.AddAttributeError(path.Root("initialization_parameters"), "Invalid initialization parameters", err.Error())
	}
	audioDegradation, audioDiagnostics := personaAudioDegradation(ctx, plan.AudioDegradation)
	diagnostics.Append(audioDiagnostics...)
	tags, tagDiagnostics := stringSet(ctx, plan.Tags)
	diagnostics.Append(tagDiagnostics...)

	return client.CreatePersonaInput{
		SilentMode:               boolPointer(plan.SilentMode),
		MultiPhoneConfig:         multiPhoneConfig,
		InitializationParameters: initializationParameters,
		CustomPersonaData:        stringPointer(plan.CustomPersonaData),
		Voice:                    stringPointer(plan.Voice),
		CustomVoiceID:            stringPointer(plan.CustomVoiceID),
		Name:                     plan.Name.ValueString(),
		PersonaPrompt:            stringPointer(plan.PersonaPrompt),
		VoiceName:                plan.VoiceName.ValueString(),
		LanguageCode:             plan.LanguageCode.ValueString(),
		BackgroundSound:          stringPointer(plan.BackgroundSound),
		BackgroundSoundVolume:    float64Pointer(plan.BackgroundSoundVolume),
		VoiceVolume:              float64Pointer(plan.VoiceVolume),
		VoiceSpeed:               float64Pointer(plan.VoiceSpeed),
		WaitSeconds:              float64Pointer(plan.WaitSeconds),
		ConversationInitiation:   stringPointer(plan.ConversationInitiation),
		InterruptionRate:         stringPointer(plan.InterruptionRate),
		MultiLanguageSTT:         boolPointer(plan.MultiLanguageSTT),
		HoldMusicTimeoutSeconds:  float64Pointer(plan.HoldMusicTimeoutSeconds),
		SituateSpeaker:           stringPointer(plan.SituateSpeaker),
		AudioDegradation:         audioDegradation,
		Tags:                     tags,
	}, diagnostics
}

func updatePersonaInput(ctx context.Context, plan personaResourceModel) (client.UpdatePersonaInput, diag.Diagnostics) {
	createInput, diagnostics := createPersonaInput(ctx, plan)
	tags := []string(nil)
	if createInput.Tags != nil {
		tags = *createInput.Tags
	}
	return client.UpdatePersonaInput{
		SilentMode:               createInput.SilentMode,
		MultiPhoneConfig:         createInput.MultiPhoneConfig,
		InitializationParameters: createInput.InitializationParameters,
		CustomPersonaData:        createInput.CustomPersonaData,
		Voice:                    createInput.Voice,
		CustomVoiceID:            createInput.CustomVoiceID,
		Name:                     createInput.Name,
		PersonaPrompt:            createInput.PersonaPrompt,
		VoiceName:                createInput.VoiceName,
		LanguageCode:             createInput.LanguageCode,
		BackgroundSound:          createInput.BackgroundSound,
		BackgroundSoundVolume:    createInput.BackgroundSoundVolume,
		VoiceVolume:              createInput.VoiceVolume,
		VoiceSpeed:               createInput.VoiceSpeed,
		WaitSeconds:              createInput.WaitSeconds,
		ConversationInitiation:   createInput.ConversationInitiation,
		InterruptionRate:         plan.InterruptionRate.ValueString(),
		MultiLanguageSTT:         createInput.MultiLanguageSTT,
		HoldMusicTimeoutSeconds:  createInput.HoldMusicTimeoutSeconds,
		SituateSpeaker:           createInput.SituateSpeaker,
		AudioDegradation:         createInput.AudioDegradation,
		Tags:                     tags,
	}, diagnostics
}

func boolPointer(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueBool()
	return &result
}

func float64Pointer(value types.Float64) *float64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	result := value.ValueFloat64()
	return &result
}

func personaMultiPhoneConfig(ctx context.Context, value types.Object) (*client.MultiPhoneConfig, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var model multiPhoneConfigModel
	diagnostics := value.As(ctx, &model, basetypes.ObjectAsOptions{})
	if diagnostics.HasError() {
		return nil, diagnostics
	}
	return &client.MultiPhoneConfig{
		PhoneNumberIndex: model.PhoneNumberIndex.ValueInt64(),
		PhoneNumberName:  stringPointer(model.PhoneNumberName),
	}, diagnostics
}

func personaAudioDegradation(ctx context.Context, value types.Object) (*client.AudioDegradationConfig, diag.Diagnostics) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	var model audioDegradationModel
	diagnostics := value.As(ctx, &model, basetypes.ObjectAsOptions{})
	if diagnostics.HasError() {
		return nil, diagnostics
	}
	var version *int64
	if !model.PresetVersion.IsNull() && !model.PresetVersion.IsUnknown() {
		value := model.PresetVersion.ValueInt64()
		version = &value
	}
	return &client.AudioDegradationConfig{
		Preset:        model.Preset.ValueString(),
		PresetVersion: version,
	}, diagnostics
}

func personaState(ctx context.Context, remote client.Persona) (personaResourceModel, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	initializationParameters, err := dynamicFromNullableJSONObject(remote.InitializationParameters)
	if err != nil {
		diagnostics.AddError("Unable to decode persona initialization parameters", err.Error())
	}
	multiPhoneConfig, multiPhoneDiagnostics := personaMultiPhoneConfigState(remote.MultiPhoneConfig)
	diagnostics.Append(multiPhoneDiagnostics...)
	audioDegradation, audioDiagnostics := personaAudioDegradationState(remote.AudioDegradation)
	diagnostics.Append(audioDiagnostics...)
	tags, tagDiagnostics := types.SetValueFrom(ctx, types.StringType, remote.Tags)
	diagnostics.Append(tagDiagnostics...)

	state := personaResourceModel{
		ID:                       types.StringValue(remote.ID),
		ResourceName:             types.StringValue(remote.ResourceName),
		Name:                     types.StringValue(remote.Name),
		PersonaPrompt:            nullableString(remote.PersonaPrompt),
		VoiceName:                nullableString(remote.VoiceName),
		LanguageCode:             nullableString(remote.LanguageCode),
		SilentMode:               nullableBool(remote.SilentMode),
		MultiPhoneConfig:         multiPhoneConfig,
		InitializationParameters: initializationParameters,
		CustomPersonaData:        nullableString(remote.CustomPersonaData),
		Voice:                    nullableString(remote.Voice),
		CustomVoiceID:            nullableString(remote.CustomVoiceID),
		BackgroundSound:          nullableString(remote.BackgroundSound),
		BackgroundSoundVolume:    nullableFloat64(remote.BackgroundSoundVolume),
		VoiceVolume:              nullableFloat64(remote.VoiceVolume),
		VoiceSpeed:               nullableFloat64(remote.VoiceSpeed),
		WaitSeconds:              nullableFloat64(remote.WaitSeconds),
		ConversationInitiation:   nullableString(remote.ConversationInitiation),
		InterruptionRate:         types.StringValue(remote.InterruptionRate),
		MultiLanguageSTT:         nullableBool(remote.MultiLanguageSTT),
		HoldMusicTimeoutSeconds:  nullableFloat64(remote.HoldMusicTimeoutSeconds),
		SituateSpeaker:           nullableString(remote.SituateSpeaker),
		AudioDegradation:         audioDegradation,
		Tags:                     tags,
		CreateTime:               types.StringValue(remote.CreateTime),
		UpdateTime:               nullableString(remote.UpdateTime),
	}
	return state, diagnostics
}

func personaMultiPhoneConfigState(remote *client.MultiPhoneConfig) (types.Object, diag.Diagnostics) {
	if remote == nil {
		return types.ObjectNull(multiPhoneConfigAttributeTypes), nil
	}
	return types.ObjectValue(multiPhoneConfigAttributeTypes, map[string]attr.Value{
		"phone_number_index": types.Int64Value(remote.PhoneNumberIndex),
		"phone_number_name":  nullableString(remote.PhoneNumberName),
	})
}

func personaAudioDegradationState(remote *client.AudioDegradationConfig) (types.Object, diag.Diagnostics) {
	if remote == nil {
		return types.ObjectNull(audioDegradationAttributeTypes), nil
	}
	presetVersion := types.Int64Null()
	if remote.PresetVersion != nil {
		presetVersion = types.Int64Value(*remote.PresetVersion)
	}
	return types.ObjectValue(audioDegradationAttributeTypes, map[string]attr.Value{
		"preset":         types.StringValue(remote.Preset),
		"preset_version": presetVersion,
	})
}

func nullableString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

func nullableBool(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*value)
}

func nullableFloat64(value *float64) types.Float64 {
	if value == nil {
		return types.Float64Null()
	}
	return types.Float64Value(*value)
}
