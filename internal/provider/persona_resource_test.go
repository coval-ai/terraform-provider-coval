package provider

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	frameworkdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	frameworkresource "github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPersonaAudioDegradationVersionAcceptsServerSelection(t *testing.T) {
	t.Parallel()

	var response frameworkresource.SchemaResponse
	newPersonaResource().Schema(context.Background(), frameworkresource.SchemaRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", response.Diagnostics)
	}

	audioDegradation, ok := response.Schema.Attributes["audio_degradation"].(resourceschema.SingleNestedAttribute)
	if !ok {
		t.Fatalf("audio_degradation schema = %T", response.Schema.Attributes["audio_degradation"])
	}
	presetVersion, ok := audioDegradation.Attributes["preset_version"].(resourceschema.Int64Attribute)
	if !ok {
		t.Fatalf("preset_version schema = %T", audioDegradation.Attributes["preset_version"])
	}
	if !presetVersion.IsOptional() || !presetVersion.IsComputed() {
		t.Fatalf("preset_version must be optional and computed: %#v", presetVersion)
	}
}

func TestPersonaSchemasMatchPublicStringAndTagConstraints(t *testing.T) {
	t.Parallel()

	var resourceResponse frameworkresource.SchemaResponse
	newPersonaResource().Schema(context.Background(), frameworkresource.SchemaRequest{}, &resourceResponse)
	backgroundSound, ok := resourceResponse.Schema.Attributes["background_sound"].(resourceschema.StringAttribute)
	if !ok {
		t.Fatalf("background_sound schema = %T", resourceResponse.Schema.Attributes["background_sound"])
	}
	if diagnostics := validateString(t.Context(), backgroundSound.Validators, "custom:"+strings.Repeat("x", 93)); diagnostics.HasError() {
		t.Errorf("100-character background_sound was rejected: %v", diagnostics)
	}
	if diagnostics := validateString(t.Context(), backgroundSound.Validators, "custom:"+strings.Repeat("x", 94)); !diagnostics.HasError() {
		t.Error("background_sound longer than 100 characters was accepted")
	}

	var dataSourceResponse frameworkdatasource.SchemaResponse
	newPersonasDataSource().Schema(context.Background(), frameworkdatasource.SchemaRequest{}, &dataSourceResponse)
	tagFilters, ok := dataSourceResponse.Schema.Attributes["tag_filters"].(datasourceschema.SetAttribute)
	if !ok {
		t.Fatalf("tag_filters schema = %T", dataSourceResponse.Schema.Attributes["tag_filters"])
	}
	whitespaceFilter := types.SetValueMust(types.StringType, []attr.Value{types.StringValue("   ")})
	if diagnostics := validateSet(t.Context(), tagFilters.Validators, whitespaceFilter); diagnostics.HasError() {
		t.Errorf("public contract permits whitespace tag filters: %v", diagnostics)
	}
	longFilter := types.SetValueMust(types.StringType, []attr.Value{types.StringValue(strings.Repeat("x", 201))})
	if diagnostics := validateSet(t.Context(), tagFilters.Validators, longFilter); !diagnostics.HasError() {
		t.Error("tag filter longer than 200 characters was accepted")
	}
}

