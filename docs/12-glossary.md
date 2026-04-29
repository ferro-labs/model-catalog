# 12 — Glossary

Shared vocabulary for this repo and any docs that reference it. When in doubt, use these terms exactly.

## Core entities

**Catalog**
The complete set of model and provider metadata published by this repo. Always refers to the *generated* JSON artifact unless prefixed (e.g., "source catalog" = the YAML files).

**Provider**
An organization or service that exposes model APIs (e.g., `openai`, `anthropic`, `bedrock`). One folder per provider in `providers/`.

**Model**
A single addressable inference target with stable pricing and capabilities (e.g., `gpt-5`, `claude-sonnet-4-5`). One YAML file per model.

**Catalog key**
The fully qualified identifier for a model: `provider/model_id` (e.g., `openai/gpt-5`, `bedrock/anthropic.claude-sonnet-4-5-v1:0`). Used as the map key in `dist/catalog.json`.

**Wrapper**
A model entry that uses `extends:` to inherit from a base model in another provider (e.g., Bedrock or Vertex re-hosting Anthropic's Claude). See [04-extends-pattern.md](./04-extends-pattern.md).

**Base model**
A model entry without `extends:` — the canonical definition that wrappers inherit from.

**Mode**
The kind of inference a model handles. Enum: `chat`, `embedding`, `image`, `audio_in`, `audio_out`, `rerank`, `moderation`, `search`. A wrapper cannot change mode.

**Lifecycle status**
Enum describing where in its life a model is: `preview`, `ga`, `deprecated`, `sunset`. Determines whether `prune-monthly` removes it.

**Override**
A dated patch file (`overrides/YYYY-MM-DD-*.yaml`) that applies on top of one or more model entries at build time. Used for bulk price changes with a clean audit trail.

**Alias**
An alternate model ID that resolves to the same entry (e.g., `gpt-5-2025-08-07` → `gpt-5`). Lets dated/legacy IDs share pricing.

## Provenance

**Source**
The URL where a piece of catalog data was verified from. Stored per field under `sources.<field>.url` in the model YAML.

**verified_at**
Date (YYYY-MM-DD) the field was last confirmed against its source. Must monotonically increase across edits.

**verified_by**
Either `manual:<github-username>` for human edits or `scraper-<name>-v<N>` for auto-PRs.

**Snapshot**
The raw HTML/JSON content of a source page at the moment it was scraped. Stored on the orphan `snapshots` branch with SHA-256 referenced in the model YAML.

**Confidence**
A scraper-emitted rating (`high`, `medium`, `low`, `conflict`) reflecting how many independent sources agreed on a given field.

## Distribution

**Manifest**
The pointer file (`dist/manifest.json`) that lists the current catalog SHA, schema version, per-provider slice URLs, and signature. Gateways fetch this most often.

**Slice (per-provider slice)**
A subset of the catalog containing only one provider's models (`dist/providers/<id>.json`). Gateways fetch only the slices for providers they're configured to use.

**Content-addressed URL**
An immutable URL whose path includes the SHA of the file (`/v1/<sha>.json`). Cacheable forever, never changes contents.

**Latest pointer**
A short-lived URL (`/v1/manifest.json`, `/v1/latest.json`) that points at the current SHA. Cached briefly with `stale-while-revalidate`.

**Pin**
A SHA value baked into a gateway binary at build time (`manifest_pin.json`) representing the last-known-good catalog version. Defends against rollback attacks.

**Hot-reload**
The act of swapping the in-memory catalog without restarting the gateway. Triggered by background ticker or admin endpoint.

## Automation

**Scraper**
A Go function implementing the `Scraper` interface that fetches model data from a single source and emits `Observation`s.

**Tier (1–5)**
Reliability rating of a scraper source: 1 = live `/v1/models` API, 2 = cloud pricing API, 3 = server-rendered HTML, 4 = provider repo Markdown docs, 5 = headless browser. See [05-scraping.md](./05-scraping.md).

**Oracle**
A cross-check source (currently OpenRouter `/api/v1/models` and `models.dev/api.json`) used to verify scraper output across many providers in one call.

**Reconciler**
The component that combines `Observation`s from multiple scrapers into a single proposed model state plus a confidence rating.

**Auto-PR**
A pull request opened by `scrape-weekly.yml` containing scraper-detected changes. Labeled with `auto-pr` plus a confidence label.

**Failure streak**
The count of consecutive scrape runs in which a given scraper failed. At 3, an issue opens; at 6, the scraper is marked disabled.

## Trust & security

**Cosign keyless**
Signature mechanism using a short-lived certificate bound to the GitHub Actions OIDC token. No long-lived keys to manage. Used to sign every published manifest.

**Rekor**
Sigstore's transparency log, where every signature is recorded immutably. Anyone can verify a signature was issued at a specific time by a specific workflow.

**Trust boundary**
The point in the pipeline beyond which content is considered authoritative. For us: the merge to `main` + `build.yml` workflow. Anything before is reviewable; anything after is signed.

**Air-gapped**
A deployment with no outbound internet access. Supported via `FERRO_MODEL_CATALOG_BASE_URL` env override pointing at an internal mirror, plus the gateway's embedded fallback.

## Repos & components

**ai-gateway**
The OSS gateway runtime in `github.com/ferro-labs/ai-gateway`. Consumes this catalog. Currently ships the catalog inline; will switch to remote consumption per [08-gateway-integration.md](./08-gateway-integration.md).

**FerroCloud**
The proprietary multi-tenant management layer. Imports `ai-gateway` as a Go library and runs one gateway instance per tenant. Adds private overlay catalog for tenant-specific pricing.

**model-catalog**
This repo. Source of truth + automation + distribution for the public catalog.

**ferrocloud-catalog-overlay**
Private repo (separate from this one) containing tenant-specific pricing overlays applied at FerroCloud build time.

**ferrocat**
The Go CLI in `tools/ferrocat`. One binary, multiple subcommands (`validate`, `build`, `diff`, `scrape`, `lint`, `split`, `prune`).

## Versioning

**CalVer**
Calendar-based versioning: `vYYYY.MM.DD[.N]`. Used for catalog releases (e.g., `v2026.04.28`, `v2026.04.28.1`). Different from SemVer used by `ai-gateway`.

**Schema version**
An integer in the manifest (`schema_version: 1`). Bumped when the catalog schema makes incompatible changes. Major bumps publish under a new URL prefix (`/v2/`); old versions stay live for ≥6 months.

**Bake period**
Time between flipping a feature flag to default and removing the old code path. For the gateway-side migration: 1 release minimum, typically 6 months ([09-migration-plan.md](./09-migration-plan.md)).

## Money

**USD per 1M tokens**
The base unit for all `pricing.input`, `pricing.output`, `pricing.cache_read`, etc. Why: matches how every provider publishes prices (eliminates conversion errors).

**`null` vs `0`**
- `null` = field is not applicable to this mode (e.g., `embedding` price on a chat model)
- `0` = the field is explicitly free (e.g., a self-hosted Ollama model)

**Tier (pricing)**
A pricing variant triggered by request metadata (Vertex `priority`/`flex`, Bedrock service tiers). Stored under `pricing.tiers[]` with a `trigger` field that the gateway matches against the response.

**Above-threshold pricing**
Different rates that kick in once a request crosses a token boundary (Gemini 2M-token tier, Claude 1M-token tier). Stored under `pricing.above_threshold`.

## File / path patterns

**`provider.yaml`**
Defines a provider. One per `providers/<id>/` folder.

**`<model-id>.yaml`**
Defines a model. Lives under `providers/<id>/models/`. Filename matches the model ID (with `:` → `_` and `/` → `__` substitution).

**`overrides/YYYY-MM-DD-<short>.yaml`**
A dated patch file. Sorted by date at build time; newer wins.

**`dist/catalog.json`**
Generated full catalog in legacy single-map shape. Backwards-compatible with current `ai-gateway` consumer.

**`dist/providers/<id>.json`**
Generated per-provider slice. Used for lazy loading.

**`dist/manifest.json`**
Generated pointer file. The most-fetched artifact.

**`snapshots/`**
Orphan branch holding raw scraped content. Not on `main`; reachable only via `git checkout snapshots`.
