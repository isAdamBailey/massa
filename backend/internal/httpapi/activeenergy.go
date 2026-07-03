package httpapi

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/isAdamBailey/massa/backend/internal/activeenergy"
)

// ActiveEnergyService is the subset of activeenergy.Service used by the API.
type ActiveEnergyService interface {
	List(ctx context.Context, userID uuid.UUID, from, to *time.Time) ([]activeenergy.Entry, error)
}

// activeEnergyEntryResponse is the JSON representation of an active energy
// entry.
type activeEnergyEntryResponse struct {
	Day              string  `json:"day"`
	ActiveEnergyKcal float64 `json:"activeEnergyKcal"`
}

func toActiveEnergyEntryResponse(e activeenergy.Entry) activeEnergyEntryResponse {
	return activeEnergyEntryResponse{
		Day:              e.Day.Format("2006-01-02"),
		ActiveEnergyKcal: e.ActiveEnergyKcal,
	}
}

// listActiveEnergy returns the caller's active energy entries, optionally
// filtered by a day range via the from and to query parameters (RFC 3339).
func (h *Handler) listActiveEnergy(w http.ResponseWriter, r *http.Request) {
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

	entries, err := h.activeEnergy.List(r.Context(), user.ID, from, to)
	if err != nil {
		log.Printf("httpapi: list active energy entries: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]activeEnergyEntryResponse, len(entries))
	for i, e := range entries {
		resp[i] = toActiveEnergyEntryResponse(e)
	}
	writeJSON(w, http.StatusOK, resp)
}
