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
	numStories = 10
)

func main() {

	port := flag.Int("p", 3000, "the port to start the web server on")
	flag.Parse()

	log.Printf("start listen on: %d", *port)

	tmpl := template.Must(template.ParseFiles("template/index.html"))
	r := mux.NewRouter()
	r.HandleFunc("/", index(tmpl))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), r); err != nil {
		log.Fatal(err)
	}
}

func index(tmpl *template.Template) http.HandlerFunc {
	client := hn.NewClient(http.DefaultClient)
	c := &cacheClient{client: client, numStories: numStories, f: hn.OnlyStory, duration: 15 * time.Minute}
	go c.refresh()

	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case err := <-c.err:
			log.Printf("could not refresh: %v", err)
			http.Error(w, "Failed to update HN", http.StatusInternalServerError)
		default:
			start := time.Now()
			stories := c.GetItems()
			data := templateData{
				Stories: stories,
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

type cacheClient struct {
	client      *hn.Client
	numStories  int
	f           hn.Filter
	rwlock      sync.RWMutex
	duration    time.Duration
	refreshTime time.Time
	items       []*hn.Item
	err         chan error
}

func (c *cacheClient) refresh() {
	ticker := time.Tick(c.duration)
	for {
		items, err := c.client.GetItems(c.numStories, c.f)
		if err != nil {
			log.Printf("ERR: %v", err)
			c.err <- err
			return
		}
		c.rwlock.Lock()
		c.items = items
		c.rwlock.Unlock()
		<-ticker
	}

}

func (c *cacheClient) GetItems() []*hn.Item {
	c.rwlock.RLock()
	defer c.rwlock.RUnlock()

	return c.items
}

type templateData struct {
	Stories []*hn.Item
	Time    time.Duration
}
