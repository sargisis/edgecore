package main

import (
	"context"
	"edgecore/internal/backend"
	"edgecore/internal/balancer"
	"edgecore/internal/config"
	"edgecore/internal/devtools"
	"edgecore/internal/proxy"
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pterm/pterm"
)

var (
	serverPool  balancer.ServerPool
	rateLimiter *proxy.RateLimiter
	devMode     = flag.Bool("dev", false, "Start test backends automatically")
)

func lbHandler(w http.ResponseWriter, r *http.Request) {
	peer := serverPool.GetLeastConnections()
	if peer != nil {
		peer.IncConnections()
		defer peer.DecConnections()
		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

func loadConfig(cfg *config.Config) {
	serverPool.Clear()
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}

	spinner, _ := pterm.DefaultSpinner.Start("Loading backends...")

	for _, target := range cfg.Backends {
		serverUrl, err := url.Parse(target)
		if err != nil {
			pterm.Error.Printf("Invalid backend URL %s: %v\n", target, err)
			continue
		}

		proxy := httputil.NewSingleHostReverseProxy(serverUrl)
		proxy.Transport = transport
		proxy.ErrorHandler = func(writer http.ResponseWriter, request *http.Request, e error) {
			pterm.Warning.Printf("[%s] %s\n", serverUrl.Host, e.Error())
			writer.WriteHeader(http.StatusBadGateway)
		}

		b := backend.NewBackend(serverUrl, proxy)
		serverPool.AddBackend(b)
		pterm.Success.Printf("Registered backend: %s\n", serverUrl)
	}

	spinner.Success("All backends loaded!")
}

func main() {
	flag.Parse()

	// ASCII Banner
	pterm.DefaultHeader.WithFullWidth().
		WithBackgroundStyle(pterm.NewStyle(pterm.BgLightBlue)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack)).
		Println("EdgeCore Load Balancer")

	pterm.Info.Println("High-Performance HTTP Load Balancer")
	pterm.Println()

	// 1. Dev Mode: Start test backends
	if *devMode {
		pterm.DefaultBox.WithTitle("Development Mode").
			WithTitleTopCenter().
			Println("Starting test backend servers...")

		devtools.StartAllBackends()
		time.Sleep(500 * time.Millisecond)
		pterm.Success.Println("Test backends ready on :8081, :8082, :8083")
		pterm.Println()
	}

	// 2. Load Config
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		pterm.Fatal.Printf("Failed to load config: %v\n", err)
	}

	loadConfig(cfg)
	rateLimiter = proxy.NewRateLimiter(cfg.RateLimit, cfg.Burst)

	// 3. Setup Signal Handling for Hot-reload + Graceful Shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for sig := range sigs {
			if sig == syscall.SIGHUP {
				pterm.Info.Println("📡 Received SIGHUP, reloading configuration...")
				newCfg, err := config.LoadConfig("config.json")
				if err == nil {
					loadConfig(newCfg)
				}
			} else {
				pterm.Warning.Println("🛑 Shutting down gracefully...")
				return
			}
		}
	}()

	// 4. Start Health Check loop
	go func() {
		t := time.NewTicker(time.Second * 30)
		for range t.C {
			serverPool.HealthCheck()
		}
	}()

	// 5. Setup Middleware Chain
	handler := http.HandlerFunc(lbHandler)
	finalHandler := proxy.Logger(proxy.RateLimitMiddleware(rateLimiter, handler))

	// 6. Setup HTTP Server with Metrics endpoint
	mux := http.NewServeMux()
	mux.Handle("/", finalHandler)
	mux.HandleFunc("/metrics", proxy.PrometheusMetrics)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}

	pterm.Println()
	pterm.DefaultBox.WithTitle("🚀 Server Started").
		WithTitleTopCenter().
		WithBoxStyle(pterm.NewStyle(pterm.FgLightGreen)).
		Printfln("Port: %d\nRate Limit: %.0f req/s\nMetrics: http://localhost:%d/metrics\nHealth: http://localhost:%d/health",
			cfg.Port, cfg.RateLimit, cfg.Port, cfg.Port)

	if *devMode {
		pterm.Info.Println("💡 Dev Mode is ON. Press Ctrl+C to stop everything.")
	}
	pterm.Println()

	// Graceful shutdown with context
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			pterm.Fatal.Printf("Server error: %v\n", err)
		}
	}()

	// Wait for interrupt signal
	<-sigs
	pterm.Warning.Println("🛑 Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		pterm.Error.Printf("Server shutdown error: %v\n", err)
	}
	pterm.Success.Println("✅ EdgeCore stopped")
}
