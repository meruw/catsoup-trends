// Test-only host: exercises the actual HTTP API and database against synthetic YouTube responses.
// Never built into the production executable.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/meruw/catsoup-trends/internal/research"
	"github.com/meruw/catsoup-trends/internal/server"
	"github.com/meruw/catsoup-trends/web"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"
)

func main() {
	dir := os.Getenv("TRENDS_TEST_DATA")
	if dir == "" {
		panic("TRENDS_TEST_DATA is required")
	}
	_ = os.MkdirAll(dir, 0700)
	_ = os.Setenv("TRENDS_YOUTUBE_API_KEY", "synthetic-test-key")
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/search" {
			switch r.URL.Query().Get("q") {
			case "quota":
				w.WriteHeader(403)
				fmt.Fprint(w, `{"error":{"errors":[{"reason":"quotaExceeded"}]}}`)
				return
			case "empty":
				fmt.Fprint(w, `{"items":[]}`)
				return
			case "slow":
				select {
				case <-time.After(3 * time.Second):
				case <-r.Context().Done():
					return
				}
			}
			if r.URL.Query().Get("pageToken") == "page2" {
				fmt.Fprint(w, `{"items":[{"id":{"videoId":"fixture0003"}}]}`)
			} else {
				fmt.Fprint(w, `{"items":[{"id":{"videoId":"fixture0001"}},{"id":{"videoId":"fixture0002"}}],"nextPageToken":"page2"}`)
			}
		} else {
			names := []string{"The shadow that moved first", "One rule in an empty city", "A door that opens somewhere else"}
			items := []any{}
			for i, name := range names {
				items = append(items, map[string]any{"id": fmt.Sprintf("fixture000%d", i+1), "snippet": map[string]any{"title": name + " [test fixture]", "channelTitle": "Synthetic animation studio", "channelId": "fixturechannel", "tags": []string{"animation", "block story"}, "description": "Synthetic reference for automated testing. Not a real YouTube video. #Animation #Historia", "publishedAt": "2026-10-12T12:00:00Z", "thumbnails": map[string]any{"medium": map[string]string{"url": fmt.Sprintf("https://i.ytimg.com/vi/fixture000%d/mqdefault.jpg", i+1)}}}, "contentDetails": map[string]string{"duration": "PT22S"}, "statistics": map[string]string{"viewCount": fmt.Sprint((i + 1) * 24000), "likeCount": fmt.Sprint((i + 1) * 800)}})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
		}
	}))
	defer provider.Close()
	store, err := research.Open(filepath.Join(dir, "trends.db"))
	if err != nil {
		panic(err)
	}
	defer store.Close()
	app, err := server.New(store, dir)
	if err != nil {
		panic(err)
	}
	app.YouTube.BaseURL = provider.URL + "/"
	topics := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		if r.URL.Query().Get("geo") == "CA" {
			w.WriteHeader(503)
			return
		}
		fmt.Fprint(w, `<rss xmlns:ht="https://trends.google.com/trending/rss"><channel><item><title>Animation festival [test fixture]</title><ht:approx_traffic>20K+</ht:approx_traffic><pubDate>Sat, 07 Nov 2026 10:00:00 -0800</pubDate></item><item><title>Roblox story [test fixture]</title><ht:approx_traffic>5K+</ht:approx_traffic></item></channel></rss>`)
	}))
	defer topics.Close()
	app.Trends.BaseURL = topics.URL
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	app.Host = listener.Addr().String()
	app.Origin = "http://" + app.Host
	assets, err := fs.Sub(web.Assets, "dist")
	if err != nil {
		panic(err)
	}
	host := &http.Server{Handler: app.Handler(assets), ReadHeaderTimeout: 5 * time.Second}
	app.Shutdown = func() { _ = host.Shutdown(context.Background()) }
	fmt.Println(app.Origin + "/#token=" + app.Token)
	_ = host.Serve(listener)
}
