package initialize

import (
	"fmt"
	"github.com/advanced-go/common/access"
	"github.com/advanced-go/common/core"
	"github.com/advanced-go/common/uri"
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

		//access.JsonString(o.Region),
		//access.JsonString(o.Zone),
		//access.JsonString(o.SubZone),
		//access.JsonString(o.App),
		//access.JsonString(o.InstanceId),

		traffic,
		core.FmtRFC3339Millis(start),
		strconv.Itoa(access.Milliseconds(duration)),

		access.JsonString(newReq.Header.Get(core.XRequestId)),
		access.JsonString(routing.From),
		access.JsonString(access.CreateTo(newReq)),
		//access.JsonString(req.Header.Get(runtime2.XRelatesTo)),
		//access.JsonString(req.Proto),
		access.JsonString(newReq.Method),
		access.JsonString(url),
		access.JsonString(newReq.URL.RawQuery),
		//access.JsonString(host),
		//access.JsonString(path),

		newResp.StatusCode,
		//access.JsonString(resp.Status),
		fmt.Sprintf("%v", newResp.ContentLength),
		access.JsonString(access.Encoding(newResp)),

		// Controller
		access.Milliseconds(controller.Timeout),
		fmt.Sprintf("%v", controller.RateLimit),
		strconv.Itoa(controller.RateBurst),
		access.JsonString(controller.Code),

		// Routing
		access.JsonString(routing.Route),
		access.JsonString(routing.To),
		fmt.Sprintf("%v", routing.Percent),
		access.JsonString(routing.Code),
	)
	fmt.Printf("%v\n", s)
	//return s
}
