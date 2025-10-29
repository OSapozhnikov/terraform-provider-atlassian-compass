package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/OSapozhnikov/terraform-provider-atlassian-compass/internal/client"
)

const (
	createComponentMutation = `
		mutation CreateComponent($name: String!, $description: String, $type: String!, $ownerId: String, $customFields: [CustomFieldInput!]) {
			compass {
				createComponent(
					input: {
						name: $name
						description: $description
						type: $type
						ownerId: $ownerId
						customFields: $customFields
					}
				) {
					success
					component {
						id
						name
						description
						type
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
					id
					name
					description
					type
					ownerId
				}
			}
		}
	`

	deleteComponentMutation = `
		mutation DeleteComponent($id: ID!) {
			compass {
				deleteComponent(id: $id) {
					success
				}
			}
		}
	`
)

type Component struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	OwnerID     string                 `json:"ownerId"`
	CustomFields map[string]interface{} `json:"customFields,omitempty"`
}

type CreateComponentResponse struct {
	Compass struct {
		CreateComponent struct {
			Success   bool      `json:"success"`
			Component Component `json:"component"`
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

func resourceComponent() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceComponentCreate,
		ReadContext:   resourceComponentRead,
		UpdateContext: resourceComponentUpdate,
		DeleteContext: resourceComponentDelete,
		Schema: map[string]*schema.Schema{
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
				Description: "Type of the Compass component (e.g., SERVICE, LIBRARY, APPLICATION)",
			},
			"owner_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Owner ID of the Compass component",
			},
			"custom_fields": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Custom fields for the Compass component",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}
}

func resourceComponentCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	compassClient := m.(*client.Client)

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	componentType := d.Get("type").(string)
	ownerID := d.Get("owner_id").(string)

	variables := map[string]interface{}{
		"name":        name,
		"type":        componentType,
	}

	if description != "" {
		variables["description"] = description
	}

	if ownerID != "" {
		variables["ownerId"] = ownerID
	}

	if customFields, ok := d.GetOk("custom_fields"); ok {
		customFieldsMap := customFields.(map[string]interface{})
		if len(customFieldsMap) > 0 {
			customFieldsInput := make([]map[string]interface{}, 0)
			for k, v := range customFieldsMap {
				customFieldsInput = append(customFieldsInput, map[string]interface{}{
					"key":   k,
					"value": v,
				})
			}
			variables["customFields"] = customFieldsInput
		}
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

	component := response.Compass.CreateComponent.Component
	d.SetId(component.ID)

	return resourceComponentRead(ctx, d, m)
}

func resourceComponentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	compassClient := m.(*client.Client)
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

	d.Set("name", component.Name)
	d.Set("description", component.Description)
	d.Set("type", component.Type)
	d.Set("owner_id", component.OwnerID)

	return nil
}

func resourceComponentUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// For now, Compass API doesn't have a direct update mutation in the provided documentation
	// We'll need to delete and recreate, or implement update if available
	// This is a placeholder that reads the current state
	
	if d.HasChangesExcept("custom_fields") {
		return diag.Errorf("updating component properties is not supported. Please delete and recreate the component.")
	}

	return resourceComponentRead(ctx, d, m)
}

func resourceComponentDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	compassClient := m.(*client.Client)
	componentID := d.Id()

	variables := map[string]interface{}{
		"id": componentID,
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

