package server

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"github.com/meruw/catsoup-trends/internal/research"
	"net/http"
)

func (s *Server) listFormats(w http.ResponseWriter, r *http.Request) {
	items, err := s.Store.Formats(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, 200, map[string]any{"items": items})
}
func (s *Server) saveFormat(w http.ResponseWriter, r *http.Request) {
	var f research.Format
	if !decode(w, r, &f) {
		return
	}
	create := r.Method == http.MethodPost
	f.ID = r.PathValue("id")
	if !create && !ideaID.MatchString(f.ID) {
		fail(w, &research.Error{Code: "invalid_format", Message: "Invalid format", Status: 400})
		return
	}
	if create {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			fail(w, err)
			return
		}
		f.ID = hex.EncodeToString(b)
	}
	saved, err := s.Store.SaveFormat(r.Context(), f, create)
	if errors.Is(err, sql.ErrNoRows) {
		err = &research.Error{Code: "not_found", Message: "Format not found", Status: 404}
	}
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, 200, saved)
}
func (s *Server) deleteFormat(w http.ResponseWriter, r *http.Request) {
	if !ideaID.MatchString(r.PathValue("id")) {
		fail(w, &research.Error{Code: "invalid_format", Message: "Invalid format", Status: 400})
		return
	}
	if err := s.Store.DeleteFormat(r.Context(), r.PathValue("id")); err != nil {
		fail(w, err)
		return
	}
	reply(w, 200, map[string]bool{"ok": true})
}
func (s *Server) formatResults(w http.ResponseWriter, r *http.Request) {
	f, err := s.Store.Format(r.Context(), r.PathValue("id"))
	if errors.Is(err, sql.ErrNoRows) {
		err = &research.Error{Code: "not_found", Message: "Format not found", Status: 404}
	}
	if err != nil {
		fail(w, err)
		return
	}
	q := f.SearchQuery(r.URL.Query().Get("page"))
	if s.searchQuery(w, r, q) {
		_ = s.Store.CheckedFormat(r.Context(), f.ID)
	}
}
