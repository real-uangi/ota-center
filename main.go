package main

import (
	"log"
	"net/http"
)

func main() {
	cfg, err := loadConfigFromEnv()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	store := NewStore(cfg.DataDir)
	server := NewServer(cfg, store)

	log.Printf("ota-center listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, server.routes()); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
