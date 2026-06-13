package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/isAdamBailey/massa/backend/internal/googlehealth"
	"github.com/isAdamBailey/massa/backend/internal/weights"
)

// WeightsService is the subset of weights.Service used by the API.
type WeightsService interface {
	List(ctx context.Context, userID uuid.UUID, from, to *time.Time) ([]weights.Entry, error)
	Create(ctx context.Context, userID uuid.UUID, weightKg float64, recordedAt time.Time) (weights.Entry, error)
	Update(ctx context.Context, userID, id uuid.UUID, weightKg float64, recordedAt time.Time) (weights.Entry, error)
	UpdateGoogleSync(ctx context.Context, userID, id uuid.UUID, dataPointID *string, status string) (weights.Entry, error)
	Delete(ctx context.Context, userID, id uuid.UUID) error
	Get(ctx context.Context, userID, id uuid.UUID) (weights.Entry, error)
	Latest(ctx context.Context, userID uuid.UUID) (weights.Entry, error)
}

// weightEntryResponse is the JSON representation of a weight entry.
type weightEntryResponse struct {
	ID               string    `json:"id"`
	WeightKg         float64   `json:"weightKg"`
	RecordedAt       time.Time `json:"recordedAt"`
	BMI              *float64  `json:"bmi,omitempty"`
	HeightUsedCm     *float64  `json:"heightUsedCm,omitempty"`
	Source           string    `json:"source"`
	GoogleSyncStatus *string   `json:"googleSyncStatus,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func toWeightEntryResponse(e weights.Entry) weightEntryResponse {
	return weightEntryResponse{
		ID:               e.ID.String(),
		WeightKg:         e.WeightKg,
		RecordedAt:       e.RecordedAt,
		BMI:              e.BMI,
		HeightUsedCm:     e.HeightUsedCm,
		Source:           e.Source,
		GoogleSyncStatus: e.GoogleSyncStatus,
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}
}

// weightEntryRequest is the JSON request body for creating or updating a
// weight entry.
type weightEntryRequest struct {
	WeightKg   float64   `json:"weightKg"`
	RecordedAt time.Time `json:"recordedAt"`
}

func (req weightEntryRequest) validate() string {
	if req.WeightKg <= 0 {
		return "weightKg must be positive"
	}
	if req.RecordedAt.IsZero() {
		return "recordedAt is required"
	}
	return ""
}

// listWeights returns the caller's weight entries, optionally filtered by a
// recorded_at date range via the from and to query parameters (RFC 3339).
func (h *Handler) listWeights(w http.ResponseWriter, r *http.Request) {
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

	entries, err := h.weights.List(r.Context(), user.ID, from, to)
	if err != nil {
		log.Printf("httpapi: list weight entries: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]weightEntryResponse, len(entries))
	for i, e := range entries {
		resp[i] = toWeightEntryResponse(e)
	}
	writeJSON(w, http.StatusOK, resp)
}

// createWeight records a new manual weight entry for the caller.
func (h *Handler) createWeight(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req weightEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if msg := req.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	entry, err := h.weights.Create(r.Context(), user.ID, req.WeightKg, req.RecordedAt)
	if err != nil {
		log.Printf("httpapi: create weight entry: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	entry = h.pushToGoogle(r.Context(), user.ID, entry)
	writeJSON(w, http.StatusCreated, toWeightEntryResponse(entry))
}

// getWeight returns a single weight entry belonging to the caller.
func (h *Handler) getWeight(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	entry, err := h.weights.Get(r.Context(), user.ID, id)
	if errors.Is(err, weights.ErrNotFound) {
		writeError(w, http.StatusNotFound, "weight entry not found")
		return
	}
	if err != nil {
		log.Printf("httpapi: get weight entry: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toWeightEntryResponse(entry))
}

// updateWeight changes the weight and/or recorded time of an existing entry.
func (h *Handler) updateWeight(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req weightEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if msg := req.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	entry, err := h.weights.Update(r.Context(), user.ID, id, req.WeightKg, req.RecordedAt)
	if errors.Is(err, weights.ErrNotFound) {
		writeError(w, http.StatusNotFound, "weight entry not found")
		return
	}
	if err != nil {
		log.Printf("httpapi: update weight entry: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	entry = h.pushToGoogle(r.Context(), user.ID, entry)
	writeJSON(w, http.StatusOK, toWeightEntryResponse(entry))
}

// deleteWeight removes a weight entry belonging to the caller.
func (h *Handler) deleteWeight(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	entry, err := h.weights.Get(r.Context(), user.ID, id)
	if errors.Is(err, weights.ErrNotFound) {
		writeError(w, http.StatusNotFound, "weight entry not found")
		return
	}
	if err != nil {
		log.Printf("httpapi: get weight entry: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if h.google != nil && h.google.Push != nil && entry.Source == "manual" && entry.GoogleDataPointID != nil &&
		entry.GoogleSyncStatus != nil && *entry.GoogleSyncStatus == "synced" {
		if pushErr := h.google.Push.DeleteWeight(r.Context(), user.ID, *entry.GoogleDataPointID); pushErr != nil && !errors.Is(pushErr, googlehealth.ErrNotConnected) {
			log.Printf("httpapi: delete weight entry from google health: %v", pushErr)
		}
	}

	err = h.weights.Delete(r.Context(), user.ID, id)
	if errors.Is(err, weights.ErrNotFound) {
		writeError(w, http.StatusNotFound, "weight entry not found")
		return
	}
	if err != nil {
		log.Printf("httpapi: delete weight entry: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// bmiLatestResponse is the JSON body returned by GET /api/bmi/latest.
type bmiLatestResponse struct {
	BMI          *float64  `json:"bmi"`
	WeightKg     float64   `json:"weightKg"`
	HeightUsedCm *float64  `json:"heightUsedCm"`
	RecordedAt   time.Time `json:"recordedAt"`
}

// bmiLatest returns the BMI computed from the caller's most recent weight
// entry.
func (h *Handler) bmiLatest(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	entry, err := h.weights.Latest(r.Context(), user.ID)
	if errors.Is(err, weights.ErrNotFound) {
		writeError(w, http.StatusNotFound, "no weight entries")
		return
	}
	if err != nil {
		log.Printf("httpapi: get latest weight entry: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, bmiLatestResponse{
		BMI:          entry.BMI,
		WeightKg:     entry.WeightKg,
		HeightUsedCm: entry.HeightUsedCm,
		RecordedAt:   entry.RecordedAt,
	})
}

// pushToGoogle pushes a manual weight entry to the caller's Google Health
// account, best-effort: if Google Health is not configured, the entry isn't
// a manual entry, or the user hasn't connected an account, entry is returned
// unchanged. Otherwise the push result is recorded on the entry via
// h.weights.UpdateGoogleSync.
func (h *Handler) pushToGoogle(ctx context.Context, userID uuid.UUID, entry weights.Entry) weights.Entry {
	if h.google == nil || h.google.Push == nil || entry.Source != "manual" {
		return entry
	}

	dataPointID := entry.ID.String()
	if entry.GoogleDataPointID != nil {
		dataPointID = *entry.GoogleDataPointID
	}
	create := entry.GoogleSyncStatus == nil || *entry.GoogleSyncStatus != "synced"

	status := "synced"
	err := h.google.Push.PushWeight(ctx, userID, dataPointID, entry.WeightKg, entry.RecordedAt, create)
	if errors.Is(err, googlehealth.ErrNotConnected) {
		return entry
	}
	if err != nil {
		log.Printf("httpapi: push weight entry to google health: %v", err)
		status = "failed"
	}

	updated, err := h.weights.UpdateGoogleSync(ctx, userID, entry.ID, &dataPointID, status)
	if err != nil {
		log.Printf("httpapi: record google sync status: %v", err)
		return entry
	}
	return updated
}

// parseOptionalTime parses an RFC 3339 timestamp, returning nil if s is
// empty.
func parseOptionalTime(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
