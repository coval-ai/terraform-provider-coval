package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/coval-ai/terraform-provider-coval/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const acceptanceTestSetPrefix = "tf-acc-test-set-"

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"coval": providerserver.NewProtocol6WithError(New("acceptance")()),
}

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func init() {
	resource.AddTestSweepers("coval_test_set", &resource.Sweeper{
		Name:         "coval_test_set",
		Dependencies: []string{"coval_test_case"},
		F:            sweepTestSets,
	})
}

func TestAccTestSetResource(t *testing.T) {
	displayName := acctest.RandomWithPrefix(acceptanceTestSetPrefix)
	resourceName := "coval_test_set.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTestSetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTestSetConfig(displayName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "display_name", displayName),
					resource.TestCheckResourceAttr(resourceName, "description", "Managed by the Terraform provider acceptance suite"),
					resource.TestCheckResourceAttr(resourceName, "test_set_type", "SCENARIO"),
					resource.TestCheckResourceAttr(resourceName, "test_set_metadata.owner", "terraform-provider-coval"),
					resource.TestCheckResourceAttr(resourceName, "parameters.customer_name.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "2"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "name"),
					resource.TestCheckResourceAttrSet(resourceName, "slug"),
					resource.TestCheckResourceAttr("data.coval_test_set.current", "display_name", displayName),
					resource.TestCheckResourceAttr("data.coval_test_sets.matching", "test_sets.#", "1"),
					resource.TestCheckResourceAttr("data.coval_test_sets.matching", "test_sets.0.display_name", displayName),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccTestSetConfig(displayName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "description", "Updated by Terraform"),
					resource.TestCheckResourceAttr(resourceName, "test_set_type", "WORKFLOW"),
					resource.TestCheckResourceAttr(resourceName, "test_set_metadata.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "parameters.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "tags.#", "0"),
					resource.TestCheckResourceAttr("data.coval_test_sets.matching", "test_sets.#", "0"),
				),
			},
		},
	})
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	apiKey := os.Getenv("COVAL_API_KEY")
	baseURL := os.Getenv("COVAL_API_BASE_URL")
	if apiKey == "" {
		t.Fatal("COVAL_API_KEY must be set for acceptance tests")
	}
	if baseURL == "" {
		t.Fatal("COVAL_API_BASE_URL must be set for acceptance tests")
	}
	apiClient, err := client.New(apiKey, baseURL)
	if err != nil {
		t.Fatalf("configure acceptance API client: %v", err)
	}
	if _, err := apiClient.ListTestSets(t.Context(), client.ListTestSetsOptions{PageSize: 1}); err != nil {
		t.Fatalf("verify acceptance API credentials with a read-only request: %v", err)
	}
}

func testAccCheckTestSetDestroy(state *terraform.State) error {
	apiClient, err := acceptanceAPIClient()
	if err != nil {
		return err
	}
	for _, instance := range state.RootModule().Resources {
		if instance.Type != "coval_test_set" {
			continue
		}
		_, err := apiClient.GetTestSet(context.Background(), instance.Primary.ID)
		if err == nil {
			return fmt.Errorf("Coval test set %s still exists after destroy", instance.Primary.ID)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("verify Coval test set %s was destroyed: %w", instance.Primary.ID, err)
		}
	}
	return nil
}

func sweepTestSets(_ string) error {
	apiClient, err := acceptanceAPIClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	options := client.ListTestSetsOptions{PageSize: 100}
	for {
		page, err := apiClient.ListTestSets(ctx, options)
		if err != nil {
			return err
		}
		for _, testSet := range page.TestSets {
			if strings.HasPrefix(testSet.DisplayName, acceptanceTestSetPrefix) {
				if err := apiClient.DeleteTestSet(ctx, testSet.ID); err != nil && !client.IsNotFound(err) {
					return fmt.Errorf("sweep Coval test set %s: %w", testSet.ID, err)
				}
			}
		}
		if page.NextPageToken == "" {
			return nil
		}
		options.PageToken = page.NextPageToken
	}
}

func acceptanceAPIClient() (*client.Client, error) {
	apiKey := os.Getenv("COVAL_API_KEY")
	baseURL := os.Getenv("COVAL_API_BASE_URL")
	if apiKey == "" || baseURL == "" {
		return nil, fmt.Errorf("COVAL_API_KEY and COVAL_API_BASE_URL must be set")
	}
	return client.New(apiKey, baseURL)
}

func testAccTestSetConfig(displayName string, cleared bool) string {
	description := "Managed by the Terraform provider acceptance suite"
	testSetType := "SCENARIO"
	metadata := `{ owner = "terraform-provider-coval", nested = { enabled = true } }`
	parameters := `{ customer_name = ["Alice", "Bob"], priority = [1, 2] }`
	tags := `["terraform", "acceptance"]`
	if cleared {
		description = "Updated by Terraform"
		testSetType = "WORKFLOW"
		metadata = `{}`
		parameters = `{}`
		tags = `[]`
	}
	filter := fmt.Sprintf("display_name=%q", displayName)
	return fmt.Sprintf(`
resource "coval_test_set" "test" {
  display_name      = %s
  description       = %s
  test_set_type     = %s
  test_set_metadata = %s
  parameters        = %s
  tags              = %s
}

data "coval_test_set" "current" {
  id = coval_test_set.test.id
}

data "coval_test_sets" "matching" {
  filter      = %s
  tag_filters = ["terraform", "acceptance"]
  depends_on  = [coval_test_set.test]
}
`, strconv.Quote(displayName), strconv.Quote(description), strconv.Quote(testSetType), metadata, parameters, tags, strconv.Quote(filter))
}
