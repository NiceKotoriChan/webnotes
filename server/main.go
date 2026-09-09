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
	dataDir := flag.String("data", "./data", "data directory (webnotes root)")
	flag.Parse()

	s, err := store.New(*dataDir)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	a := asset.New()

	// 逐仓库对账附件状态（清孤儿 tmp、uploading→ready、deleting 收尾）
	for _, ri := range s.ListRepos() {
		db, err := s.OpenRepo(ri.ID)
		if err != nil {
			log.Printf("reconcile %s: open: %v", ri.ID, err)
			continue
		}
		if err := a.Reconcile(db, s.RepoDir(ri.ID)); err != nil {
			log.Printf("reconcile %s: %v", ri.ID, err)
		}
		db.Close()
	}

	r := api.NewRouter(s, a)
	log.Printf("listening on %s, data=%s", *addr, *dataDir)
	log.Fatal(http.ListenAndServe(*addr, r))
}
