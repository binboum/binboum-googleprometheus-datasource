package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"
)

const scalarSuccessBody = `{"status":"success","data":{"resultType":"scalar","result":[1692969348.331,"2"]}}`

// TestCheckHealth_Plumbing proves NewDatasource wires a working promlib
// Service and that CheckHealth reaches a Prometheus endpoint. It is not a
// re-test of promlib's health logic.
func TestCheckHealth_Plumbing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(scalarSuccessBody))
	}))
	defer srv.Close()

	settings := backend.DataSourceInstanceSettings{
		URL:      srv.URL,
		JSONData: []byte("{}"),
	}
	inst, err := NewDatasource(context.Background(), settings)
	require.NoError(t, err)
	ds, ok := inst.(*Datasource)
	require.True(t, ok)
	require.NotNil(t, ds.Service)

	req := &backend.CheckHealthRequest{
		PluginContext: backend.PluginContext{
			DataSourceInstanceSettings: &settings,
			GrafanaConfig:              backend.NewGrafanaCfg(map[string]string{"concurrent_query_count": "10"}),
		},
	}
	res, err := ds.CheckHealth(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, backend.HealthStatusOk, res.Status)
}

func TestMapGMPHealthMessage(t *testing.T) {
	// promlib's health probe surfaces an HTTP failure as "<status> - There was
	// an error returned querying the Prometheus API." (querydata.instantQuery
	// __healthcheck__ branch returns errors.New(res.Status)).
	tests := []struct {
		name        string
		in          string
		wantRewrite bool
		wantPrefix  string
	}{
		{
			name:        "400 bad request (wrong project id)",
			in:          "400 Bad Request - There was an error returned querying the Prometheus API.",
			wantRewrite: true,
			wantPrefix:  "Invalid request — check the project ID in the data source URL.",
		},
		{
			name:        "401 unauthorized (no/invalid credentials)",
			in:          "401 Unauthorized - There was an error returned querying the Prometheus API.",
			wantRewrite: true,
			wantPrefix:  "Google Cloud rejected the request as unauthenticated",
		},
		{
			name:        "403 forbidden (missing role)",
			in:          "403 Forbidden - There was an error returned querying the Prometheus API.",
			wantRewrite: true,
			wantPrefix:  "Missing roles/monitoring.viewer on the target project.",
		},
		{
			name:        "404 not found (wrong url path)",
			in:          "404 Not Found - There was an error returned querying the Prometheus API.",
			wantRewrite: true,
			wantPrefix:  "Not found — check the data source URL path",
		},
		{
			name:        "credential discovery: default credentials",
			in:          "google: could not find default credentials. See https://...",
			wantRewrite: true,
			wantPrefix:  "Not running on a GCP runtime and no GOOGLE_APPLICATION_CREDENTIALS",
		},
		{
			name:        "credential discovery: metadata server",
			in:          "compute: Could not fetch token from metadata server",
			wantRewrite: true,
			wantPrefix:  "Not running on a GCP runtime and no GOOGLE_APPLICATION_CREDENTIALS",
		},
		{
			name:        "unknown error passthrough",
			in:          "dial tcp: connection refused",
			wantRewrite: false,
		},
		{
			name:        "other status passthrough",
			in:          "500 Internal Server Error - There was an error returned querying the Prometheus API.",
			wantRewrite: false,
		},
		{
			name:        "empty input returns empty",
			in:          "",
			wantRewrite: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapGMPHealthMessage(tt.in)
			if !tt.wantRewrite {
				require.Equal(t, tt.in, got)
				return
			}
			require.True(t, strings.HasPrefix(got, tt.wantPrefix), "got: %q", got)
			require.True(t, strings.HasSuffix(got, " (original: "+tt.in+")"), "original not appended verbatim: %q", got)
		})
	}
}
