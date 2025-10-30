package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// mockState holds simple in-memory data to emulate GraphQL resources.
type mockState struct {
	mu         sync.Mutex
	cloudID    string
	components map[string]map[string]interface{}
	links      map[string]map[string]interface{}
}

func newMockState() *mockState {
	return &mockState{
		cloudID:    "cloud-123",
		components: map[string]map[string]interface{}{},
		links:      map[string]map[string]interface{}{},
	}
}

// graphQLResponse is the envelope returned by the mock GraphQL endpoint.
type graphQLResponse struct {
	Data   interface{}   `json:"data"`
	Errors []interface{} `json:"errors,omitempty"`
}

// startMockGraphQLServer creates an httptest.Server that understands the specific
// GraphQL queries this provider issues and returns deterministic JSON.
func startMockGraphQLServer(state *mockState) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only POST /graphql is supported
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/graphql") {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		var req struct {
			Query     string                 `json:"query"`
			Variables map[string]interface{} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		q := req.Query

		// Tenant to cloudId lookup
		if strings.Contains(q, "tenantContexts") {
			// Always return one context with the configured cloudID
			writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
				"tenantContexts": []map[string]string{{"cloudId": state.cloudID}},
			}})
			return
		}

		// Create component
		if strings.Contains(q, "createComponent(") {
			vars := req.Variables
			name, _ := vars["name"].(string)
			description, _ := vars["description"].(string)
			ownerId, _ := vars["ownerId"].(string)
			// Use a deterministic ID for simplicity
			id := "cmp-1"
			state.mu.Lock()
			state.components[id] = map[string]interface{}{
				"id":          id,
				"name":        name,
				"description": description,
				// API returns typeId in read; we store the provided type into TypeID for later read mapping behavior
				"typeId":  "type-service",
				"ownerId": ownerId,
			}
			state.mu.Unlock()

			writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
				"compass": map[string]interface{}{
					"createComponent": map[string]interface{}{
						"success":          true,
						"componentDetails": state.components[id],
					},
				},
			}})
			return
		}

		// Read component by id (only when links are not requested)
		if strings.Contains(q, "query GetComponent(") && strings.Contains(q, "component(id:") && !strings.Contains(q, "links {") {
			id := ""
			if v, ok := req.Variables["id"].(string); ok {
				id = v
			}
			state.mu.Lock()
			comp := state.components[id]
			state.mu.Unlock()
			if comp == nil {
				// Return empty object to simulate not found
				writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
					"compass": map[string]interface{}{
						"component": map[string]interface{}{},
					},
				}})
				return
			}
			writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
				"compass": map[string]interface{}{
					"component": comp,
				},
			}})
			return
		}

		// Update component
		if strings.Contains(q, "updateComponent(") {
			// variables: { input: { id, name?, description?, ownerId? } }
			input, _ := req.Variables["input"].(map[string]interface{})
			id, _ := input["id"].(string)
			state.mu.Lock()
			comp := state.components[id]
			if comp != nil {
				if v, ok := input["name"].(string); ok {
					comp["name"] = v
				}
				if v, ok := input["description"].(string); ok {
					comp["description"] = v
				}
				if _, exists := input["ownerId"]; exists {
					// may be string or nil
					if v, ok := input["ownerId"].(string); ok {
						comp["ownerId"] = v
					} else {
						comp["ownerId"] = ""
					}
				}
				state.components[id] = comp
			}
			state.mu.Unlock()
			writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
				"compass": map[string]interface{}{
					"updateComponent": map[string]interface{}{
						"success":          true,
						"componentDetails": comp,
					},
				},
			}})
			return
		}

		// Delete component
		if strings.Contains(q, "deleteComponent(") {
			input, _ := req.Variables["input"].(map[string]interface{})
			id, _ := input["id"].(string)
			state.mu.Lock()
			delete(state.components, id)
			// Also delete links bound to this component
			for k, v := range state.links {
				if v["componentId"] == id {
					delete(state.links, k)
				}
			}
			state.mu.Unlock()
			writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
				"compass": map[string]interface{}{
					"deleteComponent": map[string]interface{}{"success": true},
				},
			}})
			return
		}

		// Component with links query (used by link create/read)
		if strings.Contains(q, "component(id:") && strings.Contains(q, "links {") {
			componentId := ""
			if v, ok := req.Variables["componentId"].(string); ok {
				componentId = v
			}
			// Collect links for this component
			state.mu.Lock()
			var links []map[string]interface{}
			for _, l := range state.links {
				if l["componentId"] == componentId {
					// Return only GraphQL fields
					links = append(links, map[string]interface{}{
						"id":       l["id"],
						"name":     l["name"],
						"type":     l["type"],
						"url":      l["url"],
						"objectId": l["objectId"],
					})
				}
			}
			state.mu.Unlock()
			writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
				"compass": map[string]interface{}{
					"component": map[string]interface{}{
						"links": links,
					},
				},
			}})
			return
		}

		// Create link
		if strings.Contains(q, "createComponentLink(") {
			input, _ := req.Variables["input"].(map[string]interface{})
			componentId, _ := input["componentId"].(string)
			link, _ := input["link"].(map[string]interface{})
			name, _ := link["name"].(string)
			linkType, _ := link["type"].(string)
			url, _ := link["url"].(string)
			objectId, _ := link["objectId"].(string)
			id := "lnk-1"
			state.mu.Lock()
			state.links[id] = map[string]interface{}{
				"id":          id,
				"componentId": componentId,
				"name":        name,
				"type":        linkType,
				"url":         url,
				"objectId":    objectId,
			}
			state.mu.Unlock()
			writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
				"compass": map[string]interface{}{
					"createComponentLink": map[string]interface{}{"success": true},
				},
			}})
			return
		}

		// Update link
		if strings.Contains(q, "updateComponentLink(") {
			input, _ := req.Variables["input"].(map[string]interface{})
			componentId, _ := input["componentId"].(string)
			link, _ := input["link"].(map[string]interface{})
			id, _ := link["id"].(string)
			state.mu.Lock()
			if l := state.links[id]; l != nil && l["componentId"] == componentId {
				if v, ok := link["name"].(string); ok {
					l["name"] = v
				}
				if v, ok := link["type"].(string); ok {
					l["type"] = v
				}
				if v, ok := link["url"].(string); ok {
					l["url"] = v
				}
				if _, exists := link["objectId"]; exists {
					if v, ok := link["objectId"].(string); ok {
						l["objectId"] = v
					} else {
						l["objectId"] = ""
					}
				}
				state.links[id] = l
			}
			state.mu.Unlock()
			writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
				"compass": map[string]interface{}{
					"updateComponentLink": map[string]interface{}{"success": true},
				},
			}})
			return
		}

		// Delete link
		if strings.Contains(q, "deleteComponentLink(") {
			input, _ := req.Variables["input"].(map[string]interface{})
			componentId, _ := input["componentId"].(string)
			linkID, _ := input["link"].(string)
			state.mu.Lock()
			if l := state.links[linkID]; l != nil && l["componentId"] == componentId {
				delete(state.links, linkID)
			}
			state.mu.Unlock()
			writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
				"compass": map[string]interface{}{
					"deleteComponentLink": map[string]interface{}{"success": true},
				},
			}})
			return
		}

		// Fallback: unsupported query
		writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{}})
	})

	return httptest.NewServer(handler)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
