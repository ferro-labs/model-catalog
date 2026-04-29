# 00 — Vision & OSS Strategy

## Problem statement

The FerroGateway OSS repo currently ships a single 111,365-line `models/catalog.json` containing 2,531 model entries across 75 "providers." It is fetched from `raw.githubusercontent.com` at gateway startup with the bundled copy as silent fallback.

**Concrete pains observed today** (from a fresh read of the gateway repo):

| Symptom | Evidence |
|---|---|
| Diff hell on every catalog edit | 111K-line file, single map, every PR conflicts |
| Junk keys mixed with models | e.g. `bedrock/1024-x-1024/50-steps/bedrock/amazon.nova-canvas-v1:0` — that's a *parameterized request*, not a model |
| Duplicate provider namespaces | `bedrock`, `amazon_nova`, `vertex_ai-deepseek_models`, `vertex_ai-zai_models` all repeat the same base models |
| No per-field provenance | One `updated_at` per row, can't tell what was actually re-verified |
| Pricing accuracy is unverifiable | The existing `scripts/catalog-check` only proves URLs return HTTP 200, not that prices match the page |
| No automated deprecation cleanup | `IsDeprecated()` exists but nothing prunes by `sunset_date` |
| Single live catalog URL is unsigned | One bad PR poisons every gateway via `defaultCatalogURL`, with a 1-second timeout silently masking corruption |

## Why a separate public repo

### Decoupling

| Lives in `ai-gateway` | Lives in `model-catalog` |
|---|---|
| Gateway runtime (Go) | Catalog data (YAML) |
| Routing strategies, plugins | Schema, scrapers, validators |
| HTTP server, admin API | Generated JSON artifacts |
| Provider client code | Provider metadata only |

A pricing PR no longer churns gateway commit history, doesn't re-trigger the full gateway CI/release pipeline, and doesn't block on Go reviewers. The catalog can release on its own cadence (potentially daily) while the gateway releases weekly.

### Community contributions

The realistic contributor for "Together AI just dropped a new model and the price is $X" is a developer who uses Together — not a Go engineer. Lowering the barrier to a YAML PR in a small, scoped repo radically increases the contributor pool. This is exactly the dynamic that grew BerriAI/litellm, Portkey-AI/models, and sst/models.dev.

### Reuse beyond FerroGateway

Once the catalog is a clean public resource, we can:
- Power the FerroLabs marketing-site model browser from it
- Let FerroCloud customers self-audit pricing
- Let downstream OSS projects (Aider, OpenCode, custom internal tools) consume our `dist/catalog.json` the same way they consume LiteLLM's

That makes the catalog **distribution infrastructure**, not a feature.

## Why OSS (not closed)

The catalog is **not the product**. The product is:
- The gateway runtime — routing, plugins, circuit breaking, streaming
- FerroCloud's multi-tenancy, billing, observability layer
- The integration depth across 75+ providers

A pricing/capability table for publicly-published model APIs has zero defensive value. Three reasons OSS is correct:

1. **Trust** — operators won't deploy an OSS gateway whose pricing source they can't audit. Open catalog = auditable cost tracking.
2. **Flywheel** — every external contributor improves data accuracy at no engineering cost.
3. **Brand** — being the de-facto open catalog is a marketing asset for FerroLabs (Portkey gets cited every time someone benchmarks pricing accuracy across gateways).

License: **Apache-2.0**, matching `ai-gateway`.

## What stays private

Enterprise/FerroCloud-specific overlays do **not** go here:

- Negotiated tenant pricing
- Internal cost tiers
- Pre-release model entries under NDA
- Customer-specific model aliases

These live in a private overlay repo (`ferrocloud-catalog-overlay`) that the build pipeline merges *after* the public build. Same pattern Portkey uses with their hosted config service. See [07-distribution.md](./07-distribution.md) for the merge model.

## Non-goals

To prevent scope creep, this repo will explicitly **not**:

- Rate or rank models on quality (that's a separate `model-rankings` repo)
- Provide latency benchmarks (separate `ai-gateway-performance-benchmarks` already exists)
- Host model weights, prompts, or tokenizers
- Implement any gateway logic — this is data + tooling only

## Success criteria

We declare this repo successful when:

1. The gateway no longer ships a bundled `catalog.json` (only a manifest pointer)
2. Day-0 model launches reflect in production gateways within 24h *without operator action*
3. > 50% of catalog updates are auto-PRs from scrapers, not manual edits
4. External contributors regularly PR new providers (target: ≥5 external PRs/month within 6 months)
5. Pricing accuracy verified against provider-published sources at ≥95% confidence
