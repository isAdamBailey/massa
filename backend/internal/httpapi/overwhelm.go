package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/isAdamBailey/massa/backend/internal/overwhelm"
)

// dayLayout is the date-only format used by day-keyed request and response
// bodies.
const dayLayout = "2006-01-02"

// OverwhelmService is the subset of overwhelm.Service used by the API.
type OverwhelmService interface {
	List(ctx context.Context, userID uuid.UUID, from, to *time.Time) ([]overwhelm.Entry, error)
	Upsert(ctx context.Context, userID uuid.UUID, day time.Time, level int) (overwhelm.Entry, error)
}

// overwhelmEntryResponse is the JSON representation of an overwhelm entry.
type overwhelmEntryResponse struct {
	Day            string `json:"day"`
	OverwhelmLevel int    `json:"overwhelmLevel"`
}

func toOverwhelmEntryResponse(e overwhelm.Entry) overwhelmEntryResponse {
	return overwhelmEntryResponse{
		Day:            e.Day.Format(dayLayout),
		OverwhelmLevel: e.OverwhelmLevel,
	}
}

// overwhelmEntryRequest is the JSON request body for PUT /api/overwhelm.
type overwhelmEntryRequest struct {
	Day            string `json:"day"`
	OverwhelmLevel int    `json:"overwhelmLevel"`
}

func (req overwhelmEntryRequest) validate() string {
	if _, err := time.Parse(dayLayout, req.Day); err != nil {
		return "day must be a date in YYYY-MM-DD format"
	}
	if req.OverwhelmLevel < 1 || req.OverwhelmLevel > 10 {
		return "overwhelmLevel must be between 1 and 10"
	}
	return ""
}

// listOverwhelm returns the caller's overwhelm entries, optionally filtered
// by a day range via the from and to query parameters (RFC 3339).
func (h *Handler) listOverwhelm(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	from, err := parseOptionalTime(r.URL.Query().Get("from"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid from")
		return
	}
	to, err := parseOptionalTime(r.URL.Query().Get("to"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid to")
		return
	}

	entries, err := h.overwhelm.List(r.Context(), user.ID, from, to)
	if err != nil {
		log.Printf("httpapi: list overwhelm entries: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]overwhelmEntryResponse, len(entries))
	for i, e := range entries {
		resp[i] = toOverwhelmEntryResponse(e)
	}
	writeJSON(w, http.StatusOK, resp)
}

// upsertOverwhelm records the caller's overwhelm level for a day, replacing
// any existing entry for that day.
func (h *Handler) upsertOverwhelm(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req overwhelmEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if msg := req.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	// Safe: validate rejects any day that does not parse.
	day, _ := time.Parse(dayLayout, req.Day)

	entry, err := h.overwhelm.Upsert(r.Context(), user.ID, day, req.OverwhelmLevel)
	if err != nil {
		log.Printf("httpapi: upsert overwhelm entry: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toOverwhelmEntryResponse(entry))
}
