package googlehealth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/isAdamBailey/massa/backend/internal/googlehealth"
)

func TestPushService_PushWeight_Create(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/users/me/dataTypes/weight/dataPoints", r.URL.Path)

		var body googlehealth.DataPoint
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "users/me/dataTypes/weight/dataPoints/dp-1", body.Name)
		require.NotNil(t, body.Weight)
		assert.InDelta(t, 70000, body.Weight.WeightGrams, 0.001)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name": "users/me/dataTypes/weight/dataPoints/dp-1"}`))
	}))
	defer srv.Close()

	q := newFakeQuerier()
	credRepo := googlehealth.NewPostgresCredentialsRepository(q, testKey(t))
	oauthConfig := &oauth2.Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		Endpoint:     oauth2.Endpoint{TokenURL: "http://unused.invalid/token"},
	}

	userID := uuid.New()
	expiry := time.Now().Add(time.Hour)
	require.NoError(t, credRepo.Save(context.Background(), userID, googlehealth.Credentials{
		HealthUserID:         "health-user-123",
		RefreshToken:         "refresh-token",
		AccessToken:          "access-token",
		AccessTokenExpiresAt: &expiry,
	}))

	service := googlehealth.NewPushServiceForTest(credRepo, oauthConfig, srv.URL)

	recordedAt := time.Date(2024, 1, 2, 8, 0, 0, 0, time.UTC)
	err := service.PushWeight(context.Background(), userID, "dp-1", 70, recordedAt, true)
	require.NoError(t, err)
}

func TestPushService_PushWeight_CreateFallsBackToUpdateOnConflict(t *testing.T) {
	var requests []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method)
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error": "already exists"}`))
		case http.MethodPatch:
			assert.Equal(t, "/users/me/dataTypes/weight/dataPoints/dp-1", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name": "users/me/dataTypes/weight/dataPoints/dp-1"}`))
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()

	q := newFakeQuerier()
	credRepo := googlehealth.NewPostgresCredentialsRepository(q, testKey(t))
	oauthConfig := &oauth2.Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		Endpoint:     oauth2.Endpoint{TokenURL: "http://unused.invalid/token"},
	}

	userID := uuid.New()
	expiry := time.Now().Add(time.Hour)
	require.NoError(t, credRepo.Save(context.Background(), userID, googlehealth.Credentials{
		HealthUserID:         "health-user-123",
		RefreshToken:         "refresh-token",
		AccessToken:          "access-token",
		AccessTokenExpiresAt: &expiry,
	}))

	service := googlehealth.NewPushServiceForTest(credRepo, oauthConfig, srv.URL)

	recordedAt := time.Date(2024, 1, 2, 8, 0, 0, 0, time.UTC)
	err := service.PushWeight(context.Background(), userID, "dp-1", 70, recordedAt, true)
	require.NoError(t, err)
	assert.Equal(t, []string{http.MethodPost, http.MethodPatch}, requests)
}

func TestPushService_PushWeight_Update(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/users/me/dataTypes/weight/dataPoints/dp-1", r.URL.Path)

		var body googlehealth.DataPoint
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.NotNil(t, body.Weight)
		assert.InDelta(t, 70000, body.Weight.WeightGrams, 0.001)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name": "users/me/dataTypes/weight/dataPoints/dp-1"}`))
	}))
	defer srv.Close()

	q := newFakeQuerier()
	credRepo := googlehealth.NewPostgresCredentialsRepository(q, testKey(t))
	oauthConfig := &oauth2.Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		Endpoint:     oauth2.Endpoint{TokenURL: "http://unused.invalid/token"},
	}

	userID := uuid.New()
	expiry := time.Now().Add(time.Hour)
	require.NoError(t, credRepo.Save(context.Background(), userID, googlehealth.Credentials{
		HealthUserID:         "health-user-123",
		RefreshToken:         "refresh-token",
		AccessToken:          "access-token",
		AccessTokenExpiresAt: &expiry,
	}))

	service := googlehealth.NewPushServiceForTest(credRepo, oauthConfig, srv.URL)

	recordedAt := time.Date(2024, 1, 2, 8, 0, 0, 0, time.UTC)
	err := service.PushWeight(context.Background(), userID, "dp-1", 70, recordedAt, false)
	require.NoError(t, err)
}

func TestPushService_PushWeight_NotConnected(t *testing.T) {
	q := newFakeQuerier()
	credRepo := googlehealth.NewPostgresCredentialsRepository(q, testKey(t))
	oauthConfig := &oauth2.Config{}

	service := googlehealth.NewPushServiceForTest(credRepo, oauthConfig, "http://unused.invalid")

	err := service.PushWeight(context.Background(), uuid.New(), "dp-1", 70, time.Now(), true)
	require.ErrorIs(t, err, googlehealth.ErrNotConnected)
}

func TestPushService_DeleteWeight(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/users/me/dataTypes/weight/dataPoints/dp-1", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	q := newFakeQuerier()
	credRepo := googlehealth.NewPostgresCredentialsRepository(q, testKey(t))
	oauthConfig := &oauth2.Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		Endpoint:     oauth2.Endpoint{TokenURL: "http://unused.invalid/token"},
	}

	userID := uuid.New()
	expiry := time.Now().Add(time.Hour)
	require.NoError(t, credRepo.Save(context.Background(), userID, googlehealth.Credentials{
		HealthUserID:         "health-user-123",
		RefreshToken:         "refresh-token",
		AccessToken:          "access-token",
		AccessTokenExpiresAt: &expiry,
	}))

	service := googlehealth.NewPushServiceForTest(credRepo, oauthConfig, srv.URL)

	err := service.DeleteWeight(context.Background(), userID, "dp-1")
	require.NoError(t, err)
}

func TestPushService_DeleteWeight_NotConnected(t *testing.T) {
	q := newFakeQuerier()
	credRepo := googlehealth.NewPostgresCredentialsRepository(q, testKey(t))
	oauthConfig := &oauth2.Config{}

	service := googlehealth.NewPushServiceForTest(credRepo, oauthConfig, "http://unused.invalid")

	err := service.DeleteWeight(context.Background(), uuid.New(), "dp-1")
	require.ErrorIs(t, err, googlehealth.ErrNotConnected)
}
