// Package googleauth wires Google Cloud authentication into the Prometheus
// data-source HTTP client.
//
// This package never logs the raw service-account JSON, the parsed
// private_key, or any access token. Errors are wrapped with field names
// only — see TestSecretsNeverLogged for the regression test.
package googleauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/grafana/grafana-google-sdk-go/pkg/tokenprovider"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	sdkhttpclient "github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"golang.org/x/oauth2/google"
)

// Auth-type wire values. These are the strings written to jsonData by the
// frontend. Empty/absent means the feature is disabled and ExtendOptions is
// a no-op.
const (
	AuthTypeNone               = ""
	AuthTypeADC                = "adc"
	AuthTypeServiceAccountJSON = "serviceAccountJson"
)

// MonitoringReadScope matches the OAuth2 scope used by Grafana's
// cloud-monitoring data source for reads. GMP requires the same scope.
const MonitoringReadScope = "https://www.googleapis.com/auth/monitoring.read"

// SecureKeyServiceAccountJSON is the secureJsonData key holding the raw
// service-account JSON when AuthType is AuthTypeServiceAccountJSON.
const SecureKeyServiceAccountJSON = "googleServiceAccountJson"

// jsonDataShape is the subset of jsonData this package reads. Other
// promlib-level jsonData fields are passed through unchanged.
type jsonDataShape struct {
	GoogleAuthType string `json:"googleAuthType"`
}

// providerFactory is the seam used by tests to swap in fake token providers
// without touching network or GCP. Production code uses the default
// initialised in this package's init below.
type providerFactory struct {
	NewGCE func(tokenprovider.Config) tokenprovider.TokenProvider
	NewJWT func(tokenprovider.Config) tokenprovider.TokenProvider
}

var defaultFactory = providerFactory{
	NewGCE: tokenprovider.NewGceAccessTokenProvider,
	NewJWT: tokenprovider.NewJwtAccessTokenProvider,
}

// ExtendOptions is the promlib.ExtendOptions hook, called once per per-DS
// http.Client to append the Google auth middleware.
func ExtendOptions(ctx context.Context, settings backend.DataSourceInstanceSettings, opts *sdkhttpclient.Options, logger log.Logger) error {
	return extendWithFactory(ctx, settings, opts, logger, defaultFactory)
}

// extendWithFactory is the test seam.
func extendWithFactory(_ context.Context, settings backend.DataSourceInstanceSettings, opts *sdkhttpclient.Options, logger log.Logger, factory providerFactory) error {
	var jd jsonDataShape
	if len(settings.JSONData) > 0 {
		if err := json.Unmarshal(settings.JSONData, &jd); err != nil {
			return fmt.Errorf("googleauth: parsing jsonData: %w", err)
		}
	}

	switch jd.GoogleAuthType {
	case AuthTypeNone:
		return nil

	case AuthTypeADC:
		provider := factory.NewGCE(tokenProviderConfig(settings))
		opts.Middlewares = append(opts.Middlewares, tokenprovider.AuthMiddleware(provider))
		logger.Debug("Google auth attached", "type", AuthTypeADC, "uid", settings.UID)
		return nil

	case AuthTypeServiceAccountJSON:
		raw := settings.DecryptedSecureJSONData[SecureKeyServiceAccountJSON]
		if raw == "" {
			return errors.New("googleauth: googleAuthType=serviceAccountJson requires secureJsonData.googleServiceAccountJson")
		}
		jwtCfg, err := parseServiceAccountKey(raw)
		if err != nil {
			// IMPORTANT: do not include `raw` or any private-key bytes in the error.
			return fmt.Errorf("googleauth: parsing service account JSON: %w", err)
		}
		cfg := tokenProviderConfig(settings)
		cfg.JwtTokenConfig = jwtCfg
		opts.Middlewares = append(opts.Middlewares, tokenprovider.AuthMiddleware(factory.NewJWT(cfg)))
		logger.Debug("Google auth attached", "type", AuthTypeServiceAccountJSON, "uid", settings.UID, "client_email", jwtCfg.Email)
		return nil

	default:
		return fmt.Errorf("googleauth: unsupported googleAuthType %q", jd.GoogleAuthType)
	}
}

// tokenProviderConfig builds the cache key (DataSourceID + Updated + Scopes)
// used by the SDK's token providers. Reconfiguring the data source advances
// settings.Updated, invalidating the cached token.
func tokenProviderConfig(settings backend.DataSourceInstanceSettings) tokenprovider.Config {
	return tokenprovider.Config{
		DataSourceID:      settings.ID,
		DataSourceUpdated: settings.Updated,
		Scopes:            []string{MonitoringReadScope},
	}
}

// parseServiceAccountKey parses a raw service-account JSON blob into the JWT
// provider's config. It rejects authorized-user keys and other shapes (see
// TestExtendOptions_ServiceAccountJSON_WrongType) and names any missing field
// in the error without echoing the raw blob.
func parseServiceAccountKey(raw string) (*tokenprovider.JwtTokenConfig, error) {
	jwtCfg, err := google.JWTConfigFromJSON([]byte(raw), MonitoringReadScope)
	if err != nil {
		// google.JWTConfigFromJSON's error message is safe (no key bytes).
		return nil, err
	}
	if jwtCfg.Email == "" {
		return nil, errors.New("missing required field client_email")
	}
	if jwtCfg.TokenURL == "" {
		return nil, errors.New("missing required field token_uri")
	}
	if len(jwtCfg.PrivateKey) == 0 {
		return nil, errors.New("missing required field private_key")
	}
	return &tokenprovider.JwtTokenConfig{
		Email:      jwtCfg.Email,
		URI:        jwtCfg.TokenURL,
		PrivateKey: jwtCfg.PrivateKey,
	}, nil
}
