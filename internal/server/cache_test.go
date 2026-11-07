package server

import (
	"github.com/meruw/catsoup-trends/internal/research"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCachedResultsCanBeSavedAfterCandidateEviction(t *testing.T) {
	app, host := setup(t)
	page := research.Page{Items: []research.Video{{ID: "fixture0001", ObservedAt: time.Now()}}}
	app.searchReply(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil), page, true)
	r := request(t, app, host, "POST", "/api/v1/references", `{"id":"fixture0001"}`)
	if r.StatusCode != 200 {
		t.Fatal("cached video cannot be saved")
	}
}
