package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"text/template"
	"time"

	"github.com/gorilla/mux"
	"github.com/kmollee/tophn/hn"
)

const (
	numStories = 15
)

func main() {

	port := flag.Int("p", 3000, "the port to start the web server on")
	flag.Parse()

	log.Printf("start listen on: %d", *port)

	tmpl := template.Must(template.ParseFiles("template/index.html"))
	client := hn.NewClient(http.DefaultClient)
	// c := &cacheClient{client: client, numStories: numStories, f: hn.OnlyStory, duration: 15 * time.Minute}
	c := newCacheHnClient(client, numStories, hn.OnlyStory, 15*time.Minute)
	go c.refresh()

	r := mux.NewRouter()
	r.HandleFunc("/", c.index(tmpl))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), r); err != nil {
		log.Fatal(err)
	}
}

type hnCacheClient struct {
	client     *hn.Client
	numStories int
	f          hn.Filter
	rwlock     sync.RWMutex
	duration   time.Duration
	items      []*hn.Item
	err        chan error
}

func newCacheHnClient(c *hn.Client, num int, f hn.Filter, cacheTime time.Duration) *hnCacheClient {
	return &hnCacheClient{
		client:     c,
		numStories: num,
		f:          f,
		duration:   cacheTime,
		err:        make(chan error),
	}

}

func (h *hnCacheClient) index(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case err := <-h.err:
			log.Printf("ERR: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		default:
			start := time.Now()

			var items []*hn.Item
			h.rwlock.RLock()
			items = h.items
			h.rwlock.RUnlock()

			data := templateData{
				Stories: items,
				Time:    time.Now().Sub(start),
			}
			err := tmpl.Execute(w, data)
			if err != nil {
				http.Error(w, "Failed to process the template", http.StatusInternalServerError)
				return
			}
		}
	}

}

func (h *hnCacheClient) refresh() {
	ticker := time.Tick(h.duration)
	for {
		items, err := h.client.GetItems(h.numStories, h.f)
		if err != nil {
			log.Printf("ERR: %v", err)
			h.err <- err
			return
		}
		h.rwlock.Lock()
		h.items = items
		h.rwlock.Unlock()
		<-ticker
	}
}

type templateData struct {
	Stories []*hn.Item
	Time    time.Duration
}
