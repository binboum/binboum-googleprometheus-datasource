# Contributing

Thanks for your interest in improving the Google Managed Service for Prometheus
data source.

## Dev loop

```bash
yarn install --immutable          # frontend deps (immutable lockfile)
mage -v build:backend             # backend binaries into dist/
yarn build                        # frontend bundle into dist/
yarn server                       # docker compose: Grafana + Prometheus
```

Grafana comes up on http://localhost:3000 (admin / admin) with three
provisioned data sources. For iterative work, `yarn dev` watches the frontend
and `mage -v build:linux && docker compose restart grafana` rebuilds the
backend.

## Tests

| What            | Command          |
| --------------- | ---------------- |
| Backend (Go)    | `go test ./...`  |
| Frontend (Jest) | `yarn test:ci`   |
| E2E (Playwright)| `yarn e2e` (needs `yarn server` running) |
| Lint (TS)       | `yarn lint`      |
| Lint (Go)       | `go vet ./...`   |
| Typecheck       | `yarn typecheck` |

Please keep all of the above green in a PR.

## Filing issues

Open a [GitHub issue](https://github.com/binboum/binboum-googleprometheus-datasource/issues)
with the Grafana version, plugin version, the auth mode in use, and the exact
`Save & test` message (the original error is preserved in parentheses — include
it). Do **not** paste service-account keys or access tokens. For security
issues, follow [SECURITY.md](SECURITY.md) instead of opening a public issue.

## Commits and changelog

User-visible changes go under `## [Unreleased]` in
[CHANGELOG.md](CHANGELOG.md), following
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Internal/CI-only
changes do not need a changelog entry.
