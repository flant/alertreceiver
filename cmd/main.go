package main

import (
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"alertreceiver/pkg/config"
	"alertreceiver/pkg/logging"
	"alertreceiver/pkg/madison"
	"alertreceiver/pkg/webhook"

	log "github.com/sirupsen/logrus"
)

var (
	appVersion string
)

func initLog() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {
	initLog()

	if appVersion == "" {
		appVersion = "dev"
	}

	logger := logging.NewLogger()
	logger.Info("starting alertreceiver", log.Fields{
		"version":   appVersion,
		"timestamp": time.Now().Format(time.RFC3339),
	})

	ex, err := os.Executable()
	if err != nil {
		log.Fatal("failed to get executable path: ", err)
	}

	configDir := filepath.Dir(ex)
	configPath := filepath.Join(configDir, ".env")

	if err := config.LoadConfig(configPath); err != nil {
		log.Fatal("failed to load configuration: ", err)
	}

	cfg := config.GetConfig()
	madisonClient := madison.NewClient(cfg)
	handler := webhook.NewHandler(madisonClient, logger, cfg.Dms)

	mux := http.NewServeMux()
	mux.HandleFunc("/prometheus", handler.HandlePrometheus)
	mux.HandleFunc("/health", handler.HandleHealth)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	go func() {
		logger.Info("starting server", log.Fields{
			"port":      cfg.Port,
			"timestamp": time.Now().Format(time.RFC3339),
		})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("failed to start server: ", err)
		}
	}()

	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for range ticker.C {
			handler.SendDMS()
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	<-sigs
	logger.Info("shutting down", log.Fields{
		"timestamp": time.Now().Format(time.RFC3339),
	})
	ticker.Stop()
}
