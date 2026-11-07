package research

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"
)

type Format struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Query       string `json:"query"`
	Language    string `json:"language"`
	Period      string `json:"period"`
	Order       string `json:"order"`
	Duration    string `json:"duration"`
	UpdatedAt   string `json:"updatedAt"`
	LastChecked string `json:"lastChecked"`
}

func (f *Format) Validate() error {
	f.Name = strings.TrimSpace(f.Name)
	if f.Name == "" || utf8.RuneCountInString(f.Name) > 100 {
		return &Error{"invalid_format", "Use a format name of 1–100 characters", 400}
	}
	q := f.SearchQuery("")
	if err := ValidateQuery(&q); err != nil {
		return err
	}
	f.Query = q.Text
	f.Period = q.Period
	f.Order = q.Order
	f.Duration = q.Duration
	return nil
}
func (f Format) SearchQuery(page string) Query {
	return Query{Text: f.Query, Language: f.Language, Period: f.Period, Order: f.Order, Duration: f.Duration, Page: page}
}
func (s *Store) Formats(ctx context.Context) ([]Format, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT body,last_checked FROM formats WHERE workspace=? ORDER BY name COLLATE NOCASE,id`, workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Format{}
	for rows.Next() {
		var f Format
		var body string
		if err = rows.Scan(&body, &f.LastChecked); err != nil {
			return nil, err
		}
		checked := f.LastChecked
		if err = json.Unmarshal([]byte(body), &f); err != nil {
			return nil, err
		}
		f.LastChecked = checked
		items = append(items, f)
	}
	return items, rows.Err()
}
func (s *Store) Format(ctx context.Context, id string) (Format, error) {
	var f Format
	var body, checked string
	err := s.db.QueryRowContext(ctx, `SELECT body,last_checked FROM formats WHERE workspace=? AND id=?`, workspace, id).Scan(&body, &checked)
	if err == nil {
		err = json.Unmarshal([]byte(body), &f)
		f.LastChecked = checked
	}
	return f, err
}
func (s *Store) SaveFormat(ctx context.Context, f Format, create bool) (Format, error) {
	if err := f.Validate(); err != nil {
		return f, err
	}
	f.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	f.LastChecked = ""
	b, err := json.Marshal(f)
	if err != nil {
		return f, err
	}
	if create {
		result, e := s.db.ExecContext(ctx, `INSERT INTO formats(workspace,id,name,body,last_checked) SELECT ?,?,?,?,'' WHERE (SELECT COUNT(*) FROM formats WHERE workspace=?)<100`, workspace, f.ID, f.Name, string(b), workspace)
		err = e
		if err == nil {
			n, _ := result.RowsAffected()
			if n == 0 {
				err = &Error{"format_limit", "You can save up to 100 formats. Remove an unused format first", 409}
			}
		}
	} else {
		result, e := s.db.ExecContext(ctx, `UPDATE formats SET name=?,body=?,last_checked='' WHERE workspace=? AND id=?`, f.Name, string(b), workspace, f.ID)
		err = e
		if err == nil {
			n, _ := result.RowsAffected()
			if n == 0 {
				err = sql.ErrNoRows
			}
		}
	}
	return f, err
}
func (s *Store) DeleteFormat(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM formats WHERE workspace=? AND id=?`, workspace, id)
	return err
}
func (s *Store) CheckedFormat(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE formats SET last_checked=? WHERE workspace=? AND id=?`, time.Now().UTC().Format(time.RFC3339Nano), workspace, id)
	return err
}
