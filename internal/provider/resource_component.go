package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/OSapozhnikov/terraform-provider-atlassian-compass/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	createComponentMutation = `
		mutation CreateComponent($cloudId: ID!, $name: String!, $description: String, $typeId: ID!, $ownerId: ID, $slug: String) {
			compass {
				createComponent(
					cloudId: $cloudId
					input: {
						name: $name
						description: $description
						typeId: $typeId
						ownerId: $ownerId
						slug: $slug
					}
				) {
					success
					componentDetails {
						id
						name
						description
						typeId
						ownerId
						slug
					}
				}
			}
		}
	`

	getComponentQuery = `
		query GetComponent($id: ID!) {
			compass {
				component(id: $id) {
					... on CompassComponent {
						id
						name
						description
						typeId
						ownerId
						slug
					}
				}
			}
		}
	`

	deleteComponentMutation = `
		mutation DeleteComponent($input: DeleteCompassComponentInput!) {
			compass {
				deleteComponent(input: $input) {
					success
				}
			}
		}
	`

	updateComponentMutation = `
		mutation UpdateComponent($input: UpdateCompassComponentInput!) {
			compass {
				updateComponent(input: $input) {
					success
					componentDetails {
						id
						name
						description
						typeId
						ownerId
						slug
					}
				}
			}
		}
	`
)

type Component struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Type         string                 `json:"type,omitempty"`   // Enum string (SERVICE, LIBRARY, etc.) - used in create
	TypeID       string                 `json:"typeId,omitempty"` // Type ID returned from API - used in read
	OwnerID      string                 `json:"ownerId,omitempty"`
	Slug         string                 `json:"slug,omitempty"`
	CustomFields map[string]interface{} `json:"customFields,omitempty"`
}

type CreateComponentResponse struct {
	Compass struct {
		CreateComponent struct {
			Success          bool      `json:"success"`
			ComponentDetails Component `json:"componentDetails"`
		} `json:"createComponent"`
	} `json:"compass"`
}

type GetComponentResponse struct {
	Compass struct {
		Component Component `json:"component"`
	} `json:"compass"`
}

type DeleteComponentResponse struct {
	Compass struct {
		DeleteComponent struct {
			Success bool `json:"success"`
		} `json:"deleteComponent"`
	} `json:"compass"`
}

type UpdateComponentResponse struct {
	Compass struct {
		UpdateComponent struct {
			Success          bool      `json:"success"`
			ComponentDetails Component `json:"componentDetails"`
		} `json:"updateComponent"`
	} `json:"compass"`
}

// resolveTypeToTypeID maps a user-provided type (id or name) to the Compass typeId.
// It tries exact id match, then exact name match, then case-insensitive name match.
// If multiple types match the name (case-insensitive), it returns an error.
func resolveTypeToTypeID(types []client.ComponentTypeInfo, userType string) (string, error) {
	// 1. Exact id match
	for _, t := range types {
		if t.ID == userType {
			return t.ID, nil
		}
	}
	// 2. Exact name match
	for _, t := range types {
		if t.Name == userType {
			return t.ID, nil
		}
	}
	// 3. Case-insensitive name match (fail if multiple)
	userLower := strings.ToLower(userType)
	var match *client.ComponentTypeInfo
	for i := range types {
		if strings.ToLower(types[i].Name) == userLower {
			if match != nil {
				return "", fmt.Errorf("ambiguous type %q: multiple component types match (id %s and %s); use type id instead", userType, match.ID, types[i].ID)
			}
			match = &types[i]
		}
	}
	if match != nil {
		return match.ID, nil
	}
	// 4. No match - build helpful error
	ids := make([]string, 0, len(types))
	for _, t := range types {
		ids = append(ids, fmt.Sprintf("%s (%s)", t.ID, t.Name))
	}
	sort.Strings(ids)
	return "", fmt.Errorf("no component type found for %q; available for this site: %s", userType, strings.Join(ids, ", "))
}

