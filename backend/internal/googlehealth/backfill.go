package googlehealth

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"

	"github.com/isAdamBailey/massa/backend/internal/db"
)

// BackfillService pulls a user's full weight and height history from the
// Google Health API and upserts it into the local database.
type BackfillService struct {
	q           Querier
	credentials CredentialsRepository
	syncMeta    SyncMetadataRepository
	oauthConfig *oauth2.Config
	apiBaseURL  string
}

// NewBackfillService returns a BackfillService that writes through q and
// authenticates using oauthConfig.
func NewBackfillService(q Querier, credentials CredentialsRepository, syncMeta SyncMetadataRepository, oauthConfig *oauth2.Config) *BackfillService {
	return &BackfillService{
		q:           q,
		credentials: credentials,
		syncMeta:    syncMeta,
		oauthConfig: oauthConfig,
		apiBaseURL:  baseURL,
	}
}

// NewBackfillServiceForTest returns a BackfillService that sends Google
// Health API requests to apiBaseURL instead of the real API, for use
// against an httptest.Server.
func NewBackfillServiceForTest(q Querier, credentials CredentialsRepository, syncMeta SyncMetadataRepository, oauthConfig *oauth2.Config, apiBaseURL string) *BackfillService {
	return &BackfillService{
		q:           q,
		credentials: credentials,
		syncMeta:    syncMeta,
		oauthConfig: oauthConfig,
		apiBaseURL:  apiBaseURL,
	}
}

// Run fetches the user's complete weight and height history from Google
// Health and upserts it into weight_entries and height_entries, then
// records the result in sync_metadata.
func (s *BackfillService) Run(ctx context.Context, userID uuid.UUID) error {
	creds, err := s.credentials.Get(ctx, userID)
	if err != nil {
		return err
	}

	token := &oauth2.Token{
		RefreshToken: creds.RefreshToken,
		AccessToken:  creds.AccessToken,
	}
	if creds.AccessTokenExpiresAt != nil {
		token.Expiry = *creds.AccessTokenExpiresAt
	}

	tokenSource := s.oauthConfig.TokenSource(ctx, token)
	client := newClient(oauth2.NewClient(ctx, tokenSource), s.apiBaseURL)

	if err := s.syncWeight(ctx, client, userID, creds.HealthUserID); err != nil {
		return fmt.Errorf("sync weight: %w", err)
	}
	if err := s.syncHeight(ctx, client, userID, creds.HealthUserID); err != nil {
		return fmt.Errorf("sync height: %w", err)
	}

	if err := s.persistRefreshedToken(ctx, userID, creds, tokenSource); err != nil {
		return fmt.Errorf("persist refreshed token: %w", err)
	}

	now := time.Now().UTC()
	meta, err := s.syncMeta.GetOrCreate(ctx, userID)
	if err != nil {
		return fmt.Errorf("load sync metadata: %w", err)
	}
	meta.LastFullBackfillAt = &now
	meta.LastIncrementalSyncAt = &now
	meta.WeightSyncWatermark = &now
	meta.HeightSyncWatermark = &now
	if err := s.syncMeta.Update(ctx, userID, meta); err != nil {
		return fmt.Errorf("update sync metadata: %w", err)
	}

	return nil
}

func (s *BackfillService) persistRefreshedToken(ctx context.Context, userID uuid.UUID, creds Credentials, tokenSource oauth2.TokenSource) error {
	newToken, err := tokenSource.Token()
	if err != nil {
		return err
	}

	changed := newToken.AccessToken != creds.AccessToken
	if newToken.RefreshToken != "" && newToken.RefreshToken != creds.RefreshToken {
		creds.RefreshToken = newToken.RefreshToken
		changed = true
	}
	if !changed {
		return nil
	}

	creds.AccessToken = newToken.AccessToken
	if !newToken.Expiry.IsZero() {
		expiry := newToken.Expiry
		creds.AccessTokenExpiresAt = &expiry
	}

	return s.credentials.Save(ctx, userID, creds)
}

