package main

import (
	"context"
	"fmt"
	"github.com/advanced-go/search/provider"
	"github.com/advanced-go/stdlib/access"
	"github.com/advanced-go/stdlib/core"
	fmt2 "github.com/advanced-go/stdlib/fmt"
	"github.com/advanced-go/stdlib/host"
	"github.com/advanced-go/stdlib/httpx"
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
	fmt.Printf("env     : %v\n", core.EnvStr())
}

func startup(r *http.ServeMux) (http.Handler, bool) {
	// Override access logger
	access.SetLogFn(logger)

	// Run host startup where all registered resources/packages will be sent a startup configuration message
	m := createPackageConfiguration()
	if !host.Startup(time.Second*4, m) {
		return r, false
	}

	// Initialize host proxy for all HTTP handlers,and add intermediaries
	host.SetHostTimeout(time.Second * 3)
	host.SetAuthHandler(AuthHandler, nil)
	err := host.RegisterHandler(provider.PkgPath, host.NewAccessLogIntermediary("google-search", provider.HttpExchange))
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

// TO DO : create package configuration information for startup
func createPackageConfiguration() host.ContentMap {
	return make(host.ContentMap)
}

func healthLivelinessHandler(w http.ResponseWriter, r *http.Request) {
	var status = core.StatusOK()
	if status.OK() {
		httpx.WriteResponse[core.Log](w, nil, status.HttpCode(), []byte("up"))
	} else {
		httpx.WriteResponse[core.Log](w, nil, status.HttpCode(), nil)
	}
}

func healthReadinessHandler(w http.ResponseWriter, r *http.Request) {
	var status = core.StatusOK()
	if status.OK() {
		httpx.WriteResponse[core.Log](w, nil, status.HttpCode(), []byte("up"))
	} else {
		httpx.WriteResponse[core.Log](w, nil, status.HttpCode(), nil)
	}
}

func logger(o *access.Origin, traffic string, start time.Time, duration time.Duration, req *http.Request, resp *http.Response, routeName, routeTo string, threshold int, thresholdFlags string) {
	req = access.SafeRequest(req)
	resp = access.SafeResponse(resp)
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
		"\"request-id\":%v, "+
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
		fmt2.FmtRFC3339Millis(start),
		strconv.Itoa(access.Milliseconds(duration)),

		fmt2.JsonString(req.Header.Get(httpx.XRequestId)),
		//access.FmtJsonString(req.Header.Get(runtime2.XRelatesTo)),
		//access.FmtJsonString(req.Proto),
		fmt2.JsonString(req.Method),
		fmt2.JsonString(url),
		fmt2.JsonString(req.URL.RawQuery),
		//fmt2.JsonString(host),
		//fmt2.JsonString(path),

		resp.StatusCode,
		//fmt2.JsonString(resp.Status),
		fmt.Sprintf("%v", resp.ContentLength),
		fmt2.JsonString(access.Encoding(resp)),

		fmt2.JsonString(routeName),
		//fmt2.JsonString(routeTo),

		threshold,
		fmt2.JsonString(thresholdFlags),
		//fmt2.JsonString(routeName),
	)
	fmt.Printf("%v\n", s)
	//return s
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
