package googleauth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-google-sdk-go/pkg/tokenprovider"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	sdkhttpclient "github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/stretchr/testify/require"
)

// middlewareName returns the registered name of a Middleware, or "" if it has
// no name. SDK middlewares created via NamedMiddlewareFunc satisfy the
// MiddlewareName interface; we type-assert and call.
func middlewareName(m sdkhttpclient.Middleware) string {
	if named, ok := m.(sdkhttpclient.MiddlewareName); ok {
		return named.MiddlewareName()
	}
	return ""
}

// googleAuthMiddlewareName is the constant used by grafana-google-sdk-go's
// tokenprovider.AuthMiddleware. We assert on it to confirm the *right*
// middleware was appended, not just any middleware.
const googleAuthMiddlewareName = "GoogleAuthentication"

// fakeProvider satisfies tokenprovider.TokenProvider without making any
// network call. Used everywhere instead of the real GCE/JWT providers.
type fakeProvider struct {
	token string
}

func (f *fakeProvider) GetAccessToken(_ context.Context) (string, error) {
	return f.token, nil
}

func newFakeFactory() (providerFactory, *fakeProvider, *fakeProvider) {
	gce := &fakeProvider{token: "FAKE-GCE-TOKEN"}
	jwt := &fakeProvider{token: "FAKE-JWT-TOKEN"}
	return providerFactory{
		NewGCE: func(_ tokenprovider.Config) tokenprovider.TokenProvider { return gce },
		NewJWT: func(_ tokenprovider.Config) tokenprovider.TokenProvider { return jwt },
	}, gce, jwt
}

// generateServiceAccountKeyJSON synthesises a valid service-account key JSON
// (with a real-but-throwaway RSA private key) so tests can exercise the JSON
// parse + JWT config path without committing a fake key fixture to the repo.
// Marker strings let the secrets-never-logged test prove the raw value isn't
// leaked.
func generateServiceAccountKeyJSON(t *testing.T, clientEmail string) (raw, privatePEM string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	der := x509.MarshalPKCS1PrivateKey(key)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der})
	privatePEM = string(pemBytes)

	payload := map[string]string{
		"type":         "service_account",
		"project_id":   "test-project",
		"private_key":  privatePEM,
		"client_email": clientEmail,
		"client_id":    "100000000000000000000",
		"token_uri":    "https://oauth2.googleapis.com/token",
	}
	b, err := json.Marshal(payload)
	require.NoError(t, err)
	return string(b), privatePEM
}

func newSettings(jsonData string, secure map[string]string) backend.DataSourceInstanceSettings {
	return backend.DataSourceInstanceSettings{
		ID:                      42,
		UID:                     "test-uid",
		Updated:                 time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC),
		JSONData:                json.RawMessage(jsonData),
		DecryptedSecureJSONData: secure,
	}
}

func TestExtendOptions_DisabledByDefault(t *testing.T) {
	factory, _, _ := newFakeFactory()
	opts := &sdkhttpclient.Options{}

	err := extendWithFactory(context.Background(), newSettings("", nil), opts, log.DefaultLogger, factory)

	require.NoError(t, err)
	require.Empty(t, opts.Middlewares, "no auth type ⇒ no middleware appended")
}

func TestExtendOptions_UnsupportedAuthType(t *testing.T) {
	factory, _, _ := newFakeFactory()
	opts := &sdkhttpclient.Options{}

	err := extendWithFactory(context.Background(), newSettings(`{"googleAuthType":"banana"}`, nil), opts, log.DefaultLogger, factory)

	require.Error(t, err)
	require.Contains(t, err.Error(), `unsupported googleAuthType "banana"`)
	require.Empty(t, opts.Middlewares)
}

func TestExtendOptions_MalformedJSONData(t *testing.T) {
	factory, _, _ := newFakeFactory()
	opts := &sdkhttpclient.Options{}

	err := extendWithFactory(context.Background(), newSettings(`{not json`, nil), opts, log.DefaultLogger, factory)

	require.Error(t, err)
	require.Contains(t, err.Error(), "parsing jsonData")
}

