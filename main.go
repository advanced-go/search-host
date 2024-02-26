package main

import (
	"context"
	"fmt"
	"github.com/advanced-go/core/access"
	"github.com/advanced-go/core/host"
	"github.com/advanced-go/core/http2"
	runtime2 "github.com/advanced-go/core/runtime"
	"github.com/advanced-go/search/provider"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"time"
)

const (
	portKey                 = "PORT"
	addr                    = "0.0.0.0:8081"
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
	fmt.Printf("env     : %v\n", runtime2.EnvStr())
}

func startup(r *http.ServeMux) (http.Handler, bool) {
	// Override access logger
	access.SetLogger(logger)

	// Run host startup where all registered resources/packages will be sent a startup configuration message
	m := createPackageConfiguration()
	if !host.Startup(time.Second*4, m) {
		return r, false
	}

	// Initialize messaging proxy for all HTTP handlers,and add an authorization intermediary
	host.RegisterHandler(provider.PkgPath, host.NewIntermediary(AuthHandler, provider.HttpHandler))

	// Initialize health handlers
	r.Handle(healthLivelinessPattern, http.HandlerFunc(healthLivelinessHandler))
	r.Handle(healthReadinessPattern, http.HandlerFunc(healthReadinessHandler))

	// Route all other requests to host proxy
	r.Handle("/", http.HandlerFunc(host.HttpHandler))

	// Add host metrics handler for ingress access logging
	return host.HttpHostMetricsHandler(r, ""), true
}

// TO DO : create package configuration information for startup
func createPackageConfiguration() host.ContentMap {
	return make(host.ContentMap)
}

func healthLivelinessHandler(w http.ResponseWriter, r *http.Request) {
	var status = runtime2.StatusOK()
	if status.OK() {
		http2.WriteResponse[runtime2.Log](w, []byte("up"), status, nil)
	} else {
		http2.WriteResponse[runtime2.Log](w, nil, status, nil)
	}
}

func healthReadinessHandler(w http.ResponseWriter, r *http.Request) {
	var status = runtime2.StatusOK()
	if status.OK() {
		http2.WriteResponse[runtime2.Log](w, []byte("up"), status, nil)
	} else {
		http2.WriteResponse[runtime2.Log](w, nil, status, nil)
	}
}

func logger(o *access.Origin, traffic string, start time.Time, duration time.Duration, req *http.Request, resp *http.Response, routeName, routeTo string, threshold int, thresholdFlags string) {
	if req == nil {
		req, _ = http.NewRequest("", "https://somehost.com/search?q=test", nil)
	}
	encoding := ""
	resp, encoding = Encoding(resp)
	url, _, _ := access.CreateUrlHostPath(req)
	s := fmt.Sprintf("{"+
		//"\"region\":%v, "+
		//"\"zone\":%v, "+
		//"\"sub-zone\":%v, "+
		//"\"app\":%v, "+
		//"\"instance-id\":%v, "+
		"\"traffic\":\"%v\", "+
		"\"start\":%v, "+
		"\"duration\":%v, "+
		//"\"request-id\":%v, "+
		//"\"relates-to\":%v, "+
		//"\"proto\":%v, "+
		"\"method\":%v, "+
		"\"uri\":%v, "+
		"\"query\":%v, "+
		//"\"host\":%v, "+
		//"\"path\":%v, "+
		"\"status-code\":%v, "+
		"\"bytes-written\":%v, "+
		"\"encoding\":%v, "+
		"\"route\":%v, "+
		//"\"route-to\":%v, "+
		"\"threshold\":%v, "+
		"\"threshold-flags\":%v }",
		//access.FmtJsonString(o.Region),
		//access.FmtJsonString(o.Zone),
		//access.FmtJsonString(o.SubZone),
		//access.FmtJsonString(o.App),
		//access.FmtJsonString(o.InstanceId),

		traffic,
		access.FmtTimestamp(start),
		strconv.Itoa(access.Milliseconds(duration)),

		//access.FmtJsonString(req.Header.Get(runtime2.XRequestId)),
		//access.FmtJsonString(req.Header.Get(runtime2.XRelatesTo)),
		//access.FmtJsonString(req.Proto),
		access.FmtJsonString(req.Method),
		access.FmtJsonString(url),
		access.FmtJsonString(req.URL.RawQuery),
		//access.FmtJsonString(host),
		//access.FmtJsonString(path),

		resp.StatusCode,
		//access.FmtJsonString(resp.Status),
		access.FmtJsonString(encoding),
		fmt.Sprintf("%v", resp.ContentLength),
		access.FmtJsonString(routeName),
		//access.FmtJsonString(routeTo),

		threshold,
		access.FmtJsonString(thresholdFlags),
		//access.FmtJsonString(routeName),
	)
	fmt.Printf("%v\n", s)
	//return s
}

func Encoding(resp *http.Response) (*http.Response, string) {
	encoding := ""
	if resp == nil {
		resp = &http.Response{StatusCode: http.StatusOK}
	} else {
		if resp.Header != nil {
			encoding = resp.Header.Get("Content-Encoding")
		}
	}
	return resp, encoding
}
func AuthHandler(w http.ResponseWriter, r *http.Request) {
	/*
		if r != nil {
			tokenString := r.Header.Get(host.Authorization)
			if tokenString == "" {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(w, "Missing authorization header")
			}
		}
	*/
}
