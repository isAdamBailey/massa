package googlehealth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// PushService pushes manually-recorded weight entries to a user's Google
// Health account.
type PushService struct {
	credentials CredentialsRepository
	oauthConfig *oauth2.Config
	apiBaseURL  string
}

// NewPushService returns a PushService that authenticates using oauthConfig.
func NewPushService(credentials CredentialsRepository, oauthConfig *oauth2.Config) *PushService {
	return &PushService{credentials: credentials, oauthConfig: oauthConfig, apiBaseURL: baseURL}
}

// NewPushServiceForTest returns a PushService that sends Google Health API
// requests to apiBaseURL instead of the real API, for use against an
// httptest.Server.
func NewPushServiceForTest(credentials CredentialsRepository, oauthConfig *oauth2.Config, apiBaseURL string) *PushService {
	return &PushService{credentials: credentials, oauthConfig: oauthConfig, apiBaseURL: apiBaseURL}
}

// PushWeight creates or updates the Google Health weight data point
// identified by dataPointID for userID. Returns ErrNotConnected if userID
// has not connected a Google Health account.
func (s *PushService) PushWeight(ctx context.Context, userID uuid.UUID, dataPointID string, weightKg float64, recordedAt time.Time) error {
	client, persist, err := newAuthorizedClient(ctx, s.credentials, s.oauthConfig, userID, s.apiBaseURL)
	if err != nil {
		return err
	}

	if _, err := client.UpsertWeightDataPoint(ctx, "me", dataPointID, weightKg*1000, recordedAt); err != nil {
		return fmt.Errorf("upsert weight data point: %w", err)
	}

	return persist(ctx)
}

// DeleteWeight removes the Google Health weight data point identified by
// dataPointID for userID. Returns ErrNotConnected if userID has not
// connected a Google Health account.
func (s *PushService) DeleteWeight(ctx context.Context, userID uuid.UUID, dataPointID string) error {
	client, persist, err := newAuthorizedClient(ctx, s.credentials, s.oauthConfig, userID, s.apiBaseURL)
	if err != nil {
		return err
	}

	if err := client.DeleteWeightDataPoint(ctx, "me", dataPointID); err != nil {
		return fmt.Errorf("delete weight data point: %w", err)
	}

	return persist(ctx)
}
