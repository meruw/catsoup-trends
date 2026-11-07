package server

import (
	"github.com/meruw/catsoup-trends/internal/research"
	"net/http"
	"time"
)

func (s *Server) discover(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	region, category, token := v.Get("region"), v.Get("category"), v.Get("page")
	if region == "" {
		region = "MX"
	}
	if category == "" {
		category = "20"
	}
	if err := research.ValidateDiscovery(region, category, token); err != nil {
		fail(w, err)
		return
	}
	key, gen := s.credentials()
	if key == "" {
		fail(w, &research.Error{Code: "key_required", Message: "Add a YouTube API key in Settings", Status: 409})
		return
	}
	// Empty Text cannot be a search key. Both features share the same bounded cache.
	q := research.Query{Order: "mostPopular", Language: region, Duration: category, Page: token}
	s.mu.Lock()
	hit, ok := s.cache[q]
	s.mu.Unlock()
	if ok && time.Since(hit.At) < 10*time.Minute {
		s.searchReply(w, r, hit.Page, true)
		return
	}
	if !s.acquire(w) {
		return
	}
	defer func() { <-s.slots }()
	page, err := s.YouTube.Popular(r.Context(), key, region, category, token)
	if err != nil {
		fail(w, err)
		return
	}
	s.mu.Lock()
	if gen != s.generation {
		s.mu.Unlock()
		fail(w, &research.Error{Code: "settings_changed", Message: "Settings changed. Load discovery again", Status: 409})
		return
	}
	s.verified = true
	if len(s.cache) >= 20 {
		s.cache = map[research.Query]cached{}
	}
	s.cache[q] = cached{page, time.Now()}
	s.mu.Unlock()
	s.searchReply(w, r, page, false)
}
