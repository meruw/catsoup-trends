package research

import (
	"context"
	"path/filepath"
	"testing"
)

func TestIdeasSurviveReopenAndPaginate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ideas.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for n := 0; n < 22; n++ {
		_, err = s.SaveIdea(ctx, Idea{ID: string(rune('a' + n)), Title: "Concept", Sources: []string{"fixture0001"}, Twist: "A creator observation"}, true)
		if err != nil {
			t.Fatal(err)
		}
	}
	s.Close()
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	items, err := s.Ideas(ctx, "Concept", 0)
	if err != nil || len(items) != 21 {
		t.Fatal("first page", len(items), err)
	}
	items, err = s.Ideas(ctx, "Concept", 20)
	if err != nil || len(items) != 2 || items[0].Twist != "A creator observation" {
		t.Fatal("persistence/page", err)
	}
	items, err = s.Ideas(ctx, "%", 0)
	if err != nil || len(items) != 0 {
		t.Fatal("search should be literal")
	}
}
