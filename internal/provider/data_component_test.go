package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestDataSourceComponent_ByID(t *testing.T) {
	state := newMockState()
	server := startMockGraphQLServer(state)
	defer server.Close()

	state.components["cmp-ds-1"] = map[string]interface{}{
		"id":          "cmp-ds-1",
		"name":        "svc-by-id",
		"description": "A component looked up by ID",
		"typeId":      "SERVICE",
		"ownerId":     "owner-1",
		"slug":        "svc-by-id-slug",
	}

	prov := New()
	providerFactories := map[string]func() (*schema.Provider, error){
		"compass": func() (*schema.Provider, error) { return prov, nil },
	}

	config := fmt.Sprintf(`
provider "compass" {
  email     = "test@example.com"
  api_token = "test-token"
  base_url  = "%s"
}

data "compass_component" "by_id" {
  id = "cmp-ds-1"
}
`, server.URL)

	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.compass_component.by_id", "id", "cmp-ds-1"),
					resource.TestCheckResourceAttr("data.compass_component.by_id", "name", "svc-by-id"),
					resource.TestCheckResourceAttr("data.compass_component.by_id", "slug", "svc-by-id-slug"),
					resource.TestCheckResourceAttr("data.compass_component.by_id", "description", "A component looked up by ID"),
					resource.TestCheckResourceAttr("data.compass_component.by_id", "type_id", "SERVICE"),
					resource.TestCheckResourceAttr("data.compass_component.by_id", "owner_id", "owner-1"),
				),
			},
		},
	})
}

func TestDataSourceComponent_BySlug(t *testing.T) {
	state := newMockState()
	server := startMockGraphQLServer(state)
	defer server.Close()

	prov := New()
	providerFactories := map[string]func() (*schema.Provider, error){
		"compass": func() (*schema.Provider, error) { return prov, nil },
	}

	config := fmt.Sprintf(`
provider "compass" {
  email     = "test@example.com"
  api_token = "test-token"
  base_url  = "%s"
  tenant    = "temabit"
}

data "compass_component" "by_slug" {
  slug = "temabit-product-example"
}
`, server.URL)

	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.compass_component.by_slug", "id", "ari:cloud:compass:cloud-123:component/uuid/euid"),
					resource.TestCheckResourceAttr("data.compass_component.by_slug", "name", "product-example"),
					resource.TestCheckResourceAttr("data.compass_component.by_slug", "slug", "temabit-product-example"),
					resource.TestCheckResourceAttr("data.compass_component.by_slug", "type_id", "SERVICE"),
					resource.TestCheckResourceAttr("data.compass_component.by_slug", "cloud_id", state.cloudID),
				),
			},
		},
	})
}

func TestDataSourceComponent_Validation_BothIDAndSlug(t *testing.T) {
	state := newMockState()
	server := startMockGraphQLServer(state)
	defer server.Close()

	prov := New()
	providerFactories := map[string]func() (*schema.Provider, error){
		"compass": func() (*schema.Provider, error) { return prov, nil },
	}

	config := fmt.Sprintf(`
provider "compass" {
  email     = "test@example.com"
  api_token = "test-token"
  base_url  = "%s"
}

data "compass_component" "invalid" {
  id   = "cmp-1"
  slug = "my-slug"
}
`, server.URL)

	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexpOrContains("exactly one of id or slug"),
			},
		},
	})
}

func TestDataSourceComponent_Validation_NeitherIDNorSlug(t *testing.T) {
	state := newMockState()
	server := startMockGraphQLServer(state)
	defer server.Close()

	prov := New()
	providerFactories := map[string]func() (*schema.Provider, error){
		"compass": func() (*schema.Provider, error) { return prov, nil },
	}

	config := fmt.Sprintf(`
provider "compass" {
  email     = "test@example.com"
  api_token = "test-token"
  base_url  = "%s"
}

data "compass_component" "invalid" {}
`, server.URL)

	resource.Test(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexpOrContains("either id or slug must be set"),
			},
		},
	})
}

// regexpOrContains returns a regexp that matches either the literal pattern or "contained in error message".
// resource.TestCase ExpectError accepts *regexp.Regexp; we use a regex that matches our error substrings.
func regexpOrContains(substring string) *regexp.Regexp {
	return regexp.MustCompile(".*" + regexp.QuoteMeta(substring) + ".*")
}
