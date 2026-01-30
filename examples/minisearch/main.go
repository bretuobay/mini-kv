package main

import (
	"log"

	"github.com/bretuobay/mini-kv"
)

// Placeholder for integration with MiniSearchDB.
func main() {
	db, err := minikv.Open(minikv.DefaultOptions("./minisearch"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_ = db.Set([]byte("doc:1"), []byte("metadata"))
}
