package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const acceptancePersonaPrefix = "tf-acc-persona-"

func init() {
	resource.AddTestSweepers("coval_persona", &resource.Sweeper{
		Name: "coval_persona",
		F:    sweepPersonas,
	})
}

func TestAccPersonaResource(t *testing.T) {
	name := acctest.RandomWithPrefix(acceptancePersonaPrefix)
	resourceName := "coval_persona.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPersonaDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPersonaConfig(name, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "persona_prompt", "You are a friendly customer testing the Terraform provider."),
					resource.TestCheckResourceAttr(resourceName, "voice_name", "aria"),
					resource.TestCheckResourceAttr(resourceName, "language_code", "en-US"),
					resource.TestCheckResourceAttr(resourceName, "background_sound", "office"),
					resource.TestCheckResourceAttr(resourceName, "conversation_initiation", "speak_first"),
					resource.TestCheckResourceAttr(resourceName, "interruption_rate", "LOW"),
					resource.TestCheckResourceAttr(resourceName, "initialization_parameters.customer_tier", "premium"),
					resource.TestCheckResourceAttr(resourceName, "audio_degradation.preset", "cell-poor"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "resource_name"),
					resource.TestCheckResourceAttr("data.coval_persona.current", "name", name),
					resource.TestCheckResourceAttr("data.coval_personas.matching", "personas.#", "1"),
					resource.TestCheckResourceAttr("data.coval_personas.matching", "personas.0.name", name),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccPersonaConfig(name, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name+" updated"),
					resource.TestCheckNoResourceAttr(resourceName, "persona_prompt"),
					resource.TestCheckNoResourceAttr(resourceName, "background_sound"),
					resource.TestCheckNoResourceAttr(resourceName, "initialization_parameters"),
					resource.TestCheckNoResourceAttr(resourceName, "audio_degradation"),
					resource.TestCheckResourceAttr(resourceName, "interruption_rate", "NONE"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "0"),
					resource.TestCheckResourceAttr("data.coval_personas.matching", "personas.#", "1"),
				),
			},
		},
	})
}

func testAccCheckPersonaDestroy(state *terraform.State) error {
	apiClient, err := acceptanceAPIClient()
	if err != nil {
		return err
	}
	for _, instance := range state.RootModule().Resources {
		if instance.Type != "coval_persona" {
			continue
		}
		_, err := apiClient.GetPersona(context.Background(), instance.Primary.ID)
		if err == nil {
			return fmt.Errorf("Coval persona %s still exists after destroy", instance.Primary.ID)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("verify Coval persona %s was destroyed: %w", instance.Primary.ID, err)
		}
	}
	return nil
}

func sweepPersonas(_ string) error {
	apiClient, err := acceptanceAPIClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	options := client.ListPersonasOptions{PageSize: 100}
	for {
		page, err := apiClient.ListPersonas(ctx, options)
		if err != nil {
			return err
		}
		for _, persona := range page.Personas {
			if strings.HasPrefix(persona.Name, acceptancePersonaPrefix) {
				if err := apiClient.DeletePersona(ctx, persona.ID); err != nil && !client.IsNotFound(err) {
					return fmt.Errorf("sweep Coval persona %s: %w", persona.ID, err)
				}
			}
		}
		if page.NextPageToken == "" {
			return nil
		}
		options.PageToken = page.NextPageToken
	}
}

func testAccPersonaConfig(name string, cleared bool) string {
	configuredName := name
	optionalFields := `
  persona_prompt             = "You are a friendly customer testing the Terraform provider."
  background_sound           = "office"
  background_sound_volume    = 0.3
  voice_volume               = 1.0
  voice_speed                = 1.0
  wait_seconds               = 0.5
  conversation_initiation    = "speak_first"
  interruption_rate          = "LOW"
  multi_language_stt         = true
  hold_music_timeout_seconds = 15
  situate_speaker            = "speakerphone-easy"
  initialization_parameters = {
    customer_tier = "premium"
  }
  audio_degradation = {
    preset         = "cell-poor"
    preset_version = 1
  }
  tags = ["terraform", "acceptance"]`
	if cleared {
		configuredName += " updated"
		optionalFields = `
  interruption_rate = "NONE"
  tags              = []`
	}
	filter := fmt.Sprintf("name=%q", configuredName)
	return fmt.Sprintf(`
resource "coval_persona" "test" {
  name          = %s
  voice_name    = "aria"
  language_code = "en-US"
%s
}

data "coval_persona" "current" {
  id = coval_persona.test.id
}

data "coval_personas" "matching" {
  filter     = %s
  depends_on = [coval_persona.test]
}
`, strconv.Quote(configuredName), optionalFields, strconv.Quote(filter))
}
