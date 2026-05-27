package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/dobezor/turnable-manager-panel/internal/app"
)

func main() {
	configPath := flag.String("config", "/etc/turnable-manager-panel/config.json", "panel config path")
	flag.Parse()

	cfg, err := app.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	manager, err := app.New(cfg)
	if err != nil {
		log.Fatalf("init manager: %v", err)
	}

	log.Printf("turnable-manager-panel listening on %s", cfg.ListenAddress)
	if err := http.ListenAndServe(cfg.ListenAddress, manager.Routes()); err != nil {
		log.Fatal(err)
	}
}
