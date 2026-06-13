// Package googlehealth provides OAuth2 connection management and a typed
// client for the Google Health API (v4), used to read a user's weight and
// height history.
package googlehealth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// baseURL is the Google Health API v4 endpoint.
const baseURL = "https://health.googleapis.com/v4"

// maxPageSize is the largest page size accepted by dataPoints.list.
const maxPageSize = 10000

// Identity describes the authenticated Google Health user.
type Identity struct {
	Name         string `json:"name"`
	HealthUserID string `json:"healthUserId"`
}

// ObservationSampleTime is the time at which a sample data point (weight or
// height) was recorded.
type ObservationSampleTime struct {
	PhysicalTime string `json:"physicalTime"`
	UtcOffset    string `json:"utcOffset,omitempty"`
}

// Weight is a body weight measurement.
type Weight struct {
	WeightGrams float64               `json:"weightGrams"`
	SampleTime  ObservationSampleTime `json:"sampleTime"`
	Notes       string                `json:"notes,omitempty"`
}

// Height is a body height measurement.
type Height struct {
	HeightMillimeters string                `json:"heightMillimeters"`
	SampleTime        ObservationSampleTime `json:"sampleTime"`
}

// DataPoint is a single recorded health metric. Only the fields relevant to
// weight and height are modeled; other data types are left unparsed.
type DataPoint struct {
	Name   string  `json:"name,omitempty"`
	Weight *Weight `json:"weight,omitempty"`
	Height *Height `json:"height,omitempty"`
}

// ListDataPointsResponse is the response from dataPoints.list.
type ListDataPointsResponse struct {
	DataPoints    []DataPoint `json:"dataPoints"`
	NextPageToken string      `json:"nextPageToken,omitempty"`
}

// Client is a typed wrapper around the parts of the Google Health API (v4)
// needed to read a user's weight and height history.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient returns a Client that authenticates requests using httpClient,
// typically the result of an oauth2.Config's TokenSource-backed client.
func NewClient(httpClient *http.Client) *Client {
	return newClient(httpClient, baseURL)
}

// NewClientForTest returns a Client that sends requests to apiBaseURL
// instead of the real Google Health API, for use against an
// httptest.Server.
func NewClientForTest(httpClient *http.Client, apiBaseURL string) *Client {
	return newClient(httpClient, apiBaseURL)
}

func newClient(httpClient *http.Client, apiBaseURL string) *Client {
	return &Client{httpClient: httpClient, baseURL: apiBaseURL}
}

// GetIdentity fetches the authenticated user's Google Health identity,
// which includes the health user ID used to address subsequent requests.
func (c *Client) GetIdentity(ctx context.Context) (Identity, error) {
	var identity Identity
	if err := c.get(ctx, "/users/me/identity", nil, &identity); err != nil {
		return Identity{}, err
	}
	return identity, nil
}

// ListWeightDataPoints fetches one page of the user's weight history. Pass
// an empty pageToken for the first page.
func (c *Client) ListWeightDataPoints(ctx context.Context, healthUserID, pageToken string) (ListDataPointsResponse, error) {
	return c.listDataPoints(ctx, healthUserID, "weight", pageToken)
}

// ListHeightDataPoints fetches one page of the user's height history. Pass
// an empty pageToken for the first page.
func (c *Client) ListHeightDataPoints(ctx context.Context, healthUserID, pageToken string) (ListDataPointsResponse, error) {
	return c.listDataPoints(ctx, healthUserID, "height", pageToken)
}

func (c *Client) listDataPoints(ctx context.Context, healthUserID, dataType, pageToken string) (ListDataPointsResponse, error) {
	query := url.Values{"pageSize": {strconv.Itoa(maxPageSize)}}
	if pageToken != "" {
		query.Set("pageToken", pageToken)
	}

	path := fmt.Sprintf("/users/%s/dataTypes/%s/dataPoints", url.PathEscape(healthUserID), dataType)

	var resp ListDataPointsResponse
	if err := c.get(ctx, path, query, &resp); err != nil {
		return ListDataPointsResponse{}, err
	}
	return resp, nil
}

func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	reqURL := c.baseURL + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("build request for %s: %w", path, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("request %s: unexpected status %d: %s", path, resp.StatusCode, body)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response from %s: %w", path, err)
	}

	return nil
}
