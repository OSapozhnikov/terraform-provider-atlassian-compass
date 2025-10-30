package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestResourceComponentLink_CRUD(t *testing.T) {
	state := newMockState()
	server := startMockGraphQLServer(state)
	defer server.Close()

	// Seed a component that the link will attach to (simulate that it exists in API)
	state.components["cmp-1"] = map[string]interface{}{
		"id":          "cmp-1",
		"name":        "svc-a",
		"description": "",
		"typeId":      "type-service",
		"ownerId":     "",
	}

	prov := New()
	providerFactories := map[string]func() (*schema.Provider, error){
		"compass": func() (*schema.Provider, error) { return prov, nil },
	}

	resourceName := "compass_component_link.test"
	initial := fmt.Sprintf(`
provider "compass" {
  email     = "test@example.com"
  api_token = "test-token"
  base_url  = "%s"
  tenant    = "temabit"
}

resource "compass_component_link" "test" {
  component_id = "cmp-1"
  name         = "Repo"
  type         = "REPOSITORY"
  url          = "https://example.com/repo"
}
`, server.URL)

	updated := fmt.Sprintf(`
provider "compass" {
  email     = "test@example.com"
  api_token = "test-token"
  base_url  = "%s"
  tenant    = "temabit"
}

resource "compass_component_link" "test" {
  component_id = "cmp-1"
  name         = "Repo-2"
  type         = "REPOSITORY"
  url          = "https://example.com/repo2"
  object_id    = "obj-123"
}
`, server.URL)

	resource.ParallelTest(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: initial,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Repo"),
					resource.TestCheckResourceAttr(resourceName, "type", "REPOSITORY"),
					resource.TestCheckResourceAttr(resourceName, "url", "https://example.com/repo"),
					resource.TestCheckResourceAttr(resourceName, "cloud_id", state.cloudID),
				),
			},
			{
				Config: updated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "Repo-2"),
					resource.TestCheckResourceAttr(resourceName, "url", "https://example.com/repo2"),
					resource.TestCheckResourceAttr(resourceName, "object_id", "obj-123"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"cloud_id"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[resourceName]
					if !ok {
						return "", fmt.Errorf("resource not found: %s", resourceName)
					}
					return fmt.Sprintf("%s:%s", rs.Primary.Attributes["component_id"], rs.Primary.ID), nil
				},
			},
		},
	})
}
