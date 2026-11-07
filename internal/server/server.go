package server

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/meruw/catsoup-trends/internal/research"
	"github.com/meruw/catsoup-trends/internal/secret"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type cached struct {
	Page research.Page
	At   time.Time
}
type Server struct {
	Store                        *research.Store
	YouTube                      *research.YouTube
	Token, Origin, Host, DataDir string
	Shutdown                     func()
	mu                           sync.Mutex
	key                          string
	envKey                       bool
	verified                     bool
	generation                   int
	cache                        map[research.Query]cached
	videos                       map[string]research.Video
	slots                        chan struct{}
	attempts                     []time.Time
}

func New(store *research.Store, dataDir string) (*Server, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	s := &Server{Store: store, YouTube: research.NewYouTube(), Token: hex.EncodeToString(b), DataDir: dataDir, cache: map[research.Query]cached{}, videos: map[string]research.Video{}, slots: make(chan struct{}, 2)}
	s.key = strings.TrimSpace(os.Getenv("TRENDS_YOUTUBE_API_KEY"))
	s.envKey = s.key != ""
	if !s.envKey {
		key, err := secret.Load(filepath.Join(dataDir, "youtube.key"))
		if err == nil {
			s.key = key
		} else if !errors.Is(err, os.ErrNotExist) && secret.Supported() {
			return nil, errors.New("cannot read the protected YouTube key; use the same Windows account or remove youtube.key")
		}
	}
	return s, nil
}
func (s *Server) Handler(assets fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/status", s.status)
	mux.HandleFunc("PUT /api/v1/settings/youtube", s.saveKey)
	mux.HandleFunc("DELETE /api/v1/settings/youtube", s.removeKey)
	mux.HandleFunc("GET /api/v1/search", s.search)
	mux.HandleFunc("GET /api/v1/discover", s.discover)
	mux.HandleFunc("GET /api/v1/ideas", s.listIdeas)
	mux.HandleFunc("POST /api/v1/ideas", s.saveIdea)
	mux.HandleFunc("PUT /api/v1/ideas/{id}", s.saveIdea)
	mux.HandleFunc("DELETE /api/v1/ideas/{id}", s.deleteIdea)
	mux.HandleFunc("POST /api/v1/ideas/{id}/export", s.exportIdea)
	mux.HandleFunc("GET /api/v1/references", s.list)
	mux.HandleFunc("POST /api/v1/references", s.save)
	mux.HandleFunc("GET /api/v1/references/{id}", s.getReference)
	mux.HandleFunc("POST /api/v1/exports/claude", s.exportClaude)
	mux.HandleFunc("PATCH /api/v1/references/{id}", s.note)
	mux.HandleFunc("DELETE /api/v1/references/{id}", s.remove)
	mux.HandleFunc("POST /api/v1/references/{id}/refresh", s.refresh)
	mux.HandleFunc("POST /api/v1/shutdown", func(w http.ResponseWriter, r *http.Request) {
		reply(w, 200, map[string]bool{"ok": true})
		if s.Shutdown != nil {
			go s.Shutdown()
		}
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		fail(w, &research.Error{Code: "not_found", Message: "Unknown API route or method", Status: 404})
	})
	mux.Handle("/", http.FileServer(http.FS(assets)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data: https://i.ytimg.com; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		if r.Host != s.Host {
			fail(w, &research.Error{Code: "host_rejected", Message: "Invalid local host", Status: 403})
			return
		}
		if o := r.Header.Get("Origin"); o != "" && o != s.Origin {
			fail(w, &research.Error{Code: "origin_rejected", Message: "This origin is not allowed", Status: 403})
			return
		}
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			fail(w, &research.Error{Code: "origin_rejected", Message: "Cross-site access is not allowed", Status: 403})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") || subtle.ConstantTimeCompare([]byte(got), []byte(s.Token)) != 1 {
				fail(w, &research.Error{Code: "unauthorized", Message: "Reopen Trends from its launcher to reconnect", Status: 401})
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}
func reply(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}
func fail(w http.ResponseWriter, err error) {
	var known *research.Error
	if errors.As(err, &known) {
		reply(w, known.Status, map[string]string{"code": known.Code, "message": known.Message})
		return
	}
	if errors.Is(err, context.Canceled) {
		reply(w, 408, map[string]string{"code": "cancelled", "message": "Request cancelled"})
		return
	}
	reply(w, 500, map[string]string{"code": "internal_error", "message": "The operation could not be completed. Try again"})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		fail(w, &research.Error{Code: "invalid_body", Message: "Use JSON for this request", Status: 415})
		return false
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32768))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		fail(w, &research.Error{Code: "invalid_body", Message: "Invalid or oversized request", Status: 400})
		return false
	}
	var tail any
	if d.Decode(&tail) != io.EOF {
		fail(w, &research.Error{Code: "invalid_body", Message: "Unexpected trailing data", Status: 400})
		return false
	}
	return true
}
func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	source := "none"
	if s.key != "" {
		source = "protected"
		if s.envKey {
			source = "environment"
		}
	}
	reply(w, 200, map[string]any{"keyConfigured": s.key != "", "keyVerified": s.verified, "keySource": source, "canStoreKey": secret.Supported() && !s.envKey, "version": "0.1.0-poc"})
}
func (s *Server) saveKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key string `json:"key"`
	}
	if !decode(w, r, &body) {
		return
	}
	body.Key = strings.TrimSpace(body.Key)
	if len(body.Key) < 20 || len(body.Key) > 128 || strings.ContainsAny(body.Key, " \r\n\t") {
		fail(w, &research.Error{Code: "invalid_key", Message: "Enter a valid Google API key, not an OAuth client secret", Status: 400})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.envKey {
		fail(w, &research.Error{Code: "environment_key", Message: "This key is supplied by TRENDS_YOUTUBE_API_KEY", Status: 409})
		return
	}
	if !secret.Supported() {
		fail(w, &research.Error{Code: "unsupported", Message: "Set TRENDS_YOUTUBE_API_KEY on this operating system", Status: 400})
		return
	}
	if err := secret.Save(filepath.Join(s.DataDir, "youtube.key"), body.Key); err != nil {
		fail(w, err)
		return
	}
	s.key = body.Key
	s.verified = false
	s.generation++
	s.cache = map[research.Query]cached{}
	s.videos = map[string]research.Video{}
	reply(w, 200, map[string]bool{"ok": true})
}
func (s *Server) removeKey(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.envKey {
		fail(w, &research.Error{Code: "environment_key", Message: "Remove TRENDS_YOUTUBE_API_KEY and restart Trends", Status: 409})
		return
	}
	err := os.Remove(filepath.Join(s.DataDir, "youtube.key"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fail(w, err)
		return
	}
	s.key = ""
	s.verified = false
	s.generation++
	s.cache = map[research.Query]cached{}
	s.videos = map[string]research.Video{}
	reply(w, 200, map[string]bool{"ok": true})
}
func (s *Server) credentials() (string, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.key, s.generation
}
func (s *Server) acquire(w http.ResponseWriter) bool {
	select {
	case s.slots <- struct{}{}:
	default:
		fail(w, &research.Error{Code: "busy", Message: "Another lookup is running. Try again shortly", Status: 429})
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-time.Minute)
	valid := s.attempts[:0]
	for _, t := range s.attempts {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	s.attempts = valid
	if len(s.attempts) >= 20 {
		<-s.slots
		fail(w, &research.Error{Code: "rate_limited", Message: "Too many lookups. Wait a minute before trying again", Status: 429})
		return false
	}
	s.attempts = append(s.attempts, time.Now())
	return true
}
func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	q := research.Query{Text: v.Get("q"), Order: v.Get("order"), Duration: v.Get("duration"), Period: v.Get("period"), Language: v.Get("language"), Page: v.Get("page")}
	if err := research.ValidateQuery(&q); err != nil {
		fail(w, err)
		return
	}
	key, gen := s.credentials()
	if key == "" {
		fail(w, &research.Error{Code: "key_required", Message: "Add a YouTube API key in Settings to search", Status: 409})
		return
	}
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
	page, err := s.YouTube.Search(r.Context(), key, q)
	if err != nil {
		fail(w, err)
		return
	}
	s.mu.Lock()
	if gen != s.generation {
		s.mu.Unlock()
		fail(w, &research.Error{Code: "settings_changed", Message: "Settings changed. Search again", Status: 409})
		return
	}
	s.verified = true
	if len(s.cache) >= 20 {
		s.cache = map[research.Query]cached{}
	}
	s.cache[q] = cached{page, time.Now()}
	if len(s.videos)+len(page.Items) > 400 {
		s.videos = map[string]research.Video{}
	}
	for _, item := range page.Items {
		s.videos[item.ID] = item
	}
	s.mu.Unlock()
	s.searchReply(w, r, page, false)
}
func (s *Server) searchReply(w http.ResponseWriter, r *http.Request, page research.Page, cache bool) {
	// A cached search must remain saveable after another search evicts candidates.
	s.mu.Lock()
	if len(s.videos)+len(page.Items) > 400 {
		s.videos = map[string]research.Video{}
	}
	for _, item := range page.Items {
		s.videos[item.ID] = item
	}
	s.mu.Unlock()
	saved := []string{}
	for _, v := range page.Items {
		if s.Store.Contains(r.Context(), v.ID) {
			saved = append(saved, v.ID)
		}
	}
	reply(w, 200, map[string]any{"items": page.Items, "nextPage": page.NextPage, "savedIds": saved, "cached": cache})
}
func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	offset := 0
	if text := r.URL.Query().Get("offset"); text != "" {
		var err error
		offset, err = strconv.Atoi(text)
		if err != nil || offset < 0 || offset > 100000 {
			fail(w, &research.Error{Code: "invalid_query", Message: "Invalid offset", Status: 400})
			return
		}
	}
	if len(q) > 200 {
		fail(w, &research.Error{Code: "invalid_query", Message: "Search is too long", Status: 400})
		return
	}
	refs, count, err := s.Store.List(r.Context(), q, offset)
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, 200, map[string]any{"items": refs, "total": count, "hasMore": offset+len(refs) < count})
}
func (s *Server) save(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID string `json:"id"`
	}
	if !decode(w, r, &body) {
		return
	}
	if !research.VideoID.MatchString(body.ID) {
		fail(w, &research.Error{Code: "invalid_video", Message: "Invalid YouTube video ID", Status: 400})
		return
	}
	s.mu.Lock()
	v, ok := s.videos[body.ID]
	s.mu.Unlock()
	if !ok || time.Since(v.ObservedAt) > 15*time.Minute {
		fail(w, &research.Error{Code: "search_expired", Message: "Search again before saving this reference", Status: 409})
		return
	}
	if err := s.Store.Save(r.Context(), v); err != nil {
		fail(w, err)
		return
	}
	reply(w, 200, map[string]bool{"ok": true})
}
func (s *Server) note(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Note     string            `json:"note"`
		Analysis research.Analysis `json:"analysis"`
	}
	if !decode(w, r, &body) {
		return
	}
	if !research.VideoID.MatchString(r.PathValue("id")) || !validAnalysis(body.Note, body.Analysis) {
		fail(w, &research.Error{Code: "invalid_note", Message: "Keep each analysis field and note under 1,000 characters, and keywords under 500", Status: 400})
		return
	}
	if !s.Store.Contains(r.Context(), r.PathValue("id")) {
		fail(w, &research.Error{Code: "not_found", Message: "Reference not found", Status: 404})
		return
	}
	if err := s.Store.Update(r.Context(), r.PathValue("id"), body.Note, body.Analysis); err != nil {
		fail(w, err)
		return
	}
	reply(w, 200, map[string]bool{"ok": true})
}
func (s *Server) remove(w http.ResponseWriter, r *http.Request) {
	if !research.VideoID.MatchString(r.PathValue("id")) {
		fail(w, &research.Error{Code: "invalid_video", Message: "Invalid reference", Status: 400})
		return
	}
	if err := s.Store.Delete(r.Context(), r.PathValue("id")); err != nil {
		fail(w, err)
		return
	}
	reply(w, 200, map[string]bool{"ok": true})
}
func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !research.VideoID.MatchString(id) || !s.Store.Contains(r.Context(), id) {
		fail(w, &research.Error{Code: "not_found", Message: "Reference not found", Status: 404})
		return
	}
	key, _ := s.credentials()
	if key == "" {
		fail(w, &research.Error{Code: "key_required", Message: "Add a YouTube API key in Settings", Status: 409})
		return
	}
	if !s.acquire(w) {
		return
	}
	defer func() { <-s.slots }()
	videos, err := s.YouTube.Videos(r.Context(), key, []string{id})
	if err != nil {
		fail(w, err)
		return
	}
	if len(videos) == 0 {
		_ = s.Store.ClearMetadata(r.Context(), id)
		fail(w, &research.Error{Code: "unavailable", Message: "This video is no longer available through YouTube. Your notes are preserved", Status: 404})
		return
	}
	if err = s.Store.Save(r.Context(), videos[0]); err != nil {
		fail(w, err)
		return
	}
	reply(w, 200, videos[0])
}
