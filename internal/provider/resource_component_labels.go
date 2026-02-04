package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// labelSyncedWithJSM is added by Compass when a component is synced with Jira.
// We never remove it via the API and do not store it in state to avoid plan drift.
const labelSyncedWithJSM = "synced-with-jsm"

const (
	getComponentLabelsQuery = `
		query GetComponentLabels($id: ID!) {
			compass {
				component(id: $id) {
					... on CompassComponent {
						id
						labels {
							name
						}
					}
				}
			}
		}
	`

	addComponentLabelsMutation = `
		mutation AddComponentLabels($input: AddCompassComponentLabelsInput!) {
			compass {
				addComponentLabels(input: $input) {
					success
					errors {
						message
					}
				}
			}
		}
	`

	removeComponentLabelsMutation = `
		mutation RemoveComponentLabels($input: RemoveCompassComponentLabelsInput!) {
			compass {
				removeComponentLabels(input: $input) {
					success
					errors {
						message
					}
				}
			}
		}
	`
)

type componentLabel struct {
	Name string `json:"name"`
}

type getComponentLabelsResponse struct {
	Compass struct {
		Component struct {
			ID     string           `json:"id"`
			Labels []componentLabel `json:"labels"`
		} `json:"component"`
	} `json:"compass"`
}

type addComponentLabelsResponse struct {
	Compass struct {
		AddComponentLabels struct {
			Success bool                       `json:"success"`
			Errors  []struct{ Message string } `json:"errors,omitempty"`
		} `json:"addComponentLabels"`
	} `json:"compass"`
}

type removeComponentLabelsResponse struct {
	Compass struct {
		RemoveComponentLabels struct {
			Success bool                       `json:"success"`
			Errors  []struct{ Message string } `json:"errors,omitempty"`
		} `json:"removeComponentLabels"`
	} `json:"compass"`
}

func resourceComponentLabels() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceComponentLabelsCreate,
		ReadContext:   resourceComponentLabelsRead,
		UpdateContext: resourceComponentLabelsUpdate,
		DeleteContext: resourceComponentLabelsDelete,
		Schema: map[string]*schema.Schema{
			"component_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the Compass component to manage labels for",
			},
			"cloud_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Cloud ID of the Atlassian site. If not provided, will be automatically detected from tenant configured in provider.",
			},
			"labels": {
				Type:        schema.TypeList,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "List of label names to set on the component. Managed as a set; order is normalized.",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: resourceComponentLabelsImport,
		},
	}
}

func getCloudID(ctx context.Context, d *schema.ResourceData, providerConfig *ProviderConfig) (string, diag.Diagnostics) {
	if v, ok := d.GetOk("cloud_id"); ok && v.(string) != "" {
		return v.(string), nil
	}
	if providerConfig.Tenant == "" {
		return "", diag.FromErr(fmt.Errorf("cloud_id is required when tenant is not configured in provider"))
	}
	cloudID, err := providerConfig.Client.GetCloudIDByTenant(ctx, providerConfig.Tenant)
	if err != nil {
		return "", diag.FromErr(fmt.Errorf("failed to get cloud_id from tenant %q: %w", providerConfig.Tenant, err))
	}
	return cloudID, nil
}

func labelsFromSchema(d *schema.ResourceData) []string {
	raw := d.Get("labels").([]interface{})
	return labelsFromInterfaceSlice(raw)
}

func labelsFromInterfaceSlice(raw []interface{}) []string {
	out := make([]string, 0, len(raw))
	seen := make(map[string]bool)
	for _, v := range raw {
		s, _ := v.(string)
		if s != "" && !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func setLabelsInState(d *schema.ResourceData, labels []string) error {
	return d.Set("labels", labels)
}

// setDiff returns toAdd (in desired not in current) and toRemove (in current not in desired).
// filterOutProtectedLabels returns a copy of labels without system-managed labels
// that must not be removed by this resource (e.g. synced-with-jsm from Jira sync).
func filterOutProtectedLabels(labels []string) []string {
	out := make([]string, 0, len(labels))
	for _, s := range labels {
		if s != labelSyncedWithJSM {
			out = append(out, s)
		}
	}
	return out
}

func setDiff(current, desired []string) (toAdd, toRemove []string) {
	cur := make(map[string]bool)
	for _, s := range current {
		cur[s] = true
	}
	des := make(map[string]bool)
	for _, s := range desired {
		des[s] = true
	}
	for s := range des {
		if !cur[s] {
			toAdd = append(toAdd, s)
		}
	}
	for s := range cur {
		if !des[s] {
			toRemove = append(toRemove, s)
		}
	}
	sort.Strings(toAdd)
	sort.Strings(toRemove)
	return toAdd, toRemove
}

func resourceComponentLabelsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerConfig := m.(*ProviderConfig)
	client := providerConfig.Client

	componentID := d.Get("component_id").(string)
	cloudID, diags := getCloudID(ctx, d, providerConfig)
	if diags != nil {
		return diags
	}
	if err := d.Set("cloud_id", cloudID); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set cloud_id: %w", err))
	}

	labels := labelsFromSchema(d)
	if len(labels) == 0 {
		return diag.Errorf("at least one label is required")
	}

	input := map[string]interface{}{
		"componentId": componentID,
		"labelNames":  labels,
	}
	data, err := client.ExecuteQuery(ctx, addComponentLabelsMutation, map[string]interface{}{"input": input})
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to add component labels: %w", err))
	}

	var resp addComponentLabelsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal addComponentLabels response: %w", err))
	}
	add := resp.Compass.AddComponentLabels
	if !add.Success {
		msg := "addComponentLabels returned success=false"
		if len(add.Errors) > 0 {
			msg = add.Errors[0].Message
		}
		return diag.FromErr(fmt.Errorf("failed to add component labels: %s", msg))
	}

	d.SetId(componentID)
	return resourceComponentLabelsRead(ctx, d, m)
}

func resourceComponentLabelsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerConfig := m.(*ProviderConfig)
	client := providerConfig.Client

	componentID := d.Id()
	cloudID, diags := getCloudID(ctx, d, providerConfig)
	if diags != nil {
		return diags
	}

	data, err := client.ExecuteQuery(ctx, getComponentLabelsQuery, map[string]interface{}{"id": componentID})
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read component labels: %w", err))
	}

	var resp getComponentLabelsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal getComponentLabels response: %w", err))
	}

	comp := resp.Compass.Component
	if comp.ID == "" {
		d.SetId("")
		return nil
	}

	labelNames := make([]string, 0, len(comp.Labels))
	for _, l := range comp.Labels {
		if l.Name != "" && l.Name != labelSyncedWithJSM {
			labelNames = append(labelNames, l.Name)
		}
	}
	sort.Strings(labelNames)

	if err := d.Set("component_id", componentID); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("cloud_id", cloudID); err != nil {
		return diag.FromErr(err)
	}
	if err := setLabelsInState(d, labelNames); err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceComponentLabelsUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerConfig := m.(*ProviderConfig)
	client := providerConfig.Client

	if !d.HasChange("labels") {
		return resourceComponentLabelsRead(ctx, d, m)
	}

	componentID := d.Id()
	oldRaw, _ := d.GetChange("labels")
	oldList := labelsFromInterfaceSlice(oldRaw.([]interface{}))
	newList := labelsFromSchema(d)

	toAdd, toRemove := setDiff(oldList, newList)
	toRemove = filterOutProtectedLabels(toRemove)

	// Remove first, then add
	if len(toRemove) > 0 {
		input := map[string]interface{}{
			"componentId": componentID,
			"labelNames":  toRemove,
		}
		data, err := client.ExecuteQuery(ctx, removeComponentLabelsMutation, map[string]interface{}{"input": input})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to remove component labels: %w", err))
		}
		var resp removeComponentLabelsResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return diag.FromErr(fmt.Errorf("failed to unmarshal removeComponentLabels response: %w", err))
		}
		if !resp.Compass.RemoveComponentLabels.Success {
			msg := "removeComponentLabels returned success=false"
			if len(resp.Compass.RemoveComponentLabels.Errors) > 0 {
				msg = resp.Compass.RemoveComponentLabels.Errors[0].Message
			}
			return diag.FromErr(fmt.Errorf("failed to remove component labels: %s", msg))
		}
	}
	if len(toAdd) > 0 {
		input := map[string]interface{}{
			"componentId": componentID,
			"labelNames":  toAdd,
		}
		data, err := client.ExecuteQuery(ctx, addComponentLabelsMutation, map[string]interface{}{"input": input})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to add component labels: %w", err))
		}
		var resp addComponentLabelsResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return diag.FromErr(fmt.Errorf("failed to unmarshal addComponentLabels response: %w", err))
		}
		if !resp.Compass.AddComponentLabels.Success {
			msg := "addComponentLabels returned success=false"
			if len(resp.Compass.AddComponentLabels.Errors) > 0 {
				msg = resp.Compass.AddComponentLabels.Errors[0].Message
			}
			return diag.FromErr(fmt.Errorf("failed to add component labels: %s", msg))
		}
	}

	return resourceComponentLabelsRead(ctx, d, m)
}

func resourceComponentLabelsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	providerConfig := m.(*ProviderConfig)
	client := providerConfig.Client

	componentID := d.Id()
	labels := filterOutProtectedLabels(labelsFromSchema(d))
	if len(labels) == 0 {
		d.SetId("")
		return nil
	}

	input := map[string]interface{}{
		"componentId": componentID,
		"labelNames":  labels,
	}
	data, err := client.ExecuteQuery(ctx, removeComponentLabelsMutation, map[string]interface{}{"input": input})
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to remove component labels: %w", err))
	}

	var resp removeComponentLabelsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return diag.FromErr(fmt.Errorf("failed to unmarshal removeComponentLabels response: %w", err))
	}
	if !resp.Compass.RemoveComponentLabels.Success {
		msg := "removeComponentLabels returned success=false"
		if len(resp.Compass.RemoveComponentLabels.Errors) > 0 {
			msg = resp.Compass.RemoveComponentLabels.Errors[0].Message
		}
		return diag.FromErr(fmt.Errorf("failed to remove component labels: %s", msg))
	}

	d.SetId("")
	return nil
}

func resourceComponentLabelsImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()
	componentID := id
	// Allow component_id or component_id:cloud_id (use last separator)
	for i := len(id) - 1; i >= 0; i-- {
		if id[i] == ':' || id[i] == '/' {
			componentID = id[:i]
			break
		}
	}
	d.SetId(componentID)
	if err := d.Set("component_id", componentID); err != nil {
		return nil, err
	}
	diags := resourceComponentLabelsRead(ctx, d, m)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to read imported resource: %v", diags)
	}
	return []*schema.ResourceData{d}, nil
}
