package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultTimeout = 30 * time.Second
	// GraphQL endpoint path - this provider uses ONLY GraphQL API, not REST
	graphQLPath = "/graphql"
)

type Client struct {
	baseURL    string
	email      string
	apiToken   string
	httpClient *http.Client
}

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type GraphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []GraphQLError  `json:"errors,omitempty"`
}

type GraphQLError struct {
	Message    string                 `json:"message"`
	Locations  []Location             `json:"locations,omitempty"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

type Location struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

func NewClient(baseURL, email, apiToken string) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL cannot be empty")
	}
	if email == "" {
		return nil, fmt.Errorf("email cannot be empty")
	}
	if apiToken == "" {
		return nil, fmt.Errorf("apiToken cannot be empty")
	}

	return &Client{
		baseURL:  baseURL,
		email:    email,
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}, nil
}

// ExecuteQuery executes a GraphQL query or mutation against Atlassian Compass GraphQL API.
// This provider uses ONLY GraphQL API, no REST endpoints are used.
func (c *Client) ExecuteQuery(ctx context.Context, query string, variables map[string]interface{}) (json.RawMessage, error) {
	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// POST request to GraphQL endpoint - always uses /graphql path
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+graphQLPath, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Atlassian Compass GraphQL API requires Basic Authentication
	// Format: email:api_token encoded in Base64
	authString := fmt.Sprintf("%s:%s", c.email, c.apiToken)
	authEncoded := base64.StdEncoding.EncodeToString([]byte(authString))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", authEncoded))
	req.Header.Set("X-ExperimentalApi", "compass-beta")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graphQL request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var graphQLResp GraphQLResponse
	if err := json.Unmarshal(body, &graphQLResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(graphQLResp.Errors) > 0 {
		var errMessages []string
		for _, err := range graphQLResp.Errors {
			errMessages = append(errMessages, err.Message)
		}
		return nil, fmt.Errorf("GraphQL errors: %v", errMessages)
	}

	return graphQLResp.Data, nil
}

// GetCloudIDByTenant retrieves cloud_id for a given tenant using GraphQL query.
// Tenant can be provided as just the name (e.g., "temabit") or full domain (e.g., "temabit.atlassian.net")
func (c *Client) GetCloudIDByTenant(ctx context.Context, tenant string) (string, error) {
	// Normalize tenant name - add .atlassian.net if not present
	hostName := tenant
	if hostName != "" && !strings.Contains(hostName, ".") {
		hostName = hostName + ".atlassian.net"
	}

	query := `
		query GetCloudId($hostNames: [String!]!) {
			tenantContexts(hostNames: $hostNames) {
				cloudId
			}
		}
	`

	variables := map[string]interface{}{
		"hostNames": []string{hostName},
	}

	data, err := c.ExecuteQuery(ctx, query, variables)
	if err != nil {
		return "", fmt.Errorf("failed to get cloud ID: %w", err)
	}

	type TenantContext struct {
		CloudID string `json:"cloudId"`
	}

	type TenantContextsResponse struct {
		TenantContexts []TenantContext `json:"tenantContexts"`
	}

	var response TenantContextsResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal tenant contexts response: %w", err)
	}

	if len(response.TenantContexts) == 0 {
		return "", fmt.Errorf("tenant '%s' not found or no access", hostName)
	}

	if response.TenantContexts[0].CloudID == "" {
		return "", fmt.Errorf("cloud ID not found for tenant '%s'", hostName)
	}

	return response.TenantContexts[0].CloudID, nil
}
