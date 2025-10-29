package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	createComponentLinkMutation = `
		mutation CreateComponentLink($input: CreateCompassComponentLinkInput!) {
			compass {
				createComponentLink(input: $input) {
					success
				}
			}
		}
	`

	getComponentLinkQuery = `
		query GetComponentLink($cloudId: ID!, $componentId: ID!, $linkId: ID!) {
			compass {
				component(id: $componentId) {
					... on CompassComponent {
						links {
							id
							name
							type
							url
							objectId
						}
					}
				}
			}
		}
	`

	updateComponentLinkMutation = `
		mutation UpdateComponentLink($input: UpdateCompassComponentLinkInput!) {
			compass {
				updateComponentLink(input: $input) {
					success
				}
			}
		}
	`

	deleteComponentLinkMutation = `
		mutation DeleteComponentLink($input: DeleteCompassComponentLinkInput!) {
			compass {
				deleteComponentLink(input: $input) {
					success
				}
			}
		}
	`
)

type ComponentLink struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	ObjectID string `json:"objectId,omitempty"`
}

type CreateComponentLinkResponse struct {
	Compass struct {
		CreateComponentLink struct {
			Success bool `json:"success"`
		} `json:"createComponentLink"`
	} `json:"compass"`
}

type GetComponentResponseWithLinks struct {
	Compass struct {
		Component struct {
			Links []ComponentLink `json:"links"`
		} `json:"component"`
	} `json:"compass"`
}

type UpdateComponentLinkResponse struct {
	Compass struct {
		UpdateComponentLink struct {
			Success bool `json:"success"`
		} `json:"updateComponentLink"`
	} `json:"compass"`
}

type DeleteComponentLinkResponse struct {
	Compass struct {
		DeleteComponentLink struct {
			Success bool `json:"success"`
		} `json:"deleteComponentLink"`
	} `json:"compass"`
}

func resourceComponentLink() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceComponentLinkCreate,
		ReadContext:   resourceComponentLinkRead,
		UpdateContext: resourceComponentLinkUpdate,
		DeleteContext: resourceComponentLinkDelete,
		Schema: map[string]*schema.Schema{
			"component_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the Compass component to attach the link to",
			},
			"cloud_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Cloud ID of the Atlassian site. If not provided, will be automatically detected from tenant configured in provider.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the link",
			},
			"type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Type of the link. Valid values: DOCUMENT, CHAT_CHANNEL, REPOSITORY, PROJECT, DASHBOARD, ON_CALL, OTHER_LINK",
			},
			"url": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "URL of the link",
			},
			"object_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The unique ID of the object the link points to (generally configured by integrations)",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: resourceComponentLinkImport,
		},
	}
}

func resourceComponentLinkCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerConfig := m.(*ProviderConfig)
	compassClient := providerConfig.Client

	componentID := d.Get("component_id").(string)

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
	linkType := d.Get("type").(string)
	url := d.Get("url").(string)
	objectID := d.Get("object_id").(string)

	// Validate link type
	validTypes := map[string]bool{
		"DOCUMENT":     true,
		"CHAT_CHANNEL": true,
		"REPOSITORY":   true,
		"PROJECT":      true,
		"DASHBOARD":    true,
		"ON_CALL":      true,
		"OTHER_LINK":   true,
	}
	if !validTypes[linkType] {
		return diag.Errorf("invalid link type: %s. Valid values are: DOCUMENT, CHAT_CHANNEL, REPOSITORY, PROJECT, DASHBOARD, ON_CALL, OTHER_LINK", linkType)
	}

	// Build input according to CreateCompassComponentLinkInput structure:
	// - componentId: ID!
	// - link: CreateCompassLinkInput! (contains name, type, url, objectId)
	linkInput := map[string]interface{}{
		"name": name,
		"type": linkType,
		"url":  url,
	}

	if objectID != "" {
		linkInput["objectId"] = objectID
	}

	// Build full input: CreateCompassComponentLinkInput
	// Note: cloudId might need to be included in the input, or componentId might contain it
	// Based on error, cloudId is not a separate parameter to the mutation
	createInput := map[string]interface{}{
		"componentId": componentID,
		"link":        linkInput,
	}

	variables := map[string]interface{}{
		"input": createInput,
	}

	data, err := compassClient.ExecuteQuery(ctx, createComponentLinkMutation, variables)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create component link: %w", err))
	}

	var response CreateComponentLinkResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	if !response.Compass.CreateComponentLink.Success {
		return diag.FromErr(fmt.Errorf("failed to create component link: GraphQL mutation returned success=false"))
	}

	// The mutation doesn't return the link ID, so we need to read it from the component
	// We'll use a temporary ID and then read to get the actual ID
	// Alternatively, we can query the component links to find the newly created link
	// by matching name, type, and url

	// Query component links to find the created link
	getComponentQueryTemp := `
		query GetComponent($componentId: ID!) {
			compass {
				component(id: $componentId) {
					... on CompassComponent {
						id
						links {
							id
							name
							type
							url
							objectId
						}
					}
				}
			}
		}
	`

	variablesRead := map[string]interface{}{
		"componentId": componentID,
	}

	dataRead, err := compassClient.ExecuteQuery(ctx, getComponentQueryTemp, variablesRead)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read component links after creation: %w", err))
	}

	var responseRead GetComponentResponseWithLinks
	if err := json.Unmarshal(dataRead, &responseRead); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal component links response: %w", err))
	}

	// Find the link by matching name, type, and url (since we don't have ID yet)
	var foundLink *ComponentLink
	for i := range responseRead.Compass.Component.Links {
		link := responseRead.Compass.Component.Links[i]
		if link.Name == name && link.Type == linkType && link.URL == url {
			// Also check objectId if provided
			if objectID == "" && link.ObjectID == "" {
				foundLink = &link
				break
			} else if objectID != "" && link.ObjectID == objectID {
				foundLink = &link
				break
			}
		}
	}

	if foundLink == nil {
		return diag.Errorf("failed to find created link in component. Created link may not be visible yet.")
	}

	d.SetId(foundLink.ID)

	return resourceComponentLinkRead(ctx, d, m)
}

func resourceComponentLinkRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerConfig := m.(*ProviderConfig)
	compassClient := providerConfig.Client

	linkID := d.Id()
	componentID := d.Get("component_id").(string)

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
	}

	// Query component to get all links
	// Note: We need to find the specific link by ID from the component's links
	getComponentQuery := `
		query GetComponent($componentId: ID!) {
			compass {
				component(id: $componentId) {
					... on CompassComponent {
						id
						links {
							id
							name
							type
							url
							objectId
						}
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"componentId": componentID,
	}

	data, err := compassClient.ExecuteQuery(ctx, getComponentQuery, variables)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read component link: %w", err))
	}

	var response GetComponentResponseWithLinks
	if err := json.Unmarshal(data, &response); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	// Find the specific link by ID
	var foundLink *ComponentLink
	for _, link := range response.Compass.Component.Links {
		if link.ID == linkID {
			foundLink = &link
			break
		}
	}

	if foundLink == nil {
		// Link not found, mark as deleted
		d.SetId("")
		return nil
	}

	// Set fields
	d.Set("component_id", componentID)
	d.Set("cloud_id", cloudID)
	d.Set("name", foundLink.Name)
	d.Set("type", foundLink.Type)
	d.Set("url", foundLink.URL)
	if foundLink.ObjectID != "" {
		d.Set("object_id", foundLink.ObjectID)
	}

	return nil
}

func resourceComponentLinkUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerConfig := m.(*ProviderConfig)
	compassClient := providerConfig.Client

	linkID := d.Id()
	componentID := d.Get("component_id").(string)

	// Check if any updatable fields have changed
	if !d.HasChanges("name", "type", "url", "object_id") {
		// No changes to updatable fields, just read the state
		return resourceComponentLinkRead(ctx, d, m)
	}

	// Build update input according to UpdateCompassComponentLinkInput structure:
	// - componentId: ID!
	// - link: UpdateCompassLinkInput! (contains id (required), name, type, url, objectId - all optional)
	// Only include fields that have changed
	linkInput := map[string]interface{}{
		"id": linkID,
	}

	// Only add fields that have actually changed
	if d.HasChange("name") {
		linkInput["name"] = d.Get("name").(string)
	}

	if d.HasChange("type") {
		linkType := d.Get("type").(string)
		// Validate link type
		validTypes := map[string]bool{
			"DOCUMENT":     true,
			"CHAT_CHANNEL": true,
			"REPOSITORY":   true,
			"PROJECT":      true,
			"DASHBOARD":    true,
			"ON_CALL":      true,
			"OTHER_LINK":   true,
		}
		if !validTypes[linkType] {
			return diag.Errorf("invalid link type: %s. Valid values are: DOCUMENT, CHAT_CHANNEL, REPOSITORY, PROJECT, DASHBOARD, ON_CALL, OTHER_LINK", linkType)
		}
		linkInput["type"] = linkType
	}

	if d.HasChange("url") {
		linkInput["url"] = d.Get("url").(string)
	}

	if d.HasChange("object_id") {
		objectID := d.Get("object_id").(string)
		if objectID != "" {
			linkInput["objectId"] = objectID
		} else {
			// For clearing objectId, we might need to pass null explicitly
			linkInput["objectId"] = nil
		}
	}

	// Build full input: UpdateCompassComponentLinkInput
	updateInput := map[string]interface{}{
		"componentId": componentID,
		"link":        linkInput,
	}

	variables := map[string]interface{}{
		"input": updateInput,
	}

	data, err := compassClient.ExecuteQuery(ctx, updateComponentLinkMutation, variables)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to update component link: %w", err))
	}

	var response UpdateComponentLinkResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	if !response.Compass.UpdateComponentLink.Success {
		return diag.FromErr(fmt.Errorf("failed to update component link: GraphQL mutation returned success=false"))
	}

	// Update successful, read the latest state
	return resourceComponentLinkRead(ctx, d, m)
}

func resourceComponentLinkDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerConfig := m.(*ProviderConfig)
	compassClient := providerConfig.Client

	linkID := d.Id()
	componentID := d.Get("component_id").(string)

	// Build delete input according to DeleteCompassComponentLinkInput structure:
	// - componentId: ID!
	// - link: ID!
	deleteInput := map[string]interface{}{
		"componentId": componentID,
		"link":        linkID,
	}

	variables := map[string]interface{}{
		"input": deleteInput,
	}

	data, err := compassClient.ExecuteQuery(ctx, deleteComponentLinkMutation, variables)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete component link: %w", err))
	}

	var response DeleteComponentLinkResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	if !response.Compass.DeleteComponentLink.Success {
		return diag.FromErr(fmt.Errorf("failed to delete component link: GraphQL mutation returned success=false"))
	}

	d.SetId("")
	return nil
}

func resourceComponentLinkImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	// Import format: component_id/link_id or component_id:cloud_id/link_id
	// For simplicity, we'll use component_id:link_id format
	id := d.Id()

	// Try to parse as component_id:link_id
	parts := []string{}
	if idx := len(id); idx > 0 {
		// Look for last colon or slash as separator
		for i := len(id) - 1; i >= 0; i-- {
			if id[i] == ':' || id[i] == '/' {
				parts = []string{id[:i], id[i+1:]}
				break
			}
		}
	}

	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid import format. Expected component_id:link_id or component_id/link_id, got: %s", id)
	}

	d.SetId(parts[1])               // link_id
	d.Set("component_id", parts[0]) // component_id

	// Read will auto-detect cloud_id
	diags := resourceComponentLinkRead(ctx, d, m)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to read imported resource: %v", diags)
	}

	return []*schema.ResourceData{d}, nil
}
