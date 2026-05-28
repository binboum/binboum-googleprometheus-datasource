package main

import (
	"context"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	sdkhttpclient "github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/config"

	"github.com/grafana/grafana-prometheus-datasource/pkg/promlib"

	"github.com/binboum/binboum-googleprometheus-datasource/pkg/googleauth"
)

func NewDatasource(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	plog := backend.NewLoggerWith("logger", "tsdb.google-prometheus")
	plog.Debug("Initializing Google Prometheus data source", "uid", settings.UID, "name", settings.Name)

	return &Datasource{
		Service: promlib.NewService(sdkhttpclient.NewProvider(), plog, googleauth.ExtendOptions),
	}, nil
}

type Datasource struct {
	Service *promlib.Service
}

func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	ctx = d.contextualMiddlewares(ctx)
	return d.Service.QueryData(ctx, req)
}

func (d *Datasource) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	ctx = d.contextualMiddlewares(ctx)
	return d.Service.CallResource(ctx, req, sender)
}

func (d *Datasource) GetBuildInfo(ctx context.Context, req promlib.BuildInfoRequest) (*promlib.BuildInfoResponse, error) {
	ctx = d.contextualMiddlewares(ctx)
	return d.Service.GetBuildInfo(ctx, req)
}

func (d *Datasource) GetHeuristics(ctx context.Context, req promlib.HeuristicsRequest) (*promlib.Heuristics, error) {
	ctx = d.contextualMiddlewares(ctx)
	return d.Service.GetHeuristics(ctx, req)
}

func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	ctx = d.contextualMiddlewares(ctx)
	result, err := d.Service.CheckHealth(ctx, req)
	if result != nil && result.Status != backend.HealthStatusOk {
		result.Message = mapGMPHealthMessage(result.Message)
	}
	return result, err
}

func (d *Datasource) contextualMiddlewares(ctx context.Context) context.Context {
	cfg := config.GrafanaConfigFromContext(ctx)
	return sdkhttpclient.WithContextualMiddleware(ctx,
		sdkhttpclient.ResponseLimitMiddleware(cfg.ResponseLimit()),
	)
}

// mapGMPHealthMessage rewrites the GMP failures promlib surfaces from a health
// check into user-actionable text, preserving the original message in
// parentheses. promlib's health probe reduces any non-200 response to its
// status line (querydata.instantQuery's __healthcheck__ branch discards the
// body), so GMP failures are matched on the status text: GMP returns 400
// INVALID_ARGUMENT for a bad/missing project ID (not 404), 401 for missing or
// invalid credentials, 403 for a missing IAM role, and 404 only for a wrong
// URL path. Credential discovery fails before any request reaches GMP, so its
// phrasing survives in full. Everything else passes through unchanged.
func mapGMPHealthMessage(msg string) string {
	switch {
	case strings.Contains(msg, "400 Bad Request"):
		return withOriginal("Invalid request — check the project ID in the data source URL.", msg)
	case strings.Contains(msg, "401 Unauthorized"):
		return withOriginal("Google Cloud rejected the request as unauthenticated — set Authentication type to ADC or Service Account JSON, or check the credentials.", msg)
	case strings.Contains(msg, "403 Forbidden"):
		return withOriginal("Missing roles/monitoring.viewer on the target project.", msg)
	case strings.Contains(msg, "404 Not Found"):
		return withOriginal("Not found — check the data source URL path (it should end with /location/global/prometheus).", msg)
	case strings.Contains(msg, "could not find default credentials"),
		strings.Contains(msg, "metadata server"):
		return withOriginal("Not running on a GCP runtime and no GOOGLE_APPLICATION_CREDENTIALS — use Service Account JSON, or run in an ADC-capable environment.", msg)
	}
	return msg
}

func withOriginal(rewrite, original string) string {
	return rewrite + " (original: " + original + ")"
}
