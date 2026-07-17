package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/isAdamBailey/massa/backend/internal/overwhelm"
)

// dayLayout is the date-only format used by day-keyed request and response
// bodies.
const dayLayout = "2006-01-02"

// OverwhelmService is the subset of overwhelm.Service used by the API.
type OverwhelmService interface {
	List(ctx context.Context, userID uuid.UUID, from, to *time.Time) ([]overwhelm.Entry, error)
	Upsert(ctx context.Context, userID uuid.UUID, day time.Time, level int, tagIDs []uuid.UUID) (overwhelm.Entry, error)
	ListTags(ctx context.Context, userID uuid.UUID) ([]overwhelm.Tag, error)
	CreateTag(ctx context.Context, userID uuid.UUID, name string) (overwhelm.Tag, error)
	RenameTag(ctx context.Context, userID, id uuid.UUID, name string) (overwhelm.Tag, error)
	ArchiveTag(ctx context.Context, userID, id uuid.UUID) error
}

// overwhelmTagResponse is the JSON representation of an overwhelm tag,
// either standalone (GET/POST/PATCH /api/overwhelm/tags) or attached to an
// entry.
type overwhelmTagResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toOverwhelmTagResponse(t overwhelm.Tag) overwhelmTagResponse {
	return overwhelmTagResponse{ID: t.ID.String(), Name: t.Name}
}

// overwhelmEntryResponse is the JSON representation of an overwhelm entry.
type overwhelmEntryResponse struct {
	Day            string                 `json:"day"`
	OverwhelmLevel int                    `json:"overwhelmLevel"`
	Tags           []overwhelmTagResponse `json:"tags"`
}

func toOverwhelmEntryResponse(e overwhelm.Entry) overwhelmEntryResponse {
	tags := make([]overwhelmTagResponse, len(e.Tags))
	for i, t := range e.Tags {
		tags[i] = overwhelmTagResponse{ID: t.ID.String(), Name: t.Name}
	}
	return overwhelmEntryResponse{
		Day:            e.Day.Format(dayLayout),
		OverwhelmLevel: e.OverwhelmLevel,
		Tags:           tags,
	}
}

// overwhelmEntryRequest is the JSON request body for PUT /api/overwhelm.
type overwhelmEntryRequest struct {
	Day            string   `json:"day"`
	OverwhelmLevel int      `json:"overwhelmLevel"`
	TagIDs         []string `json:"tagIds"`
}

func (req overwhelmEntryRequest) validate() string {
	if _, err := time.Parse(dayLayout, req.Day); err != nil {
		return "day must be a date in YYYY-MM-DD format"
	}
	if req.OverwhelmLevel < 1 || req.OverwhelmLevel > 10 {
		return "overwhelmLevel must be between 1 and 10"
	}
	for _, id := range req.TagIDs {
		if _, err := uuid.Parse(id); err != nil {
			return "tagIds must be valid ids"
		}
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

// upsertOverwhelm records the caller's overwhelm level and tags for a day,
// replacing any existing entry for that day.
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

	// Safe: validate rejects any day or tag id that does not parse.
	day, _ := time.Parse(dayLayout, req.Day)
	tagIDs := make([]uuid.UUID, len(req.TagIDs))
	for i, id := range req.TagIDs {
		tagIDs[i], _ = uuid.Parse(id)
	}

	entry, err := h.overwhelm.Upsert(r.Context(), user.ID, day, req.OverwhelmLevel, tagIDs)
	if err != nil {
		log.Printf("httpapi: upsert overwhelm entry: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toOverwhelmEntryResponse(entry))
}

// overwhelmTagRequest is the JSON request body for creating or renaming an
// overwhelm tag.
type overwhelmTagRequest struct {
	Name string `json:"name"`
}

func (req overwhelmTagRequest) validate() string {
	name := strings.TrimSpace(req.Name)
	if len(name) < 1 || len(name) > 30 {
		return "name must be between 1 and 30 characters"
	}
	return ""
}

// listOverwhelmTags returns the caller's active overwhelm tags.
func (h *Handler) listOverwhelmTags(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	tags, err := h.overwhelm.ListTags(r.Context(), user.ID)
	if err != nil {
		log.Printf("httpapi: list overwhelm tags: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	resp := make([]overwhelmTagResponse, len(tags))
	for i, t := range tags {
		resp[i] = toOverwhelmTagResponse(t)
	}
	writeJSON(w, http.StatusOK, resp)
}

// createOverwhelmTag creates a new overwhelm tag for the caller, or
// unarchives a previously archived tag with the same name.
func (h *Handler) createOverwhelmTag(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req overwhelmTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if msg := req.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	tag, err := h.overwhelm.CreateTag(r.Context(), user.ID, strings.TrimSpace(req.Name))
	if err != nil {
		log.Printf("httpapi: create overwhelm tag: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, toOverwhelmTagResponse(tag))
}

// renameOverwhelmTag renames one of the caller's overwhelm tags.
func (h *Handler) renameOverwhelmTag(w http.ResponseWriter, r *http.Request) {
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

	var req overwhelmTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if msg := req.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	tag, err := h.overwhelm.RenameTag(r.Context(), user.ID, id, strings.TrimSpace(req.Name))
	if errors.Is(err, overwhelm.ErrNotFound) {
		writeError(w, http.StatusNotFound, "overwhelm tag not found")
		return
	}
	if errors.Is(err, overwhelm.ErrDuplicateTag) {
		writeError(w, http.StatusConflict, "a tag with that name already exists")
		return
	}
	if err != nil {
		log.Printf("httpapi: rename overwhelm tag: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, toOverwhelmTagResponse(tag))
}

// archiveOverwhelmTag archives one of the caller's overwhelm tags, removing
// it from the picker without deleting its history.
func (h *Handler) archiveOverwhelmTag(w http.ResponseWriter, r *http.Request) {
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

	err = h.overwhelm.ArchiveTag(r.Context(), user.ID, id)
	if errors.Is(err, overwhelm.ErrNotFound) {
		writeError(w, http.StatusNotFound, "overwhelm tag not found")
		return
	}
	if err != nil {
		log.Printf("httpapi: archive overwhelm tag: %v", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusOK)
}
