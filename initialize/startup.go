package initialize

import (
	"github.com/advanced-go/stdlib/host"
	"time"
)

func Startup() bool {
	// Run host register where all registered resources/packages will be sent a register configuration message
	m := createPackageConfiguration()
	if !host.Startup(time.Second*4, m) {
		return false
	}
	return true
}

// TO DO : create package configuration information for startup
func createPackageConfiguration() host.ContentMap {
	return make(host.ContentMap)
}
