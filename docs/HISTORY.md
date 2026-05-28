# Background

Grafana has no built-in way to authenticate to Google Cloud. Its Prometheus
data source happily talks to any Prometheus-compatible endpoint, but Google
Managed Service for Prometheus (GMP) requires a Google Cloud OAuth2 access token
on every request — and Grafana offers no mechanism to mint or attach one.

Google's own answers are two extra components, and neither fits an
infrastructure-as-code / GitOps / stateless deployment.

## The `frontend` proxy

Google ships a standalone GMP Prometheus `frontend` that you deploy next to
Grafana. It authenticates to GMP with the workload's own identity and exposes an
unauthenticated Prometheus API locally; Grafana then points at the proxy instead
of at GMP.

It works for a single project, but it turns a data-source setting into a
long-running service to deploy, patch, and monitor — and it does not scale. Each
`frontend` is pinned to one project, so a Grafana querying several projects needs
one proxy per project, each running beside Grafana and each adding a network hop.

## The `datasource-syncer`

Google also ships `datasource-syncer`, a scheduled job that calls the Grafana
HTTP API — authenticated with a Grafana service-account token — to write a
freshly minted, short-lived access token into an existing data source on a
recurring basis.

That is imperative and stateful — exactly what GitOps/IaC setups try to avoid:

- **The Grafana service-account token it needs can only be created once Grafana
  is running.** That is a day-2 bootstrap step and a second long-lived secret to
  issue, store, and rotate before any syncing can happen.
- **It mutates live Grafana state out-of-band.** The data source declared in Git
  no longer matches what is running, so the declarative source of truth drifts.
- **It assumes a persistent, writable Grafana.** A stateless Grafana that
  re-provisions from config on every start has nowhere for the synced token to
  live, and the job and the provisioning fight over the same field.
- **It depends on a scheduler running on time.** A missed run means an expired
  token and a broken data source until the next sync.

## This plugin

This plugin makes GMP authentication a property of the data source itself.
Grafana mints the token in-process — from the workload's Application Default
Credentials or a provided service-account key — and attaches it to every
outbound request.

The whole configuration is therefore declarative `jsonData` / `secureJsonData`:
it can be provisioned from a file or a Kubernetes secret, committed to Git, and
applied to a stateless Grafana with no sidecar, no scheduler, and no out-of-band
mutation.

## Data flow

```
Grafana (ConfigEditor.tsx)
  ├─ DataSourceHttpSettingsOverhaul   (@grafana/prometheus, standard auth/HTTP)
  └─ GoogleAuthSection                (jsonData.googleAuthType + secureJsonData)
        │  gRPC
        ▼
pkg/main.go → datasource.Manage → pkg/datasource.go
        │  promlib.NewService(client, log, googleauth.ExtendOptions)
        ▼
pkg/googleauth.ExtendOptions
        │  tokenprovider.AuthMiddleware → Authorization: Bearer <token>
        ▼
https://monitoring.googleapis.com/v1/projects/<PROJECT_ID>/location/global/prometheus
```
