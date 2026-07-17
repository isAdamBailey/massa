package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListOverwhelm_RequiresAuth(t *testing.T) {
	r, _, _, _ := newTestRouter(allowedEmail)

	rec := doRequest(t, r, http.MethodGet, "/api/overwhelm", "", nil, nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestListOverwhelm_DateRangeFilter(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, csrfCookie := login(t, r, m, allowedEmail)
	headers := map[string]string{"X-CSRF-Token": csrfCookie.Value}

	rec := doRequest(t, r, http.MethodPut, "/api/overwhelm", `{"day":"2024-01-01","overwhelmLevel":5}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doRequest(t, r, http.MethodPut, "/api/overwhelm", `{"day":"2024-06-01","overwhelmLevel":7}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doRequest(t, r, http.MethodGet, "/api/overwhelm?from=2024-03-01T00:00:00Z", "", []*http.Cookie{sessionCookie}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"day":"2024-06-01"`)
	assert.NotContains(t, rec.Body.String(), `"day":"2024-01-01"`)
}

func TestUpsertOverwhelm_RequiresAuth(t *testing.T) {
	r, _, _, _ := newTestRouter(allowedEmail)

	rec := doRequest(t, r, http.MethodPut, "/api/overwhelm", `{"day":"2024-01-01","overwhelmLevel":5}`, nil, nil)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUpsertOverwhelm_RequiresCSRF(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, _ := login(t, r, m, allowedEmail)

	rec := doRequest(t, r, http.MethodPut, "/api/overwhelm", `{"day":"2024-01-01","overwhelmLevel":5}`, []*http.Cookie{sessionCookie}, nil)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestUpsertOverwhelm_InvalidBody(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, csrfCookie := login(t, r, m, allowedEmail)
	headers := map[string]string{"X-CSRF-Token": csrfCookie.Value}

	rec := doRequest(t, r, http.MethodPut, "/api/overwhelm", `{`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpsertOverwhelm_InvalidLevel(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, csrfCookie := login(t, r, m, allowedEmail)
	headers := map[string]string{"X-CSRF-Token": csrfCookie.Value}

	rec := doRequest(t, r, http.MethodPut, "/api/overwhelm", `{"day":"2024-01-01","overwhelmLevel":0}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = doRequest(t, r, http.MethodPut, "/api/overwhelm", `{"day":"2024-01-01","overwhelmLevel":11}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpsertOverwhelm_InvalidDay(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, csrfCookie := login(t, r, m, allowedEmail)
	headers := map[string]string{"X-CSRF-Token": csrfCookie.Value}

	rec := doRequest(t, r, http.MethodPut, "/api/overwhelm", `{"day":"not-a-date","overwhelmLevel":5}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = doRequest(t, r, http.MethodPut, "/api/overwhelm", `{"day":"2024-13-99","overwhelmLevel":5}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpsertOverwhelm(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, csrfCookie := login(t, r, m, allowedEmail)
	headers := map[string]string{"X-CSRF-Token": csrfCookie.Value}

	rec := doRequest(t, r, http.MethodPut, "/api/overwhelm", `{"day":"2026-07-16","overwhelmLevel":5}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"day":"2026-07-16"`)
	assert.Contains(t, rec.Body.String(), `"overwhelmLevel":5`)
}

func TestUpsertOverwhelm_Idempotent(t *testing.T) {
	r, _, _, m := newTestRouter(allowedEmail)

	sessionCookie, csrfCookie := login(t, r, m, allowedEmail)
	headers := map[string]string{"X-CSRF-Token": csrfCookie.Value}

	rec := doRequest(t, r, http.MethodPut, "/api/overwhelm", `{"day":"2024-01-01","overwhelmLevel":5}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doRequest(t, r, http.MethodPut, "/api/overwhelm", `{"day":"2024-01-01","overwhelmLevel":8}`, []*http.Cookie{sessionCookie, csrfCookie}, headers)
	require.Equal(t, http.StatusOK, rec.Code)

	rec = doRequest(t, r, http.MethodGet, "/api/overwhelm", "", []*http.Cookie{sessionCookie}, nil)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"overwhelmLevel":8`)
	assert.NotContains(t, rec.Body.String(), `"overwhelmLevel":5`)

	var entries []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &entries))
	assert.Len(t, entries, 1)
}
