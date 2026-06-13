package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
)

// settingsResponse is the JSON representation of a user's settings.
type settingsResponse struct {
	ManualHeightCm  *float64 `json:"manualHeightCm,omitempty"`
	UnitsPreference string   `json:"unitsPreference"`
}

// updateSettingsRequest is the JSON request body for PUT /api/settings.
type updateSettingsRequest struct {
	ManualHeightCm  *float64 `json:"manualHeightCm"`
	UnitsPreference string   `json:"unitsPreference"`
}

func (req updateSettingsRequest) validate() string {
	if req.UnitsPreference != "metric" && req.UnitsPreference != "imperial" {
		return "unitsPreference must be 'metric' or 'imperial'"
	}
	if req.ManualHeightCm != nil && *req.ManualHeightCm <= 0 {
		return "manualHeightCm must be positive"
	}
	return ""
}

// getSettings returns the caller's manual height override and units
// preference.
func (h *Handler) getSettings(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	writeJSON(w, http.StatusOK, settingsResponse{
		ManualHeightCm:  user.ManualHeightCm,
		UnitsPreference: user.UnitsPreference,
	})
}

// updateSettings sets the caller's manual height override and units
// preference.
func (h *Handler) updateSettings(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req updateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if msg := req.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	updated, err := h.users.UpdateSettings(r.Context(), user.ID, req.ManualHeightCm, req.UnitsPreference)
	if err != nil {
		log.Printf("httpapi: update user settings: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, settingsResponse{
		ManualHeightCm:  updated.ManualHeightCm,
		UnitsPreference: updated.UnitsPreference,
	})
}
