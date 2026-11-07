package research

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const firstID = "fixture0001"
const secondID = "fixture0002"

func TestYouTubeSearchJoinsMetricsAndPreservesOrder(t *testing.T) {
	calls := 0
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("key") != "test-key" {
			t.Error("missing key")
		}
		if r.URL.Path == "/search" {
			if r.URL.Query().Get("type") != "video" || r.URL.Query().Get("videoDuration") != "short" || r.URL.Query().Get("relevanceLanguage") != "es" {
				t.Error("missing filters")
			}
			fmt.Fprintf(w, `{"items":[{"id":{"videoId":"%s"}},{"id":{"videoId":"%s"}}],"nextPageToken":"next"}`, firstID, secondID)
		} else {
			fmt.Fprintf(w, `{"items":[{"id":"%s","snippet":{"title":"Second","thumbnails":{"medium":{"url":"http://127.0.0.1/private"}}},"statistics":{"viewCount":"0"}},{"id":"%s","snippet":{"title":"First","publishedAt":"2026-01-02T12:00:00Z","thumbnails":{"medium":{"url":"https://i.ytimg.com/vi/x/mqdefault.jpg"}}},"contentDetails":{"duration":"PT12S"},"statistics":{"viewCount":"500"}}]}`, secondID, firstID)
		}
	}))
	defer mock.Close()
	y := NewYouTube()
	y.BaseURL = mock.URL + "/"
	page, err := y.Search(context.Background(), "test-key", Query{Text: "animation", Language: "es"})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || len(page.Items) != 2 || page.NextPage != "next" {
		t.Fatalf("unexpected result: %+v", page)
	}
	if page.Items[0].ID != firstID || *page.Items[0].Views != 500 || page.Items[0].Likes != nil {
		t.Fatal("order or missing metrics were lost")
	}
	if page.Items[1].Thumbnail != "" || *page.Items[1].Views != 0 {
		t.Fatal("thumbnail guard or zero metric failed")
	}
}
func TestProviderFailuresAreRedactedAndCancellationPropagates(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		fmt.Fprint(w, `{"error":{"message":"SECRET_KEY","errors":[{"reason":"quotaExceeded"}]}}`)
	}))
	defer mock.Close()
	y := NewYouTube()
	y.BaseURL = mock.URL + "/"
	_, err := y.Search(context.Background(), "SECRET_KEY", Query{Text: "x"})
	if err == nil || strings.Contains(err.Error(), "SECRET_KEY") {
		t.Fatal("error not redacted")
	}
	if e, ok := err.(*Error); !ok || e.Code != "quota_exceeded" {
		t.Fatal("quota not mapped")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = y.Search(ctx, "key", Query{Text: "x"})
	if err != context.Canceled {
		t.Fatalf("cancellation lost: %v", err)
	}
}
func TestSaveRestartNotesExpiryAndRemoval(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	ctx := context.Background()
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	video := Video{ID: firstID, Title: "Provider metadata", ObservedAt: time.Now().UTC()}
	if err = store.Save(ctx, video); err != nil {
		t.Fatal(err)
	}
	analysis := Analysis{Hook: "A shadow moves first", Adaptation: "An original scene", Keywords: "suspense"}
	if err = store.Update(ctx, firstID, "Keep the silence", analysis); err != nil {
		t.Fatal(err)
	}
	if err = store.Save(ctx, video); err != nil {
		t.Fatal(err)
	}
	store.Close()
	store, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	refs, n, err := store.List(ctx, "", 0)
	if err != nil || n != 1 || refs[0].Note != "Keep the silence" || refs[0].Analysis.Hook != analysis.Hook {
		t.Fatalf("persistence/dedup: %v %+v", err, refs)
	}
	if err = store.Expire(ctx, time.Now().Add(30*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	ref, err := store.Get(ctx, firstID)
	if err != nil || ref.Video != nil || ref.Note == "" || ref.Analysis.Hook == "" {
		t.Fatalf("expiry erased user data: %v %+v", err, ref)
	}
	if err = store.Delete(ctx, firstID); err != nil {
		t.Fatal(err)
	}
	if store.Contains(ctx, firstID) {
		t.Fatal("delete failed")
	}
}
func TestWorkspaceIsolation(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_, err = s.db.Exec(`INSERT INTO references_saved(workspace,video_id,note,saved_at,analysis) VALUES('other','fixture0001','private','2026-01-01','{}')`)
	if err != nil {
		t.Fatal(err)
	}
	rows, n, err := s.List(context.Background(), "", 0)
	if err != nil || n != 0 || len(rows) != 0 || s.Contains(context.Background(), firstID) {
		t.Fatal("workspace leak")
	}
}
func TestBriefUsesOnlyCreatorMaterial(t *testing.T) {
	v := Video{Title: "DO NOT EXPORT API TITLE", Description: "DO NOT EXPORT API DESCRIPTION"}
	b := BuildBrief([]Reference{{ID: firstID, Video: &v, Note: "my note\nsecond line", Analysis: Analysis{Hook: "my hook"}}})
	if strings.Contains(b, "DO NOT EXPORT") || !strings.Contains(b, "watch?v="+firstID) || !strings.Contains(b, "> my hook") || !strings.Contains(b, "> second line") {
		t.Fatal("unsafe or incomplete export")
	}
}
func TestQueryValidation(t *testing.T) {
	for _, q := range []Query{{Text: ""}, {Text: "x", Order: "random"}, {Text: "x", Page: strings.Repeat("a", 513)}} {
		if ValidateQuery(&q) == nil {
			t.Fatal("invalid query allowed")
		}
	}
}
func TestModelAnalysisJSON(t *testing.T) {
	b, _ := json.Marshal(Reference{})
	if !strings.Contains(string(b), `"analysis":{"hook":""`) {
		t.Fatal("missing analysis defaults")
	}
}
