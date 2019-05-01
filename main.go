package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
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
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		stories, err := client.GetItems(numStories, hn.OnlyStory)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		data := templateData{
			Stories: stories,
			Time:    time.Now().Sub(start),
		}
		err = tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, "Failed to process the template", http.StatusInternalServerError)
			return
		}

	}
}

type templateData struct {
	Stories []*hn.Item
	Time    time.Duration
}
