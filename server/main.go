package main

import (
	"flag"
	"log"
	"net/http"

	"webnotes/server/api"
	"webnotes/server/asset"
	"webnotes/server/store"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dataDir := flag.String("data", "./data", "data directory")
	flag.Parse()

	s, err := store.New(*dataDir)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	a := asset.New(s)
	if err := a.EnsureSameDir(); err != nil {
		log.Fatalf("asset dir: %v", err)
	}
	if err := a.Reconcile(); err != nil {
		log.Printf("reconcile warning: %v", err)
	}

	r := api.NewRouter(s, a)
	log.Printf("listening on %s, data=%s", *addr, *dataDir)
	log.Fatal(http.ListenAndServe(*addr, r))
}
