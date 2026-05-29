# Changelog

## [0.1.1](https://github.com/binboum/binboum-googleprometheus-datasource/compare/v0.1.0...v0.1.1) (2026-05-29)


### Bug Fixes

* README catalog links and release hardening ([#5](https://github.com/binboum/binboum-googleprometheus-datasource/issues/5)) ([aa7623f](https://github.com/binboum/binboum-googleprometheus-datasource/commit/aa7623f269063263f610ea5ebdceef298ec4cc58))

## 0.1.0 (2026-05-28)


### Features

* initial release of the Google Managed Service for Prometheus data source ([#1](https://github.com/binboum/binboum-googleprometheus-datasource/issues/1)) ([538d046](https://github.com/binboum/binboum-googleprometheus-datasource/commit/538d046acd1eb69781e65814f1dd252d437b0020))

## 0.1.0 (2026-05-28)

Initial release of the Google Managed Service for Prometheus data source for Grafana.

### Features

* **auth:** Application Default Credentials (ADC) — metadata server (GCE / GKE Workload Identity / Cloud Run) or `GOOGLE_APPLICATION_CREDENTIALS`
* **auth:** service-account JSON key, parsed with `google.JWTConfigFromJSON` and exchanged for a `monitoring.read` token
* **auth:** non-blocking warning when Google auth is combined with Basic auth or Forward OAuth Identity
* **health:** `Save & test` rewrites GMP `400` / `401` / `403` / `404` and credential-discovery failures into actionable messages, preserving the original error
* **provisioning:** three reference data sources (plain Prometheus, ADC, Service Account JSON)
