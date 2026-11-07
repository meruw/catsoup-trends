package server

import (
	"github.com/meruw/catsoup-trends/internal/research"
	"net/http"
	"regexp"
	"time"
)

var topicRegion = regexp.MustCompile(`^[A-Z]{2}$`)

func (s *Server) topics(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	if region == "" {
		region = "MX"
	}
	if !topicRegion.MatchString(region) {
		fail(w, &research.Error{Code: "invalid_region", Message: "Choose a two-letter country code", Status: 400})
		return
	}
	s.mu.Lock()
	hit, ok := s.topicsCache[region]
	s.mu.Unlock()
	if ok && time.Since(hit.ObservedAt) < 10*time.Minute {
		hit.Cached = true
		reply(w, 200, hit)
		return
	}
	if !s.acquire(w) {
		return
	}
	defer func() { <-s.slots }()
	page, err := s.Trends.Topics(r.Context(), region)
	if err != nil {
		if ok && time.Since(hit.ObservedAt) < 24*time.Hour && r.Context().Err() == nil {
			hit.Cached = true
			hit.Stale = true
			reply(w, 200, hit)
			return
		}
		fail(w, err)
		return
	}
	s.mu.Lock()
	if len(s.topicsCache) >= 16 {
		s.topicsCache = map[string]research.TopicsPage{}
	}
	s.topicsCache[region] = page
	s.mu.Unlock()
	reply(w, 200, page)
}
