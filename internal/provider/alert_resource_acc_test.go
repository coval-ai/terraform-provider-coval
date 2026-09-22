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

const acceptanceAlertPrefix = "tf-acc-alert-"

func init() {
	resource.AddTestSweepers("coval_alert", &resource.Sweeper{Name: "coval_alert", F: sweepAlerts})
}

func TestAccAlertResource(t *testing.T) {
	name := acctest.RandomWithPrefix(acceptanceAlertPrefix)
	resourceName := "coval_alert.test"
	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) }, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, CheckDestroy: testAccCheckAlertDestroy,
		Steps: []resource.TestStep{
			{Config: testAccAlertConfig(name, false), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "name", name), resource.TestCheckResourceAttr(resourceName, "status", "ACTIVE"),
				resource.TestCheckResourceAttr(resourceName, "conversation_source", "SIMULATED"), resource.TestCheckResourceAttr(resourceName, "match_mode", "ALL"),
				resource.TestCheckResourceAttr(resourceName, "cooldown_seconds", "0"), resource.TestCheckResourceAttr(resourceName, "agent_ids.#", "1"),
				resource.TestCheckResourceAttr(resourceName, "conditions.#", "1"), resource.TestCheckResourceAttr(resourceName, "channels.#", "0"),
				resource.TestCheckResourceAttrSet(resourceName, "id"), resource.TestCheckResourceAttr("data.coval_alert.current", "name", name),
				testAccCheckAlertListed(resourceName, "data.coval_alerts.simulated"),
			)},
			{ResourceName: resourceName, ImportState: true, ImportStateVerify: true},
			{Config: testAccAlertConfig(name, true), Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(resourceName, "description", "Updated by Terraform"), resource.TestCheckResourceAttr(resourceName, "match_mode", "ANY"),
				resource.TestCheckResourceAttr(resourceName, "cooldown_seconds", "60"), resource.TestCheckResourceAttr(resourceName, "conditions.0.threshold_float", "0.8"),
				resource.TestCheckResourceAttr(resourceName, "channels.#", "0"),
			)},
		},
	})
}

func testAccCheckAlertDestroy(state *terraform.State) error {
	apiClient, err := acceptanceAPIClient()
	if err != nil {
		return err
	}
	for _, instance := range state.RootModule().Resources {
		if instance.Type != "coval_alert" {
			continue
		}
		_, err := apiClient.GetAlert(context.Background(), instance.Primary.ID)
		if err == nil {
			return fmt.Errorf("Coval alert %s still exists after destroy", instance.Primary.ID)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("verify Coval alert %s was destroyed: %w", instance.Primary.ID, err)
		}
	}
	return nil
}

func sweepAlerts(_ string) error {
	apiClient, err := acceptanceAPIClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	options := client.ListAlertsOptions{PageSize: 100}
	for {
		page, err := apiClient.ListAlerts(ctx, options)
		if err != nil {
			return err
		}
		for _, alert := range page.Alerts {
			if strings.HasPrefix(alert.Name, acceptanceAlertPrefix) {
				if err := apiClient.DeleteAlert(ctx, alert.ID); err != nil && !client.IsNotFound(err) {
					return fmt.Errorf("sweep Coval alert %s: %w", alert.ID, err)
				}
			}
		}
		if page.NextPageToken == nil || *page.NextPageToken == "" {
			return nil
		}
		options.PageToken = *page.NextPageToken
	}
}

func testAccCheckAlertListed(resourceName string, dataSourceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		alert, ok := state.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("alert resource %s was not found", resourceName)
		}
		dataSource, ok := state.RootModule().Resources[dataSourceName]
		if !ok {
			return fmt.Errorf("alert data source %s was not found", dataSourceName)
		}
		for key, value := range dataSource.Primary.Attributes {
			if strings.HasSuffix(key, ".id") && value == alert.Primary.ID {
				return nil
			}
		}
		return fmt.Errorf("alert %s was not returned by %s", alert.Primary.ID, dataSourceName)
	}
}

func testAccAlertConfig(name string, updated bool) string {
	description := "Managed by the Terraform provider acceptance suite"
	matchMode := "ALL"
	cooldown := 0
	threshold := 0.9
	if updated {
		description = "Updated by Terraform"
		matchMode = "ANY"
		cooldown = 60
		threshold = 0.8
	}
	return fmt.Sprintf(`
resource "coval_agent" "alert_fixture" {
  customer_agent_id = %s
  display_name      = %s
  model_type        = "MODEL_TYPE_CHAT"
  prompt            = "Answer acceptance-test questions."
  metadata          = { chat_endpoint = "https://example.com/chat" }
}

resource "coval_metric" "alert_fixture" {
  metric_name = %s
  description = "Alert acceptance fixture"
  metric_type = "METRIC_LLM_BINARY"
  prompt      = "Was the issue resolved?"
  target_condition = {
    comparison_operator = "in"
    target_values       = ["YES"]
  }
}

resource "coval_alert" "test" {
  name                = %s
  description         = %s
  conversation_source = "SIMULATED"
  match_mode           = %s
  cooldown_seconds     = %d
  agent_ids            = [coval_agent.alert_fixture.id]
  conditions = [{
    metric_id       = coval_metric.alert_fixture.id
    aggregation     = "RUN_AVERAGE"
    operator        = "LT"
    threshold_float = %g
  }]
  channels = []
}

data "coval_alert" "current" { id = coval_alert.test.id }
data "coval_alerts" "simulated" {
  conversation_source = "SIMULATED"
  depends_on          = [coval_alert.test]
}
`, strconv.Quote(name), strconv.Quote(name+" agent"), strconv.Quote(name+" metric"), strconv.Quote(name), strconv.Quote(description), strconv.Quote(matchMode), cooldown, threshold)
}
