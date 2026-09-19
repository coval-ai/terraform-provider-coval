package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// Persona is the Coval public API representation of a simulated persona.
type Persona struct {
	SilentMode               *bool                   `json:"silent_mode"`
	MultiPhoneConfig         *MultiPhoneConfig       `json:"multi_phone_config"`
	InitializationParameters json.RawMessage         `json:"initialization_parameters"`
	CustomPersonaData        *string                 `json:"custom_persona_data"`
	Voice                    *string                 `json:"voice"`
	CustomVoiceID            *string                 `json:"custom_voice_id"`
	ResourceName             string                  `json:"resource_name"`
	ID                       string                  `json:"id"`
	Name                     string                  `json:"name"`
	PersonaPrompt            *string                 `json:"persona_prompt"`
	VoiceName                *string                 `json:"voice_name"`
	LanguageCode             *string                 `json:"language_code"`
	BackgroundSound          *string                 `json:"background_sound"`
	BackgroundSoundVolume    *float64                `json:"background_sound_volume"`
	VoiceVolume              *float64                `json:"voice_volume"`
	VoiceSpeed               *float64                `json:"voice_speed"`
	WaitSeconds              *float64                `json:"wait_seconds"`
	ConversationInitiation   *string                 `json:"conversation_initiation"`
	InterruptionRate         string                  `json:"interruption_rate"`
	MultiLanguageSTT         *bool                   `json:"multi_language_stt"`
	HoldMusicTimeoutSeconds  *float64                `json:"hold_music_timeout_seconds"`
	SituateSpeaker           *string                 `json:"situate_speaker"`
	AudioDegradation         *AudioDegradationConfig `json:"audio_degradation"`
	Tags                     []string                `json:"tags"`
	CreateTime               string                  `json:"create_time"`
	UpdateTime               *string                 `json:"update_time"`
}

// MultiPhoneConfig selects the caller number used for a persona.
type MultiPhoneConfig struct {
	PhoneNumberIndex int64   `json:"phone_number_index"`
	PhoneNumberName  *string `json:"phone_number_name,omitempty"`
}

// AudioDegradationConfig applies a channel degradation preset to a persona.
type AudioDegradationConfig struct {
	Preset        string `json:"preset"`
	PresetVersion *int64 `json:"preset_version,omitempty"`
}

// CreatePersonaInput contains writable persona fields.
type CreatePersonaInput struct {
	SilentMode               *bool                   `json:"silent_mode,omitempty"`
	MultiPhoneConfig         *MultiPhoneConfig       `json:"multi_phone_config,omitempty"`
	InitializationParameters *json.RawMessage        `json:"initialization_parameters,omitempty"`
	CustomPersonaData        *string                 `json:"custom_persona_data,omitempty"`
	Voice                    *string                 `json:"voice,omitempty"`
	CustomVoiceID            *string                 `json:"custom_voice_id,omitempty"`
	Name                     string                  `json:"name"`
	PersonaPrompt            *string                 `json:"persona_prompt,omitempty"`
	VoiceName                string                  `json:"voice_name"`
	LanguageCode             string                  `json:"language_code"`
	BackgroundSound          *string                 `json:"background_sound,omitempty"`
	BackgroundSoundVolume    *float64                `json:"background_sound_volume,omitempty"`
	VoiceVolume              *float64                `json:"voice_volume,omitempty"`
	VoiceSpeed               *float64                `json:"voice_speed,omitempty"`
	WaitSeconds              *float64                `json:"wait_seconds,omitempty"`
	ConversationInitiation   *string                 `json:"conversation_initiation,omitempty"`
	InterruptionRate         *string                 `json:"interruption_rate,omitempty"`
	MultiLanguageSTT         *bool                   `json:"multi_language_stt,omitempty"`
	HoldMusicTimeoutSeconds  *float64                `json:"hold_music_timeout_seconds,omitempty"`
	SituateSpeaker           *string                 `json:"situate_speaker,omitempty"`
	AudioDegradation         *AudioDegradationConfig `json:"audio_degradation,omitempty"`
	Tags                     *[]string               `json:"tags,omitempty"`
}

