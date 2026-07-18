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

	"github.com/isAdamBailey/massa/backend/internal/bmi"
	"github.com/isAdamBailey/massa/backend/internal/db"
	"github.com/isAdamBailey/massa/backend/internal/googlehealth"
	"github.com/isAdamBailey/massa/backend/internal/heights"
)

func TestBackfillService_Run(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users/me/dataTypes/weight/dataPoints":
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
		case "/users/me/dataTypes/height/dataPoints":
			_, _ = w.Write([]byte(`{
				"dataPoints": [
					{
						"height": {"heightMillimeters": "1800", "sampleTime": {"physicalTime": "2024-01-01T08:00:00Z"}}
					}
				]
			}`))
		case "/users/me/dataTypes/active-energy-burned/dataPoints":
			_, _ = w.Write([]byte(`{
				"dataPoints": [
					{
						"activeEnergyBurned": {"kcal": 200, "interval": {"startTime": "2024-01-01T08:00:00Z", "endTime": "2024-01-01T09:00:00Z"}}
					},
					{
						"activeEnergyBurned": {"kcal": 150, "interval": {"startTime": "2024-01-01T18:00:00Z", "endTime": "2024-01-01T19:00:00Z"}}
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

	heightResolver := heights.NewResolver(q)
	service := googlehealth.NewBackfillServiceForTest(q, credRepo, syncRepo, heightResolver, oauthConfig, srv.URL)

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

	bmiValue, err := db.FromNumeric(byID.Bmi)
	require.NoError(t, err)
	assert.InDelta(t, bmi.Calculate(70.0, 180.0), bmiValue, 0.001)
	heightUsedCm, err := db.FromNumeric(byID.HeightUsedCm)
	require.NoError(t, err)
	assert.InDelta(t, 180.0, heightUsedCm, 0.001)

	heightEntries := q.heightEntries[userID]
	require.Len(t, heightEntries, 1)
	for _, entry := range heightEntries {
		heightCm, err := db.FromNumeric(entry.HeightCm)
		require.NoError(t, err)
		assert.InDelta(t, 180.0, heightCm, 0.001)
	}

	activeEnergyEntries := q.activeEnergyEntries[userID]
	require.Len(t, activeEnergyEntries, 1, "both intervals fall on the same day and should be summed into one entry")
	for _, entry := range activeEnergyEntries {
		kcal, err := db.FromNumeric(entry.ActiveEnergyKcal)
		require.NoError(t, err)
		assert.InDelta(t, 350.0, kcal, 0.001)
	}

	meta, err := syncRepo.GetOrCreate(context.Background(), userID)
	require.NoError(t, err)
	require.NotNil(t, meta.LastFullBackfillAt)
	require.NotNil(t, meta.WeightSyncWatermark)
	require.NotNil(t, meta.HeightSyncWatermark)
	require.NotNil(t, meta.ActiveEnergySyncWatermark)
}

func TestBackfillService_Run_SkipsWeightWhenManualEntryExistsForDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users/me/dataTypes/weight/dataPoints":
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
		case "/users/me/dataTypes/height/dataPoints":
			_, _ = w.Write([]byte(`{"dataPoints": []}`))
		case "/users/me/dataTypes/active-energy-burned/dataPoints":
			_, _ = w.Write([]byte(`{"dataPoints": []}`))
		default:
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	q := newFakeQuerier()
	credRepo := googlehealth.NewPostgresCredentialsRepository(q, testKey(t))
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

	// A manual entry already exists for 2024-01-01; the Google entry for
	// that same day should be skipped rather than added as a duplicate.
	manualRecordedAt, err := time.Parse(time.RFC3339, "2024-01-01T20:00:00Z")
	require.NoError(t, err)
	weightKg, err := db.ToNumeric(69.0)
	require.NoError(t, err)
	q.weightEntries[userID] = map[string]db.WeightEntry{
		"manual-2024-01-01": {
			ID:         db.ToUUID(uuid.New()),
			UserID:     db.ToUUID(userID),
			WeightKg:   weightKg,
			RecordedAt: db.ToTimestamptz(manualRecordedAt),
			Source:     "manual",
		},
	}

	heightResolver := heights.NewResolver(q)
	service := googlehealth.NewBackfillServiceForTest(q, credRepo, syncRepo, heightResolver, oauthConfig, srv.URL)

	require.NoError(t, service.Run(context.Background(), userID))

	weightEntries := q.weightEntries[userID]
	require.Len(t, weightEntries, 2, "manual entry plus the one Google entry not shadowed by it")

	_, dpSkipped := weightEntries["dp-1"]
	assert.False(t, dpSkipped, "Google entry for the day with an existing manual entry should be skipped")

	_, manualStillPresent := weightEntries["manual-2024-01-01"]
	assert.True(t, manualStillPresent)
}

func TestBackfillService_Run_SkipsSecondGoogleWeightEntryForSameDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users/me/dataTypes/weight/dataPoints":
			// Two weigh-ins recorded by Google on the same calendar day; only
			// the first one encountered should be kept.
			_, _ = w.Write([]byte(`{
				"dataPoints": [
					{
						"name": "users/health-user-123/dataTypes/weight/dataPoints/dp-1",
						"weight": {"weightGrams": 70000, "sampleTime": {"physicalTime": "2024-01-01T08:00:00Z"}}
					},
					{
						"name": "users/health-user-123/dataTypes/weight/dataPoints/dp-2",
						"weight": {"weightGrams": 70500, "sampleTime": {"physicalTime": "2024-01-01T20:00:00Z"}}
					}
				]
			}`))
		case "/users/me/dataTypes/height/dataPoints":
			_, _ = w.Write([]byte(`{"dataPoints": []}`))
		case "/users/me/dataTypes/active-energy-burned/dataPoints":
			_, _ = w.Write([]byte(`{"dataPoints": []}`))
		default:
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	q := newFakeQuerier()
	credRepo := googlehealth.NewPostgresCredentialsRepository(q, testKey(t))
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

	heightResolver := heights.NewResolver(q)
	service := googlehealth.NewBackfillServiceForTest(q, credRepo, syncRepo, heightResolver, oauthConfig, srv.URL)

	require.NoError(t, service.Run(context.Background(), userID))

	weightEntries := q.weightEntries[userID]
	require.Len(t, weightEntries, 1, "only the first weigh-in of the day should be kept")

	_, firstKept := weightEntries["dp-1"]
	assert.True(t, firstKept)
	_, secondSkipped := weightEntries["dp-2"]
	assert.False(t, secondSkipped, "second weigh-in for an already-synced day should be skipped")
}

func TestBackfillService_RunReauthRequired(t *testing.T) {
	// The token endpoint rejects the refresh token as Google does once it has
	// expired or been revoked. The expired access token forces a refresh
	// before any API call, so the failure surfaces from Run.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/token" {
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"Token has been expired or revoked."}`))
	}))
	defer srv.Close()

	q := newFakeQuerier()
	credRepo := googlehealth.NewPostgresCredentialsRepository(q, testKey(t))
	syncRepo := googlehealth.NewPostgresSyncMetadataRepository(q)
	oauthConfig := &oauth2.Config{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		Endpoint:     oauth2.Endpoint{TokenURL: srv.URL + "/token"},
	}

	userID := uuid.New()
	expired := time.Now().Add(-time.Hour)
	require.NoError(t, credRepo.Save(context.Background(), userID, googlehealth.Credentials{
		HealthUserID:         "health-user-123",
		RefreshToken:         "revoked-refresh-token",
		AccessToken:          "expired-access-token",
		AccessTokenExpiresAt: &expired,
	}))

	heightResolver := heights.NewResolver(q)
	service := googlehealth.NewBackfillServiceForTest(q, credRepo, syncRepo, heightResolver, oauthConfig, "http://unused.invalid")

	err := service.Run(context.Background(), userID)
	require.ErrorIs(t, err, googlehealth.ErrReauthRequired)
}