func resourceComponent() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceComponentCreate,
		ReadContext:   resourceComponentRead,
		UpdateContext: resourceComponentUpdate,
		DeleteContext: resourceComponentDelete,
		Schema: map[string]*schema.Schema{
			"cloud_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Cloud ID of the Atlassian site (e.g., jira-12345678-1234-1234-1234-123456789012). If not provided, will be automatically detected from tenant configured in provider.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the Compass component",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of the Compass component",
			},
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Type of the Compass component: use the type id (e.g. SERVICE, LIBRARY, APPLICATION, or full ARI for custom types) or the human-readable name (e.g. Service, Domain). Resolved to typeId via the Compass componentTypes API. Use data.compass_component_types to list available types.",
			},
			"owner_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Owner ID (Atlassian account ID) of the Compass component",
			},
			"slug": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A unique identifier for the component. If not set, the API receives null.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceComponentCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerConfig := m.(*ProviderConfig)
	compassClient := providerConfig.Client

	// Get or auto-detect cloud_id
	cloudID := ""
	if v, ok := d.GetOk("cloud_id"); ok && v.(string) != "" {
		cloudID = v.(string)
	} else {
		// Auto-detect cloud_id from tenant
		if providerConfig.Tenant == "" {
			return diag.Errorf("cloud_id is required when tenant is not configured in provider")
		}
		var err error
		cloudID, err = compassClient.GetCloudIDByTenant(ctx, providerConfig.Tenant)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to get cloud_id from tenant '%s': %w", providerConfig.Tenant, err))
		}
		// Save detected cloud_id to state
		if err := d.Set("cloud_id", cloudID); err != nil {
			return diag.FromErr(fmt.Errorf("failed to set cloud_id: %w", err))
		}
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	componentType := strings.TrimSpace(d.Get("type").(string))
	ownerID := d.Get("owner_id").(string)

	if componentType == "" {
		return diag.Errorf("type is required and cannot be empty")
	}

	types, err := compassClient.GetComponentTypes(ctx, cloudID)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to resolve component type: %w", err))
	}
	typeID, err := resolveTypeToTypeID(types, componentType)
	if err != nil {
		return diag.FromErr(err)
	}

	variables := map[string]interface{}{
		"cloudId": cloudID,
		"name":    name,
		"typeId":  typeID,
	}

	if description != "" {
		variables["description"] = description
	}

	if ownerID != "" {
		variables["ownerId"] = ownerID
	}

	if v, ok := d.GetOk("slug"); ok && v.(string) != "" {
		variables["slug"] = v.(string)
	} else {
		variables["slug"] = nil
	}

	data, err := compassClient.ExecuteQuery(ctx, createComponentMutation, variables)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create component: %w", err))
	}

	var response CreateComponentResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	if !response.Compass.CreateComponent.Success {
		return diag.FromErr(fmt.Errorf("failed to create component: GraphQL mutation returned success=false"))
	}

	component := response.Compass.CreateComponent.ComponentDetails
	d.SetId(component.ID)

	return resourceComponentRead(ctx, d, m)
}

func resourceComponentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerConfig := m.(*ProviderConfig)
	compassClient := providerConfig.Client
	componentID := d.Id()

	variables := map[string]interface{}{
		"id": componentID,
	}

	data, err := compassClient.ExecuteQuery(ctx, getComponentQuery, variables)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read component: %w", err))
	}

	var response GetComponentResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	component := response.Compass.Component

	if component.ID == "" {
		d.SetId("")
		return nil
	}

	// cloud_id is required for creating but not returned in read, so we keep it from state
	if cloudID := d.Get("cloud_id"); cloudID != nil {
		d.Set("cloud_id", cloudID)
	}
	d.Set("name", component.Name)
	d.Set("description", component.Description)
	// Handle type field - API returns typeId, but we need to preserve the original enum value
	// Since typeId is an ID (UUID), we keep the original type value from state if available
	// Otherwise, try to use typeId (though this may not match the enum value)
	if currentType := d.Get("type"); currentType != nil && currentType.(string) != "" {
		d.Set("type", currentType.(string))
	} else if component.TypeID != "" {
		// If no type in state, try using typeId (may need mapping later)
		d.Set("type", component.TypeID)
	}
	// Handle owner field
	if component.OwnerID != "" {
		d.Set("owner_id", component.OwnerID)
	}
	d.Set("slug", component.Slug)

	return nil
}

func resourceComponentUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerConfig := m.(*ProviderConfig)
	compassClient := providerConfig.Client
	componentID := d.Id()

	// cloud_id cannot be changed for existing components
	if d.HasChange("cloud_id") {
		return diag.Errorf("cloud_id cannot be changed. Please delete and recreate the component with the new cloud_id.")
	}

	// type cannot be changed via updateComponent mutation (not in UpdateCompassComponentInput)
	if d.HasChange("type") {
		return diag.Errorf("type cannot be changed. Please delete and recreate the component with the new type.")
	}

	// Check if any updatable fields have changed
	if !d.HasChanges("name", "description", "owner_id", "slug") {
		// No changes to updatable fields, just read the state
		return resourceComponentRead(ctx, d, m)
	}

	// Build update input
	input := map[string]interface{}{
		"id": componentID,
	}

	if d.HasChange("name") {
		name := d.Get("name").(string)
		input["name"] = name
	}

	if d.HasChange("description") {
		description := d.Get("description").(string)
		// Include description even if empty to allow clearing it
		input["description"] = description
	}

	if d.HasChange("owner_id") {
		ownerID := d.Get("owner_id").(string)
		// Include ownerId even if empty to allow clearing it
		if ownerID != "" {
			input["ownerId"] = ownerID
		} else {
			input["ownerId"] = nil
		}
	}

	if d.HasChange("slug") {
		if v, ok := d.GetOk("slug"); ok && v.(string) != "" {
			input["slug"] = v.(string)
		} else {
			input["slug"] = nil
		}
	}

	variables := map[string]interface{}{
		"input": input,
	}

	data, err := compassClient.ExecuteQuery(ctx, updateComponentMutation, variables)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update component: %w", err))
	}

	var response UpdateComponentResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	if !response.Compass.UpdateComponent.Success {
		return diag.FromErr(fmt.Errorf("failed to update component: GraphQL mutation returned success=false"))
	}

	// Update successful, read the latest state
	return resourceComponentRead(ctx, d, m)
}

func resourceComponentDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerConfig := m.(*ProviderConfig)
	compassClient := providerConfig.Client
	componentID := d.Id()

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"id": componentID,
		},
	}

	data, err := compassClient.ExecuteQuery(ctx, deleteComponentMutation, variables)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete component: %w", err))
	}

	var response DeleteComponentResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	if !response.Compass.DeleteComponent.Success {
		return diag.FromErr(fmt.Errorf("failed to delete component: GraphQL mutation returned success=false"))
	}

	d.SetId("")
	return nil
}
