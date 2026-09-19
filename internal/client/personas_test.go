package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"
)

const testPersonaID = "abc123def456ghi789jklm"

func TestCreatePersona(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost || request.URL.Path != "/v1/personas" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if decoded["name"] != "Provider persona" || decoded["voice_name"] != "aria" || decoded["language_code"] != "en-US" {
			t.Errorf("request body = %#v", decoded)
		}
		parameters, ok := decoded["initialization_parameters"].(map[string]any)
		if !ok || parameters["customer_tier"] != "premium" {
			t.Errorf("initialization_parameters = %#v", decoded["initialization_parameters"])
		}
		phone, ok := decoded["multi_phone_config"].(map[string]any)
		if !ok || phone["phone_number_index"] != float64(2) {
			t.Errorf("multi_phone_config = %#v", decoded["multi_phone_config"])
		}
		return testResponse(http.StatusCreated, personaEnvelopeJSON("Provider persona"), nil), nil
	})
	apiClient, err := New("secret", "https://example.test/v1", WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	parameters := json.RawMessage(`{"customer_tier":"premium"}`)
	prompt := "Act like a friendly customer"
	tags := []string{"terraform", "acceptance"}
	created, err := apiClient.CreatePersona(context.Background(), CreatePersonaInput{
		Name:                     "Provider persona",
		PersonaPrompt:            &prompt,
		VoiceName:                "aria",
		LanguageCode:             "en-US",
		InitializationParameters: &parameters,
		MultiPhoneConfig:         &MultiPhoneConfig{PhoneNumberIndex: 2},
		Tags:                     &tags,
	})
	if err != nil {
		t.Fatalf("CreatePersona(): %v", err)
	}
	if created.ID != testPersonaID || created.Name != "Provider persona" {
		t.Errorf("created = %#v", created)
	}
}

func TestGetAndUpdatePersona(t *testing.T) {
	t.Parallel()

	requestCount := 0
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestCount++
		if request.URL.Path != "/v1/personas/"+testPersonaID {
			t.Fatalf("request path = %s", request.URL.Path)
		}
		switch requestCount {
		case 1:
			if request.Method != http.MethodGet {
				t.Fatalf("request method = %s, want GET", request.Method)
			}
			return testResponse(http.StatusOK, personaEnvelopeJSON("Original"), nil), nil
		case 2:
			if request.Method != http.MethodPatch {
				t.Fatalf("request method = %s, want PATCH", request.Method)
			}
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			var decoded map[string]any
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("decode request body: %v", err)
			}
			if decoded["name"] != "Updated" || decoded["interruption_rate"] != "HIGH" {
				t.Errorf("request body = %#v", decoded)
			}
			for _, key := range []string{"persona_prompt", "voice", "audio_degradation", "initialization_parameters"} {
				if value, present := decoded[key]; !present || value != nil {
					t.Errorf("%s = %#v, present = %t; want explicit null", key, value, present)
				}
			}
			return testResponse(http.StatusOK, personaEnvelopeJSON("Updated"), nil), nil
		default:
			t.Fatalf("unexpected request %d", requestCount)
			return nil, nil
		}
	})
	apiClient, err := New("secret", "https://example.test/v1", WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	if _, err := apiClient.GetPersona(context.Background(), testPersonaID); err != nil {
		t.Fatalf("GetPersona(): %v", err)
	}
	updated, err := apiClient.UpdatePersona(context.Background(), testPersonaID, UpdatePersonaInput{
		Name:             "Updated",
		VoiceName:        "aria",
		LanguageCode:     "en-US",
		InterruptionRate: "HIGH",
		Tags:             []string{},
	})
	if err != nil {
		t.Fatalf("UpdatePersona(): %v", err)
	}
	if updated.Name != "Updated" {
		t.Errorf("updated = %#v", updated)
	}
}

func TestListPersonasEncodesOptions(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet || request.URL.Path != "/v1/personas" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("filter") != `name="Provider persona"` || query.Get("page_size") != "100" || query.Get("page_token") != "next" || query.Get("order_by") != "-create_time" {
			t.Errorf("query = %v", query)
		}
		if got := query["tag_filters"]; !reflect.DeepEqual(got, []string{"terraform", "voice"}) {
			t.Errorf("tag_filters = %#v", got)
		}
		return testResponse(http.StatusOK, `{"personas":[],"next_page_token":""}`, nil), nil
	})
	apiClient, err := New("secret", "https://example.test/v1", WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	_, err = apiClient.ListPersonas(context.Background(), ListPersonasOptions{
		Filter:     `name="Provider persona"`,
		PageSize:   100,
		PageToken:  "next",
		OrderBy:    "-create_time",
		TagFilters: []string{"terraform", "voice"},
	})
	if err != nil {
		t.Fatalf("ListPersonas(): %v", err)
	}
}

func TestDeletePersona(t *testing.T) {
	t.Parallel()

	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodDelete || request.URL.Path != "/v1/personas/"+testPersonaID {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		return testResponse(http.StatusOK, `{}`, nil), nil
	})
	apiClient, err := New("secret", "https://example.test/v1", WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	if err := apiClient.DeletePersona(context.Background(), testPersonaID); err != nil {
		t.Fatalf("DeletePersona(): %v", err)
	}
}

func personaEnvelopeJSON(name string) string {
	return `{"persona":{"resource_name":"personas/` + testPersonaID + `","id":"` + testPersonaID + `","name":"` + name + `","persona_prompt":"Act like a friendly customer","voice_name":"aria","language_code":"en-US","silent_mode":false,"multi_phone_config":{"phone_number_index":2,"phone_number_name":"secondary"},"initialization_parameters":{"customer_tier":"premium"},"custom_persona_data":null,"voice":null,"custom_voice_id":null,"background_sound":"office","background_sound_volume":0.3,"voice_volume":1.0,"voice_speed":1.0,"wait_seconds":0.5,"conversation_initiation":"speak_first","interruption_rate":"NONE","multi_language_stt":true,"hold_music_timeout_seconds":15,"situate_speaker":null,"audio_degradation":{"preset":"cell-poor","preset_version":1},"tags":["terraform"],"create_time":"2026-09-17T00:00:00Z","update_time":null}}`
}
