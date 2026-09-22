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

const acceptanceRunTemplatePrefix = "tf-acc-run-template-"

func init() {
	resource.AddTestSweepers("coval_run_template", &resource.Sweeper{Name: "coval_run_template", F: sweepRunTemplates})
}

func TestAccRunTemplateResource(t *testing.T) {
	displayName := acctest.RandomWithPrefix(acceptanceRunTemplatePrefix)
	resourceName := "coval_run_template.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) }, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, CheckDestroy: testAccCheckRunTemplateDestroy,
		Steps: []resource.TestStep{
			{Config: testAccRunTemplateConfig(displayName, false), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "display_name", displayName), resource.TestCheckResourceAttr(resourceName, "iteration_count", "1"),
				resource.TestCheckResourceAttr(resourceName, "concurrency", "1"), resource.TestCheckResourceAttr(resourceName, "agent_ids.#", "1"),
				resource.TestCheckResourceAttr(resourceName, "persona_ids.#", "1"), resource.TestCheckResourceAttr(resourceName, "test_set_ids.#", "1"),
				resource.TestCheckResourceAttr(resourceName, "metadata.owner", "terraform-provider-coval"), resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
				resource.TestCheckResourceAttrSet(resourceName, "id"), resource.TestCheckResourceAttr("data.coval_run_template.current", "display_name", displayName),
				resource.TestCheckResourceAttr("data.coval_run_templates.matching", "run_templates.#", "1"),
			)},
			{ResourceName: resourceName, ImportState: true, ImportStateVerify: true},
			{Config: testAccRunTemplateConfig(displayName, true), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "display_name", displayName+" Updated"), resource.TestCheckResourceAttr(resourceName, "iteration_count", "2"),
				resource.TestCheckResourceAttr(resourceName, "concurrency", "2"), resource.TestCheckResourceAttr(resourceName, "sub_sample_seed", "42"),
				resource.TestCheckResourceAttr(resourceName, "metadata.%", "0"), resource.TestCheckResourceAttr(resourceName, "tags.#", "0"),
				resource.TestCheckResourceAttr("data.coval_run_templates.matching", "run_templates.#", "0"),
			)},
		},
	})
}

func testAccCheckRunTemplateDestroy(state *terraform.State) error {
	apiClient, err := acceptanceAPIClient()
	if err != nil {
		return err
	}
	for _, instance := range state.RootModule().Resources {
		if instance.Type != "coval_run_template" {
			continue
		}
		_, err := apiClient.GetRunTemplate(context.Background(), instance.Primary.ID)
		if err == nil {
			return fmt.Errorf("Coval run template %s still exists after destroy", instance.Primary.ID)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("verify Coval run template %s was destroyed: %w", instance.Primary.ID, err)
		}
	}
	return nil
}

func sweepRunTemplates(_ string) error {
	apiClient, err := acceptanceAPIClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	options := client.ListRunTemplatesOptions{PageSize: 100}
	for {
		page, err := apiClient.ListRunTemplates(ctx, options)
		if err != nil {
			return err
		}
		for _, runTemplate := range page.RunTemplates {
			if strings.HasPrefix(runTemplate.DisplayName, acceptanceRunTemplatePrefix) {
				if err := apiClient.DeleteRunTemplate(ctx, runTemplate.ID); err != nil && !client.IsNotFound(err) {
					return fmt.Errorf("sweep Coval run template %s: %w", runTemplate.ID, err)
				}
			}
		}
		if page.NextPageToken == "" {
			return nil
		}
		options.PageToken = page.NextPageToken
	}
}

func testAccRunTemplateConfig(displayName string, updated bool) string {
	templateName := displayName
	description := "Managed by the Terraform provider acceptance suite"
	iterationCount := 1
	concurrency := 1
	metadata := `{ owner = "terraform-provider-coval" }`
	tags := `["terraform", "acceptance"]`
	seed := ""
	if updated {
		templateName += " Updated"
		description = "Updated by Terraform"
		iterationCount = 2
		concurrency = 2
		metadata = `{}`
		tags = `[]`
		seed = "sub_sample_seed = 42"
	}
	return fmt.Sprintf(`
resource "coval_agent" "template_fixture" {
  customer_agent_id = %s
  display_name      = %s
  model_type        = "MODEL_TYPE_CHAT"
  prompt            = "Answer acceptance-test questions."
  metadata          = { chat_endpoint = "https://example.com/chat" }
}

resource "coval_persona" "template_fixture" {
  name           = %s
  persona_prompt = "Ask a concise support question."
  voice_name     = "aria"
  language_code  = "en-US"
}

resource "coval_test_set" "template_fixture" {
  display_name = %s
  description  = "Run-template acceptance fixture"
}

resource "coval_run_template" "test" {
  display_name      = %s
  description       = %s
  agent_ids          = [coval_agent.template_fixture.id]
  persona_ids        = [coval_persona.template_fixture.id]
  test_set_ids       = [coval_test_set.template_fixture.id]
  iteration_count    = %d
  concurrency        = %d
  sub_sample_size    = 0
  %s
  metadata           = %s
  tags               = %s
}

data "coval_run_template" "current" { id = coval_run_template.test.id }
data "coval_run_templates" "matching" {
  tag_filters = ["terraform", "acceptance"]
  depends_on  = [coval_run_template.test]
}
`, strconv.Quote(displayName), strconv.Quote(displayName+" agent"), strconv.Quote(displayName+" persona"), strconv.Quote(displayName+" test set"), strconv.Quote(templateName), strconv.Quote(description), iterationCount, concurrency, seed, metadata, tags)
}
