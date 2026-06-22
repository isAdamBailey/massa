package googlehealth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/oauth2"

	"github.com/isAdamBailey/massa/backend/internal/bmi"
	"github.com/isAdamBailey/massa/backend/internal/db"
	"github.com/isAdamBailey/massa/backend/internal/heights"
)

// HeightResolver resolves the height to use for a user's BMI calculations.
type HeightResolver interface {
	Resolve(ctx context.Context, userID uuid.UUID) (float64, error)
}

// BackfillService pulls a user's full weight and height history from the
// Google Health API and upserts it into the local database.
type BackfillService struct {
	q           Querier
	credentials CredentialsRepository
	syncMeta    SyncMetadataRepository
	heights     HeightResolver
	oauthConfig *oauth2.Config
	apiBaseURL  string
}

// NewBackfillService returns a BackfillService that writes through q and
// authenticates using oauthConfig.
func NewBackfillService(q Querier, credentials CredentialsRepository, syncMeta SyncMetadataRepository, heightResolver HeightResolver, oauthConfig *oauth2.Config) *BackfillService {
	return &BackfillService{
		q:           q,
		credentials: credentials,
		syncMeta:    syncMeta,
		heights:     heightResolver,
		oauthConfig: oauthConfig,
		apiBaseURL:  baseURL,
	}
}

// NewBackfillServiceForTest returns a BackfillService that sends Google
// Health API requests to apiBaseURL instead of the real API, for use
// against an httptest.Server.
func NewBackfillServiceForTest(q Querier, credentials CredentialsRepository, syncMeta SyncMetadataRepository, heightResolver HeightResolver, oauthConfig *oauth2.Config, apiBaseURL string) *BackfillService {
	return &BackfillService{
		q:           q,
		credentials: credentials,
		syncMeta:    syncMeta,
		heights:     heightResolver,
		oauthConfig: oauthConfig,
		apiBaseURL:  apiBaseURL,
	}
}

// Run fetches the user's complete weight and height history from Google
// Health and upserts it into weight_entries and height_entries, then
// records the result in sync_metadata. If the stored credentials are no
// longer valid it returns an error wrapping ErrReauthRequired.
func (s *BackfillService) Run(ctx context.Context, userID uuid.UUID) error {
	err := s.run(ctx, userID)
	if err != nil && isReauthError(err) {
		return errors.Join(ErrReauthRequired, err)
	}
	return err
}

func (s *BackfillService) run(ctx context.Context, userID uuid.UUID) error {
	client, persist, err := newAuthorizedClient(ctx, s.credentials, s.oauthConfig, userID, s.apiBaseURL)
	if err != nil {
		return err
	}

	if err := s.syncHeight(ctx, client, userID, "me"); err != nil {
		return fmt.Errorf("sync height: %w", err)
	}
	if err := s.syncWeight(ctx, client, userID, "me"); err != nil {
		return fmt.Errorf("sync weight: %w", err)
	}

	if err := persist(ctx); err != nil {
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

func (s *BackfillService) syncWeight(ctx context.Context, client *Client, userID uuid.UUID, healthUserID string) error {
	heightCm, err := s.heights.Resolve(ctx, userID)
	if errors.Is(err, heights.ErrNoHeight) {
		heightCm = 0
	} else if err != nil {
		return fmt.Errorf("resolve height: %w", err)
	}

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

			weightKgFloat := dp.Weight.WeightGrams / 1000
			weightKg, err := db.ToNumeric(weightKgFloat)
			if err != nil {
				return fmt.Errorf("convert weight: %w", err)
			}

			var bmiValue, heightUsedCm pgtype.Numeric
			if heightCm > 0 {
				bmiValue, err = db.ToNumeric(bmi.Calculate(weightKgFloat, heightCm))
				if err != nil {
					return fmt.Errorf("convert bmi: %w", err)
				}
				heightUsedCm, err = db.ToNumeric(heightCm)
				if err != nil {
					return fmt.Errorf("convert height used: %w", err)
				}
			}

			if dataPointID := googleDataPointID(dp.Name); dataPointID != nil {
				_, err = s.q.UpsertWeightEntryByGoogleID(ctx, db.UpsertWeightEntryByGoogleIDParams{
					UserID:            db.ToUUID(userID),
					WeightKg:          weightKg,
					RecordedAt:        db.ToTimestamptz(recordedAt),
					Bmi:               bmiValue,
					HeightUsedCm:      heightUsedCm,
					GoogleDataPointID: dataPointID,
				})
			} else {
				_, err = s.q.UpsertWeightEntryByRecordedAt(ctx, db.UpsertWeightEntryByRecordedAtParams{
					UserID:       db.ToUUID(userID),
					WeightKg:     weightKg,
					RecordedAt:   db.ToTimestamptz(recordedAt),
					Bmi:          bmiValue,
					HeightUsedCm: heightUsedCm,
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
