package research

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestInitialSchemaMigratesWithoutLosingNotes(t *testing.T) {
	p := filepath.Join(t.TempDir(), "old.db")
	db, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE references_saved(workspace TEXT NOT NULL,video_id TEXT NOT NULL,note TEXT NOT NULL DEFAULT '',saved_at TEXT NOT NULL,metadata TEXT,observed_at TEXT,PRIMARY KEY(workspace,video_id));
 INSERT INTO references_saved(workspace,video_id,note,saved_at) VALUES('local','fixture0001','keep this','2026-01-01T00:00:00Z');`)
	db.Close()
	if err != nil {
		t.Fatal(err)
	}
	s, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ref, err := s.Get(context.Background(), "fixture0001")
	if err != nil || ref.Note != "keep this" || ref.Analysis.Hook != "" {
		t.Fatal("migration lost user data")
	}
	if err := s.Update(context.Background(), ref.ID, ref.Note, Analysis{Keywords: "anticipation"}); err != nil {
		t.Fatal(err)
	}
	rows, count, err := s.List(context.Background(), "anticipation", 0)
	if err != nil || count != 1 || len(rows) != 1 {
		t.Fatal("analysis is not searchable")
	}
}
