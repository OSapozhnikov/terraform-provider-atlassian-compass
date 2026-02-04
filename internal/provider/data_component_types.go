package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// GraphQL query to retrieve component types for a given cloudId.
// We deliberately keep the selection minimal and only request the fields
// we currently expose via the data source (id and name), plus __typename
// and optional QueryError.message for basic error reporting.
const componentTypesQuery = `
	query GetComponentTypes($cloudId: ID!) {
		compass {
			componentTypes(cloudId: $cloudId) {
				__typename
				... on CompassComponentTypeConnection {
					nodes {
						id
						name
					}
				}
				... on QueryError {
					message
				}
			}
		}
	}
`

// componentTypesPayload represents the minimal componentTypes union payload
// returned by the GraphQL API for this query.
type componentTypesPayload struct {
	Typename string `json:"__typename"`
	// When the union resolves to CompassComponentTypeConnection.
	Nodes []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"nodes,omitempty"`
	// When the union resolves to QueryError.
	Message string `json:"message,omitempty"`
}

// componentTypesResponse is a minimal wrapper used to extract the
// componentTypes union payload from the GraphQL response.
type componentTypesResponse struct {
	Compass struct {
		ComponentTypes componentTypesPayload `json:"componentTypes"`
	} `json:"compass"`
}

// dataSourceComponentTypes exposes Compass component types as a Terraform
// data source. It is intended as a low-level building block and for
// introspection of the available component types (including custom ones)
// for a given cloudId.
//
// Example:
//
//	data "compass_component_types" "all" {}
//
// Attributes:
//   - cloud_id: the cloudId used for the query (explicit or auto-detected)
//   - types:    list of objects with id and name for each component type
func dataSourceComponentTypes() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceComponentTypesRead,
		Schema: map[string]*schema.Schema{
			"cloud_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Cloud ID of the Atlassian site. If not provided, it will be auto-detected from the provider's tenant setting.",
			},
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Optional filter to return only the component type with this id. For built-in types this is the enum-style key (e.g. SERVICE, LIBRARY, APPLICATION); for custom types this is the full type ARI.",
			},
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Optional filter to return only component types with this human-readable name (for example \"Service\", \"Domain\").",
			},
			"types": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of available Compass component types for the given cloud_id, each with id and name.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Identifier of the component type. For built-in types this is the enum-style key (e.g. SERVICE, LIBRARY, APPLICATION), for custom types this is the full type ARI.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Human-readable name of the component type.",
						},
					},
				},
			},
		},
	}
}

func dataSourceComponentTypesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerConfig := m.(*ProviderConfig)
	compassClient := providerConfig.Client

	// Validate filter usage: allow at most one of id or name to be set.
	filterIDRaw, hasID := d.GetOk("id")
	filterNameRaw, hasName := d.GetOk("name")
	if hasID && hasName {
		return diag.Errorf("only one of id or name can be specified in compass_component_types data source")
	}

	// Resolve or auto-detect cloud_id, reusing the same logic as the
	// compass_component resource.
	cloudID := ""
	if v, ok := d.GetOk("cloud_id"); ok && v.(string) != "" {
		cloudID = v.(string)
	} else {
		if providerConfig.Tenant == "" {
			return diag.Errorf("cloud_id is required when tenant is not configured in provider")
		}
		var err error
		cloudID, err = compassClient.GetCloudIDByTenant(ctx, providerConfig.Tenant)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to get cloud_id from tenant %q: %w", providerConfig.Tenant, err))
		}
		if err := d.Set("cloud_id", cloudID); err != nil {
			return diag.FromErr(fmt.Errorf("failed to set cloud_id: %w", err))
		}
	}

	variables := map[string]interface{}{
		"cloudId": cloudID,
	}

	data, err := compassClient.ExecuteQuery(ctx, componentTypesQuery, variables)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to query component types: %w", err))
	}

	var resp componentTypesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal component types response: %w", err))
	}

	payload := resp.Compass.ComponentTypes

	// Handle QueryError union case explicitly so that users get a clear error.
	if payload.Typename == "QueryError" && payload.Message != "" {
		return diag.Errorf("componentTypes query failed for cloud_id %s: %s", cloudID, payload.Message)
	}

	if len(payload.Nodes) == 0 {
		return diag.Errorf("componentTypes query returned no component types for cloud_id %s", cloudID)
	}

	// Build list of all types first.
	types := make([]map[string]interface{}, 0, len(payload.Nodes))
	for _, n := range payload.Nodes {
		types = append(types, map[string]interface{}{
			"id":   n.ID,
			"name": n.Name,
		})
	}

	// Apply optional filters if provided.
	if hasID {
		filterID := filterIDRaw.(string)
		if filterID != "" {
			filtered := make([]map[string]interface{}, 0, 1)
			for _, t := range types {
				if t["id"] == filterID {
					filtered = append(filtered, t)
				}
			}
			if len(filtered) == 0 {
				return diag.Errorf("no component type found with id %q for cloud_id %s", filterID, cloudID)
			}
			types = filtered
		}
	} else if hasName {
		filterName := filterNameRaw.(string)
		if filterName != "" {
			filtered := make([]map[string]interface{}, 0, 1)
			for _, t := range types {
				if t["name"] == filterName {
					filtered = append(filtered, t)
				}
			}
			if len(filtered) == 0 {
				return diag.Errorf("no component type found with name %q for cloud_id %s", filterName, cloudID)
			}
			types = filtered
		}
	}

	if err := d.Set("types", types); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set types: %w", err))
	}

	// Use cloud_id as a stable ID for this data source instance.
	d.SetId(cloudID)

	return nil
}
