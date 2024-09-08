package initialize

import (
	"fmt"
	"github.com/advanced-go/stdlib/access"
	"github.com/advanced-go/stdlib/core"
	fmt2 "github.com/advanced-go/stdlib/fmt"
	"github.com/advanced-go/stdlib/httpx"
	"github.com/advanced-go/stdlib/uri"
	"strconv"
	"time"
)

func Logging() {
	// Override access logger
	access.SetLogFn(logger)
}

func logger(o core.Origin, traffic string, start time.Time, duration time.Duration, req any, resp any, routing access.Routing, controller access.Controller) {
	newReq := access.BuildRequest(req)
	newResp := access.BuildResponse(resp)
	url, parsed := uri.ParseURL(newReq.Host, newReq.URL)
	o.Host = access.Conditional(o.Host, parsed.Host)
	if controller.RateLimit == 0 {
		controller.RateLimit = -1
	}
	if controller.RateBurst == 0 {
		controller.RateBurst = -1
	}
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
		"\"from\":%v, "+
		"\"to\":%v, "+
		"\"method\":%v, "+
		"\"uri\":%v, "+
		"\"query\":%v, "+
		//"\"host\":%v, "+
		//"\"path\":%v, "+
		"\"status-code\":%v, "+
		"\"bytes\":%v, "+
		"\"encoding\":%v, "+
		"\"timeout\":%v, "+
		"\"rate-limit\":%v, "+
		"\"rate-burst\":%v, "+
		"\"cc\":%v, "+
		"\"route\":%v, "+
		"\"route-to\":%v, "+
		"\"route-percent\":%v, "+
		"\"rc\":%v }",

		//access.FmtJsonString(o.Region),
		//access.FmtJsonString(o.Zone),
		//access.FmtJsonString(o.SubZone),
		//access.FmtJsonString(o.App),
		//access.FmtJsonString(o.InstanceId),

		traffic,
		fmt2.FmtRFC3339Millis(start),
		strconv.Itoa(access.Milliseconds(duration)),

		fmt2.JsonString(newReq.Header.Get(httpx.XRequestId)),
		fmt2.JsonString(routing.From),
		fmt2.JsonString(access.CreateTo(newReq)),
		//access.FmtJsonString(req.Header.Get(runtime2.XRelatesTo)),
		//access.FmtJsonString(req.Proto),
		fmt2.JsonString(newReq.Method),
		fmt2.JsonString(url),
		fmt2.JsonString(newReq.URL.RawQuery),
		//fmt2.JsonString(host),
		//fmt2.JsonString(path),

		newResp.StatusCode,
		//fmt2.JsonString(resp.Status),
		fmt.Sprintf("%v", newResp.ContentLength),
		fmt2.JsonString(access.Encoding(newResp)),

		// Controller
		access.Milliseconds(controller.Timeout),
		fmt.Sprintf("%v", controller.RateLimit),
		strconv.Itoa(controller.RateBurst),
		fmt2.JsonString(controller.Code),

		// Routing
		fmt2.JsonString(routing.Route),
		fmt2.JsonString(routing.To),
		fmt.Sprintf("%v", routing.Percent),
		fmt2.JsonString(routing.Code),
	)
	fmt.Printf("%v\n", s)
	//return s
}