// UpdatePersonaInput contains the complete writable persona state sent to PATCH.
// Fields intentionally do not use omitempty so Terraform null values clear the
// corresponding nullable API fields.
type UpdatePersonaInput struct {
	SilentMode               *bool                   `json:"silent_mode"`
	MultiPhoneConfig         *MultiPhoneConfig       `json:"multi_phone_config"`
	InitializationParameters *json.RawMessage        `json:"initialization_parameters"`
	CustomPersonaData        *string                 `json:"custom_persona_data"`
	Voice                    *string                 `json:"voice"`
	CustomVoiceID            *string                 `json:"custom_voice_id"`
	Name                     string                  `json:"name"`
	PersonaPrompt            *string                 `json:"persona_prompt"`
	VoiceName                string                  `json:"voice_name"`
	LanguageCode             string                  `json:"language_code"`
	BackgroundSound          *string                 `json:"background_sound"`
	BackgroundSoundVolume    *float64                `json:"background_sound_volume"`
	VoiceVolume              *float64                `json:"voice_volume"`
	VoiceSpeed               *float64                `json:"voice_speed"`
	WaitSeconds              *float64                `json:"wait_seconds"`
	ConversationInitiation   *string                 `json:"conversation_initiation"`
	InterruptionRate         string                  `json:"interruption_rate"`
	MultiLanguageSTT         *bool                   `json:"multi_language_stt"`
	HoldMusicTimeoutSeconds  *float64                `json:"hold_music_timeout_seconds"`
	SituateSpeaker           *string                 `json:"situate_speaker"`
	AudioDegradation         *AudioDegradationConfig `json:"audio_degradation"`
	Tags                     []string                `json:"tags"`
}

// ListPersonasOptions filters and paginates personas.
type ListPersonasOptions struct {
	Filter     string
	PageSize   int
	PageToken  string
	OrderBy    string
	TagFilters []string
}

// ListPersonasOutput is one page of personas.
type ListPersonasOutput struct {
	Personas      []Persona `json:"personas"`
	NextPageToken string    `json:"next_page_token"`
}

type personaEnvelope struct {
	Persona Persona `json:"persona"`
}

// CreatePersona creates a persona.
func (c *Client) CreatePersona(ctx context.Context, input CreatePersonaInput) (Persona, error) {
	var response personaEnvelope
	if err := c.Do(ctx, http.MethodPost, "personas", input, &response); err != nil {
		return Persona{}, err
	}
	return response.Persona, nil
}

// GetPersona retrieves a persona by ID.
func (c *Client) GetPersona(ctx context.Context, id string) (Persona, error) {
	var response personaEnvelope
	if err := c.Do(ctx, http.MethodGet, "personas/"+url.PathEscape(id), nil, &response); err != nil {
		return Persona{}, err
	}
	return response.Persona, nil
}

// UpdatePersona changes a persona in place.
func (c *Client) UpdatePersona(ctx context.Context, id string, input UpdatePersonaInput) (Persona, error) {
	var response personaEnvelope
	if err := c.Do(ctx, http.MethodPatch, "personas/"+url.PathEscape(id), input, &response); err != nil {
		return Persona{}, err
	}
	return response.Persona, nil
}

// DeletePersona deletes a persona.
func (c *Client) DeletePersona(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "personas/"+url.PathEscape(id), nil, nil)
}

// ListPersonas returns one page of personas.
func (c *Client) ListPersonas(ctx context.Context, options ListPersonasOptions) (ListPersonasOutput, error) {
	query := url.Values{}
	if options.Filter != "" {
		query.Set("filter", options.Filter)
	}
	if options.PageSize > 0 {
		query.Set("page_size", strconv.Itoa(options.PageSize))
	}
	if options.PageToken != "" {
		query.Set("page_token", options.PageToken)
	}
	if options.OrderBy != "" {
		query.Set("order_by", options.OrderBy)
	}
	for _, tag := range options.TagFilters {
		query.Add("tag_filters", tag)
	}

	requestPath := "personas"
	if encoded := query.Encode(); encoded != "" {
		requestPath += "?" + encoded
	}

	var response ListPersonasOutput
	if err := c.Do(ctx, http.MethodGet, requestPath, nil, &response); err != nil {
		return ListPersonasOutput{}, fmt.Errorf("list personas: %w", err)
	}
	return response, nil
}
