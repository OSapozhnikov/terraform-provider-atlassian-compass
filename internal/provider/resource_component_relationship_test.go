package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestResourceComponentRelationship_CRUD(t *testing.T) {
	state := newMockState()
	server := startMockGraphQLServer(state)
	defer server.Close()

	state.components["cmp-1"] = map[string]interface{}{
		"id": "cmp-1", "name": "svc-a", "description": "", "typeId": "type-service", "ownerId": "",
	}
	state.components["cmp-2"] = map[string]interface{}{
		"id": "cmp-2", "name": "svc-b", "description": "", "typeId": "type-service", "ownerId": "",
	}

	prov := New()
	providerFactories := map[string]func() (*schema.Provider, error){
		"compass": func() (*schema.Provider, error) { return prov, nil },
	}

	resourceName := "compass_component_relationship.test"
	config := fmt.Sprintf(`
provider "compass" {
  email     = "test@example.com"
  api_token = "test-token"
  base_url  = "%s"
  tenant    = "temabit"
}

resource "compass_component_relationship" "test" {
  start_node_id     = "cmp-1"
  end_node_id       = "cmp-2"
  relationship_type = "DEPENDS_ON"
}
`, server.URL)

	resource.ParallelTest(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "start_node_id", "cmp-1"),
					resource.TestCheckResourceAttr(resourceName, "end_node_id", "cmp-2"),
					resource.TestCheckResourceAttr(resourceName, "relationship_type", "DEPENDS_ON"),
					resource.TestCheckResourceAttr(resourceName, "cloud_id", state.cloudID),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
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
					return fmt.Sprintf("%s:%s:%s",
						rs.Primary.Attributes["start_node_id"],
						rs.Primary.Attributes["end_node_id"],
						rs.Primary.Attributes["relationship_type"]), nil
				},
			},
		},
	})
}
