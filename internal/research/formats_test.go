package research

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestFormatsPersistenceAndIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trends.db")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	v := Video{ID: "fixture0001", Title: "Keep this", ObservedAt: time.Now()}
	if e = s.Save(ctx, v); e != nil {
		t.Fatal(e)
	}
	// Simulate an existing schema-2 database with references before migration.
	if _, e = s.db.Exec("DROP TABLE formats; PRAGMA user_version=2"); e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	if !s.Contains(ctx, v.ID) {
		t.Fatal("migration lost reference")
	}
	f := Format{ID: "one", Name: "Custom town", Query: `"en este pueblo" roblox`, Language: "es", Period: "30", Order: "viewCount", Duration: "short"}
	if _, e = s.SaveFormat(ctx, f, true); e != nil {
		t.Fatal(e)
	}
	if e = s.CheckedFormat(ctx, f.ID); e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	items, e := s.Formats(ctx)
	if e != nil || len(items) != 1 || items[0].Query != f.Query || items[0].LastChecked == "" {
		t.Fatalf("lost format: %+v %v", items, e)
	}
	f.Query = "in this town minecraft"
	f.Language = "en"
	if _, e = s.SaveFormat(ctx, f, false); e != nil {
		t.Fatal(e)
	}
	updated, _ := s.Format(ctx, f.ID)
	if updated.LastChecked != "" {
		t.Fatal("changed query kept old check")
	}
	if e = s.DeleteFormat(ctx, f.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Format(ctx, f.ID); e != sql.ErrNoRows {
		t.Fatal("format remained")
	}
	if !s.Contains(ctx, v.ID) {
		t.Fatal("format deleted saved reference")
	}
	for i := 0; i < 100; i++ {
		f.ID = fmt.Sprint(i)
		if _, e = s.SaveFormat(ctx, f, true); e != nil {
			t.Fatal(e)
		}
	}
	f.ID = "overflow"
	if _, e = s.SaveFormat(ctx, f, true); e == nil {
		t.Fatal("format cap not enforced")
	}
}