func TestBackfillService_Run_SecondRunFiltersByWatermark(t *testing.T) {
	empty := `{"dataPoints": []}`
	var weightFilters, heightFilters, activeEnergyFilters []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		filter := r.URL.Query().Get("filter")
		switch r.URL.Path {
		case "/users/me/dataTypes/weight/dataPoints":
			weightFilters = append(weightFilters, filter)
		case "/users/me/dataTypes/height/dataPoints":
			heightFilters = append(heightFilters, filter)
		case "/users/me/dataTypes/active-energy-burned/dataPoints":
			activeEnergyFilters = append(activeEnergyFilters, filter)
		default:
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(empty))
	}))
	defer srv.Close()

	q := newFakeQuerier()
	credRepo := googlehealth.NewPostgresCredentialsRepository(q, testKey(t))
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

	heightResolver := heights.NewResolver(q)
	service := googlehealth.NewBackfillServiceForTest(q, credRepo, syncRepo, heightResolver, oauthConfig, srv.URL)

	require.NoError(t, service.Run(context.Background(), userID))
	require.Len(t, weightFilters, 1)
	assert.Empty(t, weightFilters[0], "first run has no watermark yet, so it pulls full history")

	metaAfterFirstRun, err := syncRepo.GetOrCreate(context.Background(), userID)
	require.NoError(t, err)
	require.NotNil(t, metaAfterFirstRun.LastFullBackfillAt)

	require.NoError(t, service.Run(context.Background(), userID))
	require.Len(t, weightFilters, 2)
	require.Len(t, heightFilters, 2)
	require.Len(t, activeEnergyFilters, 2)
	assert.NotEmpty(t, weightFilters[1], "second run has a watermark, so it should bound the fetch")
	assert.Contains(t, weightFilters[1], "weight.sample_time.physical_time >=")
	assert.NotEmpty(t, heightFilters[1])
	assert.Contains(t, heightFilters[1], "height.sample_time.physical_time >=")
	assert.NotEmpty(t, activeEnergyFilters[1])
	assert.Contains(t, activeEnergyFilters[1], "activeEnergyBurned.interval.start_time >=")

	metaAfterSecondRun, err := syncRepo.GetOrCreate(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, metaAfterFirstRun.LastFullBackfillAt.Unix(), metaAfterSecondRun.LastFullBackfillAt.Unix(),
		"only the first, unfiltered run counts as a full backfill")
}

