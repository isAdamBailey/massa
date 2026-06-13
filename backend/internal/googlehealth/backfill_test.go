package googlehealth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/isAdamBailey/massa/backend/internal/db"
	"github.com/isAdamBailey/massa/backend/internal/googlehealth"
)

func TestBackfillService_Run(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users/health-user-123/dataTypes/weight/dataPoints":
			_, _ = w.Write([]byte(`{
				"dataPoints": [
					{
						"name": "users/health-user-123/dataTypes/weight/dataPoints/dp-1",
						"weight": {"weightGrams": 70000, "sampleTime": {"physicalTime": "2024-01-01T08:00:00Z"}}
					},
					{
						"weight": {"weightGrams": 71500, "sampleTime": {"physicalTime": "2024-01-02T08:00:00Z"}}
					}
				]
			}`))
		case "/users/health-user-123/dataTypes/height/dataPoints":
			_, _ = w.Write([]byte(`{
				"dataPoints": [
					{
						"height": {"heightMillimeters": "1800", "sampleTime": {"physicalTime": "2024-01-01T08:00:00Z"}}
					}
				]
			}`))
		default:
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	q := newFakeQuerier()
	key := testKey(t)
	credRepo := googlehealth.NewPostgresCredentialsRepository(q, key)
	syncRepo := googlehealth.NewPostgresSyncMetadataRepository(q)
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

	service := googlehealth.NewBackfillServiceForTest(q, credRepo, syncRepo, oauthConfig, srv.URL)

	require.NoError(t, service.Run(context.Background(), userID))

	weightEntries := q.weightEntries[userID]
	require.Len(t, weightEntries, 2)

	byID, ok := weightEntries["dp-1"]
	require.True(t, ok, "expected weight entry keyed by Google data point ID")
	weightKg, err := db.FromNumeric(byID.WeightKg)
	require.NoError(t, err)
	assert.InDelta(t, 70.0, weightKg, 0.001)
	require.NotNil(t, byID.GoogleDataPointID)
	assert.Equal(t, "dp-1", *byID.GoogleDataPointID)

	heightEntries := q.heightEntries[userID]
	require.Len(t, heightEntries, 1)
	for _, entry := range heightEntries {
		heightCm, err := db.FromNumeric(entry.HeightCm)
		require.NoError(t, err)
		assert.InDelta(t, 180.0, heightCm, 0.001)
	}

	meta, err := syncRepo.GetOrCreate(context.Background(), userID)
	require.NoError(t, err)
	require.NotNil(t, meta.LastFullBackfillAt)
	require.NotNil(t, meta.WeightSyncWatermark)
	require.NotNil(t, meta.HeightSyncWatermark)
}

func TestBackfillService_RunNotConnected(t *testing.T) {
	q := newFakeQuerier()
	credRepo := googlehealth.NewPostgresCredentialsRepository(q, testKey(t))
	syncRepo := googlehealth.NewPostgresSyncMetadataRepository(q)
	oauthConfig := &oauth2.Config{}

	service := googlehealth.NewBackfillServiceForTest(q, credRepo, syncRepo, oauthConfig, "http://unused.invalid")

	err := service.Run(context.Background(), uuid.New())
	require.ErrorIs(t, err, googlehealth.ErrNotConnected)
}
