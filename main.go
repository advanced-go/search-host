package main

import (
	"context"
	"fmt"
	"github.com/advanced-go/core/access"
	"github.com/advanced-go/core/handler"
	"github.com/advanced-go/core/http2"
	"github.com/advanced-go/core/messaging"
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
	addr                    = "0.0.0.0:8081"
	writeTimeout            = time.Second * 300
	readTimeout             = time.Second * 15
	idleTimeout             = time.Second * 60
	healthLivelinessPattern = "/health/liveness"
	healthReadinessPattern  = "/health/readiness"
)

func main() {
	// Initialize runtime environment - defaults to debug
	runtime2.SetStageEnvironment()

	start := time.Now()
	displayRuntime()
	handler, status := startup(http.NewServeMux())
	if !status.OK() {
		os.Exit(1)
	}
	fmt.Println(fmt.Sprintf("started : %v", time.Since(start)))
	httpPort := os.Getenv("PORT")
	if httpPort == "" {
		httpPort = addr
	}
	srv := http.Server{
		Addr: httpPort, //addr,
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

func displayRuntime() {
	fmt.Printf("addr    : %v\n", addr)
	fmt.Printf("vers    : %v\n", runtime.Version())
	fmt.Printf("os      : %v\n", runtime.GOOS)
	fmt.Printf("arch    : %v\n", runtime.GOARCH)
	fmt.Printf("cpu     : %v\n", runtime.NumCPU())
	fmt.Printf("env     : %v\n", runtime2.EnvStr())
}

func startup(r *http.ServeMux) (http.Handler, runtime2.Status) {
	// Set error handling formatter and/or logger
	runtime2.SetErrorFormatter(nil)
	runtime2.SetErrorLogger(nil)

	// Set access logging handler and options
	access.SetLogger(logger)
	//access.EnableInternalLogging()
	access.EnableTestLogger()
	//access.SetFormatter(logFormatter)

	// Run startup where all registered resources/packages will be sent a startup message which may contain
	// package configuration information such as authentication, default values...
	m := createPackageConfiguration()
	status := messaging.Startup[runtime2.Log](time.Second*4, m)
	if !status.OK() {
		return r, status
	}

	// Initialize messaging for all HTTP handlers
	messaging.Handle(provider.PkgPath, provider.HttpHandler)

	// Initialize health handlers
	r.Handle(healthLivelinessPattern, http.HandlerFunc(healthLivelinessHandler))
	r.Handle(healthReadinessPattern, http.HandlerFunc(healthReadinessHandler))

	// Route all other requests to messaging
	r.Handle("/", http.HandlerFunc(messaging.HttpHandler))

	// Add host metrics handler
	return handler.HttpHostMetricsHandler(r, ""), runtime2.StatusOK()
}

// TO DO : create package configuration information for startup
func createPackageConfiguration() messaging.ContentMap {
	return make(messaging.ContentMap)
}

func healthLivelinessHandler(w http.ResponseWriter, r *http.Request) {
	var status = runtime2.NewStatusOK()
	if status.OK() {
		http2.WriteResponse[runtime2.Log](w, []byte("up"), status, nil)
	} else {
		http2.WriteResponse[runtime2.Log](w, nil, status, nil)
	}
}

func healthReadinessHandler(w http.ResponseWriter, r *http.Request) {
	var status = runtime2.NewStatusOK()
	if status.OK() {
		http2.WriteResponse[runtime2.Log](w, []byte("up"), status, nil)
	} else {
		http2.WriteResponse[runtime2.Log](w, nil, status, nil)
	}
}

func logger(o access.Origin, traffic string, start time.Time, duration time.Duration, req *http.Request, resp *http.Response, routeName, routeTo string, threshold int, thresholdFlags string) {
	if req == nil {
		req, _ = http.NewRequest("", "https://somehost.com/search?q=test", nil)
	}
	if resp == nil {
		resp = &http.Response{StatusCode: http.StatusOK}
	}
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
		//"\"host\":%v, "+
		//"\"path\":%v, "+
		"\"status-code\":%v, "+
		//"\"status\":%v, "+
		//"\"route\":%v, "+
		//"\"route-to\":%v, "+
		//"\"threshold\":%v, "+
		"\"route\":%v }",
		//"\"threshold-flags\":%v }",
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
		//access.FmtJsonString(host),
		//access.FmtJsonString(path),

		resp.StatusCode,
		//access.FmtJsonString(resp.Status),

		//access.FmtJsonString(routeName),
		//access.FmtJsonString(routeTo),
		//threshold,
		//access.FmtJsonString(thresholdFlags),
		access.FmtJsonString(routeName),
	)
	fmt.Printf("%v\n", s)
	//return s
}
