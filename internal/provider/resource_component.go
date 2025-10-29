package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	createComponentMutation = `
		mutation CreateComponent($cloudId: ID!, $name: String!, $description: String, $type: CompassComponentType!, $ownerId: ID) {
			compass {
				createComponent(
					cloudId: $cloudId
					input: {
						name: $name
						description: $description
						type: $type
						ownerId: $ownerId
					}
				) {
					success
					componentDetails {
						id
						name
						description
						typeId
						ownerId
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
				Description: "Type of the Compass component. Valid values: SERVICE, LIBRARY, APPLICATION, INFRASTRUCTURE, DATABASE, DOCUMENTATION",
			},
			"owner_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Owner ID (Atlassian account ID) of the Compass component",
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
	componentType := d.Get("type").(string)
	ownerID := d.Get("owner_id").(string)

	// Validate component type - must be valid CompassComponentType enum value
	validTypes := map[string]bool{
		"SERVICE":        true,
		"LIBRARY":        true,
		"APPLICATION":    true,
		"INFRASTRUCTURE": true,
		"DATABASE":       true,
		"DOCUMENTATION":  true,
	}
	if !validTypes[componentType] {
		return diag.Errorf("invalid component type: %s. Valid values are: SERVICE, LIBRARY, APPLICATION, INFRASTRUCTURE, DATABASE, DOCUMENTATION", componentType)
	}

	variables := map[string]interface{}{
		"cloudId": cloudID,
		"name":    name,
		"type":    componentType,
	}

	if description != "" {
		variables["description"] = description
	}

	if ownerID != "" {
		variables["ownerId"] = ownerID
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
	if !d.HasChanges("name", "description", "owner_id") {
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
			// For clearing owner, we might need to pass null explicitly
			// Try passing empty string first, API should handle it
			input["ownerId"] = nil
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
