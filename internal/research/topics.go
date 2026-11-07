package research

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Topic struct {
	Title       string `json:"title"`
	Traffic     string `json:"traffic"`
	PublishedAt string `json:"publishedAt"`
	URL         string `json:"url"`
}
type TopicsPage struct {
	Items      []Topic   `json:"items"`
	Region     string    `json:"region"`
	ObservedAt time.Time `json:"observedAt"`
	Cached     bool      `json:"cached"`
	Stale      bool      `json:"stale"`
}
type GoogleTrends struct {
	Client  *http.Client
	BaseURL string
}

func NewGoogleTrends() *GoogleTrends {
	return &GoogleTrends{Client: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}, BaseURL: "https://trends.google.com/trending/rss"}
}
func (g *GoogleTrends) Topics(ctx context.Context, region string) (TopicsPage, error) {
	result := TopicsPage{Items: []Topic{}, Region: region}
	if !countryCode.MatchString(region) {
		return result, &Error{"invalid_region", "Choose a two-letter country code", 400}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.BaseURL+"?geo="+url.QueryEscape(region), nil)
	if err != nil {
		return result, err
	}
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")
	resp, err := g.Client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		return result, &Error{"topics_unavailable", "Google Trends could not be reached. Try again later", 502}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return result, &Error{"topics_unavailable", "Google Trends is temporarily unavailable for this country", 502}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024+1))
	if err != nil || len(body) > 2*1024*1024 {
		return result, &Error{"topics_unavailable", "Google Trends returned an unreadable feed", 502}
	}
	result, err = ParseTopics(body, region)
	if err == nil {
		result.ObservedAt = time.Now().UTC()
	}
	return result, err
}
func ParseTopics(body []byte, region string) (TopicsPage, error) {
	result := TopicsPage{Items: []Topic{}, Region: region}
	var feed struct {
		XMLName xml.Name `xml:"rss"`
		Channel *struct {
			Items []struct {
				Title   string `xml:"title"`
				Traffic string `xml:"approx_traffic"`
				Date    string `xml:"pubDate"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if xml.Unmarshal(body, &feed) != nil || feed.Channel == nil {
		return result, &Error{"topics_unavailable", "Google Trends returned an unexpected feed", 502}
	}
	seen := map[string]bool{}
	for _, item := range feed.Channel.Items {
		title := strings.TrimSpace(item.Title)
		if title == "" || len(title) > 1000 || seen[strings.ToLower(title)] {
			continue
		}
		seen[strings.ToLower(title)] = true
		published := ""
		for _, layout := range []string{time.RFC1123Z, "Mon, 2 Jan 2006 15:04:05 -0700", time.RFC1123} {
			if t, e := time.Parse(layout, item.Date); e == nil {
				published = t.UTC().Format(time.RFC3339)
				break
			}
		}
		traffic := strings.TrimSpace(item.Traffic)
		if len(traffic) > 40 {
			traffic = ""
		}
		link := "https://trends.google.com/trends/explore?" + url.Values{"geo": {region}, "q": {title}}.Encode()
		result.Items = append(result.Items, Topic{Title: title, Traffic: traffic, PublishedAt: published, URL: link})
		if len(result.Items) == 100 {
			break
		}
	}
	return result, nil
}
