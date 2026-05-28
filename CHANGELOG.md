# Changelog

## 0.1.0 (2026-05-28)

Initial release of the Google Managed Service for Prometheus data source for Grafana.

### Features

* **auth:** Application Default Credentials (ADC) — metadata server (GCE / GKE Workload Identity / Cloud Run) or `GOOGLE_APPLICATION_CREDENTIALS`
* **auth:** service-account JSON key, parsed with `google.JWTConfigFromJSON` and exchanged for a `monitoring.read` token
* **auth:** non-blocking warning when Google auth is combined with Basic auth or Forward OAuth Identity
* **health:** `Save & test` rewrites GMP `400` / `401` / `403` / `404` and credential-discovery failures into actionable messages, preserving the original error
* **provisioning:** three reference data sources (plain Prometheus, ADC, Service Account JSON)
