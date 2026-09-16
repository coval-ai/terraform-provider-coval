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

func init() {
	resource.AddTestSweepers("coval_test_case", &resource.Sweeper{
		Name: "coval_test_case",
		F:    sweepTestCases,
	})
}

func TestAccTestCaseResource(t *testing.T) {
	primaryDisplayName := acctest.RandomWithPrefix(acceptanceTestSetPrefix)
	secondaryDisplayName := acctest.RandomWithPrefix(acceptanceTestSetPrefix)
	resourceName := "coval_test_case.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTestCaseAndTestSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTestCaseConfig(primaryDisplayName, secondaryDisplayName, "scenario"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "test_set_id", "coval_test_set.primary", "id"),
					resource.TestCheckResourceAttr(resourceName, "input_str", "Ask whether a damaged order can be refunded"),
					resource.TestCheckResourceAttr(resourceName, "expected_behaviors.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "expected_output_json.eligible", "true"),
					resource.TestCheckResourceAttr(resourceName, "simulation_metadata_input.channel", "voice"),
					resource.TestCheckResourceAttr(resourceName, "metric_input.policy_window_days", "30"),
					resource.TestCheckResourceAttr(resourceName, "description", "Refund-policy regression"),
					resource.TestCheckResourceAttr(resourceName, "input_type", "SCENARIO"),
					resource.TestCheckResourceAttr(resourceName, "user_notes", "Created by the Terraform provider acceptance suite"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "name"),
					resource.TestCheckResourceAttrSet(resourceName, "create_time"),
					resource.TestCheckResourceAttrPair("data.coval_test_case.current", "id", resourceName, "id"),
					resource.TestCheckResourceAttr("data.coval_test_case.current", "input_str", "Ask whether a damaged order can be refunded"),
					resource.TestCheckResourceAttr("data.coval_test_cases.matching", "test_cases.#", "1"),
					resource.TestCheckResourceAttrPair("data.coval_test_cases.matching", "test_cases.0.id", resourceName, "id"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccTestCaseConfig(primaryDisplayName, secondaryDisplayName, "script"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "test_set_id", "coval_test_set.secondary", "id"),
					resource.TestCheckResourceAttr(resourceName, "input_str", "Follow the scripted refund flow"),
					resource.TestCheckResourceAttr(resourceName, "expected_behaviors.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "description", ""),
					resource.TestCheckResourceAttr(resourceName, "input_type", "SCRIPT"),
					resource.TestCheckResourceAttr(resourceName, "user_notes", ""),
					resource.TestCheckResourceAttr("data.coval_test_cases.matching", "test_cases.#", "1"),
					resource.TestCheckResourceAttrPair("data.coval_test_cases.matching", "test_cases.0.id", resourceName, "id"),
				),
			},
			{
				Config: testAccTestCaseConfig(primaryDisplayName, secondaryDisplayName, "scenario-after-script"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "test_set_id", "coval_test_set.primary", "id"),
					resource.TestCheckResourceAttr(resourceName, "input_type", "SCENARIO"),
					resource.TestCheckNoResourceAttr(resourceName, "script_turns"),
				),
			},
		},
	})
}

func TestAccTestCaseResourceComputedReferences(t *testing.T) {
	displayName := acctest.RandomWithPrefix(acceptanceTestSetPrefix)
	resourceName := "coval_test_case.test"
	testSetResourceName := "coval_test_set.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTestCaseAndTestSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTestCaseComputedReferencesConfig(displayName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(resourceName, "test_set_id", testSetResourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "input_type", "SCRIPT"),
					resource.TestCheckResourceAttrPair(resourceName, "script_turns.0", testSetResourceName, "name"),
					resource.TestCheckResourceAttrPair(resourceName, "simulation_metadata_input.test_set_name", testSetResourceName, "name"),
				),
			},
		},
	})
}

func testAccCheckTestCaseAndTestSetDestroy(state *terraform.State) error {
	apiClient, err := acceptanceAPIClient()
	if err != nil {
		return err
	}
	for _, instance := range state.RootModule().Resources {
		switch instance.Type {
		case "coval_test_case":
			_, err := apiClient.GetTestCase(context.Background(), instance.Primary.ID)
			if err == nil {
				return fmt.Errorf("Coval test case %s still exists after destroy", instance.Primary.ID)
			}
			if !client.IsNotFound(err) {
				return fmt.Errorf("verify Coval test case %s was destroyed: %w", instance.Primary.ID, err)
			}
		case "coval_test_set":
			_, err := apiClient.GetTestSet(context.Background(), instance.Primary.ID)
			if err == nil {
				return fmt.Errorf("Coval test set %s still exists after destroy", instance.Primary.ID)
			}
			if !client.IsNotFound(err) {
				return fmt.Errorf("verify Coval test set %s was destroyed: %w", instance.Primary.ID, err)
			}
		}
	}
	return nil
}

