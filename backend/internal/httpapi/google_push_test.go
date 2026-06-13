package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isAdamBailey/massa/backend/internal/googlehealth"
)

func connectGoogle(t *testing.T, env *googleTestEnv) {
	t.Helper()

	user, err := env.users.GetByEmail(t.Context(), allowedEmail)
	require.NoError(t, err)
	require.NoError(t, env.credentials.Save(t.Context(), user.ID, googlehealth.Credentials{
		HealthUserID: testHealthUserID,
		RefreshToken: "refresh-token",
	}))
}

func TestCreateWeight_PushesToGoogleHealth(t *testing.T) {
	env := newGoogleTestEnv(t)
	sessionCookie, csrfCookie := login(t, env.router, env.mailer, allowedEmail)
	connectGoogle(t, env)

	headers := map[string]string{"X-CSRF-Token": csrfCookie.Value}
	rec := doRequest(t, env.router, http.MethodPost, "/api/weights", `{"weightKg":70,"recordedAt":"2024-01-01T08:00:00Z"}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), `"googleSyncStatus":"synced"`)

	upserts, _ := env.pushLog.snapshot()
	require.Len(t, upserts, 1)
	require.NotNil(t, upserts[0].Weight)
	assert.InDelta(t, 70000, upserts[0].Weight.WeightGrams, 0.001)
}

func TestCreateWeight_NotConnected_NoSync(t *testing.T) {
	env := newGoogleTestEnv(t)
	sessionCookie, csrfCookie := login(t, env.router, env.mailer, allowedEmail)

	headers := map[string]string{"X-CSRF-Token": csrfCookie.Value}
	rec := doRequest(t, env.router, http.MethodPost, "/api/weights", `{"weightKg":70,"recordedAt":"2024-01-01T08:00:00Z"}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.NotContains(t, rec.Body.String(), `"googleSyncStatus"`)

	upserts, _ := env.pushLog.snapshot()
	assert.Empty(t, upserts)
}

func TestUpdateWeight_PushesToGoogleHealth(t *testing.T) {
	env := newGoogleTestEnv(t)
	sessionCookie, csrfCookie := login(t, env.router, env.mailer, allowedEmail)
	connectGoogle(t, env)
	headers := map[string]string{"X-CSRF-Token": csrfCookie.Value}

	rec := doRequest(t, env.router, http.MethodPost, "/api/weights", `{"weightKg":70,"recordedAt":"2024-01-01T08:00:00Z"}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusCreated, rec.Code)

	var created struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	rec = doRequest(t, env.router, http.MethodPatch, "/api/weights/"+created.ID, `{"weightKg":72,"recordedAt":"2024-01-02T08:00:00Z"}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"googleSyncStatus":"synced"`)

	upserts, _ := env.pushLog.snapshot()
	require.Len(t, upserts, 2)
	require.NotNil(t, upserts[1].Weight)
	assert.InDelta(t, 72000, upserts[1].Weight.WeightGrams, 0.001)
}

func TestDeleteWeight_DeletesFromGoogleHealth(t *testing.T) {
	env := newGoogleTestEnv(t)
	sessionCookie, csrfCookie := login(t, env.router, env.mailer, allowedEmail)
	connectGoogle(t, env)
	headers := map[string]string{"X-CSRF-Token": csrfCookie.Value}

	rec := doRequest(t, env.router, http.MethodPost, "/api/weights", `{"weightKg":70,"recordedAt":"2024-01-01T08:00:00Z"}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusCreated, rec.Code)

	var created struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	rec = doRequest(t, env.router, http.MethodDelete, "/api/weights/"+created.ID, "", []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusOK, rec.Code)

	_, deleted := env.pushLog.snapshot()
	require.Len(t, deleted, 1)
	assert.Equal(t, created.ID, deleted[0])
}
