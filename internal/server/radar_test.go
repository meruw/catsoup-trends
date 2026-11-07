package server

import (
	"encoding/json"
	"fmt"
	"github.com/meruw/catsoup-trends/internal/research"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTopicsCacheAndOfflineFallback(t *testing.T) {
	app, host := setup(t)
	app.key = ""
	calls := 0
	down := false
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if down {
			w.WriteHeader(503)
			return
		}
		fmt.Fprint(w, `<rss><channel><item><title>Animation</title></item></channel></rss>`)
	}))
	defer provider.Close()
	app.Trends.BaseURL = provider.URL
	for i := 0; i < 2; i++ {
		resp := request(t, app, host, "GET", "/api/v1/topics?region=MX", "")
		if resp.StatusCode != 200 {
			t.Fatal("topics required key")
		}
	}
	if calls != 1 {
		t.Fatal("cache not used")
	}
	app.mu.Lock()
	p := app.topicsCache["MX"]
	p.ObservedAt = time.Now().Add(-11 * time.Minute)
	app.topicsCache["MX"] = p
	app.mu.Unlock()
	down = true
	resp := request(t, app, host, "GET", "/api/v1/topics?region=MX", "")
	var stale research.TopicsPage
	json.NewDecoder(resp.Body).Decode(&stale)
	if resp.StatusCode != 200 || !stale.Stale || !stale.Cached {
		t.Fatal("stale feed not labeled")
	}
	app.mu.Lock()
	p.ObservedAt = time.Now().Add(-25 * time.Hour)
	app.topicsCache["MX"] = p
	app.mu.Unlock()
	if request(t, app, host, "GET", "/api/v1/topics?region=MX", "").StatusCode != 502 {
		t.Fatal("served over-age feed")
	}
	if request(t, app, host, "GET", "/api/v1/topics?region=invalid", "").StatusCode != 400 {
		t.Fatal("accepted bad region")
	}
}
func TestFormatAPIUsesSavedQueryAndPersistsCheck(t *testing.T) {
	app, host := setup(t)
	seen := ""
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Query().Get("q")
		if r.URL.Query().Get("relevanceLanguage") != "es" {
			t.Error("lost language")
		}
		fmt.Fprint(w, `{"items":[]}`)
	}))
	defer provider.Close()
	app.YouTube.BaseURL = provider.URL + "/"
	resp := request(t, app, host, "POST", "/api/v1/formats", `{"name":"One rule","query":"en este pueblo roblox","language":"es","period":"30","order":"viewCount","duration":"short"}`)
	var f research.Format
	json.NewDecoder(resp.Body).Decode(&f)
	if resp.StatusCode != 200 || !ideaID.MatchString(f.ID) {
		t.Fatal("create failed")
	}
	path := "/api/v1/formats/" + f.ID
	if request(t, app, host, "GET", path+"/results?q=override", "").StatusCode != 200 || seen != f.Query {
		t.Fatal("did not use saved query")
	}
	checked, _ := app.Store.Format(t.Context(), f.ID)
	if checked.LastChecked == "" {
		t.Fatal("check not persisted")
	}
	if request(t, app, host, "POST", "/api/v1/formats", `{"name":"Bad","query":"ok","order":"invalid"}`).StatusCode != 400 {
		t.Fatal("accepted invalid filters")
	}
	if request(t, app, host, "DELETE", path, "").StatusCode != 200 {
		t.Fatal("delete failed")
	}
	if request(t, app, host, "GET", path+"/results", "").StatusCode != 404 {
		t.Fatal("deleted format still searchable")
	}
}
