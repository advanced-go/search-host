package main

import (
	"context"
	"fmt"
	"github.com/advanced-go/search-host/initialize"
	"github.com/advanced-go/stdlib/core"
	"github.com/advanced-go/stdlib/host"
	"github.com/advanced-go/stdlib/httpx"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"time"
)

const (
	portKey                 = "PORT"
	addr                    = "0.0.0.0:8084"
	writeTimeout            = time.Second * 300
	readTimeout             = time.Second * 15
	idleTimeout             = time.Second * 60
	healthLivelinessPattern = "/health/liveness"
	healthReadinessPattern  = "/health/readiness"
)

func main() {
	//os.Setenv(portKey, "0.0.0.0:8082")
	port := os.Getenv(portKey)
	if port == "" {
		port = addr
	}
	start := time.Now()
	displayRuntime(port)
	handler, ok := startup(http.NewServeMux())
	if !ok {
		os.Exit(1)
	}
	fmt.Println(fmt.Sprintf("started : %v", time.Since(start)))
	srv := http.Server{
		Addr: port,
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: writeTimeout,
		ReadTimeout:  readTimeout,
		IdleTimeout:  idleTimeout,
		Handler:      handler,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP server Shutdown: %v", err)
		} else {
			log.Printf("HTTP server Shutdown")
		}
		close(idleConnsClosed)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Fatalf("HTTP server ListenAndServe: %v", err)
	}
	<-idleConnsClosed
}

func displayRuntime(port string) {
	fmt.Printf("addr    : %v\n", port)
	fmt.Printf("vers    : %v\n", runtime.Version())
	fmt.Printf("os      : %v\n", runtime.GOOS)
	fmt.Printf("arch    : %v\n", runtime.GOARCH)
	fmt.Printf("cpu     : %v\n", runtime.NumCPU())
	fmt.Printf("env     : %v\n", core.EnvStr())
}

func startup(r *http.ServeMux) (http.Handler, bool) {
	// Initialize logging
	initialize.Logging()

	// Initialize configuration and host startup
	if !initialize.Startup() {
		return r, false
	}

	// Initialize host proxy for all HTTP handlers,and add intermediaries
	err := initialize.Host()
	if err != nil {
		log.Printf(err.Error())
		return r, false
	}

	// Initialize health handlers
	r.Handle(healthLivelinessPattern, http.HandlerFunc(healthLivelinessHandler))
	r.Handle(healthReadinessPattern, http.HandlerFunc(healthReadinessHandler))

	// Route all other requests to host proxy
	r.Handle("/", http.HandlerFunc(host.HttpHandler))
	return r, true
}

func healthLivelinessHandler(w http.ResponseWriter, r *http.Request) {
	writeHealthResponse(w, core.StatusOK())
}

func healthReadinessHandler(w http.ResponseWriter, r *http.Request) {
	writeHealthResponse(w, core.StatusOK())

}

func writeHealthResponse(w http.ResponseWriter, status *core.Status) {
	if status.OK() {
		httpx.WriteResponse[core.Log](w, nil, status.HttpCode(), []byte("up"), nil)
	} else {
		httpx.WriteResponse[core.Log](w, nil, status.HttpCode(), nil, nil)
	}
}
