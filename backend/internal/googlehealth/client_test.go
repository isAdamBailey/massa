package googlehealth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isAdamBailey/massa/backend/internal/googlehealth"
)

func TestClient_GetIdentity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/users/me/identity", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name": "users/me/identity", "healthUserId": "abc123"}`))
	}))
	defer srv.Close()

	client := googlehealth.NewClientForTest(srv.Client(), srv.URL)

	identity, err := client.GetIdentity(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "abc123", identity.HealthUserID)
}

func TestClient_ListWeightDataPoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/users/abc123/dataTypes/weight/dataPoints", r.URL.Path)
		assert.Equal(t, "10000", r.URL.Query().Get("pageSize"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"dataPoints": [
				{"weight": {"weightGrams": 70123.4, "sampleTime": {"physicalTime": "2024-01-01T08:00:00Z", "utcOffset": "0s"}}}
			],
			"nextPageToken": "next-page"
		}`))
	}))
	defer srv.Close()

	client := googlehealth.NewClientForTest(srv.Client(), srv.URL)

	resp, err := client.ListWeightDataPoints(context.Background(), "abc123", "")
	require.NoError(t, err)
	require.Len(t, resp.DataPoints, 1)
	require.NotNil(t, resp.DataPoints[0].Weight)
	assert.InDelta(t, 70123.4, resp.DataPoints[0].Weight.WeightGrams, 0.001)
	assert.Equal(t, "2024-01-01T08:00:00Z", resp.DataPoints[0].Weight.SampleTime.PhysicalTime)
	assert.Equal(t, "next-page", resp.NextPageToken)
}

func TestClient_ListHeightDataPoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/users/abc123/dataTypes/height/dataPoints", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"dataPoints": [
				{"height": {"heightMillimeters": "1800", "sampleTime": {"physicalTime": "2024-01-01T08:00:00Z"}}}
			]
		}`))
	}))
	defer srv.Close()

	client := googlehealth.NewClientForTest(srv.Client(), srv.URL)

	resp, err := client.ListHeightDataPoints(context.Background(), "abc123", "")
	require.NoError(t, err)
	require.Len(t, resp.DataPoints, 1)
	require.NotNil(t, resp.DataPoints[0].Height)
	assert.Equal(t, "1800", resp.DataPoints[0].Height.HeightMillimeters)
}

func TestClient_CreateWeightDataPoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/users/abc123/dataTypes/weight/dataPoints", r.URL.Path)

		var body googlehealth.DataPoint
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "users/abc123/dataTypes/weight/dataPoints/dp-1", body.Name)
		require.NotNil(t, body.Weight)
		assert.InDelta(t, 70123.4, body.Weight.WeightGrams, 0.001)
		assert.Equal(t, "2024-01-02T08:00:00Z", body.Weight.SampleTime.PhysicalTime)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name": "users/abc123/dataTypes/weight/dataPoints/dp-1"}`))
	}))
	defer srv.Close()

	client := googlehealth.NewClientForTest(srv.Client(), srv.URL)

	recordedAt := time.Date(2024, 1, 2, 8, 0, 0, 0, time.UTC)
	resp, err := client.CreateWeightDataPoint(context.Background(), "abc123", "dp-1", 70123.4, recordedAt)
	require.NoError(t, err)
	assert.Equal(t, "users/abc123/dataTypes/weight/dataPoints/dp-1", resp.Name)
}

func TestClient_UpdateWeightDataPoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/users/abc123/dataTypes/weight/dataPoints/dp-1", r.URL.Path)

		var body googlehealth.DataPoint
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.NotNil(t, body.Weight)
		assert.InDelta(t, 70123.4, body.Weight.WeightGrams, 0.001)
		assert.Equal(t, "2024-01-02T08:00:00Z", body.Weight.SampleTime.PhysicalTime)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name": "users/abc123/dataTypes/weight/dataPoints/dp-1"}`))
	}))
	defer srv.Close()

	client := googlehealth.NewClientForTest(srv.Client(), srv.URL)

	recordedAt := time.Date(2024, 1, 2, 8, 0, 0, 0, time.UTC)
	resp, err := client.UpdateWeightDataPoint(context.Background(), "abc123", "dp-1", 70123.4, recordedAt)
	require.NoError(t, err)
	assert.Equal(t, "users/abc123/dataTypes/weight/dataPoints/dp-1", resp.Name)
}

func TestClient_DeleteWeightDataPoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/users/abc123/dataTypes/weight/dataPoints/dp-1", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := googlehealth.NewClientForTest(srv.Client(), srv.URL)

	err := client.DeleteWeightDataPoint(context.Background(), "abc123", "dp-1")
	require.NoError(t, err)
}

func TestClient_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "invalid token"}`))
	}))
	defer srv.Close()

	client := googlehealth.NewClientForTest(srv.Client(), srv.URL)

	_, err := client.GetIdentity(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}
