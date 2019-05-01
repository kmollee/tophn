/*Package hn :implement hackernew api client*/
package hn

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	apiBase    = "https://hacker-news.firebaseio.com/v0"
	UnLimitIDs = -1
)

// Item :represnet single itme of HN API.
// type could be "story", "job", "comment"
type Item struct {
	Author      string `json:"by"`
	ID          int    `json:"id"`
	Score       int    `json:"score"`
	Title       string `json:"title"`
	Time        int    `json:"time"`
	Type        string `json:"type"`
	Descendants int    `json:"descendants"`

	// Only one of these should exist
	Text string `json:"text"`
	URL  string `json:"url"`
}

// Client is an API client used to interact with the Hacker News API
type Client struct {
	client *http.Client
	apiURL string
}

// NewClient :create a HN API client with http client
func NewClient(c *http.Client) *Client {
	return &Client{client: c, apiURL: apiBase}
}

func fakeNewClient(c *http.Client, url string) *Client {
	return &Client{client: c, apiURL: url}
}

// GetItemIDs :return n amount of HN item ids
func (c *Client) GetItemIDs(n int) ([]int, error) {
	story := fmt.Sprintf("%s/%s", c.apiURL, "topstories.json")
	resp, err := c.client.Get(story)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	d := json.NewDecoder(resp.Body)
	var items []int
	err = d.Decode(&items)
	if err != nil {
		return nil, err
	}

	if n == UnLimitIDs {
		return items, nil
	}
	return items[:n], nil
}

// GetItem :return sepecific HN item
func (c *Client) GetItem(id int) (*Item, error) {

	// u := path.Join(c.apiURL, "item", string(id))
	u := fmt.Sprintf("%s/item/%d.json", c.apiURL, id)
	resp, err := c.client.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	d := json.NewDecoder(resp.Body)
	var item Item
	err = d.Decode(&item)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// Filter : filter items
type Filter func(item *Item) bool

// OnlyStory :filter out only story
func OnlyStory(item *Item) bool {
	return item.Type == "story" && item.URL != ""
}

// GetItems :get HN n item's that fit fitler
func (c *Client) GetItems(n int, f Filter) ([]*Item, error) {
	ids, err := c.GetItemIDs(UnLimitIDs)
	if err != nil {
		return nil, err
	}

	items := make([]*Item, n)
	at := 0
	for _, id := range ids {
		item, err := c.GetItem(id)
		if err != nil {
			return nil, err
		}
		if f(item) {
			items[at] = item
			at++
		}
		if at == n {
			return items, nil
		}
	}
	return items, nil
}

func ItemsEncode(items []*Item) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(items)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func ItemsDecode(b []byte) ([]Item, error) {
	r := bytes.NewReader(b)
	dec := gob.NewDecoder(r)
	var items []Item
	err := dec.Decode(&items)
	if err != nil {
		return nil, err
	}
	return items, nil
}
