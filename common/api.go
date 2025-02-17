package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// APIConfig holds the configuration for the API client
type APIConfig struct {
	APIKey   string
	Endpoint string
}

// API represents the main API client
type API struct {
	apiKey string
	apiURI string
	client *http.Client
}

// NewAPI creates a new API instance
func NewAPI(config APIConfig) *API {
	return &API{
		apiKey: config.APIKey,
		apiURI: config.Endpoint,
		client: &http.Client{},
	}
}

// SetEndpoint updates the API endpoint
func (a *API) SetEndpoint(endpoint string) {
	a.apiURI = endpoint
}

// RequestOptions contains the options for making a request
type RequestOptions struct {
	Timeout time.Duration
	Headers map[string]string
	Params  map[string]string
	JSON    interface{}
}

// Request makes an HTTP request to the API
func (a *API) Request(method, path string, options RequestOptions) (interface{}, error) {
	// Set default timeout if not provided
	if options.Timeout == 0 {
		options.Timeout = 10 * time.Second
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()

	fullURL, err := url.Parse(a.apiURI + path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Add query parameters if any
	if options.Params != nil {
		q := fullURL.Query()
		for key, value := range options.Params {
			q.Add(key, value)
		}
		fullURL.RawQuery = q.Encode()
	}

	// Prepare request body if JSON data is provided
	var body io.Reader
	if options.JSON != nil {
		jsonData, err := json.Marshal(options.JSON)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	// Create the request with the context
	req, err := http.NewRequestWithContext(ctx, method, fullURL.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers; setting Content-Type only if a JSON body is provided can be done conditionally.
	if options.JSON != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	// Set the Authorization header if an API key is provided and not already set
	if a.apiKey != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", a.apiKey)
	}
	for key, value := range options.Headers {
		req.Header.Set(key, value)
	}

	// Execute the request
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// If the response status code is not in the 2xx range, read and report the response body.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error! status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Decode the JSON response into a generic interface
	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}
