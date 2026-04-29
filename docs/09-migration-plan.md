# 09 — Migration Plan

How we move from today's `ai-gateway/models/catalog.json` (111K lines, 2,531 entries, 75 providers) to the new repo without breaking any running gateway.

**Guiding principle**: every phase is independently revertible. No phase requires all later phases to be deployed.

## Phase 0 — Inventory & freeze

**Duration**: 1 day. **Risk**: none.

Before touching anything:

1. Snapshot current `models/catalog.json` to `model-catalog/dist/catalog.json` byte-for-byte
2. Snapshot to a release tag: `v2026.04.28-pre-migration`
3. Diff against the existing in-repo file every morning during migration to ensure nothing changes outside this plan
4. **Freeze direct edits** to `ai-gateway/models/catalog.json` — all changes must go through the new repo from this point. (Document this in the gateway repo's CONTRIBUTING.md.)

**Deliverable**: `model-catalog` repo holds an exact byte-identical copy of today's catalog under `dist/`. Gateway behavior unchanged.

## Phase 1 — Decouple distribution

**Duration**: 1–2 days. **Risk**: low (one-line URL change).

1. Set up CDN (Cloudflare Pages, free tier) at `catalog.ferrolabs.ai/v1/`
2. Publish the unchanged `dist/catalog.json` to `https://catalog.ferrolabs.ai/v1/legacy/catalog.json`
3. In `ai-gateway/models/catalog.go`, change `defaultCatalogURL` to point at the new CDN URL (still the same JSON shape)
4. Verify gateway still works in dev + staging
5. Release as `ai-gateway` patch version

**Roll back**: revert the one-line URL change.

**Deliverable**: catalog is now served from infrastructure FerroLabs controls, and edits no longer require a gateway-repo PR.

## Phase 2 — Split into per-model YAML

**Duration**: 3–5 days. **Risk**: medium (large mechanical refactor).

1. Build `tools/ferrocat split` — a one-shot tool that:
   - Reads `dist/catalog.json` (the legacy format)
   - Emits `providers/<id>/provider.yaml` + `providers/<id>/models/<model>.yaml` for every entry
   - Produces a deterministic output (run twice → byte-identical)
2. Run it. Inspect output. Adjust until happy.
3. Build `tools/ferrocat build` — generates `dist/catalog.json` from the YAML files
4. **Regression test**: `build`'s output must be byte-identical to the input from step 1. Add this as `validate.yml` CI check.
5. Replace the committed `dist/catalog.json` with the build pipeline's output (still byte-identical).
6. Add `validate.yml` and `diff.yml` workflows. Wire schema validation.

**Roll back**: keep the legacy single-JSON copy at `legacy/catalog.json`; gateway can repoint there if anything goes wrong.

**Deliverable**: source of truth is now per-model YAML, but the published artifact at `catalog.ferrolabs.ai/v1/legacy/catalog.json` remains unchanged. Gateway behavior unchanged.

## Phase 3 — Schema upgrade & lint

**Duration**: 3–5 days. **Risk**: low (additive only).

1. Adopt the full schema from [03-schema.md](./03-schema.md). Existing fields stay; new fields (provenance, tiered pricing, above-threshold) start as null/optional.
2. Add lint rules to `ferrocat validate`:
   - Key shape regex
   - No parameter segments (`steps`, `width`, `height`, `quality`)
   - No duplicate model IDs across providers
3. Run `ferrocat lint --strict`. Expect 800–1,000 violations on the existing data:
   - ~80 junk Bedrock keys (`bedrock/1024-x-1024/...`) → delete
   - ~600 Vertex/Bedrock duplicates → flagged for `extends` migration in Phase 4
   - ~50 misformatted model IDs → fix individually
4. Open one cleanup PR per category, merge after review.
5. Build `dist/v2/catalog.json` (new shape) alongside `dist/catalog.json` (legacy shape). Both published.

**Roll back**: legacy `dist/catalog.json` is unchanged; gateway keeps using it.

**Deliverable**: the catalog is clean, lint-enforced, and richer in schema, but the gateway still consumes the legacy shape.

## Phase 4 — Collapse with `extends`

**Duration**: 1 week. **Risk**: low (validated by build regression test).

1. Implement `extends` resolution in `ferrocat build` — see [04-extends-pattern.md](./04-extends-pattern.md)
2. Migrate Bedrock partner models: ~140 wrapper files, each ~10 lines, replacing ~140 full files
3. Migrate Vertex AI partner models: ~140 wrappers
4. Migrate OpenRouter (~300 wrappers), Together, Fireworks, Groq, DeepInfra, Novita
5. After each migration batch, build and **byte-diff** against pre-migration build → ensure zero unintended changes
6. Source-file count drops from ~2,400 → ~1,200; generated catalog unchanged

**Roll back**: revert the wrapper PRs, base files are untouched.

**Deliverable**: editing surface halved, single price update propagates to all wrappers, blast-radius diff visible in PRs.

## Phase 5 — Gateway-side richer consumer

**Duration**: 1 week. **Risk**: medium (real gateway code changes).

1. Add `manifest.json`, `verify.go`, `load.go`, `slice.go`, `reload.go` to `ai-gateway/models/` per [08-gateway-integration.md](./08-gateway-integration.md)
2. Ship behind feature flag `FERRO_USE_REMOTE_CATALOG=false` (default off)
3. Internal canary: enable on FerroCloud staging, observe 1 week
4. Flip default to `true` in next gateway release
5. Old code path stays for 1 release as compatibility (logs a deprecation warning when used)

**Roll back**: flip the feature flag back to off.

**Deliverable**: gateways now do per-provider lazy loading, signature verification, and hot-reload. Memory and startup time both drop measurably.

## Phase 6 — Automation + scrapers

**Duration**: 4–6 weeks (overlaps with Phase 5). **Risk**: low (auto-PRs are review-gated).

Per the schedule in [05-scraping.md](./05-scraping.md):

1. Week 1: oracle scrapers (OpenRouter, models.dev) — gives instant cross-check
2. Week 2–3: Tier 1 `/v1/models` scrapers
3. Week 4: Tier 2 cloud pricing APIs
4. Week 5–6: Tier 3 HTML
5. Week 7+: Tier 5 headless

Each scraper goes live behind a `--scraper-allowlist` flag and is enabled one at a time after watching its output for 2 weeks.

**Roll back**: disable any individual scraper via `tools/ferrocat/scrape/registry.yaml`.

**Deliverable**: ≥50% of catalog updates become auto-PRs. Pricing accuracy verified continuously.

## Phase 7 — Sunset `ai-gateway/models/catalog.json`

**Duration**: 2 days, scheduled 6 months after Phase 5 lands. **Risk**: low (consumers had time to migrate).

1. Remove `ai-gateway/models/catalog.json` and `catalog_backup.json` from the gateway repo
2. Replace embedded fallback with a tiny pinned-manifest pointer
3. Old env var `FERRO_MODEL_CATALOG_URL` removed (was deprecated in Phase 5)
4. Final commit message in `ai-gateway`: `models: catalog source moved to ferro-labs/model-catalog`

**Roll back**: re-add the file from git history (still recoverable for years).

**Deliverable**: gateway repo no longer carries 111K lines of pricing data. Catalog repo is the single source of truth.

## Migration risk matrix

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Byte-diff regression in `ferrocat build` | Medium | High (silent price drift) | CI regression check against snapshot baseline, every PR |
| CDN downtime during cutover | Low | Medium (gateways fall back to embedded) | Three-tier fallback (CDN → GitHub Releases → embedded) from day one |
| Schema bug breaks production | Low | High | Feature flag in Phase 5; canary; auto-rollback if `catalog.signature_invalid` events spike |
| Scraper produces bad PR that gets merged | Low | Medium (one bad price) | Confidence gating; high-confidence requires ≥2 source agreement; humans review all merges |
| Old env var removed too early | Medium | Low (single-line operator config change) | 6-month deprecation window with loud logs |
| External contributors push malicious YAML | Low | High (poisoned price) | CODEOWNERS review required for `providers/**`; signed releases |

## Total timeline

| Phase | Calendar duration | Cumulative |
|---|---|---|
| 0 — inventory & freeze | 1 day | Day 1 |
| 1 — decouple distribution | 1–2 days | Day 3 |
| 2 — split into YAML | 3–5 days | Day 8 |
| 3 — schema + lint | 3–5 days | Day 13 |
| 4 — `extends` collapse | 1 week | Day 20 |
| 5 — gateway-side consumer | 1 week | Day 27 |
| 6 — scrapers + automation | 4–6 weeks (overlapping) | Day 50 |
| 7 — sunset old file | 2 days | Day 180 (after 6mo bake) |

**MVP** (Phases 0–4) lands in ~3 weeks. **Production-ready** (Phases 0–5 + first oracle scrapers) lands in ~5 weeks. **Full automation** by ~10 weeks.
