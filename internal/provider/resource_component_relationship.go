package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	createRelationshipMutation = `
		mutation CreateRelationship($input: CreateCompassRelationshipInput!) {
			compass {
				createRelationship(input: $input) {
					success
					errors {
						message
					}
				}
			}
		}
	`

	deleteRelationshipMutation = `
		mutation DeleteRelationship($input: DeleteCompassRelationshipInput!) {
			compass {
				deleteRelationship(input: $input) {
					success
					errors {
						message
					}
				}
			}
		}
	`

	// Query without relationshipType filter so API returns all relationships; we filter client-side.
	// Passing relationshipType in the query can cause empty results depending on API behavior.
	getComponentRelationshipsQuery = `
		query GetComponentRelationships($componentId: ID!, $first: Int) {
			compass {
				component(id: $componentId) {
					... on CompassComponent {
						id
						relationships(query: { first: $first }) {
							... on CompassRelationshipConnection {
								edges {
									node {
										relationshipType
										startNode {
											id
										}
										endNode {
											id
										}
									}
								}
							}
						}
					}
				}
			}
		}
	`
)

type CompassRelationship struct {
	ID               string `json:"id"`
	RelationshipType string `json:"relationshipType"`
	StartNode        struct {
		ID string `json:"id"`
	} `json:"startNode"`
	EndNode struct {
		ID string `json:"id"`
	} `json:"endNode"`
}

type CreateRelationshipResponse struct {
	Compass struct {
		CreateRelationship struct {
			Success                    bool                       `json:"success"`
			Errors                     []struct{ Message string } `json:"errors,omitempty"`
			CreatedCompassRelationship *CompassRelationship       `json:"createdCompassRelationship,omitempty"`
		} `json:"createRelationship"`
	} `json:"compass"`
}

type DeleteRelationshipResponse struct {
	Compass struct {
		DeleteRelationship struct {
			Success bool                       `json:"success"`
			Errors  []struct{ Message string } `json:"errors,omitempty"`
		} `json:"deleteRelationship"`
	} `json:"compass"`
}

type GetComponentRelationshipsResponse struct {
	Compass struct {
		Component struct {
			ID            string `json:"id"`
			Relationships struct {
				Edges []struct {
					Node CompassRelationship `json:"node"`
				} `json:"edges"`
			} `json:"relationships"`
		} `json:"component"`
	} `json:"compass"`
}

// parseRelationshipCompositeID parses composite ID "start_node_id:end_node_id:relationship_type".
// When both IDs are Compass ARIs (ari:cloud:compass:...), the boundary is the second occurrence of ":ari:cloud:compass:".
func parseRelationshipCompositeID(compositeID string) (startNodeID, endNodeID, relationshipType string, err error) {
	if compositeID == "" {
		return "", "", "", fmt.Errorf("empty relationship ID")
	}
	const ariPrefix = ":ari:cloud:compass:"
	if idx := strings.Index(compositeID, ariPrefix); idx >= 0 {
		// Full ARI format: start_ari:ari:cloud:compass:...:relationship_type
		startNodeID = compositeID[:idx]
		rest := compositeID[idx+1:] // "ari:cloud:compass:...:DEPENDS_ON"
		lastColon := strings.LastIndex(rest, ":")
		if lastColon < 0 {
			return "", "", "", fmt.Errorf("invalid relationship ID: no relationship_type after end_node_id")
		}
		endNodeID = rest[:lastColon]
		relationshipType = rest[lastColon+1:]
		return startNodeID, endNodeID, relationshipType, nil
	}
	// Simple IDs (e.g. cmp-1:cmp-2:DEPENDS_ON)
	parts := strings.Split(compositeID, ":")
	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("invalid relationship ID format. Expected start_node_id:end_node_id:relationship_type, got: %s", compositeID)
	}
	relationshipType = parts[len(parts)-1]
	endNodeID = parts[len(parts)-2]
	startNodeID = strings.Join(parts[:len(parts)-2], ":")
	return startNodeID, endNodeID, relationshipType, nil
}

