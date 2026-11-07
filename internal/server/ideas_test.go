package server

import (
	"encoding/json"
	"github.com/meruw/catsoup-trends/internal/research"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIdeasLifecycleAndExportBoundary(t *testing.T) {
	app, host := setup(t)
	app.videos["fixture0001"] = research.Video{ID: "fixture0001", Title: "PRIVATE_PROVIDER_TITLE", Description: "PROVIDER_DESCRIPTION", Tags: []string{"PROVIDER_TAG"}, ObservedAt: time.Now()}
	request(t, app, host, "POST", "/api/v1/references", `{"id":"fixture0001"}`)
	request(t, app, host, "PATCH", "/api/v1/references/fixture0001", `{"note":"creator note","analysis":{"hook":"creator hook"}}`)
	r := request(t, app, host, "POST", "/api/v1/ideas", `{"title":"New story","sources":["fixture0001"],"world":"Blocks","hook":"Original hook"}`)
	if r.StatusCode != 200 {
		t.Fatal("create failed", r.StatusCode)
	}
	var idea research.Idea
	json.NewDecoder(r.Body).Decode(&idea)
	if !ideaID.MatchString(idea.ID) || idea.UpdatedAt == "" {
		t.Fatal("missing identity")
	}
	for _, body := range []string{`{"title":""}`, `{"title":"x","sources":["bad"]}`, `{"title":"x","sources":["fixture0001","fixture0001"]}`, `{"title":"x","secret":"bad"}`} {
		if got := request(t, app, host, "POST", "/api/v1/ideas", body).StatusCode; got != 400 {
			t.Fatal("invalid accepted", got)
		}
	}
	if got := request(t, app, host, "POST", "/api/v1/ideas", `{"title":"x","sources":["fixture9999"]}`).StatusCode; got != 404 {
		t.Fatal("missing source accepted")
	}
	r = request(t, app, host, "POST", "/api/v1/ideas/"+idea.ID+"/export", "")
	data, _ := io.ReadAll(r.Body)
	if !strings.Contains(string(data), "creator hook") || !strings.Contains(string(data), "Original hook") {
		t.Fatal("lost author notes")
	}
	for _, secret := range []string{"PRIVATE_PROVIDER_TITLE", "PROVIDER_DESCRIPTION", "PROVIDER_TAG", "test-key-for-unit-tests"} {
		if strings.Contains(string(data), secret) {
			t.Fatal("export leaked provider data")
		}
	}
	request(t, app, host, "DELETE", "/api/v1/references/fixture0001", "")
	r = request(t, app, host, "POST", "/api/v1/ideas/"+idea.ID+"/export", "")
	data, _ = io.ReadAll(r.Body)
	if r.StatusCode != 200 || !strings.Contains(string(data), "fixture0001") {
		t.Fatal("deleting source broke idea")
	}
	r = request(t, app, host, "PUT", "/api/v1/ideas/"+idea.ID, `{"title":"Updated story","sources":["fixture0001"],"twist":"New twist"}`)
	if r.StatusCode != 200 {
		t.Fatal("update failed")
	}
	r = request(t, app, host, "GET", "/api/v1/ideas?q=Updated", "")
	data, _ = io.ReadAll(r.Body)
	if !strings.Contains(string(data), "New twist") {
		t.Fatal("find failed")
	}
	request(t, app, host, "DELETE", "/api/v1/ideas/"+idea.ID, "")
	if request(t, app, host, "POST", "/api/v1/ideas/"+idea.ID+"/export", "").StatusCode != 404 {
		t.Fatal("delete failed")
	}
	if request(t, app, host, "PUT", "/api/v1/ideas/"+idea.ID, `{"title":"Missing"}`).StatusCode != 404 {
		t.Fatal("update recreated deleted idea")
	}
}
func TestDiscoveryCacheAndTags(t *testing.T) {
	app, host := setup(t)
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		q := r.URL.Query()
		if r.URL.Path != "/videos" || q.Get("chart") != "mostPopular" || q.Get("regionCode") != "MX" || q.Get("videoCategoryId") != "20" {
			t.Error("bad discovery request")
		}
		io.WriteString(w, `{"items":[{"id":"fixture0001","snippet":{"title":"Popular fixture","tags":["animation","Minecraft"],"publishedAt":"2026-11-01T00:00:00Z"}}],"nextPageToken":"more"}`)
	}))
	defer upstream.Close()
	app.YouTube.BaseURL = upstream.URL + "/"
	for range 2 {
		r := request(t, app, host, "GET", "/api/v1/discover?region=MX&category=20", "")
		data, _ := io.ReadAll(r.Body)
		if r.StatusCode != 200 || !strings.Contains(string(data), "Minecraft") {
			t.Fatal("missing metadata", string(data))
		}
	}
	if calls != 1 {
		t.Fatal("cache missed")
	}
	if request(t, app, host, "POST", "/api/v1/references", `{"id":"fixture0001"}`).StatusCode != 200 {
		t.Fatal("discovery not saveable")
	}
	if request(t, app, host, "GET", "/api/v1/discover?region=invalid", "").StatusCode != 400 {
		t.Fatal("invalid accepted")
	}
}
