package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
	"github.com/kmollee/tophn/hn"
)

const (
	bucketName  = "Bucket"
	itemPerPage = 10
)

type hnDB struct {
	db    *bolt.DB
	dates []string
}

func (h *hnDB) init() error {
	err := h.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return h.refresh()
}

func (h *hnDB) List(tmpl *template.Template) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		p := mux.Vars(r)["page"]
		dates := h.dates
		if len(p) > 0 {
			page, err := strconv.Atoi(p)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			log.Printf("Page: %d", page)
			dates = h.paginate(itemPerPage*page, itemPerPage)
		}

		err := tmpl.Execute(w, map[string]interface{}{
			"dates": dates,
			"Title": "list",
		})
		if err != nil {
			http.Error(w, "Failed to process the template", http.StatusInternalServerError)
			return
		}
	}
}

func (h *hnDB) Get(tmpl *template.Template) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		date := mux.Vars(r)["date"]
		b, err := h.read(date)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		items, err := hn.ItemsDecode(b)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = tmpl.Execute(w, map[string]interface{}{
			"Stories": items,
			"Time":    time.Now().Sub(start),
			"Title":   fmt.Sprintf("Date: %s", date),
			"date":    date,
		})
		if err != nil {
			http.Error(w, "Failed to process the template", http.StatusInternalServerError)
			return
		}
	}
}

func (h *hnDB) read(date string) ([]byte, error) {
	var v []byte
	err := h.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		v = b.Get([]byte(date))
		if v == nil {
			return fmt.Errorf("key %s is not exist", date)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (h *hnDB) update(date string, v []byte) error {
	err := h.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		err := b.Put([]byte(date), v)
		return err
	})
	return err
}

func (h *hnDB) refresh() error {

	dates := []string{}

	err := h.db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte(bucketName))
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			dates = append(dates, string(k))
		}

		return nil
	})
	if err != nil {
		return err
	}
	h.dates = dates
	return nil
}

func (h *hnDB) paginate(skip int, size int) []string {
	if skip > len(h.dates) {
		skip = len(h.dates)
	}

	end := skip + size
	if end > len(h.dates) {
		end = len(h.dates)
	}
	return h.dates[skip:end]
}

// func pager(pageNo int) {
// 	var start int = pageNo/(PSIZE-1)*(PSIZE-1) + 1
// 	if pageNo%(PSIZE-1) == 0 {
// 		start -= PSIZE - 1
// 	}

// 	for i := start; i < start+PSIZE; i++ {
// 		if i == pageNo {
// 			fmt.Printf("(%d) ", i)
// 		} else {
// 			fmt.Printf("%d ", i)
// 		}
// 	}
// 	fmt.Print("\n")
// }
