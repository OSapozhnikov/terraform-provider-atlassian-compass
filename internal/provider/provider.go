package provider

import (
	"context"
	"fmt"

	"github.com/OSapozhnikov/terraform-provider-atlassian-compass/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	providerName = "compass"
)

// Provider returns a *schema.Provider.
func New() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"email": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("COMPASS_EMAIL", nil),
				Description: "Email address of your Atlassian account. Can also be set via COMPASS_EMAIL environment variable.",
			},
			"api_token": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("COMPASS_API_TOKEN", nil),
				Description: "API token for Atlassian Compass. Get it from https://id.atlassian.com/manage/api-tokens. Can also be set via COMPASS_API_TOKEN environment variable.",
				Sensitive:   true,
			},
			"base_url": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("COMPASS_BASE_URL", "https://api.atlassian.com"),
				Description: "Base URL for Atlassian Compass GraphQL API. Defaults to https://api.atlassian.com (do not include /graphql path, it will be added automatically)",
			},
			"tenant": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("COMPASS_TENANT", nil),
				Description: "Tenant name for automatic cloud_id detection (e.g., 'temabit' for temabit.atlassian.net). Can also be set via COMPASS_TENANT environment variable.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"compass_component":      resourceComponent(),
			"compass_component_link": resourceComponentLink(),
		},
		ConfigureContextFunc: configureProvider,
	}
}

// ProviderConfig holds provider configuration including tenant for cloud_id lookup
type ProviderConfig struct {
	Client *client.Client
	Tenant string
}

func configureProvider(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	email := d.Get("email").(string)
	apiToken := d.Get("api_token").(string)
	baseURL := d.Get("base_url").(string)
	tenant := ""
	if v, ok := d.GetOk("tenant"); ok {
		tenant = v.(string)
	}

	if email == "" {
		return nil, diag.FromErr(fmt.Errorf("email is required"))
	}

	if apiToken == "" {
		return nil, diag.FromErr(fmt.Errorf("api_token is required"))
	}

	compassClient, err := client.NewClient(baseURL, email, apiToken)
	if err != nil {
		return nil, diag.FromErr(fmt.Errorf("failed to create Compass client: %w", err))
	}

	return &ProviderConfig{
		Client: compassClient,
		Tenant: tenant,
	}, nil
}
