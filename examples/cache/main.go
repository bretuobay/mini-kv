package main

import (
	"log"
	"time"

	"github.com/bretuobay/mini-kv"
)

func main() {
	db, err := minikv.Open(minikv.DefaultOptions("./cache"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.SetWithTTL([]byte("token"), []byte("abc123"), 2*time.Second); err != nil {
		log.Fatal(err)
	}
	if ttl, err := db.TTL([]byte("token")); err == nil {
		log.Printf("ttl: %s", ttl)
	}
}
