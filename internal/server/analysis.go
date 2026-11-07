package server

import (
	"database/sql"
	"errors"
	"github.com/meruw/catsoup-trends/internal/research"
	"net/http"
	"unicode/utf8"
)

func (s *Server) getReference(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !research.VideoID.MatchString(id) {
		fail(w, &research.Error{Code: "invalid_video", Message: "Invalid reference", Status: 400})
		return
	}
	ref, err := s.Store.Get(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		fail(w, &research.Error{Code: "not_found", Message: "Reference not found", Status: 404})
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	reply(w, 200, ref)
}
func validAnalysis(note string, a research.Analysis) bool {
	if utf8.RuneCountInString(note) > 1000 || utf8.RuneCountInString(a.Keywords) > 500 {
		return false
	}
	for _, v := range []string{a.Hook, a.Structure, a.Twist, a.Adaptation} {
		if utf8.RuneCountInString(v) > 1000 {
			return false
		}
	}
	return true
}
func (s *Server) exportClaude(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []string `json:"ids"`
	}
	if !decode(w, r, &body) {
		return
	}
	if len(body.IDs) == 0 || len(body.IDs) > 10 {
		fail(w, &research.Error{Code: "invalid_selection", Message: "Select 1–10 saved references", Status: 400})
		return
	}
	refs := []research.Reference{}
	seen := map[string]bool{}
	for _, id := range body.IDs {
		if !research.VideoID.MatchString(id) || seen[id] {
			fail(w, &research.Error{Code: "invalid_selection", Message: "Choose unique saved references", Status: 400})
			return
		}
		seen[id] = true
		ref, err := s.Store.Get(r.Context(), id)
		if errors.Is(err, sql.ErrNoRows) {
			fail(w, &research.Error{Code: "not_found", Message: "One of the selected references no longer exists", Status: 404})
			return
		}
		if err != nil {
			fail(w, err)
			return
		}
		refs = append(refs, ref)
	}
	reply(w, 200, map[string]string{"markdown": research.BuildBrief(refs)})
}
