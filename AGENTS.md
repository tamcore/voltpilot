# AGENTS.md

Guidance for AI agents working in this repo.

## What this is

voltpilot is a PWA + Go backend that gets a user to the nearest available charger of a chosen Charge Point Operator (CPO), using the public EnBW e-mobility API. SvelteKit frontend is embedded into the Go binary via `//go:embed` (build tag `prodfrontend`).

## Privacy boundary (non-negotiable)

- Deployment specifics — kube-context names, namespaces, public FQDNs, the registry, and any EnBW subscription key — live **only** in `AGENTS.md.local`, which is gitignored. Never copy those values into tracked files (Makefile defaults, Helm values, workflows, README, code, or commit messages). Pass them via env at runtime.
- The EnBW subscription key is scraped at runtime and must not be committed. The Helm chart only renders a Secret when a key seed is supplied at deploy time.

## Layout

- `cmd/server` — entrypoint; `internal/api` — router/handlers/middleware.
- `internal/enbw` — EnBW client + key manager (scrape + 401/403 refresh).
- `internal/chargers` — typed views: filter by operator/AC-DC/availability, rank by distance, build nav deep links.
- `internal/cache` — 45s in-process TTL cache. No Redis, no database.
- `web/` — SvelteKit static PWA; types mirror the Go JSON shapes.
- `charts/voltpilot` — Helm chart (nginx ingress + cert-manager).

## Conventions

- Conventional Commits (enforced in CI).
- Immutable data; small focused files; explicit error handling.
- TDD where practical. Run `make test` and `cd web && npm run check && npm run test:unit` before pushing.
- The EnBW API only filters spatially (bounding box). Operator / AC-DC / availability filtering is done in `internal/chargers`. AC/DC is inferred from plug types + power at list level; the exact `tariffGroup` (AC_CHARGER/DC_CHARGER) is used only on the detail view.
- HTTP 403 from EnBW means throttle, not auth failure — back off, don't hammer.

## CI

- `test.yaml` — Go vet/test + coverage gate, frontend check + unit tests.
- `e2e.yaml` — builds the embedded binary, runs Playwright (mocks `/api`, fully offline).
- `release.yaml` — goreleaser on `v*` tags (multi-arch GHCR images).
- `commit-lint.yaml` — Conventional Commits on PRs.
