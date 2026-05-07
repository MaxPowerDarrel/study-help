package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	_ "time/tzdata"

	"study-help/internal/auth"
	"study-help/internal/config"
	"study-help/internal/db"
	"study-help/internal/esv"
	"study-help/internal/highlights"
	"study-help/internal/scripture"
	"study-help/internal/server"
	"study-help/internal/youversion"
)

const metricsAddr = "127.0.0.1:9090"

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	counter := &server.ESVCallCounter{}
	dailyCounter := &server.DailyCounter{}
	reg := scripture.NewRegistry(
		scripture.ESV,
		esv.NewProvider(cfg.ESVAPIKey),
		youversion.NewProvider(cfg.YouVersionAppKey),
	)
	authSvc := auth.NewService(database, auth.Config{
		IsDev:      cfg.IsDev(),
		SessionTTL: 30 * 24 * time.Hour,
	})
	authLimiter := auth.NewLimiter()
	highlightsSvc := highlights.NewService(database, highlights.Config{Registry: reg})
	srv := server.New(cfg, database, counter, dailyCounter, reg, authSvc, authLimiter, highlightsSvc)
	metricsSrv := server.NewMetricsServer(metricsAddr, counter, dailyCounter)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	go func() {
		log.Printf("metrics listening on %s", metricsSrv.Addr)
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("metrics server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Print("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("metrics shutdown: %v", err)
	}
}