func TestExtendOptions_ADC_AppendsMiddleware(t *testing.T) {
	factory, _, _ := newFakeFactory()
	opts := &sdkhttpclient.Options{}

	err := extendWithFactory(context.Background(), newSettings(`{"googleAuthType":"adc"}`, nil), opts, log.DefaultLogger, factory)

	require.NoError(t, err)
	require.Len(t, opts.Middlewares, 1)
	require.Equal(t, googleAuthMiddlewareName, middlewareName(opts.Middlewares[0]))
}

func TestExtendOptions_ServiceAccountJSON_AppendsMiddleware(t *testing.T) {
	factory, _, _ := newFakeFactory()
	raw, _ := generateServiceAccountKeyJSON(t, "sa@test-project.iam.gserviceaccount.com")
	opts := &sdkhttpclient.Options{}

	err := extendWithFactory(
		context.Background(),
		newSettings(`{"googleAuthType":"serviceAccountJson"}`, map[string]string{SecureKeyServiceAccountJSON: raw}),
		opts, log.DefaultLogger, factory,
	)

	require.NoError(t, err)
	require.Len(t, opts.Middlewares, 1)
	require.Equal(t, googleAuthMiddlewareName, middlewareName(opts.Middlewares[0]))
}

func TestExtendOptions_ServiceAccountJSON_MissingSecret(t *testing.T) {
	factory, _, _ := newFakeFactory()
	opts := &sdkhttpclient.Options{}

	err := extendWithFactory(
		context.Background(),
		newSettings(`{"googleAuthType":"serviceAccountJson"}`, nil),
		opts, log.DefaultLogger, factory,
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "googleServiceAccountJson")
	require.Empty(t, opts.Middlewares)
}

func TestExtendOptions_ServiceAccountJSON_MalformedJSON(t *testing.T) {
	factory, _, _ := newFakeFactory()
	const marker = "MARKER-PRIVATE-KEY-SHOULD-NEVER-LEAK"
	opts := &sdkhttpclient.Options{}

	err := extendWithFactory(
		context.Background(),
		newSettings(`{"googleAuthType":"serviceAccountJson"}`, map[string]string{SecureKeyServiceAccountJSON: marker}),
		opts, log.DefaultLogger, factory,
	)

	require.Error(t, err)
	require.NotContains(t, err.Error(), marker, "raw secret must never appear in the error message")
}

func TestExtendOptions_ServiceAccountJSON_WrongType(t *testing.T) {
	factory, _, _ := newFakeFactory()
	// "authorized_user" keys aren't service-account keys — JWTConfigFromJSON rejects them.
	raw := `{"type":"authorized_user","client_id":"x","client_secret":"y","refresh_token":"z"}`
	opts := &sdkhttpclient.Options{}

	err := extendWithFactory(
		context.Background(),
		newSettings(`{"googleAuthType":"serviceAccountJson"}`, map[string]string{SecureKeyServiceAccountJSON: raw}),
		opts, log.DefaultLogger, factory,
	)

	require.Error(t, err)
}

func TestExtendOptions_PreservesExistingMiddlewareOrder(t *testing.T) {
	factory, _, _ := newFakeFactory()
	seeded := sdkhttpclient.NamedMiddlewareFunc("seeded-first", func(_ sdkhttpclient.Options, next http.RoundTripper) http.RoundTripper {
		return next
	})
	opts := &sdkhttpclient.Options{Middlewares: []sdkhttpclient.Middleware{seeded}}

	err := extendWithFactory(context.Background(), newSettings(`{"googleAuthType":"adc"}`, nil), opts, log.DefaultLogger, factory)

	require.NoError(t, err)
	require.Len(t, opts.Middlewares, 2)
	require.Equal(t, "seeded-first", middlewareName(opts.Middlewares[0]))
	require.Equal(t, googleAuthMiddlewareName, middlewareName(opts.Middlewares[1]))
}

