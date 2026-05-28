package main

import (
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

func main() {
	// datasource.Manage blocks until Grafana asks the plugin to exit. The
	// NewDatasource factory is called per-DS-instance, and Dispose is invoked
	// when the instance is reconfigured. Lifecycle and gRPC plumbing are owned
	// by the SDK; we keep this file minimal so the operational shape is obvious
	// at a glance.
	if err := datasource.Manage("binboum-googleprometheus-datasource", NewDatasource, datasource.ManageOpts{}); err != nil {
		log.DefaultLogger.Error(err.Error())
		os.Exit(1)
	}
}
