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

const acceptanceScheduledRunPrefix = "tf-acc-scheduled-run-"

func init() {
	resource.AddTestSweepers("coval_scheduled_run", &resource.Sweeper{Name: "coval_scheduled_run", F: sweepScheduledRuns})
}

func TestAccScheduledRunResource(t *testing.T) {
	displayName := acctest.RandomWithPrefix(acceptanceScheduledRunPrefix)
	resourceName := "coval_scheduled_run.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) }, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, CheckDestroy: testAccCheckScheduledRunDestroy,
		Steps: []resource.TestStep{
			{Config: testAccScheduledRunConfig(displayName, false), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "display_name", displayName), resource.TestCheckResourceAttr(resourceName, "schedule_expression", "rate(1 day)"),
				resource.TestCheckResourceAttr(resourceName, "schedule_timezone", "UTC"), resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
				resource.TestCheckNoResourceAttr(resourceName, "last_run_id"), resource.TestCheckResourceAttrSet(resourceName, "id"),
				resource.TestCheckResourceAttr("data.coval_scheduled_run.current", "display_name", displayName),
				testAccCheckScheduledRunListed(resourceName, "data.coval_scheduled_runs.disabled"),
			)},
			{ResourceName: resourceName, ImportState: true, ImportStateVerify: true},
			{Config: testAccScheduledRunConfig(displayName, true), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "display_name", displayName+" Updated"), resource.TestCheckResourceAttr(resourceName, "schedule_expression", "rate(2 days)"),
				resource.TestCheckResourceAttr(resourceName, "schedule_timezone", "America/Los_Angeles"), resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
			)},
		},
	})
}

func testAccCheckScheduledRunDestroy(state *terraform.State) error {
	apiClient, err := acceptanceAPIClient()
	if err != nil {
		return err
	}
	for _, instance := range state.RootModule().Resources {
		if instance.Type != "coval_scheduled_run" {
			continue
		}
		_, err := apiClient.GetScheduledRun(context.Background(), instance.Primary.ID)
		if err == nil {
			return fmt.Errorf("Coval scheduled run %s still exists after destroy", instance.Primary.ID)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("verify Coval scheduled run %s was destroyed: %w", instance.Primary.ID, err)
		}
	}
	return nil
}

func sweepScheduledRuns(_ string) error {
	apiClient, err := acceptanceAPIClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	options := client.ListScheduledRunsOptions{PageSize: 100}
	for {
		page, err := apiClient.ListScheduledRuns(ctx, options)
		if err != nil {
			return err
		}
		for _, scheduledRun := range page.ScheduledRuns {
			if strings.HasPrefix(scheduledRun.DisplayName, acceptanceScheduledRunPrefix) {
				if err := apiClient.DeleteScheduledRun(ctx, scheduledRun.ID); err != nil && !client.IsNotFound(err) {
					return fmt.Errorf("sweep Coval scheduled run %s: %w", scheduledRun.ID, err)
				}
			}
		}
		if page.NextPageToken == nil || *page.NextPageToken == "" {
			return nil
		}
		options.PageToken = *page.NextPageToken
	}
}

func testAccCheckScheduledRunListed(resourceName string, dataSourceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		scheduledRun, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("scheduled run resource %s was not found", resourceName)
		}
		dataSource, ok := state.RootModule().Resources[dataSourceName]
		if !ok {
			return fmt.Errorf("scheduled run data source %s was not found", dataSourceName)
		}
		for key, value := range dataSource.Primary.Attributes {
			if strings.HasSuffix(key, ".id") && value == scheduledRun.Primary.ID {
				return nil
			}
		}
		return fmt.Errorf("scheduled run %s was not returned by %s", scheduledRun.Primary.ID, dataSourceName)
	}
}

func testAccScheduledRunConfig(displayName string, updated bool) string {
	name := displayName
	expression := "rate(1 day)"
	timezone := "UTC"
	if updated {
		name += " Updated"
		expression = "rate(2 days)"
		timezone = "America/Los_Angeles"
	}
	return fmt.Sprintf(`
resource "coval_agent" "schedule_fixture" {
  customer_agent_id = %s
  display_name      = %s
  model_type        = "MODEL_TYPE_CHAT"
  prompt            = "Answer acceptance-test questions."
  metadata          = { chat_endpoint = "https://example.com/chat" }
}

resource "coval_persona" "schedule_fixture" {
  name           = %s
  persona_prompt = "Ask a concise support question."
  voice_name     = "aria"
  language_code  = "en-US"
}

resource "coval_test_set" "schedule_fixture" {
  display_name = %s
  description  = "Scheduled-run acceptance fixture"
}

resource "coval_run_template" "schedule_fixture" {
  display_name  = %s
  agent_ids     = [coval_agent.schedule_fixture.id]
  persona_ids   = [coval_persona.schedule_fixture.id]
  test_set_ids  = [coval_test_set.schedule_fixture.id]
}

resource "coval_scheduled_run" "test" {
  display_name        = %s
  run_template_id     = coval_run_template.schedule_fixture.id
  schedule_expression = %s
  schedule_timezone   = %s
  enabled             = false
}

data "coval_scheduled_run" "current" { id = coval_scheduled_run.test.id }
data "coval_scheduled_runs" "disabled" {
  enabled    = false
  depends_on = [coval_scheduled_run.test]
}
`, strconv.Quote(displayName), strconv.Quote(displayName+" agent"), strconv.Quote(displayName+" persona"), strconv.Quote(displayName+" test set"), strconv.Quote(displayName+" template"), strconv.Quote(name), strconv.Quote(expression), strconv.Quote(timezone))
}