func sweepTestCases(_ string) error {
	apiClient, err := acceptanceAPIClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	testSetOptions := client.ListTestSetsOptions{PageSize: 100}
	for {
		page, err := apiClient.ListTestSets(ctx, testSetOptions)
		if err != nil {
			return err
		}
		for _, testSet := range page.TestSets {
			if !strings.HasPrefix(testSet.DisplayName, acceptanceTestSetPrefix) {
				continue
			}
			if err := sweepTestCasesInTestSet(ctx, apiClient, testSet.ID); err != nil {
				return err
			}
		}
		if page.NextPageToken == "" {
			return nil
		}
		testSetOptions.PageToken = page.NextPageToken
	}
}

func sweepTestCasesInTestSet(ctx context.Context, apiClient *client.Client, testSetID string) error {
	options := client.ListTestCasesOptions{
		Filter:   "test_set_id=" + strconv.Quote(testSetID),
		PageSize: 100,
	}
	for {
		page, err := apiClient.ListTestCases(ctx, options)
		if err != nil {
			return fmt.Errorf("list Coval test cases in test set %s: %w", testSetID, err)
		}
		for _, testCase := range page.TestCases {
			if err := apiClient.DeleteTestCase(ctx, testCase.ID); err != nil && !client.IsNotFound(err) {
				return fmt.Errorf("sweep Coval test case %s: %w", testCase.ID, err)
			}
		}
		if page.NextPageToken == "" {
			return nil
		}
		options.PageToken = page.NextPageToken
	}
}

func testAccTestCaseConfig(primaryDisplayName string, secondaryDisplayName string, phase string) string {
	testSetReference := "coval_test_set.primary.id"
	inputString := "Ask whether a damaged order can be refunded"
	expectedBehaviors := `["Explain the refund policy", "Offer the next step"]`
	expectedOutput := `{ eligible = true, resolution = "refund" }`
	description := "Refund-policy regression"
	inputType := "SCENARIO"
	scriptTurns := "null"
	simulationMetadata := `{ channel = "voice" }`
	metricInput := `{ policy_window_days = 30 }`
	userNotes := "Created by the Terraform provider acceptance suite"

	switch phase {
	case "script":
		testSetReference = "coval_test_set.secondary.id"
		inputString = "Follow the scripted refund flow"
		expectedBehaviors = `[]`
		expectedOutput = `{}`
		description = ""
		inputType = "SCRIPT"
		scriptTurns = `["Hello", { type = "dtmf", digits = "1#" }, { type = "skip" }]`
		simulationMetadata = `{}`
		metricInput = `{}`
		userNotes = ""
	case "scenario-after-script":
		inputString = "Return to the scenario flow"
		expectedBehaviors = `[]`
		expectedOutput = `{}`
		description = ""
		simulationMetadata = `{}`
		metricInput = `{}`
		userNotes = ""
	}

	return fmt.Sprintf(`
resource "coval_test_set" "primary" {
  display_name  = %s
  description   = "Test Case acceptance-test parent"
  test_set_type = "SCENARIO"
}

resource "coval_test_set" "secondary" {
  display_name  = %s
  description   = "Test Case acceptance-test move target"
  test_set_type = "SCENARIO"
}

resource "coval_test_case" "test" {
  test_set_id              = %s
  input_str                = %s
  expected_behaviors       = %s
  expected_output_json     = %s
  description              = %s
  input_type               = %s
  script_turns             = %s
  simulation_metadata_input = %s
  metric_input             = %s
  user_notes               = %s
}

data "coval_test_case" "current" {
  id = coval_test_case.test.id
}

data "coval_test_cases" "matching" {
  filter     = format("test_set_id=\"%%s\"", %s)
  depends_on = [coval_test_case.test]
}
`,
		strconv.Quote(primaryDisplayName),
		strconv.Quote(secondaryDisplayName),
		testSetReference,
		strconv.Quote(inputString),
		expectedBehaviors,
		expectedOutput,
		strconv.Quote(description),
		strconv.Quote(inputType),
		scriptTurns,
		simulationMetadata,
		metricInput,
		strconv.Quote(userNotes),
		testSetReference,
	)
}

func testAccTestCaseComputedReferencesConfig(displayName string) string {
	return fmt.Sprintf(`
resource "coval_test_set" "test" {
  display_name  = %s
  description   = "Computed-reference acceptance-test parent"
  test_set_type = "SCRIPT"
}

resource "coval_test_case" "test" {
  test_set_id = coval_test_set.test.id
  input_str   = "Use values computed during this Terraform apply"
  input_type  = "SCRIPT"

  script_turns = [
    coval_test_set.test.name,
  ]

  simulation_metadata_input = {
    test_set_name = coval_test_set.test.name
  }
}
`, strconv.Quote(displayName))
}
