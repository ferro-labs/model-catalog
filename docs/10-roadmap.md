# 10 — Roadmap & Milestones

Week-by-week execution. Each week ends with a concrete, demo-able deliverable.

## Week 1 — Bootstrap

**Goal**: this repo can hold the catalog and rebuild a byte-identical artifact.

- [ ] Repo bootstrap: `README.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, `Makefile`
- [ ] `go.mod` with minimum dependencies (yaml, jsonschema, slog)
- [ ] `schema/provider.schema.json`, `schema/model.schema.json`, `schema/manifest.schema.json`
- [ ] `tools/ferrocat` skeleton with `validate`, `build`, `split`, `lint`, `diff` subcommands stubbed
- [ ] `tools/ferrocat split` — converts the existing `ai-gateway/models/catalog.json` into per-model YAML files
- [ ] `tools/ferrocat build` — generates `dist/catalog.json` from YAML
- [ ] **Regression test**: `split → build` produces byte-identical output to input
- [ ] `.github/workflows/validate.yml` running on every PR

**Demo**: `make build` produces a `dist/catalog.json` byte-identical to today's `ai-gateway/models/catalog.json`.

## Week 2 — Schema + lint + first contributors

**Goal**: catalog cleanup and external-contributor-ready.

- [ ] Implement `ferrocat lint --strict`
- [ ] Run lint on existing data; merge fix PRs:
  - [ ] Delete junk Bedrock keys (`bedrock/1024-x-1024/...`)
  - [ ] Fix misformatted model IDs
  - [ ] Flag duplicate models for Phase 4 `extends`
- [ ] `.github/workflows/diff.yml` — render PR diff comment
- [ ] `.github/CODEOWNERS`, PR template, issue templates
- [ ] `CONTRIBUTING.md` with the "5-minute path to add a new model" walkthrough
- [ ] Public announcement: blog post on FerroLabs site

**Demo**: external contributor adds a model via PR, gets schema-validated diff comment, CODEOWNER approves, merges.

## Week 3 — `extends` and CDN

**Goal**: editing surface halved, public CDN live.

- [ ] Implement `extends` resolution in `ferrocat build`
- [ ] Migrate first wave: Bedrock partner models (~140 wrappers)
- [ ] Set up `catalog.ferrolabs.ai` (Cloudflare Pages, free tier)
- [ ] `.github/workflows/build.yml` — auto-publish `dist/` on merge to `main`
- [ ] Cosign keyless signing of `manifest.json`
- [ ] First public release: `v2026.04.28`
- [ ] Confirm `https://catalog.ferrolabs.ai/v1/manifest.json` returns signed manifest

**Demo**: a price update PR merges, builds, signs, and publishes to the CDN within 5 minutes. Anyone can `cosign verify-blob` it.

## Week 4 — Gateway-side consumer (alpha)

**Goal**: gateway can consume the new catalog source behind a feature flag.

- [ ] In `ai-gateway/models/`: add `load.go`, `verify.go`, `manifest.go`, `slice.go`, `reload.go`
- [ ] Embed cosign trust policy
- [ ] Implement three-tier fallback (CDN → GitHub Releases → embedded)
- [ ] Per-provider lazy loading based on configured targets
- [ ] Atomic.Value swap for hot-reload
- [ ] Background ticker (default 1h)
- [ ] `POST /admin/catalog/reload` admin endpoint
- [ ] All behind `FERRO_USE_REMOTE_CATALOG=false` flag (default off)
- [ ] Tests: `load_test.go`, `verify_test.go`, `slice_test.go`, `reload_test.go`

**Demo**: gateway running with flag `=true` correctly loads only the providers it needs, hot-reloads when a new manifest is published, and refuses tampered signatures.

## Week 5 — First scrapers (oracles)

**Goal**: cross-check the catalog against external sources, prove the value of automation.

- [ ] `tools/ferrocat scrape` skeleton
- [ ] `scrape/oracle/openrouter.go` — fetch `/api/v1/models`, normalize 300+ models
- [ ] `scrape/oracle/models_dev.go` — fetch `models.dev/api.json`
- [ ] Reconciler: group observations by `(provider, model_id)`
- [ ] Confidence scoring + cross-source agreement
- [ ] Snapshot writer (orphan `snapshots` branch)
- [ ] First weekly scrape run; expect to find ~30% of catalog with at least one stale or wrong field

**Demo**: a Monday-morning auto-PR titled "scrape: detected 23 price changes, 4 new models, 2 deprecations" with full provenance.

## Week 6 — More scrapers + cleanup wave

**Goal**: 10 scrapers running, biggest providers covered.

