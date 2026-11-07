package server

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"github.com/meruw/catsoup-trends/internal/research"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

var ideaID = regexp.MustCompile(`^[a-f0-9]{32}$`)

func (s *Server) listIdeas(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	offset := 0
	var err error
	if value := r.URL.Query().Get("offset"); value != "" {
		offset, err = strconv.Atoi(value)
	}
	if err != nil || offset < 0 || offset > 100000 || len(q) > 200 {
		fail(w, &research.Error{Code: "invalid_query", Message: "Invalid idea search", Status: 400})
		return
	}
	items, err := s.Store.Ideas(r.Context(), q, offset)
	if err != nil {
		fail(w, err)
		return
	}
	more := len(items) > 20
	if more {
		items = items[:20]
	}
	reply(w, 200, map[string]any{"items": items, "hasMore": more})
}
func (s *Server) saveIdea(w http.ResponseWriter, r *http.Request) {
	var i research.Idea
	if !decode(w, r, &i) {
		return
	}
	i.Title = strings.TrimSpace(i.Title)
	valid := i.Title != "" && utf8.RuneCountInString(i.Title) <= 160 && len(i.Sources) <= 10
	seen := map[string]bool{}
	for _, id := range i.Sources {
		if !research.VideoID.MatchString(id) || seen[id] {
			valid = false
		}
		seen[id] = true
	}
	for _, text := range []string{i.Hook, i.Structure, i.Twist, i.World, i.Notes} {
		if utf8.RuneCountInString(text) > 3000 {
			valid = false
		}
	}
	create := r.Method == http.MethodPost
	if !create && !ideaID.MatchString(r.PathValue("id")) {
		valid = false
	}
	if !valid {
		fail(w, &research.Error{Code: "invalid_idea", Message: "Use a title up to 160 characters, up to 10 unique source videos, and fields up to 3,000 characters", Status: 400})
		return
	}
	i.ID = r.PathValue("id")
	if create {
		for _, id := range i.Sources {
			if !s.Store.Contains(r.Context(), id) {
				fail(w, &research.Error{Code: "not_found", Message: "Save the selected references before combining them", Status: 404})
				return
			}
		}
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			fail(w, err)
			return
		}
		i.ID = hex.EncodeToString(b)
	}
	if i.Sources == nil {
		i.Sources = []string{}
	}
	saved, err := s.Store.SaveIdea(r.Context(), i, create)
	if errors.Is(err, sql.ErrNoRows) {
		fail(w, &research.Error{Code: "not_found", Message: "Idea not found", Status: 404})
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, 200, saved)
}
func (s *Server) deleteIdea(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !ideaID.MatchString(id) {
		fail(w, &research.Error{Code: "invalid_idea", Message: "Invalid idea", Status: 400})
		return
	}
	if err := s.Store.DeleteIdea(r.Context(), id); err != nil {
		fail(w, err)
		return
	}
	reply(w, 200, map[string]bool{"ok": true})
}
func (s *Server) exportIdea(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !ideaID.MatchString(id) {
		fail(w, &research.Error{Code: "invalid_idea", Message: "Invalid idea", Status: 400})
		return
	}
	i, err := s.Store.Idea(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		fail(w, &research.Error{Code: "not_found", Message: "Idea not found", Status: 404})
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	refs := []research.Reference{}
	for _, source := range i.Sources {
		ref, err := s.Store.Get(r.Context(), source)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			fail(w, err)
			return
		}
		refs = append(refs, ref)
	}
	reply(w, 200, map[string]string{"markdown": research.BuildIdeaBrief(i, refs)})
}