func (s *BackfillService) syncWeight(ctx context.Context, client *Client, userID uuid.UUID, healthUserID string) error {
	pageToken := ""
	for {
		resp, err := client.ListWeightDataPoints(ctx, healthUserID, pageToken)
		if err != nil {
			return err
		}

		for _, dp := range resp.DataPoints {
			if dp.Weight == nil {
				continue
			}

			recordedAt, err := time.Parse(time.RFC3339, dp.Weight.SampleTime.PhysicalTime)
			if err != nil {
				return fmt.Errorf("parse sample time %q: %w", dp.Weight.SampleTime.PhysicalTime, err)
			}

			weightKg, err := db.ToNumeric(dp.Weight.WeightGrams / 1000)
			if err != nil {
				return fmt.Errorf("convert weight: %w", err)
			}

			if dataPointID := googleDataPointID(dp.Name); dataPointID != nil {
				_, err = s.q.UpsertWeightEntryByGoogleID(ctx, db.UpsertWeightEntryByGoogleIDParams{
					UserID:            db.ToUUID(userID),
					WeightKg:          weightKg,
					RecordedAt:        db.ToTimestamptz(recordedAt),
					GoogleDataPointID: dataPointID,
				})
			} else {
				_, err = s.q.UpsertWeightEntryByRecordedAt(ctx, db.UpsertWeightEntryByRecordedAtParams{
					UserID:     db.ToUUID(userID),
					WeightKg:   weightKg,
					RecordedAt: db.ToTimestamptz(recordedAt),
				})
			}
			if err != nil {
				return fmt.Errorf("upsert weight entry: %w", err)
			}
		}

		if resp.NextPageToken == "" {
			return nil
		}
		pageToken = resp.NextPageToken
	}
}

func (s *BackfillService) syncHeight(ctx context.Context, client *Client, userID uuid.UUID, healthUserID string) error {
	pageToken := ""
	for {
		resp, err := client.ListHeightDataPoints(ctx, healthUserID, pageToken)
		if err != nil {
			return err
		}

		for _, dp := range resp.DataPoints {
			if dp.Height == nil {
				continue
			}

			recordedAt, err := time.Parse(time.RFC3339, dp.Height.SampleTime.PhysicalTime)
			if err != nil {
				return fmt.Errorf("parse sample time %q: %w", dp.Height.SampleTime.PhysicalTime, err)
			}

			heightMM, err := strconv.ParseInt(dp.Height.HeightMillimeters, 10, 64)
			if err != nil {
				return fmt.Errorf("parse height millimeters %q: %w", dp.Height.HeightMillimeters, err)
			}

			heightCm, err := db.ToNumeric(float64(heightMM) / 10)
			if err != nil {
				return fmt.Errorf("convert height: %w", err)
			}

			if dataPointID := googleDataPointID(dp.Name); dataPointID != nil {
				_, err = s.q.UpsertHeightEntryByGoogleID(ctx, db.UpsertHeightEntryByGoogleIDParams{
					UserID:            db.ToUUID(userID),
					HeightCm:          heightCm,
					RecordedAt:        db.ToTimestamptz(recordedAt),
					GoogleDataPointID: dataPointID,
				})
			} else {
				_, err = s.q.UpsertHeightEntryByRecordedAt(ctx, db.UpsertHeightEntryByRecordedAtParams{
					UserID:     db.ToUUID(userID),
					HeightCm:   heightCm,
					RecordedAt: db.ToTimestamptz(recordedAt),
				})
			}
			if err != nil {
				return fmt.Errorf("upsert height entry: %w", err)
			}
		}

		if resp.NextPageToken == "" {
			return nil
		}
		pageToken = resp.NextPageToken
	}
}

// googleDataPointID extracts the trailing {data_point} segment from a
// DataPoint.Name (format users/{user}/dataTypes/{data_type}/dataPoints/{data_point}),
// or returns nil if name is empty (most weight/height data points do not
// populate it).
func googleDataPointID(name string) *string {
	if name == "" {
		return nil
	}
	idx := strings.LastIndex(name, "/")
	if idx == -1 || idx == len(name)-1 {
		return nil
	}
	id := name[idx+1:]
	return &id
}
