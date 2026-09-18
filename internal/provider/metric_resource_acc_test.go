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

const acceptanceMetricPrefix = "tf-acc-metric-"

func init() {
	resource.AddTestSweepers("coval_metric", &resource.Sweeper{Name: "coval_metric", F: sweepMetrics})
}

func TestAccMetricResource(t *testing.T) {
	name := acctest.RandomWithPrefix(acceptanceMetricPrefix)
	resourceName := "coval_metric.test"
	resource.Test(t, resource.TestCase{PreCheck: func() { testAccPreCheck(t) }, ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, CheckDestroy: testAccCheckMetricDestroy, Steps: []resource.TestStep{
		{Config: testAccMetricConfig(name, false), Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(resourceName, "metric_name", name), resource.TestCheckResourceAttr(resourceName, "metric_type", "METRIC_LLM_BINARY"),
			resource.TestCheckResourceAttr(resourceName, "prompt", "Did the agent resolve the customer's issue?"), resource.TestCheckResourceAttr(resourceName, "target_condition.comparison_operator", "in"),
			resource.TestCheckResourceAttr(resourceName, "target_condition.target_values.#", "1"), resource.TestCheckResourceAttr(resourceName, "tags.#", "2"), resource.TestCheckResourceAttrSet(resourceName, "id"),
			resource.TestCheckResourceAttr("data.coval_metric.current", "metric_name", name), resource.TestCheckResourceAttr("data.coval_metrics.matching", "metrics.#", "1"),
		)},
		{ResourceName: resourceName, ImportState: true, ImportStateVerify: true},
		{Config: testAccMetricConfig(name, true), Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(resourceName, "description", "Updated by the Terraform provider acceptance suite"), resource.TestCheckResourceAttr(resourceName, "prompt", "Was the customer's issue fully resolved?"),
			resource.TestCheckResourceAttr(resourceName, "tags.#", "0"), resource.TestCheckResourceAttr("data.coval_metrics.matching", "metrics.#", "1"),
		)},
	}})
}

func testAccCheckMetricDestroy(state *terraform.State) error {
	apiClient, err := acceptanceAPIClient()
	if err != nil {
		return err
	}
	for _, instance := range state.RootModule().Resources {
		if instance.Type != "coval_metric" {
			continue
		}
		_, err := apiClient.GetMetric(context.Background(), instance.Primary.ID)
		if err == nil {
			return fmt.Errorf("Coval metric %s still exists after destroy", instance.Primary.ID)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("verify Coval metric %s was destroyed: %w", instance.Primary.ID, err)
		}
	}
	return nil
}

func sweepMetrics(_ string) error {
	apiClient, err := acceptanceAPIClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	options := client.ListMetricsOptions{PageSize: 100}
	for {
		page, err := apiClient.ListMetrics(ctx, options)
		if err != nil {
			return err
		}
		for _, metric := range page.Metrics {
			if strings.HasPrefix(metric.MetricName, acceptanceMetricPrefix) {
				if err := apiClient.DeleteMetric(ctx, metric.ID); err != nil && !client.IsNotFound(err) {
					return fmt.Errorf("sweep Coval metric %s: %w", metric.ID, err)
				}
			}
		}
		if page.NextPageToken == "" {
			return nil
		}
		options.PageToken = page.NextPageToken
	}
}

func testAccMetricConfig(name string, updated bool) string {
	description := "Managed by the Terraform provider acceptance suite"
	prompt := "Did the agent resolve the customer's issue?"
	tags := `["terraform", "acceptance"]`
	if updated {
		description = "Updated by the Terraform provider acceptance suite"
		prompt = "Was the customer's issue fully resolved?"
		tags = `[]`
	}
	filter := fmt.Sprintf("metric_name=%q", name)
	return fmt.Sprintf(`
resource "coval_metric" "test" {
  metric_name = %s
  description = %s
  metric_type = "METRIC_LLM_BINARY"
  prompt      = %s
  target_condition = {
    comparison_operator = "in"
    target_values       = ["YES"]
  }
  tags = %s
}

data "coval_metric" "current" {
  id = coval_metric.test.id
}

data "coval_metrics" "matching" {
  filter     = %s
  depends_on = [coval_metric.test]
}
`, strconv.Quote(name), strconv.Quote(description), strconv.Quote(prompt), tags, strconv.Quote(filter))
}
