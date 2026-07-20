package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/tobydoescode/unifi-api-exporter/internal/collector"
	"github.com/tobydoescode/unifi-api-exporter/internal/config"
	"github.com/tobydoescode/unifi-api-exporter/internal/unifi"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	client, err := unifi.New(cfg.URL, cfg.User, cfg.Pass, cfg.Site, cfg.Insecure)
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	col := collector.New(reg, cfg.Site)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go poll(ctx, client, col, cfg.PollInterval, cfg.ScrapeTimeout())

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{Addr: cfg.ListenAddr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}

	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	}()

	log.Printf("listening on %s, polling %s every %s", cfg.ListenAddr, cfg.URL, cfg.PollInterval)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
	<-shutdownDone
}

func poll(ctx context.Context, client *unifi.Client, col *collector.Collector, interval, timeout time.Duration) {
	scrape := func() {
		start := time.Now()
		cctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		devs, err := client.Devices(cctx)
		col.Observe(devs, time.Since(start), err)
		if err != nil {
			log.Printf("scrape error: %v", err)
		}
	}
	scrape()
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			scrape()
		}
	}
}
