package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestResourceComponent_CRUD(t *testing.T) {
	state := newMockState()
	server := startMockGraphQLServer(state)
	defer server.Close()

	prov := New()
	providerFactories := map[string]func() (*schema.Provider, error){
		"compass": func() (*schema.Provider, error) { return prov, nil },
	}

	resourceName := "compass_component.test"
	initial := fmt.Sprintf(`
provider "compass" {
  email     = "test@example.com"
  api_token = "test-token"
  base_url  = "%s"
  tenant    = "temabit"
}

resource "compass_component" "test" {
  name        = "svc-a"
  description = "desc-1"
  type        = "SERVICE"
}
`, server.URL)

	updated := fmt.Sprintf(`
provider "compass" {
  email     = "test@example.com"
  api_token = "test-token"
  base_url  = "%s"
  tenant    = "temabit"
}

resource "compass_component" "test" {
  name        = "svc-a-upd"
  description = ""
  type        = "SERVICE"
  owner_id    = "owner-xyz"
}
`, server.URL)

	resource.ParallelTest(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: initial,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "svc-a"),
					resource.TestCheckResourceAttr(resourceName, "description", "desc-1"),
					resource.TestCheckResourceAttr(resourceName, "type", "SERVICE"),
					resource.TestCheckResourceAttr(resourceName, "cloud_id", state.cloudID),
				),
			},
			{
				Config: updated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", "svc-a-upd"),
					resource.TestCheckResourceAttr(resourceName, "description", ""),
					resource.TestCheckResourceAttr(resourceName, "owner_id", "owner-xyz"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"cloud_id", "type"},
			},
		},
	})
}
