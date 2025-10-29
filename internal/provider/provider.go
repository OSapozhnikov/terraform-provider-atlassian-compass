package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/OSapozhnikov/terraform-provider-atlassian-compass/internal/client"
)

const (
	providerName = "compass"
)

// Provider returns a *schema.Provider.
func New() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"api_token": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("COMPASS_API_TOKEN", nil),
				Description: "API token for Atlassian Compass. Can also be set via COMPASS_API_TOKEN environment variable.",
				Sensitive:   true,
			},
			"base_url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("COMPASS_BASE_URL", "https://api.atlassian.com"),
				Description: "Base URL for Atlassian Compass API. Defaults to https://api.atlassian.com",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"compass_component": resourceComponent(),
		},
		ConfigureContextFunc: configureProvider,
	}
}

func configureProvider(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	apiToken := d.Get("api_token").(string)
	baseURL := d.Get("base_url").(string)

	if apiToken == "" {
		return nil, diag.FromErr(fmt.Errorf("api_token is required"))
	}

	compassClient, err := client.NewClient(baseURL, apiToken)
	if err != nil {
		return nil, diag.FromErr(fmt.Errorf("failed to create Compass client: %w", err))
	}

	return compassClient, nil
}

