# 11 — Competitor Analysis

How the four established LLM-catalog systems work, what they got right, what they missed, and where Ferro Model Catalog improves on them.

## Summary table

| System | Repo strategy | Storage | Update flow | Distribution | Validation | Provenance |
|---|---|---|---|---|---|---|
| **LiteLLM** | Single file in main repo | One JSON file (`model_prices_and_context_window.json`) | Manual PR; `aliases[]` to dedupe; hot-reload endpoint | `raw.githubusercontent.com` | Spec doc, schema in code | None per-field |
| **models.dev (SST)** | **Separate repo** ([sst/models.dev](https://github.com/sst/models.dev)) | TOML per model (`providers/<id>/models/<model>.toml`); `extends` for wrappers | PRs only | Generated `_api.json` + REST API | GitHub Action validates schema, types, ranges | None per-field |
| **Portkey** | **Separate repo** ([Portkey-AI/models](https://github.com/Portkey-AI/models)) | Per-provider JSON (`pricing/<provider>.json`); cents/token unit | PRs with source link | CDN: `configs.portkey.ai`; 24h client cache; air-gapped support | OpenAPI schema | None per-field |
| **OpenRouter** | Closed-source | Internal DB | Internal only | Dynamic `/api/v1/models`; real-time uptime per provider | Internal | None exposed |
| **Vercel AI Gateway** | Closed-source | Internal DB | Internal only | Dynamic `/v1/models` | Internal | None exposed |
| **Ferro Model Catalog** (this repo) | Separate public repo | Per-model YAML; `extends` inheritance | PRs + scraper auto-PRs with cross-source confidence | CDN + GH Releases + jsDelivr; signed; per-provider slices | JSON Schema + lint + monotonic verified_at | **Per-field**: url + verified_at + snapshot_sha + confidence |

## LiteLLM — the OG catalog

**What's good:**
- Most cited LLM cost map; broad community adoption
- `aliases[]` field lets dated/legacy IDs share one entry
- Hot-reload via `/reload/model_cost_map` and `/schedule/model_cost_map_reload?hours=N`
- `LITELLM_MODEL_COST_MAP_URL` env var to override source
- Bonus features for tier-based pricing: `input_cost_per_token_above_200k_tokens`, `input_cost_per_token_priority`, `input_cost_per_token_flex`

**What we improve on:**
- **Single-file diff hell** — every PR conflicts on the same JSON. We use per-model YAML.
- **No per-field provenance** — `verified_at` lives at the row level (or not at all). We track every field's source URL and verification timestamp.
- **No automated deprecation cleanup** — open issue [#21240](https://github.com/BerriAI/litellm/issues/21240) requests this; we ship it day one via `prune-monthly.yml`.
- **No signing** — anyone can MITM a catalog fetch. We sign every release with cosign keyless.
- **No scrapers** — every update is a manual PR. We automate ≥50% with cross-source verification.

**What we keep:**
- The hot-reload endpoint pattern (`POST /admin/catalog/reload`)
- `aliases[]` for dated variants
- Optional URL override env var (so air-gapped operators can point at private mirrors)

## models.dev (SST) — the cleanest split

**What's good:**
- **Separate public repo** with permissive license; community contributions flow naturally
- TOML per model — readable, comment-friendly, schema-validated
- **`extends` inheritance** — exactly the pattern that solves the Bedrock/Vertex wrapper explosion. We adopt this directly.
- GitHub Action validates schema, types, ranges, TOML syntax on every PR
- `bun run compare:migrations` shows generated-output diff before/after `extends` migrations
- Used internally by `opencode` (the SST coding agent) — proven runtime consumer

**What we improve on:**
- **TOML is fine but YAML wins for our needs** — anchors, multi-line strings for source notes, comments per field
- **No formal scraper layer** — community-only updates mean drift is real
- **No signing or content-addressed URLs** — fine for free-tier, weak for production gateways
- **No per-field provenance** — same gap as LiteLLM
- **No per-provider slicing** — consumers fetch the whole `_api.json` even if they need 3 providers

**What we keep:**
- The repo split + per-model file pattern
- The `extends` keyword and merge semantics
- The `compare:migrations` UX as `ferrocat diff`'s wrapper-impact view
- Generated `dist/api.json` for browsers/UIs

## Portkey — the production-grade reference

**What's good:**
- **Per-provider JSON files** (`pricing/<provider>.json`) — minimal merge conflicts, clear ownership
- Powers cost attribution for "200+ enterprises running 400B+ tokens daily" — battle-tested
- Strong air-gapped story: customers can mount the JSON files as volumes or pull from a Helm-distributed local copy
- 24-hour gateway-side cache with central control-plane refresh
- Documented multi-deployment-mode pricing strategy (SaaS instant, hybrid 24h, air-gapped configurable)
- "Cents per token" base unit — no decimal-place ambiguity (we use USD/M tokens because that matches every provider's published page)

**What we improve on:**
- **JSON files mean no comments** — provenance has nowhere to live in-line
- **No `extends`** — Bedrock and Vertex partner models duplicated by hand
- **No signing** — same issue as everyone else
- **No scrapers shipped publicly** — Portkey has internal price update tooling but it's not in the OSS repo
- **Cents-per-token unit is a footgun** — readers misread `0.003` as $0.003 vs $30 per 1M tokens. USD/M is what providers publish.

**What we keep:**
- Per-provider sliced distribution (their 24h cache pattern — we make it 1h with hot-reload)
- The "PR with source link" requirement (we enforce via schema)
- Air-gapped deployment support (`FERRO_MODEL_CATALOG_BASE_URL` env override + GH Releases as offline mirror)

## OpenRouter — the dynamic API model

**What's good:**
- `GET /api/v1/models` returns 300+ models with pricing in one call — incredible reference oracle
- Real-time uptime tracking per provider endpoint
- Rich schema: `architecture` (input/output modalities, tokenizer), `top_provider`, `supported_parameters`, `default_parameters`, `expiration_date`
- Permaslug system: `canonical_slug` is permanent, `id` can change (we adopt this distinction with `aliases[]`)
- `/v1/models?supported_parameters=tools` — queryable by capability

**What we improve on:**
- **Closed-source** — operators can't audit, contribute, or self-host
- **No write path for the community** — bug fixes require Discord pings
- **Cents/USD/string-typed pricing** in JSON (`"prompt": "0.000003"`) — slightly awkward
- **Tied to OpenRouter's commercial routing** — pricing reflects OpenRouter's 5.5% margin, not raw provider price

**What we keep:**
- The rich schema — we're a superset
- Use `/api/v1/models` as our **highest-leverage oracle scraper** (Phase A, Week 1) — 300+ models cross-checked in one HTTP call

## Vercel AI Gateway — the platform-native model

**What's good:**
- Tightly integrated with Vercel platform; OIDC auth
- Live `/v1/models` for connected providers
- "Cache Components" pattern for client-side caching

**What we improve on:**
- **Closed-source, vendor-locked** — same critique as OpenRouter
- **Vercel-bound pricing model** — usage rolled into Vercel bandwidth and function execution
- **No exportable artifact** — you can't take the catalog elsewhere

We learn from their UX (manifest-pointer pattern is similar) but the product positioning is different.

## What nobody else has

These are the **specific differentiators** that make Ferro Model Catalog defensible without being closed:

| Feature | LiteLLM | models.dev | Portkey | OpenRouter | **Ferro** |
|---|---|---|---|---|---|
| Separate public repo | ✗ | ✓ | ✓ | ✗ | ✓ |
| Per-model file | ✗ | ✓ | ✗ (per-provider) | n/a | ✓ |
| `extends` inheritance | ✗ | ✓ | ✗ | n/a | ✓ |
| Per-field provenance (URL + timestamp + snapshot) | ✗ | ✗ | ✗ | ✗ | ✓ |
| Cosign-signed releases | ✗ | ✗ | ✗ | n/a | ✓ |
| Cross-source scrapers (5 tiers) | ✗ | ✗ | ✗ | n/a | ✓ |
| Confidence scoring per field | ✗ | ✗ | ✗ | ✗ | ✓ |
| Per-provider lazy loading | ✗ | ✗ | ✓ | n/a | ✓ |
| Auto-prune deprecated entries | ✗ (open issue) | ✗ | ✗ | ✓ (404s) | ✓ |
| Air-gapped support | ✓ | ✗ | ✓ | ✗ | ✓ |
| Hot-reload without restart | ✓ | ✗ | ✓ (24h cache refresh) | n/a | ✓ |
| Snapshot evidence (raw HTML stored) | ✗ | ✗ | ✗ | ✗ | ✓ |
| Wrapper blast-radius diffs | ✗ | partial | ✗ | n/a | ✓ |

## Why we don't try to compete on closed dimensions

OpenRouter and Vercel AI Gateway have things we can't replicate without becoming a routing service:
- Real-time uptime per endpoint (requires running production traffic)
- Live latency benchmarks
- Margin-aware pricing tiers

These belong in the **gateway runtime** (`ai-gateway`) and **performance benchmark repo** (`ai-gateway-performance-benchmarks`), not the catalog. The catalog is metadata.

## Lessons we explicitly imported

| From | Lesson | Where applied |
|---|---|---|
| LiteLLM | Hot-reload endpoint without restart | `POST /admin/catalog/reload` ([08-gateway-integration.md](./08-gateway-integration.md)) |
| LiteLLM | URL override env var for air-gap | `FERRO_MODEL_CATALOG_BASE_URL` |
| LiteLLM | `aliases[]` field for dated variants | Schema ([03-schema.md](./03-schema.md)) |
| models.dev | Per-model file structure | Repo layout ([02-repo-structure.md](./02-repo-structure.md)) |
| models.dev | `extends` inheritance | [04-extends-pattern.md](./04-extends-pattern.md) |
| models.dev | Generated diff comparison tool | `ferrocat diff` |
| Portkey | Per-provider sliced distribution | [07-distribution.md](./07-distribution.md) |
| Portkey | Air-gapped deployment story | Three-tier fallback (CDN → GH Releases → embedded) |
| OpenRouter | `/api/v1/models` as oracle | Phase A scraper Week 1 ([10-roadmap.md](./10-roadmap.md)) |
| OpenRouter | Rich schema (architecture, supported_parameters) | Schema superset |
