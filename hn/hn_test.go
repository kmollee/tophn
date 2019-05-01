package hn

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testingHTTPClient(handler http.Handler) (string, *http.Client, func()) {
	s := httptest.NewServer(handler)

	cli := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, s.Listener.Addr().String())
			},
		},
	}

	return s.URL, cli, s.Close
}

func TestGetHNItems(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		l := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		enc := json.NewEncoder(w)
		err := enc.Encode(l)
		assert.Nil(t, err)
	}))
	defer ts.Close()

	// Use Client & URL from our local test server
	client := ts.Client()
	api := fakeNewClient(client, ts.URL)
	items, err := api.GetItemIDs(5)
	if err != nil {
		t.Errorf("could not get item: %v", err)
	}

	expect := []int{0, 1, 2, 3, 4}
	assert.Equal(t, expect, items)
}

func TestGetItem(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		item := `{
  "by" : "dhouston",
  "descendants" : 71,
  "id" : 8863,
  "kids" : [ 8952, 9224, 8917, 8884, 8887, 8943, 8869, 8958, 9005, 9671, 8940, 9067, 8908, 9055, 8865, 8881, 8872, 8873, 8955, 10403, 8903, 8928, 9125, 8998, 8901, 8902, 8907, 8894, 8878, 8870, 8980, 8934, 8876 ],
  "score" : 111,
  "time" : 1175714200,
  "title" : "My YC app: Dropbox - Throw away your USB drive",
  "type" : "story",
  "url" : "http://www.getdropbox.com/u/2/screencast.html"
}`
		fmt.Fprint(w, item)
	}))
	defer ts.Close()

	// Use Client & URL from our local test server
	client := ts.Client()
	api := fakeNewClient(client, ts.URL)
	item, err := api.GetItem(8863)
	if err != nil {
		t.Errorf("could not get item: %v", err)
	}

	expect := Item{
		Author: "dhouston",
		ID:     8863,
		Score:  111,
		Title:  "My YC app: Dropbox - Throw away your USB drive",
		Time:   1175714200,
		Type:   "story",
		URL:    "http://www.getdropbox.com/u/2/screencast.html",
	}

	assert.Equal(t, expect, *item)
}
