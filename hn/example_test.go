package hn_test

import (
	"fmt"
	"net/http"

	"github.com/kmollee/tophn/hn"
)

func ExampleClient() {

	client := hn.NewClient(http.DefaultClient)
	ids, err := client.GetItems(5)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 5; i++ {
		item, err := client.GetItem(ids[i])
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s (by %s)\n", item.Title, item.Author)
	}
}
