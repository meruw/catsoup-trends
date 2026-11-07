package server

import (
	"bytes"
	"encoding/json"
	"github.com/meruw/catsoup-trends/internal/research"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func setup(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	t.Setenv("TRENDS_YOUTUBE_API_KEY", "test-key-for-unit-tests")
	dir := t.TempDir()
	s, e := research.Open(filepath.Join(dir, "test.db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	app, e := New(s, dir)
	if e != nil {
		t.Fatal(e)
	}
	host := httptest.NewUnstartedServer(nil)
	app.Host = host.Listener.Addr().String()
	app.Origin = "http://" + app.Host
	host.Config.Handler = app.Handler(fstest.MapFS{"index.html": {Data: []byte("test app")}})
	host.Start()
	t.Cleanup(host.Close)
	return app, host
}
func request(t *testing.T, app *Server, host *httptest.Server, method, path, body string) *http.Response {
	t.Helper()
	r, _ := http.NewRequest(method, host.URL+path, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+app.Token)
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	response, err := host.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { response.Body.Close() })
	return response
}
func TestLocalSecurity(t *testing.T) {
	app, host := setup(t)
	for _, kind := range []string{"no token", "foreign origin", "foreign host", "cross site"} {
		r, _ := http.NewRequest("GET", host.URL+"/api/v1/status", nil)
		r.Header.Set("Authorization", "Bearer "+app.Token)
		switch kind {
		case "no token":
			r.Header.Del("Authorization")
		case "foreign origin":
			r.Header.Set("Origin", "https://evil.example")
		case "foreign host":
			r.Host = "evil.example"
		case "cross site":
			r.Header.Set("Sec-Fetch-Site", "cross-site")
		}
		resp, err := host.Client().Do(r)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode < 400 {
			t.Fatalf("allowed %s", kind)
		}
	}
	resp := request(t, app, host, "GET", "/api/v1/status", "")
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || bytes.Contains(data, []byte("test-key-for-unit-tests")) || bytes.Contains(data, []byte(app.Token)) {
		t.Fatal("status leaked credentials")
	}
	if resp.Header.Get("Content-Security-Policy") == "" {
		t.Fatal("missing CSP")
	}
}
func TestReferenceLifecycleAndExportAPI(t *testing.T) {
	app, host := setup(t)
	id := "fixture0001"
	app.videos[id] = research.Video{ID: id, Title: "Fixture", ObservedAt: time.Now()}
	for i := 0; i < 2; i++ {
		if r := request(t, app, host, "POST", "/api/v1/references", `{"id":"fixture0001"}`); r.StatusCode != 200 {
			t.Fatal("save failed")
		}
	}
	if r := request(t, app, host, "PATCH", "/api/v1/references/"+id, `{"note":"test note","analysis":{"hook":"first moment","adaptation":"original idea"}}`); r.StatusCode != 200 {
		t.Fatal("analysis save failed")
	}
	r := request(t, app, host, "GET", "/api/v1/references/"+id, "")
	var ref research.Reference
	json.NewDecoder(r.Body).Decode(&ref)
	if ref.Analysis.Hook != "first moment" || ref.Note != "test note" {
		t.Fatalf("lost note: %+v", ref)
	}
	r = request(t, app, host, "POST", "/api/v1/exports/claude", `{"ids":["fixture0001"]}`)
	data, _ := io.ReadAll(r.Body)
	if r.StatusCode != 200 || !bytes.Contains(data, []byte("first moment")) {
		t.Fatal("export failed")
	}
	r = request(t, app, host, "POST", "/api/v1/exports/claude", `{"ids":["fixture0001","fixture0001"]}`)
	if r.StatusCode != 400 {
		t.Fatal("duplicate export accepted")
	}
	r = request(t, app, host, "DELETE", "/api/v1/references/"+id, "")
	if r.StatusCode != 200 {
		t.Fatal("delete failed")
	}
	r = request(t, app, host, "GET", "/api/v1/references/"+id, "")
	if r.StatusCode != 404 {
		t.Fatal("missing reference not mapped")
	}
}
func TestRequestBounds(t *testing.T) {
	app, host := setup(t)
	for _, body := range []string{`{"id":"../../bad"}`, `{"id":"fixture0001","unexpected":true}`, `{"id":"fixture0001"}{}`, strings.Repeat("x", 40000)} {
		r := request(t, app, host, "POST", "/api/v1/references", body)
		if r.StatusCode < 400 {
			t.Fatal("invalid input accepted")
		}
	}
	r := request(t, app, host, "GET", "/api/v1/references?offset=-1", "")
	if r.StatusCode != 400 {
		t.Fatal("negative offset accepted")
	}
	r = request(t, app, host, "POST", "/api/v1/references", `{"id":"fixture0001"}`)
	if r.StatusCode != 409 {
		t.Fatal("unseen video accepted")
	}
}
func TestSearchCachePreventsRepeatedProviderRequests(t *testing.T) {
	app, host := setup(t)
	calls := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path == "/search" {
			io.WriteString(w, `{"items":[{"id":{"videoId":"fixture0001"}}]}`)
		} else {
			io.WriteString(w, `{"items":[{"id":"fixture0001","snippet":{"title":"Fixture"},"statistics":{"viewCount":"10"}}]}`)
		}
	}))
	defer provider.Close()
	app.YouTube.BaseURL = provider.URL + "/"
	for i := 0; i < 2; i++ {
		r := request(t, app, host, "GET", "/api/v1/search?q=test", "")
		if r.StatusCode != 200 {
			t.Fatal("search failed")
		}
	}
	if calls != 2 {
		t.Fatalf("cache missed: %d provider calls", calls)
	}
}
