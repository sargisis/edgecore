package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pterm/pterm"

	"github.com/sargisis/edgecore/internal/backend"
	"github.com/sargisis/edgecore/internal/balancer"
	"github.com/sargisis/edgecore/internal/config"
	"github.com/sargisis/edgecore/internal/devtools"
	"github.com/sargisis/edgecore/internal/proxy"
)

var (
	serverPool   balancer.ServerPool
	rateLimiter  *proxy.RateLimiter
	devMode      = flag.Bool("dev", false, "Start test backends automatically")
	configPath   *string
	shutdownChan = make(chan struct{})
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
	// Allow overriding config path via environment variable with CLI flag taking precedence.
	cfgEnv := os.Getenv("EDGECORE_CONFIG")
	defaultConfigPath := "config.json"
	if cfgEnv != "" {
		defaultConfigPath = cfgEnv
	}
	configPath = flag.String("config", defaultConfigPath, "Path to configuration file")

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
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		pterm.Fatal.Printf("Failed to load config: %v\n", err)
	}

	if err := cfg.Validate(); err != nil {
		pterm.Fatal.Printf("Invalid config: %v\n", err)
	}

	loadConfig(cfg)
	rateLimiter = proxy.NewRateLimiter(cfg.RateLimit, cfg.Burst)

	// 3. Setup Signal Handling for Hot-reload + Graceful Shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for sig := range sigs {
			switch sig {
			case syscall.SIGHUP:
				pterm.Info.Println("📡 Received SIGHUP, reloading configuration...")
				newCfg, err := config.LoadConfig(*configPath)
				if err != nil {
					pterm.Error.Printf("Failed to reload config: %v\n", err)
					continue
				}
				if err := newCfg.Validate(); err != nil {
					pterm.Error.Printf("Reloaded config is invalid: %v\n", err)
					continue
				}
				loadConfig(newCfg)
			case syscall.SIGINT, syscall.SIGTERM:
				pterm.Warning.Println("🛑 Shutting down gracefully...")
				close(shutdownChan)
				return
			}
		}
	}()

	// 4. Start Health Check loop with graceful stop and slight jitter
	go func() {
		// Add small random jitter before starting health checks to avoid thundering herd
		jitter := time.Duration(rand.Intn(5000)) * time.Millisecond
		time.Sleep(jitter)

		t := time.NewTicker(time.Second * 30)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				serverPool.HealthCheck()
			case <-shutdownChan:
				return
			}
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
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           mux,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
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

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			pterm.Fatal.Printf("Server error: %v\n", err)
		}
	}()

	// Wait for shutdown signal
	<-shutdownChan

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		pterm.Error.Printf("Server shutdown error: %v\n", err)
	}
	pterm.Success.Println("✅ EdgeCore stopped")
}
