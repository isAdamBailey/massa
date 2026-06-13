// Package googlehealth provides OAuth2 connection management and a typed
// client for the Google Health API (v4), used to read a user's weight and
// height history.
package googlehealth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
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

// CreateWeightDataPoint creates a new weight data point in healthUserID's
// weight history, using dataPointID as the client-supplied data point ID
// (4-63 characters, lowercase letters/numbers/hyphens).
func (c *Client) CreateWeightDataPoint(ctx context.Context, healthUserID, dataPointID string, weightGrams float64, recordedAt time.Time) (DataPoint, error) {
	path := fmt.Sprintf("/users/%s/dataTypes/weight/dataPoints", url.PathEscape(healthUserID))
	body := DataPoint{
		Name: weightDataPointName(healthUserID, dataPointID),
		Weight: &Weight{
			WeightGrams: weightGrams,
			SampleTime:  ObservationSampleTime{PhysicalTime: recordedAt.UTC().Format(time.RFC3339Nano)},
		},
	}

	var resp DataPoint
	if err := c.post(ctx, path, body, &resp); err != nil {
		return DataPoint{}, err
	}
	return resp, nil
}

// UpdateWeightDataPoint replaces the weight data point identified by
// dataPointID in healthUserID's weight history. The data point must already
// exist; use CreateWeightDataPoint for new data points.
func (c *Client) UpdateWeightDataPoint(ctx context.Context, healthUserID, dataPointID string, weightGrams float64, recordedAt time.Time) (DataPoint, error) {
	path := weightDataPointPath(healthUserID, dataPointID)
	body := DataPoint{
		Weight: &Weight{
			WeightGrams: weightGrams,
			SampleTime:  ObservationSampleTime{PhysicalTime: recordedAt.UTC().Format(time.RFC3339Nano)},
		},
	}

	var resp DataPoint
	if err := c.patch(ctx, path, body, &resp); err != nil {
		return DataPoint{}, err
	}
	return resp, nil
}

// DeleteWeightDataPoint removes the weight data point identified by
// dataPointID from healthUserID's weight history.
func (c *Client) DeleteWeightDataPoint(ctx context.Context, healthUserID, dataPointID string) error {
	return c.delete(ctx, weightDataPointPath(healthUserID, dataPointID))
}

func weightDataPointPath(healthUserID, dataPointID string) string {
	return fmt.Sprintf("/users/%s/dataTypes/weight/dataPoints/%s", url.PathEscape(healthUserID), url.PathEscape(dataPointID))
}

// weightDataPointName returns the resource name to set on a DataPoint when
// creating it with a client-supplied dataPointID.
func weightDataPointName(healthUserID, dataPointID string) string {
	return fmt.Sprintf("users/%s/dataTypes/weight/dataPoints/%s", healthUserID, dataPointID)
}

func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	reqURL := c.baseURL + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}
	return c.do(ctx, http.MethodGet, reqURL, nil, out)
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPost, c.baseURL+path, body, out)
}

func (c *Client) patch(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPatch, c.baseURL+path, body, out)
}

func (c *Client) delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, c.baseURL+path, nil, nil)
}

func (c *Client) do(ctx context.Context, method, reqURL string, body, out any) error {
	var bodyReader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request body for %s: %w", reqURL, err)
		}
		bodyReader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("build request for %s: %w", reqURL, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request %s: %w", reqURL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("request %s: unexpected status %d: %w", reqURL, resp.StatusCode, &apiError{statusCode: resp.StatusCode, body: respBody})
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response from %s: %w", reqURL, err)
	}

	return nil
}

// apiError represents a non-2xx HTTP response from the Google Health API.
type apiError struct {
	statusCode int
	body       []byte
}

func (e *apiError) Error() string {
	return fmt.Sprintf("status %d: %s", e.statusCode, e.body)
}

// isConflict reports whether err is an apiError with a 409 Conflict status,
// indicating the data point being created already exists.
func isConflict(err error) bool {
	var apiErr *apiError
	return errors.As(err, &apiErr) && apiErr.statusCode == http.StatusConflict
}
