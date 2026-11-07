package research

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const feedFixture = `<rss xmlns:ht="https://trends.google.com/trending/rss"><channel><item><title>café &amp; animation</title><ht:approx_traffic>20K+</ht:approx_traffic><pubDate>Sat, 07 Nov 2026 10:00:00 -0800</pubDate><link>javascript:alert(1)</link></item><item><title>CAFÉ &amp; animation</title></item><item><title></title></item><item><title>Roblox</title><pubDate>invalid</pubDate></item></channel></rss>`

func TestTopicsParseSourceData(t *testing.T) {
	p, e := ParseTopics([]byte(feedFixture), "MX")
	if e != nil {
		t.Fatal(e)
	}
	if len(p.Items) != 2 || p.Items[0].Title != "café & animation" || p.Items[0].Traffic != "20K+" || p.Items[0].PublishedAt != "2026-11-07T18:00:00Z" {
		t.Fatalf("unexpected feed: %+v", p)
	}
	if !strings.HasPrefix(p.Items[0].URL, "https://trends.google.com/trends/explore?") || p.Items[1].PublishedAt != "" {
		t.Fatal("unsafe link or invented date")
	}
	for _, bad := range []string{`<html/>`, `<rss/>`, `<rss><channel>`, `<!DOCTYPE rss [<!ENTITY secret SYSTEM "file:///etc/passwd">]><rss><channel><item><title>&secret;</title></item></channel></rss>`} {
		if _, e := ParseTopics([]byte(bad), "MX"); e == nil {
			t.Fatal("accepted invalid feed")
		}
	}
}
func TestTopicsHTTPBoundaries(t *testing.T) {
	mode := "ok"
	calls := 0
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("geo") != "MX" {
			t.Error("region lost")
		}
		switch mode {
		case "redirect":
			http.Redirect(w, r, "/other", 302)
		case "large":
			fmt.Fprint(w, strings.Repeat("x", 2*1024*1024+1))
		case "error":
			w.WriteHeader(429)
		default:
			fmt.Fprint(w, feedFixture)
		}
	}))
	defer host.Close()
	g := NewGoogleTrends()
	g.BaseURL = host.URL
	for _, m := range []string{"ok", "redirect", "large", "error"} {
		mode = m
		p, e := g.Topics(context.Background(), "MX")
		if m == "ok" {
			if e != nil || p.ObservedAt.IsZero() {
				t.Fatal(e)
			}
		} else if e == nil {
			t.Fatal("accepted", m)
		}
	}
	if calls != 4 {
		t.Fatal("followed a redirect")
	}
	if _, e := g.Topics(context.Background(), "MX&url=evil"); e == nil || calls != 4 {
		t.Fatal("accepted invalid region")
	}
}