func resourceComponentRelationship() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceComponentRelationshipCreate,
		ReadContext:   resourceComponentRelationshipRead,
		DeleteContext: resourceComponentRelationshipDelete,
		Schema: map[string]*schema.Schema{
			"start_node_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the component at the start of the relationship (e.g. the dependent, or the child)",
			},
			"end_node_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the component at the end of the relationship (e.g. the one we depend on, or the parent)",
			},
			"relationship_type": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Type of relationship. Valid values: DEPENDS_ON, CHILD_OF",
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					relType := val.(string)
					if relType != "DEPENDS_ON" && relType != "CHILD_OF" {
						errs = append(errs, fmt.Errorf("%q must be either DEPENDS_ON or CHILD_OF, got: %s", key, relType))
					}
					return warns, errs
				},
			},
			"cloud_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Cloud ID of the Atlassian site. If not provided, will be automatically detected from tenant configured in provider.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: resourceComponentRelationshipImport,
		},
	}
}

func resourceComponentRelationshipCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerConfig := m.(*ProviderConfig)
	compassClient := providerConfig.Client

	startNodeID := d.Get("start_node_id").(string)
	endNodeID := d.Get("end_node_id").(string)
	relationshipType := d.Get("relationship_type").(string)

	// Validate relationship type
	if relationshipType != "DEPENDS_ON" && relationshipType != "CHILD_OF" {
		return diag.Errorf("relationship_type must be either DEPENDS_ON or CHILD_OF, got: %s", relationshipType)
	}

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

	// Build GraphQL input
	input := map[string]interface{}{
		"startNodeId":      startNodeID,
		"endNodeId":        endNodeID,
		"relationshipType": relationshipType,
	}

	variables := map[string]interface{}{
		"input": input,
	}

	data, err := compassClient.ExecuteQuery(ctx, createRelationshipMutation, variables)
	if err != nil {
		// If relationship already exists, adopt it into state instead of failing
		if strings.Contains(err.Error(), "already exists") {
			compositeID := fmt.Sprintf("%s:%s:%s", startNodeID, endNodeID, relationshipType)
			d.SetId(compositeID)
			_ = d.Set("start_node_id", startNodeID)
			_ = d.Set("end_node_id", endNodeID)
			_ = d.Set("relationship_type", relationshipType)
			return resourceComponentRelationshipRead(ctx, d, m)
		}
		return diag.FromErr(fmt.Errorf("failed to create relationship: %w", err))
	}

	var response CreateRelationshipResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	createResp := response.Compass.CreateRelationship
	if !createResp.Success {
		msg := "createRelationship returned success=false"
		if len(createResp.Errors) > 0 {
			msg = createResp.Errors[0].Message
			// Adopt existing relationship if API says it already exists
			if strings.Contains(strings.ToLower(msg), "already exists") {
				compositeID := fmt.Sprintf("%s:%s:%s", startNodeID, endNodeID, relationshipType)
				d.SetId(compositeID)
				_ = d.Set("start_node_id", startNodeID)
				_ = d.Set("end_node_id", endNodeID)
				_ = d.Set("relationship_type", relationshipType)
				return resourceComponentRelationshipRead(ctx, d, m)
			}
		}
		return diag.FromErr(fmt.Errorf("failed to create relationship: %s", msg))
	}

	// Use composite ID format: start_node_id:end_node_id:relationship_type
	compositeID := fmt.Sprintf("%s:%s:%s", startNodeID, endNodeID, relationshipType)
	d.SetId(compositeID)

	// Set all fields in state
	if err := d.Set("start_node_id", startNodeID); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set start_node_id: %w", err))
	}
	if err := d.Set("end_node_id", endNodeID); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set end_node_id: %w", err))
	}
	if err := d.Set("relationship_type", relationshipType); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set relationship_type: %w", err))
	}

	return resourceComponentRelationshipRead(ctx, d, m)
}

func resourceComponentRelationshipRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerConfig := m.(*ProviderConfig)
	compassClient := providerConfig.Client

	// Parse composite ID or get from state
	compositeID := d.Id()
	var startNodeID, endNodeID, relationshipType string

	if compositeID != "" {
		var parseErr error
		startNodeID, endNodeID, relationshipType, parseErr = parseRelationshipCompositeID(compositeID)
		if parseErr != nil {
			return diag.FromErr(parseErr)
		}
	} else {
		// Fallback to state fields if ID is not set
		startNodeID = d.Get("start_node_id").(string)
		endNodeID = d.Get("end_node_id").(string)
		relationshipType = d.Get("relationship_type").(string)
	}

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

	// Query component relationships for start_node_id (OUTWARD by default).
	variables := map[string]interface{}{
		"componentId": startNodeID,
		"first":       100,
	}

	data, err := compassClient.ExecuteQuery(ctx, getComponentRelationshipsQuery, variables)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read component relationships: %w", err))
	}

	var response GetComponentRelationshipsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	var foundRelationship *CompassRelationship
	for _, edge := range response.Compass.Component.Relationships.Edges {
		rel := edge.Node
		if rel.StartNode.ID == startNodeID &&
			rel.EndNode.ID == endNodeID &&
			rel.RelationshipType == relationshipType {
			foundRelationship = &rel
			break
		}
	}

	if foundRelationship == nil {
		// Do not clear ID when we got no edges (avoids "inconsistent result after apply").
		// Only clear ID when we got edges but none matched (relationship was removed).
		hasEdges := len(response.Compass.Component.Relationships.Edges) > 0
		if hasEdges {
			// We got some edges but none matched — relationship was likely removed
			d.SetId("")
			return nil
		}
		// Zero edges: leave ID and state unchanged, refresh attributes from state
		if err := d.Set("start_node_id", startNodeID); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("end_node_id", endNodeID); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("relationship_type", relationshipType); err != nil {
			return diag.FromErr(err)
		}
		if err := d.Set("cloud_id", cloudID); err != nil {
			return diag.FromErr(err)
		}
		return nil
	}

	// Set all fields in state
	if err := d.Set("start_node_id", startNodeID); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set start_node_id: %w", err))
	}
	if err := d.Set("end_node_id", endNodeID); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set end_node_id: %w", err))
	}
	if err := d.Set("relationship_type", relationshipType); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set relationship_type: %w", err))
	}
	if err := d.Set("cloud_id", cloudID); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set cloud_id: %w", err))
	}

	return nil
}

func resourceComponentRelationshipDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerConfig := m.(*ProviderConfig)
	compassClient := providerConfig.Client

	compositeID := d.Id()
	startNodeID, endNodeID, relationshipType, err := parseRelationshipCompositeID(compositeID)
	if err != nil {
		return diag.FromErr(err)
	}

	// Build GraphQL input for delete
	input := map[string]interface{}{
		"startNodeId":      startNodeID,
		"endNodeId":        endNodeID,
		"relationshipType": relationshipType,
	}

	variables := map[string]interface{}{
		"input": input,
	}

	data, err := compassClient.ExecuteQuery(ctx, deleteRelationshipMutation, variables)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to delete relationship: %w", err))
	}

	var response DeleteRelationshipResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal response: %w", err))
	}

	deleteResp := response.Compass.DeleteRelationship
	if !deleteResp.Success {
		msg := "deleteRelationship returned success=false"
		if len(deleteResp.Errors) > 0 {
			msg = deleteResp.Errors[0].Message
		}
		return diag.FromErr(fmt.Errorf("failed to delete relationship: %s", msg))
	}

	d.SetId("")
	return nil
}

func resourceComponentRelationshipImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()
	startNodeID, endNodeID, relationshipType, err := parseRelationshipCompositeID(id)
	if err != nil {
		return nil, err
	}

	// Validate relationship type
	if relationshipType != "DEPENDS_ON" && relationshipType != "CHILD_OF" {
		return nil, fmt.Errorf("invalid relationship_type: %s. Must be DEPENDS_ON or CHILD_OF", relationshipType)
	}

	// Set the composite ID
	d.SetId(id)

	// Set fields in state
	if err := d.Set("start_node_id", startNodeID); err != nil {
		return nil, fmt.Errorf("failed to set start_node_id: %w", err)
	}
	if err := d.Set("end_node_id", endNodeID); err != nil {
		return nil, fmt.Errorf("failed to set end_node_id: %w", err)
	}
	if err := d.Set("relationship_type", relationshipType); err != nil {
		return nil, fmt.Errorf("failed to set relationship_type: %w", err)
	}

	// Read to populate remaining fields (cloud_id)
	diags := resourceComponentRelationshipRead(ctx, d, m)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to read imported resource: %v", diags)
	}

	return []*schema.ResourceData{d}, nil
}
