package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/yourname/click-counter/internal/storage"
)

func minuteUTC(t time.Time) time.Time {
	return t.UTC().Truncate(time.Minute)
}

func parseRFC3339(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, errors.New("empty time")
	}
	return time.Parse(time.RFC3339, s)
}

type Handlers struct {
	repo storage.Repository
}

func NewHandlers(r storage.Repository) *Handlers {
	return &Handlers{repo: r}
}

// GET /counter/{bannerID} — увеличить счётчик кликов
func (h *Handlers) Counter(w http.ResponseWriter, r *http.Request) {
	bannerID := chi.URLParam(r, "bannerID")
	if bannerID == "" {
		http.Error(w, "missing bannerID", http.StatusBadRequest)
		return
	}

	bucket := minuteUTC(time.Now())

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.repo.Inc(ctx, bannerID, bucket, 1); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// POST /stats/{bannerID} — статистика по одному баннеру
func (h *Handlers) Stats(w http.ResponseWriter, r *http.Request) {
	bannerID := chi.URLParam(r, "bannerID")
	if bannerID == "" {
		http.Error(w, "missing bannerID", http.StatusBadRequest)
		return
	}

	var req StatsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	from, err := parseRFC3339(req.From)
	if err != nil {
		http.Error(w, "invalid from", 400)
		return
	}
	to, err := parseRFC3339(req.To)
	if err != nil {
		http.Error(w, "invalid to", 400)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	items, err := h.repo.Range(ctx, bannerID, from, to)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	resp := StatsResponse{Stats: make([]StatPoint, 0, len(items))}
	for _, it := range items {
		resp.Stats = append(resp.Stats, StatPoint{TS: it.TS, V: it.Count})
	}

	// w.Header().Set("Content-Type", "application/json")
	// _ = json.NewEncoder(w).Encode(resp)
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(resp)
}

// POST /stats — общая статистика по всем баннерам
func (h *Handlers) StatsAll(w http.ResponseWriter, r *http.Request) {
	var req StatsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	from, err := parseRFC3339(req.From)
	if err != nil {
		http.Error(w, "invalid from", 400)
		return
	}
	to, err := parseRFC3339(req.To)
	if err != nil {
		http.Error(w, "invalid to", 400)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	stats, err := h.repo.RangeAll(ctx, from, to)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	type Resp struct {
		BannerID string    `json:"bannerId"`
		TS       time.Time `json:"ts"`
		V        int64     `json:"v"`
	}

	resp := struct {
		Stats []Resp `json:"stats"`
	}{Stats: make([]Resp, 0, len(stats))}

	for _, s := range stats {
		resp.Stats = append(resp.Stats, Resp{
			BannerID: s.BannerID,
			TS:       s.TS,
			V:        s.Count,
		})
	}

	// w.Header().Set("Content-Type", "application/json")
	//_ = json.NewEncoder(w).Encode(resp)
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(resp)
}
