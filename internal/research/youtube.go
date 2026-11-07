package research

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var VideoID = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
var countryCode = regexp.MustCompile(`^[A-Z]{2}$`)

type YouTube struct {
	Client  *http.Client
	BaseURL string
}

func NewYouTube() *YouTube {
	return &YouTube{Client: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}, BaseURL: "https://www.googleapis.com/youtube/v3/"}
}
func (y *YouTube) get(ctx context.Context, key, endpoint string, values url.Values, out any) error {
	values.Set("key", key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, y.BaseURL+endpoint+"?"+values.Encode(), nil)
	if err != nil {
		return &Error{"provider_error", "Could not prepare the YouTube request", 502}
	}
	resp, err := y.Client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return &Error{"network_error", "YouTube could not be reached. Check your connection and try again", 502}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024+1))
	if err != nil || len(body) > 2*1024*1024 {
		return &Error{"provider_error", "YouTube returned an unreadable response", 502}
	}
	if resp.StatusCode != http.StatusOK {
		var detail struct {
			Error struct {
				Errors []struct {
					Reason string `json:"reason"`
				} `json:"errors"`
			} `json:"error"`
		}
		_ = json.Unmarshal(body, &detail)
		for _, e := range detail.Error.Errors {
			switch e.Reason {
			case "videoChartNotFound":
				return &Error{"chart_unavailable", "YouTube has no popular chart for this country and category. Choose another combination", 404}
			case "quotaExceeded", "dailyLimitExceeded", "rateLimitExceeded":
				return &Error{"quota_exceeded", "YouTube quota is exhausted or temporarily limited. Check your Google Cloud quota before retrying", 429}
			case "keyInvalid", "accessNotConfigured", "ipRefererBlocked", "forbidden":
				return &Error{"key_rejected", "YouTube rejected this API key. Check its restrictions and enable YouTube Data API v3", 400}
			}
		}
		if resp.StatusCode == 400 || resp.StatusCode == 401 || resp.StatusCode == 403 {
			return &Error{"youtube_rejected", "YouTube rejected the request. Check the API key, API access, and search options", 400}
		}
		return &Error{"provider_error", "YouTube is unavailable. Try again later", 502}
	}
	if json.Unmarshal(body, out) != nil {
		return &Error{"provider_error", "YouTube returned an unexpected response", 502}
	}
	return nil
}
func ValidateQuery(q *Query) error {
	q.Text = strings.TrimSpace(q.Text)
	if q.Text == "" || len(q.Text) > 200 {
		return &Error{"invalid_query", "Enter a search of 1–200 characters", 400}
	}
	if q.Order == "" {
		q.Order = "relevance"
	}
	if q.Duration == "" {
		q.Duration = "short"
	}
	if q.Period == "" {
		q.Period = "any"
	}
	if !oneOf(q.Order, "relevance", "viewCount", "date") || !oneOf(q.Duration, "short", "any") || !oneOf(q.Period, "any", "7", "30", "90") || !oneOf(q.Language, "", "es", "en") || len(q.Page) > 512 {
		return &Error{"invalid_query", "Choose supported search options", 400}
	}
	return nil
}
func oneOf(s string, values ...string) bool {
	for _, v := range values {
		if s == v {
			return true
		}
	}
	return false
}
func (y *YouTube) Search(ctx context.Context, key string, q Query) (Page, error) {
	if err := ValidateQuery(&q); err != nil {
		return Page{}, err
	}
	values := url.Values{"part": {"snippet"}, "type": {"video"}, "maxResults": {"20"}, "q": {q.Text}, "order": {q.Order}, "videoDuration": {q.Duration}}
	if q.Page != "" {
		values.Set("pageToken", q.Page)
	}
	if q.Language != "" {
		values.Set("relevanceLanguage", q.Language)
	}
	if q.Period != "any" {
		var days int
		fmt.Sscan(q.Period, &days)
		values.Set("publishedAfter", time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339))
	}
	var search struct {
		Items []struct {
			ID struct {
				VideoID string `json:"videoId"`
			} `json:"id"`
		} `json:"items"`
		Next string `json:"nextPageToken"`
	}
	if err := y.get(ctx, key, "search", values, &search); err != nil {
		return Page{}, err
	}
	ids := make([]string, 0, len(search.Items))
	for _, item := range search.Items {
		if VideoID.MatchString(item.ID.VideoID) {
			ids = append(ids, item.ID.VideoID)
		}
	}
	result := Page{Items: []Video{}, NextPage: search.Next}
	if len(ids) == 0 {
		return result, nil
	}
	videos, err := y.Videos(ctx, key, ids)
	if err != nil {
		return Page{}, err
	}
	indexed := map[string]Video{}
	for _, v := range videos {
		indexed[v.ID] = v
	}
	for _, id := range ids {
		if v, ok := indexed[id]; ok {
			result.Items = append(result.Items, v)
		}
	}
	return result, nil
}
func (y *YouTube) Videos(ctx context.Context, key string, ids []string) ([]Video, error) {
	if len(ids) == 0 || len(ids) > 50 {
		return nil, &Error{"invalid_video", "Invalid video selection", 400}
	}
	for _, id := range ids {
		if !VideoID.MatchString(id) {
			return nil, &Error{"invalid_video", "Invalid YouTube video ID", 400}
		}
	}
	page, err := y.videoPage(ctx, key, url.Values{"part": {"snippet,contentDetails,statistics"}, "id": {strings.Join(ids, ",")}})
	return page.Items, err
}
func ValidateDiscovery(region, category, page string) error {
	if !countryCode.MatchString(region) || !oneOf(category, "0", "10", "20", "24") || len(page) > 512 {
		return &Error{"invalid_query", "Choose a two-letter country and a supported category", 400}
	}
	return nil
}
func (y *YouTube) Popular(ctx context.Context, key, region, category, page string) (Page, error) {
	if err := ValidateDiscovery(region, category, page); err != nil {
		return Page{}, err
	}
	values := url.Values{"part": {"snippet,contentDetails,statistics"}, "chart": {"mostPopular"}, "regionCode": {region}, "videoCategoryId": {category}, "maxResults": {"20"}}
	if page != "" {
		values.Set("pageToken", page)
	}
	return y.videoPage(ctx, key, values)
}
func (y *YouTube) videoPage(ctx context.Context, key string, values url.Values) (Page, error) {
	var response struct {
		Next  string `json:"nextPageToken"`
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Tags         []string  `json:"tags"`
				Title        string    `json:"title"`
				ChannelTitle string    `json:"channelTitle"`
				ChannelID    string    `json:"channelId"`
				Description  string    `json:"description"`
				PublishedAt  time.Time `json:"publishedAt"`
				Thumbnails   map[string]struct {
					URL string `json:"url"`
				} `json:"thumbnails"`
			} `json:"snippet"`
			ContentDetails struct {
				Duration string `json:"duration"`
			} `json:"contentDetails"`
			Statistics struct {
				Views    *uint64 `json:"viewCount,string"`
				Likes    *uint64 `json:"likeCount,string"`
				Comments *uint64 `json:"commentCount,string"`
			} `json:"statistics"`
		} `json:"items"`
	}
	if err := y.get(ctx, key, "videos", values, &response); err != nil {
		return Page{}, err
	}
	out := []Video{}
	for _, v := range response.Items {
		if !VideoID.MatchString(v.ID) {
			continue
		}
		thumb := v.Snippet.Thumbnails["medium"].URL
		u, e := url.Parse(thumb)
		if e != nil || u.Scheme != "https" || u.Host != "i.ytimg.com" {
			thumb = ""
		}
		out = append(out, Video{ID: v.ID, Tags: v.Snippet.Tags, Title: v.Snippet.Title, Channel: v.Snippet.ChannelTitle, ChannelID: v.Snippet.ChannelID, Description: v.Snippet.Description, Thumbnail: thumb, PublishedAt: v.Snippet.PublishedAt, ObservedAt: time.Now().UTC(), Duration: v.ContentDetails.Duration, Views: v.Statistics.Views, Likes: v.Statistics.Likes, Comments: v.Statistics.Comments})
	}
	return Page{Items: out, NextPage: response.Next}, nil
}
