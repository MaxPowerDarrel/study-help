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

	"study-help/internal/config"
	"study-help/internal/esv"
	"study-help/internal/scripture"
	"study-help/internal/server"
	"study-help/internal/youversion"
)

const metricsAddr = "127.0.0.1:9090"

func main() {
	cfg := config.Load()

	counter := &server.ESVCallCounter{}
	dailyCounter := &server.DailyCounter{}
	reg := scripture.NewRegistry(
		scripture.ESV,
		esv.NewProvider(cfg.ESVAPIKey),
		youversion.NewProvider(cfg.YouVersionAppKey),
	)
	srv := server.New(cfg, counter, dailyCounter, reg)
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
