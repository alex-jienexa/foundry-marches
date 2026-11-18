package relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

const maxActorSearchResults = 500

var (
	ErrActorListLimitReached = errors.New("Foundry Actor search reached Relay's result limit; refusing an incomplete world sync")
	ErrInvalidActorSearch    = errors.New("invalid Foundry Actor search response")
)

// ActorReference is the small, stable identity returned by Relay's world search.
// UUID is the canonical key used for subsequent GetActorSnapshot calls.
type ActorReference struct {
	UUID         string  `json:"uuid"`
	ID           string  `json:"id,omitempty"`
	Name         string  `json:"name"`
	Image        *string `json:"img,omitempty"`
	DocumentType string  `json:"documentType,omitempty"`
}

// ListActors returns every world Actor visible to the configured Relay client.
// Compendium documents are excluded. Relay currently caps one search response at
// 500 results; reaching that cap is treated as an error rather than silently
// presenting a partial list as a complete world inventory.
func (c *Client) ListActors(ctx context.Context, worldID string) ([]ActorReference, error) {
	if strings.TrimSpace(worldID) == "" {
		return nil, errors.New("world ID is required")
	}
	if worldID != c.worldID {
		return nil, fmt.Errorf("%w: client world %q, requested world %q", ErrWorldMismatch, c.worldID, worldID)
	}

	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/search"
	query := endpoint.Query()
	query.Set("clientId", c.clientID)
	query.Set("filter", "Actor")
	query.Set("excludeCompendiums", "true")
	query.Set("minified", "true")
	query.Set("limit", fmt.Sprint(maxActorSearchResults))
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create Foundry Actor search request: %w", err)
	}
	request.Header.Set("x-api-key", c.apiKey)
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("search Foundry Actors: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, c.maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read Foundry Actor search response: %w", err)
	}
	if int64(len(body)) > c.maxResponseBytes {
		return nil, ErrResponseTooLarge
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("%w: HTTP %d", ErrUnexpectedStatus, response.StatusCode)
	}

	actors, err := decodeActorSearchResponse(body)
	if err != nil {
		return nil, err
	}
	if len(actors) >= maxActorSearchResults {
		return actors, ErrActorListLimitReached
	}
	return actors, nil
}

func decodeActorSearchResponse(payload []byte) ([]ActorReference, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(payload, &root); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidActorSearch, err)
	}

	data := json.RawMessage(payload)
	if nested, ok := root["data"]; ok && len(nested) > 0 {
		data = nested
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(data, &rows); err != nil {
		var resultObject map[string]json.RawMessage
		if objectErr := json.Unmarshal(data, &resultObject); objectErr != nil {
			return nil, fmt.Errorf("%w: expected an array or result object", ErrInvalidActorSearch)
		}
		for _, key := range []string{"results", "entities", "matches"} {
			if candidate := resultObject[key]; len(candidate) > 0 {
				if err := json.Unmarshal(candidate, &rows); err == nil {
					break
				}
			}
		}
	}
	if rows == nil {
		return nil, fmt.Errorf("%w: no Actor result array found", ErrInvalidActorSearch)
	}

	actors := make([]ActorReference, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for index, row := range rows {
		var actor ActorReference
		if err := json.Unmarshal(row, &actor); err != nil {
			return nil, fmt.Errorf("%w: result %d: %v", ErrInvalidActorSearch, index, err)
		}
		if actor.UUID == "" && actor.ID != "" {
			actor.UUID = "Actor." + actor.ID
		}
		if !strings.HasPrefix(actor.UUID, "Actor.") || actor.Name == "" {
			return nil, fmt.Errorf("%w: result %d has no canonical Actor UUID or name", ErrInvalidActorSearch, index)
		}
		if _, exists := seen[actor.UUID]; exists {
			continue
		}
		seen[actor.UUID] = struct{}{}
		actors = append(actors, actor)
	}
	sort.Slice(actors, func(i, j int) bool { return actors[i].UUID < actors[j].UUID })
	return actors, nil
}