func TestCreatePersonaInputConvertsNestedAndDynamicValues(t *testing.T) {
	t.Parallel()

	multiPhone, diagnostics := types.ObjectValue(multiPhoneConfigAttributeTypes, map[string]attr.Value{
		"phone_number_index": types.Int64Value(2),
		"phone_number_name":  types.StringValue("secondary"),
	})
	if diagnostics.HasError() {
		t.Fatalf("build multi-phone config: %v", diagnostics)
	}
	audioDegradation, diagnostics := types.ObjectValue(audioDegradationAttributeTypes, map[string]attr.Value{
		"preset":         types.StringValue("cell-poor"),
		"preset_version": types.Int64Value(1),
	})
	if diagnostics.HasError() {
		t.Fatalf("build audio degradation: %v", diagnostics)
	}
	parametersObject, diagnostics := types.ObjectValue(
		map[string]attr.Type{"customer_tier": types.StringType},
		map[string]attr.Value{"customer_tier": types.StringValue("premium")},
	)
	if diagnostics.HasError() {
		t.Fatalf("build initialization parameters: %v", diagnostics)
	}
	tags, diagnostics := types.SetValueFrom(context.Background(), types.StringType, []string{"terraform", "voice"})
	if diagnostics.HasError() {
		t.Fatalf("build tags: %v", diagnostics)
	}

	input, diagnostics := createPersonaInput(context.Background(), personaResourceModel{
		Name:                     types.StringValue("Provider persona"),
		VoiceName:                types.StringValue("aria"),
		LanguageCode:             types.StringValue("en-US"),
		SilentMode:               types.BoolValue(false),
		MultiPhoneConfig:         multiPhone,
		InitializationParameters: types.DynamicValue(parametersObject),
		InterruptionRate:         types.StringValue("NONE"),
		AudioDegradation:         audioDegradation,
		Tags:                     tags,
	})
	if diagnostics.HasError() {
		t.Fatalf("createPersonaInput() diagnostics: %v", diagnostics)
	}
	if input.MultiPhoneConfig == nil || input.MultiPhoneConfig.PhoneNumberIndex != 2 || input.MultiPhoneConfig.PhoneNumberName == nil || *input.MultiPhoneConfig.PhoneNumberName != "secondary" {
		t.Errorf("MultiPhoneConfig = %#v", input.MultiPhoneConfig)
	}
	if input.AudioDegradation == nil || input.AudioDegradation.Preset != "cell-poor" || input.AudioDegradation.PresetVersion == nil || *input.AudioDegradation.PresetVersion != 1 {
		t.Errorf("AudioDegradation = %#v", input.AudioDegradation)
	}
	if input.InitializationParameters == nil || string(*input.InitializationParameters) != `{"customer_tier":"premium"}` {
		t.Errorf("InitializationParameters = %s", valueOrNil(input.InitializationParameters))
	}
	if input.Tags == nil || len(*input.Tags) != 2 {
		t.Errorf("Tags = %#v", input.Tags)
	}
}

func TestPersonaStatePreservesNullableFields(t *testing.T) {
	t.Parallel()

	state, diagnostics := personaState(context.Background(), client.Persona{
		ResourceName:             "personas/abc123def456ghi789jklm",
		ID:                       "abc123def456ghi789jklm",
		Name:                     "Provider persona",
		VoiceName:                stringValue("aria"),
		LanguageCode:             stringValue("en-US"),
		InitializationParameters: json.RawMessage(`{"customer_tier":"premium"}`),
		InterruptionRate:         "NONE",
		Tags:                     []string{},
		CreateTime:               "2026-09-17T00:00:00Z",
	})
	if diagnostics.HasError() {
		t.Fatalf("personaState() diagnostics: %v", diagnostics)
	}
	if !state.PersonaPrompt.IsNull() || !state.AudioDegradation.IsNull() || !state.MultiPhoneConfig.IsNull() {
		t.Errorf("nullable fields were not null: %#v", state)
	}
	if state.InitializationParameters.IsNull() || state.InitializationParameters.IsUnknown() {
		t.Fatal("initialization_parameters was not decoded")
	}
	if state.InterruptionRate.ValueString() != "NONE" || !state.UpdateTime.IsNull() {
		t.Errorf("state = %#v", state)
	}
}

func TestIsJSONObject(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		value string
		want  bool
	}{
		{name: "object", value: `{"customer_tier":"premium"}`, want: true},
		{name: "empty object", value: `{}`, want: true},
		{name: "array", value: `[]`, want: false},
		{name: "null", value: `null`, want: false},
		{name: "invalid JSON", value: `{`, want: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := isJSONObject(test.value); got != test.want {
				t.Errorf("isJSONObject(%q) = %t, want %t", test.value, got, test.want)
			}
		})
	}
}

func stringValue(value string) *string {
	return &value
}

func valueOrNil(value *json.RawMessage) string {
	if value == nil {
		return "<nil>"
	}
	return string(*value)
}
