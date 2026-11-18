package relay

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/alex-jienexa/foundry-marches/internal/foundrysync/pf2e"
	"github.com/alex-jienexa/foundry-marches/internal/foundrysync/pf2e/calculator"
)

const defaultMaxResponseBytes int64 = 16 << 20

var (
	ErrInvalidConfig     = errors.New("invalid Foundry Relay client configuration")
	ErrResponseTooLarge  = errors.New("Foundry Relay response exceeds configured size limit")
	ErrUnexpectedStatus  = errors.New("Foundry Relay returned an unsuccessful HTTP status")
	ErrDocumentMismatch  = errors.New("Foundry Relay returned a different document than requested")
	ErrWorldMismatch     = errors.New("Foundry Relay client is configured for a different world")
)

// Config contains the connection details for one trusted, administrator-configured
// Relay instance. API keys must be loaded from a secret store by the composition root.
type Config struct {
	WorldID          string
	BaseURL          string
	ClientID         string
	APIKey           string
	HTTPClient       *http.Client
	MaxResponseBytes int64
}

// Client is a read-only HTTP adapter for the self-hosted Foundry REST API Relay.
type Client struct {
	worldID          string
	baseURL          *url.URL
	clientID         string
	apiKey           string
	httpClient       *http.Client
	maxResponseBytes int64
}

func NewClient(config Config) (*Client, error) {
	if strings.TrimSpace(config.WorldID) == "" ||
		strings.TrimSpace(config.BaseURL) == "" ||
		strings.TrimSpace(config.ClientID) == "" ||
		strings.TrimSpace(config.APIKey) == "" {
		return nil, ErrInvalidConfig
	}

	baseURL, err := url.Parse(config.BaseURL)
	if err != nil || baseURL.Host == "" || (baseURL.Scheme != "http" && baseURL.Scheme != "https") ||
		baseURL.User != nil || baseURL.RawQuery != "" || baseURL.Fragment != "" {
		return nil, fmt.Errorf("%w: BaseURL must be an absolute HTTP(S) URL without credentials, query or fragment", ErrInvalidConfig)
	}
	baseURL.Path = strings.TrimRight(baseURL.Path, "/")

	httpClient := &http.Client{Timeout: 30 * time.Second}
	if config.HTTPClient != nil {
		clientCopy := *config.HTTPClient
		httpClient = &clientCopy
	}
	// The Relay API key is a custom header, so net/http does not treat it as
	// sensitive on cross-origin redirects. Never forward credentials via redirects.
	httpClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	maxResponseBytes := config.MaxResponseBytes
	if maxResponseBytes <= 0 {
		maxResponseBytes = defaultMaxResponseBytes
	}

	return &Client{
		worldID: config.WorldID, baseURL: baseURL, clientID: config.ClientID, apiKey: config.APIKey,
		httpClient: httpClient, maxResponseBytes: maxResponseBytes,
	}, nil
}

// GetActorSnapshot retrieves and validates a complete Actor document.
// The returned bytes are the Actor JSON itself, not the Relay response envelope.
func (c *Client) GetActorSnapshot(ctx context.Context, worldID, actorUUID string) ([]byte, error) {
	if strings.TrimSpace(worldID) == "" || strings.TrimSpace(actorUUID) == "" {
		return nil, errors.New("world ID and Actor UUID are required")
	}
	if worldID != c.worldID {
		return nil, fmt.Errorf("%w: client world %q, requested world %q", ErrWorldMismatch, c.worldID, worldID)
	}
	payload, err := c.getDocument(ctx, actorUUID)
	if err != nil {
		return nil, err
	}
	decoded, err := pf2e.DecodeRelayResponse(payload)
	if err != nil {
		return nil, fmt.Errorf("decode Actor %q from Relay: %w", actorUUID, err)
	}
	if decoded.Character.Identity.UUID != actorUUID {
		return nil, fmt.Errorf("%w: requested %q, received %q", ErrDocumentMismatch, actorUUID, decoded.Character.Identity.UUID)
	}
	return append([]byte(nil), decoded.RawSnapshot...), nil
}

// ResolveCompendiumItem implements calculator.CompendiumItemResolver.
// It returns the exact Item JSON document for the requested compendium UUID.
func (c *Client) ResolveCompendiumItem(ctx context.Context, uuid string) ([]byte, error) {
	if strings.TrimSpace(uuid) == "" {
		return nil, errors.New("compendium Item UUID is required")
	}
	payload, err := c.getDocument(ctx, uuid)
	if err != nil {
		return nil, err
	}
	decoded, err := pf2e.DecodeItemResponse(payload)
	if err != nil {
		return nil, fmt.Errorf("decode compendium Item %q from Relay: %w", uuid, err)
	}
	return append([]byte(nil), decoded.RawSnapshot...), nil
}

func (c *Client) getDocument(ctx context.Context, uuid string) ([]byte, error) {
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/get"
	query := endpoint.Query()
	query.Set("uuid", uuid)
	query.Set("clientId", c.clientID)
	endpoint.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create Foundry Relay request: %w", err)
	}
	request.Header.Set("x-api-key", c.apiKey)
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request Foundry Relay document %q: %w", uuid, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, c.maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read Foundry Relay response: %w", err)
	}
	if int64(len(body)) > c.maxResponseBytes {
		return nil, ErrResponseTooLarge
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("%w: HTTP %d", ErrUnexpectedStatus, response.StatusCode)
	}
	return body, nil
}

var _ calculator.CompendiumItemResolver = (*Client)(nil)
