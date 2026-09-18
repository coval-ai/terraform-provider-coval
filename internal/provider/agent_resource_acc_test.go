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

const acceptanceAgentPrefix = "tf-acc-agent-"

func init() {
	resource.AddTestSweepers("coval_agent", &resource.Sweeper{Name: "coval_agent", F: sweepAgents})
}

func TestAccAgentResource(t *testing.T) {
	name := acctest.RandomWithPrefix(acceptanceAgentPrefix)
	resourceName := "coval_agent.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) }, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, CheckDestroy: testAccCheckAgentDestroy,
		Steps: []resource.TestStep{
			{Config: testAccAgentConfig(name, false), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "display_name", name), resource.TestCheckResourceAttr(resourceName, "model_type", "MODEL_TYPE_CHAT"),
				resource.TestCheckResourceAttr(resourceName, "prompt", "Help customers with acceptance-test questions."), resource.TestCheckResourceAttr(resourceName, "attributes.owner", "terraform-provider-coval"),
				resource.TestCheckResourceAttr(resourceName, "tags.#", "2"), resource.TestCheckResourceAttrSet(resourceName, "id"),
				resource.TestCheckResourceAttr("data.coval_agent.current", "display_name", name), resource.TestCheckResourceAttr("data.coval_agents.matching", "agents.#", "1"),
			)},
			{ResourceName: resourceName, ImportState: true, ImportStateVerify: true},
			{Config: testAccAgentConfig(name, true), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "display_name", name+" Updated"), resource.TestCheckResourceAttr(resourceName, "model_type", "MODEL_TYPE_CHAT_A2A"),
				resource.TestCheckNoResourceAttr(resourceName, "prompt"),
				resource.TestCheckNoResourceAttr(resourceName, "language"), resource.TestCheckNoResourceAttr(resourceName, "attributes"),
				resource.TestCheckResourceAttr(resourceName, "workflows.%", "0"), resource.TestCheckResourceAttr(resourceName, "tags.#", "0"),
				resource.TestCheckResourceAttr("data.coval_agents.matching", "agents.#", "1"),
			)},
		},
	})
}

func testAccCheckAgentDestroy(state *terraform.State) error {
	apiClient, err := acceptanceAPIClient()
	if err != nil {
		return err
	}
	for _, instance := range state.RootModule().Resources {
		if instance.Type != "coval_agent" {
			continue
		}
		_, err := apiClient.GetAgent(context.Background(), instance.Primary.ID)
		if err == nil {
			return fmt.Errorf("Coval agent %s still exists after destroy", instance.Primary.ID)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("verify Coval agent %s was destroyed: %w", instance.Primary.ID, err)
		}
	}
	return nil
}

func sweepAgents(_ string) error {
	apiClient, err := acceptanceAPIClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	options := client.ListAgentsOptions{PageSize: 100}
	for {
		page, err := apiClient.ListAgents(ctx, options)
		if err != nil {
			return err
		}
		for _, agent := range page.Agents {
			if strings.HasPrefix(agent.DisplayName, acceptanceAgentPrefix) {
				if err := apiClient.DeleteAgent(ctx, agent.ID); err != nil && !client.IsNotFound(err) {
					return fmt.Errorf("sweep Coval agent %s: %w", agent.ID, err)
				}
			}
		}
		if page.NextPageToken == "" {
			return nil
		}
		options.PageToken = page.NextPageToken
	}
}

func testAccAgentConfig(name string, updated bool) string {
	displayName := name
	modelType := "MODEL_TYPE_CHAT"
	prompt := `prompt = "Help customers with acceptance-test questions."`
	language := `language = "en"`
	attributes := `attributes = { owner = "terraform-provider-coval" }`
	workflows := `workflows = { routing = "support" }`
	tags := `tags = ["terraform", "acceptance"]`
	if updated {
		displayName += " Updated"
		modelType = "MODEL_TYPE_CHAT_A2A"
		prompt = ""
		language = ""
		attributes = ""
		workflows = "workflows = {}"
		tags = "tags = []"
	}
	filter := fmt.Sprintf("display_name=%q", displayName)
	return fmt.Sprintf(`
resource "coval_agent" "test" {
  customer_agent_id = %s
  display_name      = %s
  model_type       = %s
  %s
  %s
  %s
  metadata = {
    chat_endpoint = "https://example.com/chat"
  }
  %s
  %s
}

data "coval_agent" "current" {
  id = coval_agent.test.id
}

data "coval_agents" "matching" {
  filter     = %s
  depends_on = [coval_agent.test]
}
`, strconv.Quote(name), strconv.Quote(displayName), strconv.Quote(modelType), prompt, language, attributes, workflows, tags, strconv.Quote(filter))
}