func TestBackfillService_Run_ActiveEnergyRefreshesTodayAndYesterdayOnly(t *testing.T) {
	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	older := now.AddDate(0, 0, -10).Format("2006-01-02")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/users/me/dataTypes/weight/dataPoints", "/users/me/dataTypes/height/dataPoints":
			_, _ = w.Write([]byte(`{"dataPoints": []}`))
		case "/users/me/dataTypes/active-energy-burned/dataPoints":
			_, _ = w.Write([]byte(`{
				"dataPoints": [
					{"activeEnergyBurned": {"kcal": 111, "interval": {"startTime": "` + today + `T08:00:00Z", "endTime": "` + today + `T09:00:00Z"}}},
					{"activeEnergyBurned": {"kcal": 222, "interval": {"startTime": "` + yesterday + `T08:00:00Z", "endTime": "` + yesterday + `T09:00:00Z"}}},
					{"activeEnergyBurned": {"kcal": 333, "interval": {"startTime": "` + older + `T08:00:00Z", "endTime": "` + older + `T09:00:00Z"}}}
				]
			}`))
		default:
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	q := newFakeQuerier()
	credRepo := googlehealth.NewPostgresCredentialsRepository(q, testKey(t))
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

	// Pre-populate all three days as if a previous sync already stored them,
	// with a stale kcal value that should only be overwritten for
	// today/yesterday.
	staleKcal, err := db.ToNumeric(1.0)
	require.NoError(t, err)
	q.activeEnergyEntries[userID] = map[string]db.ActiveEnergyEntry{
		today:     {UserID: db.ToUUID(userID), Day: db.ToDate(now), ActiveEnergyKcal: staleKcal},
		yesterday: {UserID: db.ToUUID(userID), Day: db.ToDate(now.AddDate(0, 0, -1)), ActiveEnergyKcal: staleKcal},
		older:     {UserID: db.ToUUID(userID), Day: db.ToDate(now.AddDate(0, 0, -10)), ActiveEnergyKcal: staleKcal},
	}

	heightResolver := heights.NewResolver(q)
	service := googlehealth.NewBackfillServiceForTest(q, credRepo, syncRepo, heightResolver, oauthConfig, srv.URL)

	require.NoError(t, service.Run(context.Background(), userID))

	entries := q.activeEnergyEntries[userID]

	todayKcal, err := db.FromNumeric(entries[today].ActiveEnergyKcal)
	require.NoError(t, err)
	assert.InDelta(t, 111.0, todayKcal, 0.001, "today should always be refreshed")

	yesterdayKcal, err := db.FromNumeric(entries[yesterday].ActiveEnergyKcal)
	require.NoError(t, err)
	assert.InDelta(t, 222.0, yesterdayKcal, 0.001, "yesterday should always be refreshed")

	olderKcal, err := db.FromNumeric(entries[older].ActiveEnergyKcal)
	require.NoError(t, err)
	assert.InDelta(t, 1.0, olderKcal, 0.001, "an older day already stored should be left untouched")
}

func TestBackfillService_RunNotConnected(t *testing.T) {
	q := newFakeQuerier()
	credRepo := googlehealth.NewPostgresCredentialsRepository(q, testKey(t))
	syncRepo := googlehealth.NewPostgresSyncMetadataRepository(q)
	oauthConfig := &oauth2.Config{}

	heightResolver := heights.NewResolver(q)
	service := googlehealth.NewBackfillServiceForTest(q, credRepo, syncRepo, heightResolver, oauthConfig, "http://unused.invalid")

	err := service.Run(context.Background(), uuid.New())
	require.ErrorIs(t, err, googlehealth.ErrNotConnected)
}
