# Google Managed Service for Prometheus — Grafana data source

[![CI](https://github.com/binboum/binboum-googleprometheus-datasource/actions/workflows/ci.yml/badge.svg)](https://github.com/binboum/binboum-googleprometheus-datasource/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](https://github.com/binboum/binboum-googleprometheus-datasource/blob/main/LICENSE)
[![Grafana](https://img.shields.io/badge/Grafana-%E2%89%A5%2011.5.0-orange.svg)](https://grafana.com/grafana/download)

Query **[Google Managed Service for Prometheus (GMP)](https://cloud.google.com/stackdriver/docs/managed-prometheus)**
from Grafana.

The plugin wraps the upstream Prometheus client (`@grafana/prometheus` /
`grafana-prometheus-datasource/pkg/promlib`) and injects a Google Cloud OAuth2
access token on every request — so a GMP endpoint behaves like any other
Prometheus data source.

## Features

- **Native GMP authentication** — Application Default Credentials (ADC) or a
  pasted service-account JSON key; the Google OAuth2 token is added to every
  request automatically.
- **Drop-in Prometheus experience** — query, alerting, exemplars, metadata and
  build-info all work through the upstream `@grafana/prometheus` client, with no
  change to how you write or run queries.

![Querying a Prometheus endpoint through the data source in Explore](https://raw.githubusercontent.com/binboum/binboum-googleprometheus-datasource/main/src/img/screenshots/explore-query.png)

## Install

### Signed (Grafana plugin catalog) — *in progress*

Submission to the Grafana plugin catalog is in progress; the signed install is
not available yet. Once published, you will be able to install it with the
Grafana CLI and restart Grafana:

```bash
grafana cli plugins install binboum-googleprometheus-datasource
```

Until then, use the unsigned install below.

### Unsigned (manual / air-gapped)

Download `binboum-googleprometheus-datasource-<version>.zip` from the
[Releases](https://github.com/binboum/binboum-googleprometheus-datasource/releases)
page, unpack it into Grafana's plugin directory, and allow the unsigned
plugin to load:

```bash
unzip binboum-googleprometheus-datasource-0.1.0.zip -d /var/lib/grafana/plugins/
# grafana.ini  ->  [plugins] allow_loading_unsigned_plugins = binboum-googleprometheus-datasource
# or env:
export GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=binboum-googleprometheus-datasource
```

To build from source instead, see [CONTRIBUTING.md](https://github.com/binboum/binboum-googleprometheus-datasource/blob/main/CONTRIBUTING.md).

## Usage

1. Add a new data source of type **Google Managed Service for Prometheus**.
2. Set the URL to your GMP query endpoint:
   `https://monitoring.googleapis.com/v1/projects/<PROJECT_ID>/location/global/prometheus`
3. Pick an authentication mode under **Google Cloud authentication** (below).
4. Click **Save & test**.

## Authentication

Choose a mode in the **Google Cloud authentication** section of the config page
(`jsonData.googleAuthType`):

- **`adc` — Google Cloud ADC**
  Application Default Credentials, read from the metadata server (GCE / GKE
  Workload Identity / Cloud Run) or `GOOGLE_APPLICATION_CREDENTIALS`.

- **`serviceAccountJson` — Service Account JSON**
  A service-account JSON key you paste in, parsed with `google.JWTConfigFromJSON`
  and exchanged for an access token.

How tokens are handled:

- Acquired through `grafana-google-sdk-go`'s `tokenprovider` — the same library
  Grafana's Cloud Monitoring data source uses.
- Cached and refreshed (10 s before expiry) by the SDK, which injects the
  `Authorization: Bearer` header on each request.
- Scope is fixed at `monitoring.read` (read-only); the required IAM role is
  `roles/monitoring.viewer`.

### Standard auth picker

The config page also shows Grafana's standard Prometheus auth picker — Basic
auth, Forward OAuth Identity, TLS settings, custom headers.

- None of it is required for GMP, but it stays available for non-GMP Prometheus
  endpoints.
- If Google Cloud authentication is enabled, it wins: its `Authorization` header
  is applied to outgoing requests.
- Enabling it alongside Basic auth or Forward OAuth Identity raises a
  non-blocking warning (you can still save):

  > Google Cloud authentication will override &lt;those methods&gt; for outgoing requests.

![Google Cloud authentication on the data source config page](https://raw.githubusercontent.com/binboum/binboum-googleprometheus-datasource/main/src/img/screenshots/config-google-auth.png)

### Health check

**Save & test** runs the upstream Prometheus health check, then rewrites the GMP
failures it recognizes into actionable hints — always keeping the original error,
appended as `(original: …)`:

- **`400 Bad Request`** → Invalid request — check the project ID in the data
  source URL.
- **`401 Unauthorized`** → Google Cloud rejected the request as unauthenticated —
  set the Authentication type to ADC or Service Account JSON, or check the
  credentials.
- **`403 Forbidden`** → Missing `roles/monitoring.viewer` on the target project.
- **`404 Not Found`** → Not found — check the data source URL path (it should end
  with `/location/global/prometheus`).
- **No usable credentials** → Not running on a GCP runtime and no
  `GOOGLE_APPLICATION_CREDENTIALS` — use Service Account JSON, or run in an
  ADC-capable environment.

Each mapping was validated against the live GMP API; any other error passes
through unchanged.

## Provisioning

`provisioning/datasources/datasources.yml` ships three reference examples
(plain Prometheus, ADC, Service Account JSON). Copy a single block into your
own Grafana provisioning and adapt the `url` and credentials:

```yaml
apiVersion: 1
datasources:
  - name: GMP (ADC)
    type: binboum-googleprometheus-datasource
    access: proxy
    url: https://monitoring.googleapis.com/v1/projects/<PROJECT_ID>/location/global/prometheus
    jsonData:
      httpMethod: POST
      googleAuthType: adc

  - name: GMP (Service Account)
    type: binboum-googleprometheus-datasource
    access: proxy
    url: https://monitoring.googleapis.com/v1/projects/<PROJECT_ID>/location/global/prometheus
    jsonData:
      httpMethod: POST
      googleAuthType: serviceAccountJson
    secureJsonData:
      googleServiceAccountJson: |
        { "type": "service_account", "project_id": "...", ... }
```

## Compatibility

Requires **Grafana ≥ 11.5.0** (`dependencies.grafanaDependency` in `plugin.json`).

The config editor uses `DataSourceHttpSettingsOverhaul` from `@grafana/prometheus`
13.x, which ships with Grafana 11.5 and later. On older releases that component
isn't exported, so the configuration editor fails to render.

## Contributing & support

Issues and pull requests are welcome — [CONTRIBUTING.md](https://github.com/binboum/binboum-googleprometheus-datasource/blob/main/CONTRIBUTING.md)
covers the development setup, build, and the full test and lint matrix, plus how
to file a useful bug report; see [docs/HISTORY.md](https://github.com/binboum/binboum-googleprometheus-datasource/blob/main/docs/HISTORY.md)
for the design background. Report security issues privately via
[SECURITY.md](https://github.com/binboum/binboum-googleprometheus-datasource/blob/main/SECURITY.md),
not a public issue.

## Security

- Service-account JSON is stored in Grafana's encrypted `secureJsonData` and is
  never echoed back over the API.
- No code path constructs an error or log line containing the raw JSON, the
  parsed `private_key`, or any access token. `TestSecretsNeverLogged` is the
  regression test.
- The token cache key is `(DataSourceID, settings.Updated, scopes)`.
  Reconfiguring the data source advances `Updated`, invalidating the cached
  token; rotating the secret takes effect on the next request.

To report a vulnerability, see [SECURITY.md](https://github.com/binboum/binboum-googleprometheus-datasource/blob/main/SECURITY.md).

## License

Apache-2.0 — see [LICENSE](https://github.com/binboum/binboum-googleprometheus-datasource/blob/main/LICENSE).

## Trademarks

This is an independent, community-maintained plugin. It is not affiliated with,
endorsed by, or sponsored by Google or Grafana Labs. "Google Managed Service for
Prometheus" and "Google Cloud" are trademarks of Google LLC; "Grafana" and
"Prometheus" are trademarks of their respective owners.