- [ ] `scrape/api/openai.go`, `anthropic.go`, `groq.go`, `together.go`, `fireworks.go`
- [ ] `scrape/api/mistral.go`, `cohere.go`, `deepseek.go`, `xai.go`, `cerebras.go`
- [ ] `tools/ferrocat prune` — finds entries with `sunset_date < today - 90d`
- [ ] `.github/workflows/prune-monthly.yml`
- [ ] Migrate remaining wrapper providers to `extends`: Vertex AI, Together, Fireworks, OpenRouter, DeepInfra, Novita

**Demo**: catalog passes its first fully-automated week — at least one auto-PR per provider, all merged after CODEOWNER review, zero manual edits required.

## Week 7 — Cloud pricing APIs

**Goal**: Bedrock, Vertex, Azure prices auto-verified.

- [ ] `scrape/pricing_api/aws_bedrock.go` — uses AWS Pricing API
- [ ] `scrape/pricing_api/azure.go` — uses `prices.azure.com`
- [ ] `scrape/pricing_api/vertex.go` — uses GCP Cloud Billing Catalog
- [ ] These cover the majority of "wrapper" pricing automatically
- [ ] Wire into the weekly scrape pipeline

**Demo**: a Bedrock price change announced by AWS reflects in the catalog within 24h without any human touching a YAML.

## Week 8 — HTML scrapers (server-rendered)

**Goal**: cover the providers without a model API.

- [ ] `scrape/html/mistral.go`, `cohere.go`, `groq.go`, `deepseek.go` — `goquery`-based scrapers
- [ ] Rate limiting, robots.txt respect, ETag caching

**Demo**: weekly scrape now covers all top-15 providers from at least 2 sources each.

## Week 9 — Headless browser (the stubborn last mile)

**Goal**: SPA pricing pages.

- [ ] `scrape/browser/chromedp_backend.go` — default
- [ ] `scrape/browser/cloudflare_backend.go` — opt-in
- [ ] `scrape/browser/pages/openai_pricing.go` — OpenAI pricing page (React)
- [ ] `scrape/browser/pages/anthropic_pricing.go`
- [ ] `scrape/browser/pages/gemini_pricing.go`
- [ ] Screenshot evidence committed alongside HTML snapshots

**Demo**: an OpenAI price update detected from the live pricing page within 24h, with a PNG snapshot attached to the PR.

## Week 10 — Production migration

**Goal**: gateway default flips to remote-catalog mode.

- [ ] Run gateway with `FERRO_USE_REMOTE_CATALOG=true` on FerroCloud staging for 2 weeks
- [ ] No `catalog.signature_invalid` events; no `catalog.fallback_to_embedded` events
- [ ] Memory and startup-time improvements measured and documented
- [ ] Flip default to `true` in next gateway release
- [ ] Update `ai-gateway` CHANGELOG and migration guide

**Demo**: production gateway uses the new catalog source by default. Day-0 model launches reflect within 1h via the background ticker.

## Beyond Week 10 — quarterly themes

| Quarter | Theme | Concrete goals |
|---|---|---|
| Q1 (Weeks 11–22) | Coverage | All 75 providers covered by at least 1 scraper; ≥50% by 2 scrapers; ≥95% pricing accuracy |
| Q2 | Community | ≥5 external maintainers added to CODEOWNERS; first community-built provider plugin; quarterly retro |
| Q3 | UX | Catalog-browser UI on `ferrolabs.ai` powered by `dist/catalog.json` directly; FerroCloud customers can see provenance per model |
| Q4 | Moat | Schema v2 with mode-specific extensions (multimodal, batch, fine-tune); deprecate schema v1 |

## Stop criteria for Week 10 ship

We do **not** flip to default-on until all of:

- ≥99% of catalog `Get()` calls succeed (measured in staging)
- ≥1h latency target met for hot-reload after manifest publish
- Three-tier fallback verified (CDN, GitHub Releases, embedded) all working in chaos test
- Cosign signature rejection verified (tampered manifest correctly refused)
- Per-provider lazy load verified (memory drop measured)
- `manifest_pin.json` rollback-protection tested

Any one of these failing pushes the flip out by another week.

## Definition of done for the entire migration

(From [00-vision.md](./00-vision.md) success criteria, restated as a checklist.)

- [ ] Gateway no longer ships `catalog.json` in its binary (only a manifest pin)
- [ ] Day-0 launches: median time-to-prod < 24h
- [ ] >50% of catalog updates are scraper auto-PRs
- [ ] ≥5 external contributors PR'd at least once
- [ ] Pricing accuracy ≥95% (measured by manual sample audit, monthly)
