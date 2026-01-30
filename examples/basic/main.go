package main

import (
	"log"

	"mini-kv"
)

func main() {
	db, err := minikv.Open(minikv.DefaultOptions("./data"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Set([]byte("hello"), []byte("world")); err != nil {
		log.Fatal(err)
	}
	value, err := db.Get([]byte("hello"))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s", value)
}
