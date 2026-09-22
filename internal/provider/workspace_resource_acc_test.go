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

const acceptanceWorkspacePrefix = "tf-acc-workspace"

func init() {
	resource.AddTestSweepers("coval_workspace", &resource.Sweeper{
		Name:         "coval_workspace",
		Dependencies: []string{"coval_test_set"},
		F:            sweepWorkspaces,
	})
}

func TestAccWorkspaceResource(t *testing.T) {
	testID := acctest.RandomWithPrefix(acceptanceWorkspacePrefix)
	displayName := "Terraform acceptance " + testID
	workspaceResourceName := "coval_workspace.test"
	testSetResourceName := "coval_test_set.scoped"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckWorkspaceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkspaceConfig(testID, displayName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(workspaceResourceName, "display_name", displayName),
					resource.TestCheckResourceAttr(workspaceResourceName, "status", "ACTIVE"),
					resource.TestCheckResourceAttr(workspaceResourceName, "workspace_type", "CUSTOM"),
					resource.TestCheckResourceAttrSet(workspaceResourceName, "id"),
					resource.TestCheckResourceAttrPair(testSetResourceName, "workspace_id", workspaceResourceName, "id"),
					resource.TestCheckResourceAttrPair("data.coval_workspace.current", "id", workspaceResourceName, "id"),
					resource.TestCheckResourceAttrPair("data.coval_test_set.scoped", "workspace_id", workspaceResourceName, "id"),
					resource.TestCheckResourceAttrPair("data.coval_test_set.scoped", "id", testSetResourceName, "id"),
					resource.TestCheckResourceAttr("data.coval_test_sets.scoped", "test_sets.#", "1"),
					resource.TestCheckResourceAttr("data.coval_test_sets.default", "test_sets.#", "0"),
					testAccCheckWorkspaceListed(workspaceResourceName, "data.coval_workspaces.all"),
				),
			},
			{
				ResourceName:    testSetResourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
			{
				ResourceName:    workspaceResourceName,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithResourceIdentity,
			},
			{
				Config: testAccWorkspaceConfig(testID, displayName+" updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(workspaceResourceName, "display_name", displayName+" updated"),
					resource.TestCheckResourceAttrPair(testSetResourceName, "workspace_id", workspaceResourceName, "id"),
				),
			},
		},
	})
}

func testAccWorkspaceConfig(testID string, displayName string) string {
	testSetDisplayName := "Scoped test set " + testID
	filter := fmt.Sprintf("display_name=%q", testSetDisplayName)
	return fmt.Sprintf(`
resource "coval_workspace" "test" {
  display_name = %s
}

resource "coval_test_set" "scoped" {
  workspace_id = coval_workspace.test.id
  display_name = %s
  description  = "Managed by the Terraform provider workspace acceptance suite"
  tags         = ["terraform", "acceptance"]
}

data "coval_workspace" "current" {
  id = coval_workspace.test.id
}

data "coval_workspaces" "all" {
  depends_on = [coval_workspace.test]
}

data "coval_test_set" "scoped" {
  workspace_id = coval_workspace.test.id
  id           = coval_test_set.scoped.id
}

data "coval_test_sets" "scoped" {
  workspace_id = coval_workspace.test.id
  filter       = %s
  depends_on   = [coval_test_set.scoped]
}

data "coval_test_sets" "default" {
  filter     = %s
  depends_on = [coval_test_set.scoped]
}
`, strconv.Quote(displayName), strconv.Quote(testSetDisplayName), strconv.Quote(filter), strconv.Quote(filter))
}

func testAccCheckWorkspaceListed(workspaceResourceName string, dataSourceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		workspace, ok := state.RootModule().Resources[workspaceResourceName]
		if !ok {
			return fmt.Errorf("workspace resource %s was not found", workspaceResourceName)
		}
		dataSource, ok := state.RootModule().Resources[dataSourceName]
		if !ok {
			return fmt.Errorf("workspace data source %s was not found", dataSourceName)
		}
		for key, value := range dataSource.Primary.Attributes {
			if strings.HasSuffix(key, ".id") && value == workspace.Primary.ID {
				return nil
			}
		}
		return fmt.Errorf("workspace %s was not returned by %s", workspace.Primary.ID, dataSourceName)
	}
}

func testAccCheckWorkspaceDestroy(state *terraform.State) error {
	apiClient, err := acceptanceAPIClient()
	if err != nil {
		return err
	}
	for _, instance := range state.RootModule().Resources {
		if instance.Type != "coval_workspace" {
			continue
		}
		_, err := apiClient.GetWorkspace(context.Background(), instance.Primary.ID)
		if err == nil {
			return fmt.Errorf("Coval workspace %s still exists after destroy", instance.Primary.ID)
		}
		if !client.IsNotFound(err) {
			return fmt.Errorf("verify Coval workspace %s was destroyed: %w", instance.Primary.ID, err)
		}
	}
	return nil
}

func sweepWorkspaces(_ string) error {
	apiClient, err := acceptanceAPIClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	workspaces, err := apiClient.ListWorkspaces(ctx)
	if err != nil {
		return err
	}
	for _, workspace := range workspaces {
		if workspace.WorkspaceType != "CUSTOM" ||
			!strings.HasPrefix(workspace.DisplayName, "Terraform acceptance "+acceptanceWorkspacePrefix) {
			continue
		}
		if err := sweepWorkspaceTestSets(ctx, apiClient.ForWorkspace(workspace.ID)); err != nil {
			return err
		}
		if err := apiClient.DeleteWorkspace(ctx, workspace.ID); err != nil && !client.IsNotFound(err) {
			return fmt.Errorf("sweep Coval workspace %s: %w", workspace.ID, err)
		}
	}
	return nil
}

func sweepWorkspaceTestSets(ctx context.Context, apiClient *client.Client) error {
	options := client.ListTestSetsOptions{PageSize: 100}
	for {
		page, err := apiClient.ListTestSets(ctx, options)
		if err != nil {
			return err
		}
		for _, testSet := range page.TestSets {
			if err := apiClient.DeleteTestSet(ctx, testSet.ID); err != nil && !client.IsNotFound(err) {
				return fmt.Errorf("sweep Coval test set %s: %w", testSet.ID, err)
			}
		}
		if page.NextPageToken == "" {
			return nil
		}
		options.PageToken = page.NextPageToken
	}
}
