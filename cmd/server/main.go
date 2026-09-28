package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"timetable-editor/internal/backend"
	"timetable-editor/internal/config"
	"timetable-editor/internal/database"
	"timetable-editor/internal/handler"
	"timetable-editor/internal/router"
)

// main loads configuration, wires the app layers together, and serves HTTP.
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	box := backend.NewSecretBox(cfg.MasterKey)
	store, err := database.Open(cfg.DatabasePath, box)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer store.Close()

	service := backend.NewService(store)
	webHandler, err := handler.New(service, cfg.PublicBaseURL)
	if err != nil {
		log.Fatalf("prepare web handlers: %v", err)
	}

	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		log.Fatalf("listen on %s: %v", cfg.Address, err)
	}
	server := &http.Server{
		Addr: cfg.Address, Handler: router.New(webHandler),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.Serve(listener) }()

	if cfg.OpenBrowser && cfg.PublicBaseURL == "" {
		if err := openBrowser(localURL(listener.Addr())); err != nil {
			log.Printf("could not open a browser: %v", err)
		}
	}
	log.Printf("timetable filter listening on %s", listener.Addr())
	if err := <-serverErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

// localURL builds a browser-friendly address for wildcard listeners.
func localURL(address net.Addr) string {
	host, port, err := net.SplitHostPort(address.String())
	if err != nil || host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
}
