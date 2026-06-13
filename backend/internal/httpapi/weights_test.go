package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListWeights_RequiresAuth(t *testing.T) {
	r, _, _, _ := newTestRouter(allowedEmail)

	rec := doRequest(t, r, http.MethodGet, "/api/weights", "", nil, nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestCreateWeight_RequiresCSRF(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, _ := login(t, r, m, allowedEmail)

	rec := doRequest(t, r, http.MethodPost, "/api/weights", `{"weightKg":70,"recordedAt":"2024-01-01T08:00:00Z"}`, []*http.Cookie{sessionCookie}, nil)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestCreateWeight_InvalidBody(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, csrfCookie := login(t, r, m, allowedEmail)
	headers := map[string]string{"X-CSRF-Token": csrfCookie.Value}

	rec := doRequest(t, r, http.MethodPost, "/api/weights", `{"weightKg":0,"recordedAt":"2024-01-01T08:00:00Z"}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = doRequest(t, r, http.MethodPost, "/api/weights", `{"weightKg":70}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateAndListWeights(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, csrfCookie := login(t, r, m, allowedEmail)
	headers := map[string]string{"X-CSRF-Token": csrfCookie.Value}

	rec := doRequest(t, r, http.MethodPost, "/api/weights", `{"weightKg":70,"recordedAt":"2024-01-01T08:00:00Z"}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), `"weightKg":70`)
	assert.Contains(t, rec.Body.String(), `"bmi":`)

	rec = doRequest(t, r, http.MethodGet, "/api/weights", "", []*http.Cookie{sessionCookie}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"weightKg":70`)
}

func TestListWeights_DateRangeFilter(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, csrfCookie := login(t, r, m, allowedEmail)
	headers := map[string]string{"X-CSRF-Token": csrfCookie.Value}

	rec := doRequest(t, r, http.MethodPost, "/api/weights", `{"weightKg":70,"recordedAt":"2024-01-01T08:00:00Z"}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusCreated, rec.Code)

	rec = doRequest(t, r, http.MethodPost, "/api/weights", `{"weightKg":75,"recordedAt":"2024-06-01T08:00:00Z"}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusCreated, rec.Code)

	rec = doRequest(t, r, http.MethodGet, "/api/weights?from=2024-05-01T00:00:00Z&to=2024-07-01T00:00:00Z", "", []*http.Cookie{sessionCookie}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"weightKg":75`)
	assert.NotContains(t, rec.Body.String(), `"weightKg":70`)
}

func TestListWeights_InvalidDateRange(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, _ := login(t, r, m, allowedEmail)

	rec := doRequest(t, r, http.MethodGet, "/api/weights?from=not-a-date", "", []*http.Cookie{sessionCookie}, nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetUpdateDeleteWeight(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, csrfCookie := login(t, r, m, allowedEmail)
	headers := map[string]string{"X-CSRF-Token": csrfCookie.Value}

	rec := doRequest(t, r, http.MethodPost, "/api/weights", `{"weightKg":70,"recordedAt":"2024-01-01T08:00:00Z"}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusCreated, rec.Code)

	var created struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	rec = doRequest(t, r, http.MethodGet, "/api/weights/"+created.ID, "", []*http.Cookie{sessionCookie}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"weightKg":70`)

	rec = doRequest(t, r, http.MethodPatch, "/api/weights/"+created.ID, `{"weightKg":72,"recordedAt":"2024-01-02T08:00:00Z"}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"weightKg":72`)

	rec = doRequest(t, r, http.MethodDelete, "/api/weights/"+created.ID, "", []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doRequest(t, r, http.MethodGet, "/api/weights/"+created.ID, "", []*http.Cookie{sessionCookie}, nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetWeight_InvalidID(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, _ := login(t, r, m, allowedEmail)

	rec := doRequest(t, r, http.MethodGet, "/api/weights/not-a-uuid", "", []*http.Cookie{sessionCookie}, nil)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestBMILatest(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, csrfCookie := login(t, r, m, allowedEmail)
	headers := map[string]string{"X-CSRF-Token": csrfCookie.Value}

	rec := doRequest(t, r, http.MethodGet, "/api/bmi/latest", "", []*http.Cookie{sessionCookie}, nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	rec = doRequest(t, r, http.MethodPost, "/api/weights", `{"weightKg":70,"recordedAt":"2024-01-01T08:00:00Z"}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusCreated, rec.Code)

	rec = doRequest(t, r, http.MethodGet, "/api/bmi/latest", "", []*http.Cookie{sessionCookie}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"weightKg":70`)
	assert.Contains(t, rec.Body.String(), `"bmi":`)
}

func TestGetSettings_Defaults(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, _ := login(t, r, m, allowedEmail)

	rec := doRequest(t, r, http.MethodGet, "/api/settings", "", []*http.Cookie{sessionCookie}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"unitsPreference":"metric"`)
}

func TestUpdateSettings(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, csrfCookie := login(t, r, m, allowedEmail)
	headers := map[string]string{"X-CSRF-Token": csrfCookie.Value}

	rec := doRequest(t, r, http.MethodPut, "/api/settings", `{"manualHeightCm":180.5,"unitsPreference":"imperial"}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"manualHeightCm":180.5`)
	assert.Contains(t, rec.Body.String(), `"unitsPreference":"imperial"`)

	rec = doRequest(t, r, http.MethodGet, "/api/settings", "", []*http.Cookie{sessionCookie}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"manualHeightCm":180.5`)
}

func TestUpdateSettings_InvalidUnitsPreference(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, csrfCookie := login(t, r, m, allowedEmail)
	headers := map[string]string{"X-CSRF-Token": csrfCookie.Value}

	rec := doRequest(t, r, http.MethodPut, "/api/settings", `{"unitsPreference":"furlongs"}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateSettings_RequiresCSRF(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, _ := login(t, r, m, allowedEmail)

	rec := doRequest(t, r, http.MethodPut, "/api/settings", `{"unitsPreference":"metric"}`, []*http.Cookie{sessionCookie}, nil)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
