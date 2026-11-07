package research

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	_ "modernc.org/sqlite"
	"time"
)

const workspace = "local"

type Store struct{ db *sql.DB }

func Open(path string) (*Store, error) {
	db, e := sql.Open("sqlite", path)
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	_, e = db.Exec(`PRAGMA busy_timeout=5000; PRAGMA secure_delete=ON;
 CREATE TABLE IF NOT EXISTS references_saved (
 workspace TEXT NOT NULL, video_id TEXT NOT NULL, note TEXT NOT NULL DEFAULT '',
 saved_at TEXT NOT NULL, metadata TEXT, observed_at TEXT, analysis TEXT NOT NULL DEFAULT '{}',
 PRIMARY KEY(workspace,video_id));
 CREATE INDEX IF NOT EXISTS references_order ON references_saved(workspace,saved_at DESC,video_id);
 PRAGMA user_version=1;`)
	if e != nil {
		db.Close()
		return nil, e
	}
	var hasAnalysis bool
	rows, e := db.Query("PRAGMA table_info(references_saved)")
	if e != nil {
		db.Close()
		return nil, e
	}
	for rows.Next() {
		var cid, notnull, pk int
		var name, kind string
		var def any
		if e = rows.Scan(&cid, &name, &kind, &notnull, &def, &pk); e != nil {
			rows.Close()
			db.Close()
			return nil, e
		}
		if name == "analysis" {
			hasAnalysis = true
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		db.Close()
		return nil, e
	}
	if !hasAnalysis {
		if _, e = db.Exec("ALTER TABLE references_saved ADD COLUMN analysis TEXT NOT NULL DEFAULT '{}'"); e != nil {
			db.Close()
			return nil, e
		}
	}
	s := &Store{db}
	if e = s.Expire(context.Background(), time.Now()); e != nil {
		db.Close()
		return nil, e
	}
	return s, nil
}
func (s *Store) Close() error { return s.db.Close() }

// Only bookmarks and user-authored notes survive expiry. Public API metadata is refreshed or cleared before 30 days.
func (s *Store) Expire(ctx context.Context, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE references_saved SET metadata=NULL,observed_at=NULL WHERE observed_at < ?`, now.UTC().Add(-29*24*time.Hour).Format(time.RFC3339Nano))
	return err
}
func (s *Store) Save(ctx context.Context, v Video) error {
	if !VideoID.MatchString(v.ID) {
		return errors.New("invalid id")
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO references_saved(workspace,video_id,saved_at,metadata,observed_at) VALUES(?,?,?,?,?)
 ON CONFLICT(workspace,video_id) DO UPDATE SET metadata=excluded.metadata,observed_at=excluded.observed_at`, workspace, v.ID, time.Now().UTC().Format(time.RFC3339Nano), string(data), v.ObservedAt.UTC().Format(time.RFC3339Nano))
	return err
}
func (s *Store) Note(ctx context.Context, id, note string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE references_saved SET note=? WHERE workspace=? AND video_id=?`, note, workspace, id)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (s *Store) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM references_saved WHERE workspace=? AND video_id=?`, workspace, id)
	return err
}
func (s *Store) Contains(ctx context.Context, id string) bool {
	var n int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM references_saved WHERE workspace=? AND video_id=?`, workspace, id).Scan(&n)
	return n > 0
}
func (s *Store) List(ctx context.Context, q string, offset int) ([]Reference, int, error) {
	if err := s.Expire(ctx, time.Now()); err != nil {
		return nil, 0, err
	}
	filter := `workspace=? AND (?='' OR instr(lower(coalesce(metadata,'')),lower(?))>0 OR instr(lower(note),lower(?))>0 OR instr(lower(analysis),lower(?))>0 OR instr(video_id,?)>0)`
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM references_saved WHERE "+filter, workspace, q, q, q, q, q).Scan(&count); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, "SELECT video_id,note,saved_at,metadata,analysis FROM references_saved WHERE "+filter+" ORDER BY saved_at DESC,video_id LIMIT 20 OFFSET ?", workspace, q, q, q, q, q, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []Reference{}
	for rows.Next() {
		var r Reference
		var saved, analysis string
		var metadata sql.NullString
		if err = rows.Scan(&r.ID, &r.Note, &saved, &metadata, &analysis); err != nil {
			return nil, 0, err
		}
		r.SavedAt, _ = time.Parse(time.RFC3339Nano, saved)
		if err = json.Unmarshal([]byte(analysis), &r.Analysis); err != nil {
			return nil, 0, err
		}
		if metadata.Valid {
			var v Video
			if err = json.Unmarshal([]byte(metadata.String), &v); err != nil {
				return nil, 0, err
			}
			r.Video = &v
		}
		out = append(out, r)
	}
	return out, count, rows.Err()
}