// TestSecretsNeverLogged is the regression test for the security promise made
// in the package doc-comment. Generates a real SA JSON with a known marker
// in the client_email, runs ExtendOptions with a recording logger, asserts
// neither the raw blob nor the private-key bytes appear anywhere.
func TestSecretsNeverLogged(t *testing.T) {
	factory, _, _ := newFakeFactory()
	raw, privatePEM := generateServiceAccountKeyJSON(t, "marker@test-project.iam.gserviceaccount.com")
	rec := newRecordingLogger()
	opts := &sdkhttpclient.Options{}

	err := extendWithFactory(
		context.Background(),
		newSettings(`{"googleAuthType":"serviceAccountJson"}`, map[string]string{SecureKeyServiceAccountJSON: raw}),
		opts, rec, factory,
	)
	require.NoError(t, err)

	all := rec.String()
	require.NotContains(t, all, privatePEM, "private key bytes leaked into logs")
	require.NotContains(t, all, raw, "raw service-account JSON leaked into logs")
}

func TestParseServiceAccountKey_HappyPath(t *testing.T) {
	raw, _ := generateServiceAccountKeyJSON(t, "ok@test-project.iam.gserviceaccount.com")

	cfg, err := parseServiceAccountKey(raw)

	require.NoError(t, err)
	require.Equal(t, "ok@test-project.iam.gserviceaccount.com", cfg.Email)
	require.Equal(t, "https://oauth2.googleapis.com/token", cfg.URI)
	require.NotEmpty(t, cfg.PrivateKey)
}

func TestParseServiceAccountKey_MissingFields(t *testing.T) {
	type tc struct {
		name        string
		modify      func(m map[string]string)
		wantInError string
	}
	cases := []tc{
		{
			name:        "missing client_email",
			modify:      func(m map[string]string) { delete(m, "client_email") },
			wantInError: "client_email",
		},
		// google.JWTConfigFromJSON defaults missing token_uri to a Google
		// fallback, so it's not a missing-field error we can synthesise here.
		// missing private_key cannot be cleanly tested either: the parser
		// fails earlier when it can't parse an empty PEM.
	}

	base, _ := generateServiceAccountKeyJSON(t, "x@y.iam.gserviceaccount.com")
	var basePayload map[string]string
	require.NoError(t, json.Unmarshal([]byte(base), &basePayload))

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			payload := make(map[string]string, len(basePayload))
			for k, v := range basePayload {
				payload[k] = v
			}
			c.modify(payload)
			b, err := json.Marshal(payload)
			require.NoError(t, err)

			_, err = parseServiceAccountKey(string(b))

			require.Error(t, err)
			require.Contains(t, err.Error(), c.wantInError)
		})
	}
}

// --- recording logger ---

type recordingLogger struct {
	log.Logger
	buf strings.Builder
}

func newRecordingLogger() *recordingLogger {
	r := &recordingLogger{Logger: log.DefaultLogger}
	return r
}

func (r *recordingLogger) String() string { return r.buf.String() }

func (r *recordingLogger) Debug(msg string, args ...interface{}) { r.write("DEBUG", msg, args) }
func (r *recordingLogger) Info(msg string, args ...interface{})  { r.write("INFO", msg, args) }
func (r *recordingLogger) Warn(msg string, args ...interface{})  { r.write("WARN", msg, args) }
func (r *recordingLogger) Error(msg string, args ...interface{}) { r.write("ERROR", msg, args) }

func (r *recordingLogger) write(level, msg string, args []interface{}) {
	r.buf.WriteString(level)
	r.buf.WriteByte(' ')
	r.buf.WriteString(msg)
	for _, a := range args {
		r.buf.WriteByte(' ')
		switch v := a.(type) {
		case string:
			r.buf.WriteString(v)
		default:
			b, _ := json.Marshal(v)
			r.buf.Write(b)
		}
	}
	r.buf.WriteByte('\n')
}
