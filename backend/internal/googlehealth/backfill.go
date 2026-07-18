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

// watermarkOverlap is subtracted from a stored watermark before building a
// filter, so a sync that starts slightly late (or a data point that lands
// with a delay on Google's side) is still covered. Harmless to overlap
// further back since weight/height/active-energy syncing never overwrites a
// day that's already stored (active energy's today/yesterday refresh is the
// deliberate exception).
const watermarkOverlap = 48 * time.Hour

func (s *BackfillService) run(ctx context.Context, userID uuid.UUID) error {
	client, persist, err := newAuthorizedClient(ctx, s.credentials, s.oauthConfig, userID, s.apiBaseURL)
	if err != nil {
		return err
	}

	meta, err := s.syncMeta.GetOrCreate(ctx, userID)
	if err != nil {
		return fmt.Errorf("load sync metadata: %w", err)
	}
	// A user with no watermarks yet has never completed a sync; run a full,
	// unfiltered pull and record it as a full backfill. Otherwise, bound each
	// data type's fetch to what's changed since it last synced.
	isFullBackfill := meta.WeightSyncWatermark == nil && meta.HeightSyncWatermark == nil && meta.ActiveEnergySyncWatermark == nil

	if err := s.syncHeight(ctx, client, userID, "me", meta.HeightSyncWatermark); err != nil {
		return fmt.Errorf("sync height: %w", err)
	}
	if err := s.syncWeight(ctx, client, userID, "me", meta.WeightSyncWatermark); err != nil {
		return fmt.Errorf("sync weight: %w", err)
	}
	if err := s.syncActiveEnergy(ctx, client, userID, "me", meta.ActiveEnergySyncWatermark); err != nil {
		return fmt.Errorf("sync active energy: %w", err)
	}

	if err := persist(ctx); err != nil {
		return fmt.Errorf("persist refreshed token: %w", err)
	}

	now := time.Now().UTC()
	if isFullBackfill {
		meta.LastFullBackfillAt = &now
	}
	meta.LastIncrementalSyncAt = &now
	meta.WeightSyncWatermark = &now
	meta.HeightSyncWatermark = &now
	meta.ActiveEnergySyncWatermark = &now
	if err := s.syncMeta.Update(ctx, userID, meta); err != nil {
		return fmt.Errorf("update sync metadata: %w", err)
	}

	return nil
}

// sampleTimeFilter returns an AIP-160 filter bounding field to values at or
// after watermark minus watermarkOverlap (and no later than floor, if given,
// so callers can guarantee a minimum coverage window regardless of how
// recent the watermark is), or "" (no filter, full history) if watermark is
// nil.
func sampleTimeFilter(field string, watermark *time.Time, floor *time.Time) string {
	if watermark == nil {
		return ""
	}
	since := watermark.Add(-watermarkOverlap)
	if floor != nil && floor.Before(since) {
		since = *floor
	}
	return fmt.Sprintf(`%s >= "%s"`, field, since.UTC().Format(time.RFC3339))
}

func (s *BackfillService) syncWeight(ctx context.Context, client *Client, userID uuid.UUID, healthUserID string, watermark *time.Time) error {
	heightCm, err := s.heights.Resolve(ctx, userID)
	if errors.Is(err, heights.ErrNoHeight) {
		heightCm = 0
	} else if err != nil {
		return fmt.Errorf("resolve height: %w", err)
	}

	filter := sampleTimeFilter("weight.sample_time.physical_time", watermark, nil)

	pageToken := ""
	for {
		resp, err := client.ListWeightDataPoints(ctx, healthUserID, pageToken, filter)
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

			exists, err := s.q.ExistsWeightEntryForDate(ctx, db.ExistsWeightEntryForDateParams{
				UserID: db.ToUUID(userID),
				Date:   db.ToDate(recordedAt),
			})
			if err != nil {
				return fmt.Errorf("check weight entry for date: %w", err)
			}
			if exists {
				continue
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

func (s *BackfillService) syncHeight(ctx context.Context, client *Client, userID uuid.UUID, healthUserID string, watermark *time.Time) error {
	filter := sampleTimeFilter("height.sample_time.physical_time", watermark, nil)

	pageToken := ""
	for {
		resp, err := client.ListHeightDataPoints(ctx, healthUserID, pageToken, filter)
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

			exists, err := s.q.ExistsHeightEntryForDate(ctx, db.ExistsHeightEntryForDateParams{
				UserID: db.ToUUID(userID),
				Date:   db.ToDate(recordedAt),
			})
			if err != nil {
				return fmt.Errorf("check height entry for date: %w", err)
			}
			if exists {
				continue
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

// syncActiveEnergy pulls the user's active energy burned history from
// Google Health and upserts one row per calendar day, summing the kcal of
// every interval data point observed that day (active energy is an interval
// data type with many readings per day, unlike the single daily
// weight/height reading). Unlike weight/height, active energy is
// Google-only, so today and yesterday are always re-pulled and overwritten
// to reflect their running/finalized totals; older days already stored are
// left untouched.
func (s *BackfillService) syncActiveEnergy(ctx context.Context, client *Client, userID uuid.UUID, healthUserID string, watermark *time.Time) error {
	dailyKcal := make(map[string]float64)

	// now is computed once and reused for both the fetch's lower bound and
	// the today/yesterday freshness check below, so a sync that straddles a
	// UTC midnight boundary can't classify a day differently between them.
	now := time.Now().UTC()
	startOfYesterday := now.AddDate(0, 0, -1).Truncate(24 * time.Hour)
	// Always cover at least yesterday and today, regardless of the stored
	// watermark, since those two days are refreshed on every sync.
	filter := sampleTimeFilter("activeEnergyBurned.interval.start_time", watermark, &startOfYesterday)

	pageToken := ""
	for {
		resp, err := client.ListActiveEnergyDataPoints(ctx, healthUserID, pageToken, filter)
		if err != nil {
			return err
		}

		for _, dp := range resp.DataPoints {
			if dp.ActiveEnergyBurned == nil {
				continue
			}

			startTime, err := time.Parse(time.RFC3339, dp.ActiveEnergyBurned.Interval.StartTime)
			if err != nil {
				return fmt.Errorf("parse interval start time %q: %w", dp.ActiveEnergyBurned.Interval.StartTime, err)
			}

			day := startTime.Format("2006-01-02")
			dailyKcal[day] += dp.ActiveEnergyBurned.Kcal
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	today := now.Format("2006-01-02")
	yesterday := startOfYesterday.Format("2006-01-02")

	for day, kcal := range dailyKcal {
		dayTime, err := time.Parse("2006-01-02", day)
		if err != nil {
			return fmt.Errorf("parse day %q: %w", day, err)
		}

		if day != today && day != yesterday {
			exists, err := s.q.ExistsActiveEnergyForDate(ctx, db.ExistsActiveEnergyForDateParams{
				UserID: db.ToUUID(userID),
				Day:    db.ToDate(dayTime),
			})
			if err != nil {
				return fmt.Errorf("check active energy entry for date: %w", err)
			}
			if exists {
				continue
			}
		}

		kcalNumeric, err := db.ToNumeric(kcal)
		if err != nil {
			return fmt.Errorf("convert active energy kcal: %w", err)
		}

		if _, err := s.q.UpsertActiveEnergyByDay(ctx, db.UpsertActiveEnergyByDayParams{
			UserID:           db.ToUUID(userID),
			Day:              db.ToDate(dayTime),
			ActiveEnergyKcal: kcalNumeric,
		}); err != nil {
			return fmt.Errorf("upsert active energy entry: %w", err)
		}
	}

	return nil
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
