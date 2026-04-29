# 01 — Architecture Overview

The catalog is a **three-layer system**: source-of-truth files, automation pipelines, distribution infrastructure. Each layer has clear responsibilities and contracts.

## High-level diagram

```
┌──────────────────────────────────────────────────────────────────────────┐
│                       LAYER 1 — SOURCE OF TRUTH                          │
│                                                                          │
│   providers/<id>/provider.yaml + providers/<id>/models/*.yaml            │
│   Human-edited, schema-validated, git-versioned                          │
│                                                                          │
└────────────────────┬─────────────────────────────────────┬───────────────┘
                     │                                     │
                     │ on PR                               │ on merge
                     ▼                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                        LAYER 2 — AUTOMATION                              │
│                                                                          │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│   │ validate │  │   diff   │  │   build  │  │  scrape  │  │  prune   │  │
│   │  (PR)    │  │   (PR)   │  │ (merge)  │  │ (cron)   │  │ (cron)   │  │
│   └──────────┘  └──────────┘  └──────────┘  └──────────┘  └──────────┘  │
│        │             │             │              │             │        │
│        └─────────────┴─────────────┼──────────────┴─────────────┘        │
│                                    │                                     │
│                                    ▼                                     │
│                          dist/catalog.json                               │
│                          dist/providers/<id>.json                        │
│                          dist/manifest.json (signed)                     │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     │
                                     │ git tag + release
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                       LAYER 3 — DISTRIBUTION                             │
│                                                                          │
│   catalog.ferrolabs.ai/v1/manifest.json    ← pointer (small, frequent)   │
│   catalog.ferrolabs.ai/v1/<sha>.json       ← immutable, content-hashed   │
│   catalog.ferrolabs.ai/v1/providers/<id>.json                            │
│                                                                          │
└────────────────────────────────────┬─────────────────────────────────────┘
                                     │
                                     │ HTTPS + signature verification
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                      CONSUMERS                                           │
│                                                                          │
│   • ai-gateway (OSS runtime)        ← lazy-loads providers it needs      │
│   • FerroCloud (multi-tenant)       ← merges with private overlay        │
│   • ai-gateway-gtm-site             ← powers public model browser        │
│   • Third-party tools (Aider, etc.) ← consume our public artifacts       │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

## Layer 1 — Source of truth

**Format:** YAML, one file per model, organized by provider.

**Why YAML, not JSON or TOML:**
- Comments — every price field can carry a `# source: …` line for provenance
- Anchors and `extends:` for inheritance (Bedrock/Vertex wrappers, see [04-extends-pattern.md](./04-extends-pattern.md))
- Human-reviewable diffs in pull requests
- Schema-friendly (JSON Schema works on YAML directly)

JSON files are **never edited by hand** — they are generated artifacts in `dist/`.

**Authority:**
- Provider section (`providers/<id>/provider.yaml`) defines auth style, base URLs, env vars, region list
- Model files define pricing, capabilities, lifecycle, sources
- Override files (`overrides/YYYY-MM-DD-*.yaml`) apply timestamped patches without losing history

## Layer 2 — Automation

Five GitHub Actions, each with a single responsibility:

| Workflow | Trigger | Job |
|---|---|---|
| `validate.yml` | every PR | JSON Schema check + key-shape regex + dedupe + monotonic `verified_at` |
| `diff.yml` | every PR | Render before/after JSON diff as a PR comment |
| `build.yml` | merge to main | Generate `dist/`, sign manifest, push to CDN |
| `scrape-weekly.yml` | cron (Mon 06:00 UTC) | Run all scrapers, open PRs with detected price/capability deltas |
| `prune-monthly.yml` | cron (1st of month) | Auto-PR removing entries whose `sunset_date < today − 90d` |

Each workflow is documented in [06-automation.md](./06-automation.md).

The scrapers (`tools/ferrocat scrape`) are the engine that fixes consistency — they cross-reference up to five sources per model and only auto-merge when ≥2 agree. See [05-scraping.md](./05-scraping.md).

## Layer 3 — Distribution

Static artifacts served from a CDN with three guarantees:

1. **Versioned & content-addressed** — `<sha>.json` URLs are immutable; the `manifest.json` pointer indicates the current version
2. **Signed** — every release is signed with cosign (keyless, OIDC-bound to this repo)
3. **Sliced** — per-provider files (`providers/<id>.json`) so a gateway using only OpenAI doesn't download Bedrock data

Gateway-side, this replaces the current single-URL fetch with:
- Pin a SHA at build time (embedded `manifest.json`)
- Refuse remote updates that don't roll forward from it
- Verify signature before swapping in memory
- Optional hot-reload via admin endpoint

Details in [07-distribution.md](./07-distribution.md) and [08-gateway-integration.md](./08-gateway-integration.md).

## Trust boundary

The build pipeline is the trust boundary. Anything **before** build is human-readable and reviewer-checkable; anything **after** build is signed and immutable.

| Boundary | Who can write | What enforces it |
|---|---|---|
| `providers/` and `overrides/` (YAML) | Repo collaborators + accepted external PRs | GitHub branch protection + `validate.yml` |
| `dist/` (JSON, signed) | Only the `build.yml` workflow on `main` | OIDC + cosign keyless signing |
| CDN | Only the deploy workflow with scoped credentials | Cloudflare API token + GitHub OIDC |

This is how `dist/` can be safely committed to the repo *and* be considered a release artifact — the signed manifest proves it came from a trusted CI run.

## Key design decisions

| Decision | Why |
|---|---|
| Per-model YAML files (not single JSON) | Fixes diff hell, makes PRs reviewable, enables `extends` |
| Build to JSON (not just serve YAML) | Gateways stay zero-dep on YAML libs; existing consumers keep working |
| `extends` inheritance | Collapses 600+ Bedrock/Vertex wrapper rows onto base entries |
| Per-field provenance (`sources.pricing.url + verified_at`) | Lets us prove pricing accuracy and target re-verification surgically |
| Content-addressed URLs + manifest pointer | Standard CDN cache-busting; gateways can pin a known-good SHA |
| Signed releases | Defends against compromised contributor credentials or CDN takeover |
| Per-provider slices | Bandwidth + memory savings for typical 3–5-provider deployments |
| Public OSS catalog + private overlay | Keeps community asset open while permitting tenant-specific pricing |

## Non-decisions (deferred)

These can be added without breaking the architecture:

- A queryable HTTP API in front of the CDN (serve query params like models.dev)
- Live benchmarking integration (separate repo)
- Web UI for browsing the catalog (lives in the FerroLabs site)
- gRPC delivery — JSON over HTTPS is fine for catalog-scale data
