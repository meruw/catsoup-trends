package research

import "time"

type Video struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Channel     string    `json:"channel"`
	ChannelID   string    `json:"channelId"`
	Description string    `json:"description"`
	Thumbnail   string    `json:"thumbnail"`
	PublishedAt time.Time `json:"publishedAt"`
	ObservedAt  time.Time `json:"observedAt"`
	Duration    string    `json:"duration"`
	Views       *uint64   `json:"views"`
	Likes       *uint64   `json:"likes"`
	Comments    *uint64   `json:"comments"`
}
type Reference struct {
	Analysis Analysis  `json:"analysis"`
	ID       string    `json:"id"`
	Video    *Video    `json:"video"`
	Note     string    `json:"note"`
	SavedAt  time.Time `json:"savedAt"`
}
type Page struct {
	Items    []Video `json:"items"`
	NextPage string  `json:"nextPage"`
}
type Query struct {
	Text     string
	Order    string
	Duration string
	Period   string
	Language string
	Page     string
}
type Error struct {
	Code    string
	Message string
	Status  int
}

func (e *Error) Error() string { return e.Message }
