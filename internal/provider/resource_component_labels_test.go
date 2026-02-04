package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestResourceComponentLabels_CRUD(t *testing.T) {
	state := newMockState()
	server := startMockGraphQLServer(state)
	defer server.Close()

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

	resourceName := "compass_component_labels.test"
	initial := fmt.Sprintf(`
provider "compass" {
  email     = "test@example.com"
  api_token = "test-token"
  base_url  = "%s"
  tenant    = "temabit"
}

resource "compass_component_labels" "test" {
  component_id = "cmp-1"
  labels       = ["production"]
}
`, server.URL)

	updated := fmt.Sprintf(`
provider "compass" {
  email     = "test@example.com"
  api_token = "test-token"
  base_url  = "%s"
  tenant    = "temabit"
}

resource "compass_component_labels" "test" {
  component_id = "cmp-1"
  labels       = ["production", "staging", "team-backend"]
}
`, server.URL)

	resource.ParallelTest(t, resource.TestCase{
		IsUnitTest:        true,
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: initial,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "component_id", "cmp-1"),
					resource.TestCheckResourceAttr(resourceName, "cloud_id", state.cloudID),
					resource.TestCheckResourceAttr(resourceName, "id", "cmp-1"),
					resource.TestCheckResourceAttr(resourceName, "labels.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "labels.*", "production"),
				),
			},
			{
				Config: updated,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "labels.#", "3"),
					resource.TestCheckTypeSetElemAttr(resourceName, "labels.*", "production"),
					resource.TestCheckTypeSetElemAttr(resourceName, "labels.*", "staging"),
					resource.TestCheckTypeSetElemAttr(resourceName, "labels.*", "team-backend"),
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
					return rs.Primary.Attributes["component_id"], nil
				},
			},
		},
	})
}

func TestParseComponentLabelsImportID(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		wantComp  string
		wantCloud string
	}{
		{
			name:      "ARI only - no truncation at slash",
			id:        "ari:cloud:compass:abc:component/123",
			wantComp:  "ari:cloud:compass:abc:component/123",
			wantCloud: "",
		},
		{
			name:      "ARI with cloud_id suffix (colon)",
			id:        "ari:cloud:compass:abc:component/123:a1250265-f505-432c-90ff-5d28665aa42c",
			wantComp:  "ari:cloud:compass:abc:component/123",
			wantCloud: "a1250265-f505-432c-90ff-5d28665aa42c",
		},
		{
			name:      "ARI with cloud_id suffix (slash)",
			id:        "ari:cloud:compass:abc:component/123/a1250265-f505-432c-90ff-5d28665aa42c",
			wantComp:  "ari:cloud:compass:abc:component/123",
			wantCloud: "a1250265-f505-432c-90ff-5d28665aa42c",
		},
		{
			name:      "simple id",
			id:        "cmp-1",
			wantComp:  "cmp-1",
			wantCloud: "",
		},
		{
			name:      "simple id with cloud_id",
			id:        "cmp-1:a1250265-f505-432c-90ff-5d28665aa42c",
			wantComp:  "cmp-1",
			wantCloud: "a1250265-f505-432c-90ff-5d28665aa42c",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotComp, gotCloud := parseComponentLabelsImportID(tt.id)
			if gotComp != tt.wantComp || gotCloud != tt.wantCloud {
				t.Errorf("parseComponentLabelsImportID(%q) = component_id %q, cloud_id %q; want %q, %q",
					tt.id, gotComp, gotCloud, tt.wantComp, tt.wantCloud)
			}
		})
	}
}
