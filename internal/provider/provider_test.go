package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// mockState holds simple in-memory data to emulate GraphQL resources.
type mockState struct {
	mu              sync.Mutex
	cloudID         string
	components      map[string]map[string]interface{}
	links           map[string]map[string]interface{}
	componentLabels map[string][]string               // componentId -> label names
	relationships   map[string]map[string]interface{} // relationshipId -> { id, startNodeId, endNodeId, type }
	nextRelID       int
}

func newMockState() *mockState {
	return &mockState{
		cloudID:         "cloud-123",
		components:      map[string]map[string]interface{}{},
		links:           map[string]map[string]interface{}{},
		componentLabels: map[string][]string{},
		relationships:   map[string]map[string]interface{}{},
		nextRelID:       1,
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

		// Component types lookup
		if strings.Contains(q, "componentTypes(") {
			// Return a minimal but valid componentTypes payload (id matches real API: SERVICE, etc.).
			writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
				"compass": map[string]interface{}{
					"componentTypes": map[string]interface{}{
						"__typename": "CompassComponentTypeConnection",
						"nodes": []map[string]interface{}{
							{"id": "SERVICE", "name": "Service"},
						},
					},
				},
			}})
			return
		}

		// componentByReference (data source compass_component by slug)
		if strings.Contains(q, "componentByReference(") {
			reference, _ := req.Variables["reference"].(map[string]interface{})
			slugObj, _ := reference["slug"].(map[string]interface{})
			slug, _ := slugObj["slug"].(string)
			cloudID, _ := slugObj["cloudId"].(string)
			if slug == "" || cloudID == "" {
				writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
					"compass": map[string]interface{}{
						"componentByReference": map[string]interface{}{
							"__typename": "QueryError",
							"message":    "slug and cloudId required",
						},
					},
				}})
				return
			}
			comp := map[string]interface{}{
				"__typename":  "CompassComponent",
				"id":          "ari:cloud:compass:" + cloudID + ":component/uuid/euid",
				"name":        "product-example",
				"slug":        slug,
				"description": nil,
				"url":         "https://example.atlassian.net/compass/component/euid",
				"typeId":      "SERVICE",
				"ownerId":     "",
			}
			writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
				"compass": map[string]interface{}{
					"componentByReference": comp,
				},
			}})
			return
		}

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
			typeId, _ := vars["typeId"].(string)
			if typeId == "" {
				typeId = "SERVICE"
			}
			slug, _ := vars["slug"].(string)
			id := "cmp-1"
			state.mu.Lock()
			state.components[id] = map[string]interface{}{
				"id":          id,
				"name":        name,
				"description": description,
				"typeId":      typeId,
				"ownerId":     ownerId,
				"slug":        slug,
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

		// Get component with labels (used by compass_component_labels)
		if strings.Contains(q, "GetComponentLabels") && strings.Contains(q, "labels {") {
			id := ""
			if v, ok := req.Variables["id"].(string); ok {
				id = v
			}
			state.mu.Lock()
			labels := state.componentLabels[id]
			if labels == nil {
				labels = []string{}
			}
			labelMaps := make([]map[string]string, 0, len(labels))
			for _, n := range labels {
				labelMaps = append(labelMaps, map[string]string{"name": n})
			}
			state.mu.Unlock()
			writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
				"compass": map[string]interface{}{
					"component": map[string]interface{}{
						"id":     id,
						"labels": labelMaps,
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
					if v, ok := input["ownerId"].(string); ok {
						comp["ownerId"] = v
					} else {
						comp["ownerId"] = ""
					}
				}
				if _, exists := input["slug"]; exists {
					if v, ok := input["slug"].(string); ok {
						comp["slug"] = v
					} else {
						comp["slug"] = ""
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

		// Add component labels
		if strings.Contains(q, "addComponentLabels(") {
			input, _ := req.Variables["input"].(map[string]interface{})
			componentId, _ := input["componentId"].(string)
			labelNamesRaw, _ := input["labelNames"].([]interface{})
			state.mu.Lock()
			cur := state.componentLabels[componentId]
			seen := make(map[string]bool)
			for _, s := range cur {
				seen[s] = true
			}
			for _, v := range labelNamesRaw {
				if s, ok := v.(string); ok && s != "" && !seen[s] {
					seen[s] = true
					cur = append(cur, s)
				}
			}
			state.componentLabels[componentId] = cur
			state.mu.Unlock()
			writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
				"compass": map[string]interface{}{
					"addComponentLabels": map[string]interface{}{"success": true},
				},
			}})
			return
		}

		// Remove component labels
		if strings.Contains(q, "removeComponentLabels(") {
			input, _ := req.Variables["input"].(map[string]interface{})
			componentId, _ := input["componentId"].(string)
			labelNamesRaw, _ := input["labelNames"].([]interface{})
			toRemove := make(map[string]bool)
			for _, v := range labelNamesRaw {
				if s, ok := v.(string); ok {
					toRemove[s] = true
				}
			}
			state.mu.Lock()
			cur := state.componentLabels[componentId]
			var newCur []string
			for _, s := range cur {
				if !toRemove[s] {
					newCur = append(newCur, s)
				}
			}
			state.componentLabels[componentId] = newCur
			state.mu.Unlock()
			writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
				"compass": map[string]interface{}{
					"removeComponentLabels": map[string]interface{}{"success": true},
				},
			}})
			return
		}

		// Create relationship
		if strings.Contains(q, "createRelationship(") {
			input, _ := req.Variables["input"].(map[string]interface{})
			startNodeId, _ := input["startNodeId"].(string)
			endNodeId, _ := input["endNodeId"].(string)
			relType := ""
			if s, ok := input["relationshipType"].(string); ok {
				relType = s
			} else if rt, ok := input["relationshipType"].(map[string]interface{}); ok {
				relType, _ = rt["type"].(string)
			}
			state.mu.Lock()
			relID := fmt.Sprintf("rel-%d", state.nextRelID)
			state.nextRelID++
			state.relationships[relID] = map[string]interface{}{
				"id":          relID,
				"startNodeId": startNodeId,
				"endNodeId":   endNodeId,
				"type":        relType,
			}
			state.mu.Unlock()
			writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
				"compass": map[string]interface{}{
					"createRelationship": map[string]interface{}{
						"success": true,
						"createdCompassRelationship": map[string]interface{}{
							"id":               relID,
							"startNodeId":      startNodeId,
							"endNodeId":        endNodeId,
							"relationshipType": relType,
						},
					},
				},
			}})
			return
		}

		// Get component relationships (GetComponentRelationships) — CompassRelationshipConnectionResult with edges
		if strings.Contains(q, "GetComponentRelationships") && strings.Contains(q, "relationships") {
			componentId, _ := req.Variables["componentId"].(string)
			state.mu.Lock()
			var edges []map[string]interface{}
			for _, rel := range state.relationships {
				if rel["startNodeId"] == componentId {
					edges = append(edges, map[string]interface{}{
						"node": map[string]interface{}{
							"id":               rel["id"],
							"relationshipType": rel["type"],
							"startNode":        map[string]interface{}{"id": rel["startNodeId"]},
							"endNode":          map[string]interface{}{"id": rel["endNodeId"]},
						},
					})
				}
			}
			state.mu.Unlock()
			writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
				"compass": map[string]interface{}{
					"component": map[string]interface{}{
						"id": componentId,
						"relationships": map[string]interface{}{
							"edges": edges,
						},
					},
				},
			}})
			return
		}

		// Delete relationship
		if strings.Contains(q, "deleteRelationship(") {
			input, _ := req.Variables["input"].(map[string]interface{})
			startNodeId, _ := input["startNodeId"].(string)
			endNodeId, _ := input["endNodeId"].(string)
			relType := ""
			if s, ok := input["relationshipType"].(string); ok {
				relType = s
			} else if rt, ok := input["relationshipType"].(map[string]interface{}); ok {
				relType, _ = rt["type"].(string)
			}
			state.mu.Lock()
			// Find and delete relationship by startNodeId, endNodeId, and type
			for relID, rel := range state.relationships {
				if rel["startNodeId"] == startNodeId && rel["endNodeId"] == endNodeId && rel["type"] == relType {
					delete(state.relationships, relID)
					break
				}
			}
			state.mu.Unlock()
			writeJSON(w, http.StatusOK, graphQLResponse{Data: map[string]interface{}{
				"compass": map[string]interface{}{
					"deleteRelationship": map[string]interface{}{"success": true},
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
