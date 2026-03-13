package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// componentByIDQuery retrieves a single component by its ID (ARI or internal ID).
// Requested fields align with componentByReference for consistent output.
const componentByIDQuery = `
	query GetComponent($id: ID!) {
		compass {
			component(id: $id) {
				__typename
				... on CompassComponent {
					id
					name
					slug
					description
					url
					typeId
					ownerId
				}
				... on QueryError {
					message
				}
			}
		}
	}
`

// componentByReferenceQuery retrieves a single component by slug and cloudId.
// Variable: reference: { slug: { slug: "...", cloudId: "..." } }
const componentByReferenceQuery = `
	query ComponentByReference($reference: ComponentReferenceInput!) {
		compass {
			componentByReference(reference: $reference) {
				__typename
				... on CompassComponent {
					id
					name
					slug
					description
					url
					typeId
					ownerId
				}
				... on QueryError {
					message
				}
			}
		}
	}
`

// componentResultPayload represents the union CompassComponentResult (CompassComponent | QueryError).
type componentResultPayload struct {
	Typename   string `json:"__typename"`
	ID         string `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	Slug       string `json:"slug,omitempty"`
	Description string `json:"description,omitempty"`
	URL        string `json:"url,omitempty"`
	TypeID     string `json:"typeId,omitempty"`
	OwnerID    string `json:"ownerId,omitempty"`
	Message    string `json:"message,omitempty"`
}

type componentByIDResponse struct {
	Compass struct {
		Component componentResultPayload `json:"component"`
	} `json:"compass"`
}

type componentByReferenceResponse struct {
	Compass struct {
		ComponentByReference componentResultPayload `json:"componentByReference"`
	} `json:"compass"`
}

// dataSourceComponent exposes a single Compass component as a Terraform data source.
// Lookup is by id (Query component) or by slug (Query componentByReference); exactly one must be set.
//
// Example by ID:
//
//	data "compass_component" "by_id" { id = "ari:cloud:compass:...:component/..." }
//
// Example by slug:
//
//	data "compass_component" "by_slug" { slug = "my-component-slug" }
//	# cloud_id optional when tenant is set in provider
func dataSourceComponent() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceComponentRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "ARI or internal ID of the Compass component. Use this for lookup via Query component(id). After read, holds the component's ARI. Exactly one of id or slug must be set.",
			},
			"slug": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Slug of the Compass component. Use this for lookup via Query componentByReference. After read, holds the component's slug. Exactly one of id or slug must be set.",
			},
			"cloud_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Cloud ID of the Atlassian site. Required for slug lookup when tenant is not set in the provider; otherwise auto-detected from tenant.",
			},
			// Computed attributes
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the component.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Description of the component.",
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL to the component in Compass.",
			},
			"type_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Type ID of the component (e.g. SERVICE, or custom type ARI).",
			},
			"owner_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Owner team or user ID of the component.",
			},
		},
	}
}

func dataSourceComponentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerConfig := m.(*ProviderConfig)
	compassClient := providerConfig.Client

	idRaw, hasID := d.GetOk("id")
	slugRaw, hasSlug := d.GetOk("slug")

	if hasID && hasSlug {
		return diag.Errorf("exactly one of id or slug must be set in compass_component data source")
	}
	if !hasID && !hasSlug {
		return diag.Errorf("either id or slug must be set in compass_component data source")
	}

	var payload componentResultPayload

	if hasID {
		id := idRaw.(string)
		if id == "" {
			return diag.Errorf("id cannot be empty in compass_component data source")
		}
		variables := map[string]interface{}{"id": id}
		data, err := compassClient.ExecuteQuery(ctx, componentByIDQuery, variables)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to query component by id: %w", err))
		}
		var resp componentByIDResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return diag.FromErr(fmt.Errorf("failed to unmarshal component response: %w", err))
		}
		payload = resp.Compass.Component
	} else {
		slug := slugRaw.(string)
		if slug == "" {
			return diag.Errorf("slug cannot be empty in compass_component data source")
		}
		cloudID := ""
		if v, ok := d.GetOk("cloud_id"); ok && v.(string) != "" {
			cloudID = v.(string)
		} else {
			if providerConfig.Tenant == "" {
				return diag.Errorf("cloud_id is required for slug lookup when tenant is not configured in provider")
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
		reference := map[string]interface{}{
			"slug": map[string]interface{}{
				"slug":    slug,
				"cloudId": cloudID,
			},
		}
		variables := map[string]interface{}{"reference": reference}
		data, err := compassClient.ExecuteQuery(ctx, componentByReferenceQuery, variables)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to query component by slug: %w", err))
		}
		var resp componentByReferenceResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return diag.FromErr(fmt.Errorf("failed to unmarshal componentByReference response: %w", err))
		}
		payload = resp.Compass.ComponentByReference
	}

	if payload.Typename == "QueryError" && payload.Message != "" {
		return diag.Errorf("compass component query failed: %s", payload.Message)
	}
	if payload.ID == "" {
		return diag.Errorf("compass component query returned no component")
	}

	if err := d.Set("id", payload.ID); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set id: %w", err))
	}
	if err := d.Set("name", payload.Name); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set name: %w", err))
	}
	if err := d.Set("slug", payload.Slug); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set slug: %w", err))
	}
	if err := d.Set("description", payload.Description); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set description: %w", err))
	}
	if err := d.Set("url", payload.URL); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set url: %w", err))
	}
	if err := d.Set("type_id", payload.TypeID); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set type_id: %w", err))
	}
	if err := d.Set("owner_id", payload.OwnerID); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set owner_id: %w", err))
	}

	d.SetId(payload.ID)
	return nil
}
