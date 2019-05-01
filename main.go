package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
	"github.com/kmollee/tophn/hn"
)

const (
	defaultNumStories = 15
	defaultPort       = 3000
)

var (
	port       int
	numStories int
)

func init() {
	p := os.Getenv("PORT")
	if p == "" {
		port = defaultPort
	} else {
		v, err := strconv.Atoi(p)
		if err != nil {
			log.Fatal(err)
		}
		port = v
	}

	n := os.Getenv("numStories")
	if n == "" {
		numStories = defaultNumStories
	} else {
		v, err := strconv.Atoi(n)
		if err != nil {
			log.Fatal(err)
		}
		numStories = v
	}

}

func main() {
	log.Printf("listen on:%d", port)
	db, err := bolt.Open("my.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	hdb := &hnDB{db: db, dates: make([]string, 0)}
	err = hdb.init()
	if err != nil {
		log.Fatal(err)
	}

	tmpl := template.Must(template.ParseFiles("template/base.html", "template/index.html", "template/nav.html"))
	tmplList := template.Must(template.ParseFiles("template/base.html", "template/list.html", "template/nav.html"))
	client := hn.NewClient(http.DefaultClient)
	// c := &cacheClient{client: client, numStories: numStories, f: hn.OnlyStory, duration: 15 * time.Minute}
	c := newCacheHnClient(client, numStories, hn.OnlyStory, 15*time.Minute, hdb)
	go c.refresh()

	r := mux.NewRouter()
	r.HandleFunc("/", c.index(tmpl))
	r.HandleFunc("/archive/{page:[0-9]+}", hdb.List(tmplList))
	r.HandleFunc("/archive", hdb.List(tmplList))
	r.HandleFunc("/archive/date/{date}", hdb.Get(tmpl))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), r); err != nil {
		log.Fatal(err)
	}
}

type hnCacheClient struct {
	client      *hn.Client
	numStories  int
	f           hn.Filter
	rwlock      sync.RWMutex
	duration    time.Duration
	items       []*hn.Item
	err         chan error
	refreshTime time.Time
	db          *hnDB
}

func newCacheHnClient(c *hn.Client, num int, f hn.Filter, cacheTime time.Duration, db *hnDB) *hnCacheClient {
	return &hnCacheClient{
		client:     c,
		numStories: num,
		f:          f,
		duration:   cacheTime,
		err:        make(chan error),
		db:         db,
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

			err := tmpl.Execute(w, map[string]interface{}{
				"Stories": items,
				"Time":    time.Now().Sub(start),
				"Title":   "Index",
			})
			if err != nil {
				http.Error(w, "Failed to process the template", http.StatusInternalServerError)
				return
			}
		}
	}

}

func getToday() time.Time {
	y := time.Now().Year()  //年
	m := time.Now().Month() //月
	d := time.Now().Day()   //日
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

func (h *hnCacheClient) refresh() {
	ticker := time.Tick(h.duration)
	today := getToday()
	daily := make(chan string)

	// calculate is next day
	go func() {
		t := time.Tick(time.Minute)
		nextDay := today.AddDate(0, 0, 1)
		for {
			if time.Now().After(nextDay) {
				daily <- nextDay.Format("2006-01-02")
				nextDay = nextDay.AddDate(0, 0, 1)
			}
			<-t
		}
	}()

	update := func() {
		items, err := h.client.GetItems(h.numStories, h.f)
		if err != nil {
			log.Printf("ERR: %v", err)
			h.err <- err
			return
		}
		h.rwlock.Lock()
		h.items = items
		h.rwlock.Unlock()
	}

	update()

	for {
		select {
		case date := <-daily:
			b, err := hn.ItemsEncode(h.items)
			if err != nil {
				log.Fatal(err)
			}
			err = h.db.update(date, b)
			if err != nil {
				log.Fatal(err)
			}
			err = h.db.refresh()
			if err != nil {
				log.Fatal(err)
			}
		case <-ticker:
			update()
		}
	}
}

type templateData struct {
	Stories []*hn.Item
	Time    time.Duration
}
