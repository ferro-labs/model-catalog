# 02 — Repo Structure

```
model-catalog/
├── README.md                              # Public-facing intro + quick start for contributors
├── LICENSE                                # Apache-2.0
├── CONTRIBUTING.md                        # How to add/update a model (5-min path)
├── CODE_OF_CONDUCT.md
├── SECURITY.md                            # Disclosure policy for malicious PRs
├── .gitignore
│
├── docs/                                  # ← THIS FOLDER (planning + design docs)
│   ├── README.md
│   ├── 00-vision.md
│   ├── 01-architecture.md
│   └── …
│
├── schema/                                # JSON Schema (draft 2020-12)
│   ├── provider.schema.json
│   ├── model.schema.json
│   ├── override.schema.json
│   └── manifest.schema.json
│
├── providers/                             # ← Source of truth, human-edited
│   ├── openai/
│   │   ├── provider.yaml
│   │   └── models/
│   │       ├── gpt-5.yaml
│   │       ├── gpt-5-mini.yaml
│   │       ├── gpt-4o.yaml
│   │       └── …
│   ├── anthropic/
│   │   ├── provider.yaml
│   │   └── models/
│   │       ├── claude-sonnet-4-5.yaml
│   │       ├── claude-haiku-4-5.yaml
│   │       └── …
│   ├── bedrock/
│   │   ├── provider.yaml
│   │   └── models/
│   │       └── anthropic.claude-sonnet-4-5.yaml   # uses extends: anthropic/claude-sonnet-4-5
│   ├── vertex_ai/
│   │   ├── provider.yaml
│   │   └── models/
│   │       ├── gemini-3-pro.yaml
│   │       └── claude-sonnet-4-5.yaml             # extends: anthropic/claude-sonnet-4-5
│   ├── openrouter/
│   ├── groq/
│   ├── mistral/
│   └── …                                          # ~30 providers at launch, ~75 long-term
│
├── overrides/                             # Dated patches with audit trail
│   ├── 2026-04-15-anthropic-price-cut.yaml
│   ├── 2026-05-01-openai-gpt-5-launch.yaml
│   └── README.md                          # When and why to use overrides
│
├── snapshots/                             # Raw scraped HTML/JSON (separate orphan branch)
│   └── (see snapshots branch)
│
├── dist/                                  # ← Generated artifacts, committed for CDN
│   ├── catalog.json                       # Full flat map (current shape, backwards-compat)
│   ├── catalog.minified.json
│   ├── providers/
│   │   ├── openai.json
│   │   ├── anthropic.json
│   │   └── …
│   ├── manifest.json                      # version, sha, generated_at, signature
│   └── manifest.json.sig                  # cosign signature
│
├── tools/                                 # Go tooling
│   └── ferrocat/                          # Single CLI binary
│       ├── main.go
│       ├── cmd/
│       │   ├── validate.go                # ferrocat validate
│       │   ├── build.go                   # ferrocat build
│       │   ├── diff.go                    # ferrocat diff <ref>
│       │   ├── scrape.go                  # ferrocat scrape [--provider X]
│       │   ├── lint.go                    # ferrocat lint (key shape, dedupe)
│       │   ├── split.go                   # ferrocat split <legacy.json>  (one-time migration)
│       │   └── prune.go                   # ferrocat prune (deprecated entries)
│       ├── internal/
│       │   ├── schema/                    # Schema loader + validator
│       │   ├── extends/                   # Inheritance resolution
│       │   ├── diff/                      # Pretty PR diffs
│       │   └── snapshot/                  # SHA + screenshot helpers
│       └── scrape/
│           ├── interface.go               # Backend abstraction
│           ├── oracle/                    # Cross-check sources
│           │   ├── openrouter.go          # OpenRouter /api/v1/models
│           │   └── models_dev.go          # models.dev api.json
│           ├── api/                       # Provider /v1/models endpoints
│           │   ├── openai.go
│           │   ├── anthropic.go
│           │   ├── groq.go
│           │   └── …
│           ├── pricing_api/               # Cloud provider pricing APIs
│           │   ├── aws_bedrock.go         # AWS Pricing API
│           │   ├── azure.go               # prices.azure.com
│           │   └── vertex.go              # GCP Cloud Billing Catalog
│           ├── html/                      # HTML scrapers (server-rendered)
│           │   ├── mistral.go
│           │   ├── cohere.go
│           │   └── …
│           ├── browser/                   # Headless Chromium for SPA pages
│           │   ├── chromedp_backend.go    # Default (free, GitHub Actions)
│           │   ├── cloudflare_backend.go  # Optional (paid, --backend=cloudflare)
│           │   └── pages/
│           │       ├── openai_pricing.go
│           │       ├── anthropic_pricing.go
│           │       └── gemini_pricing.go
│           └── docs/                      # Markdown docs scrapers
│               ├── github_openapi.go      # github.com/openai/openai-openapi
│               └── google_genai_docs.go
│
├── .github/
│   ├── CODEOWNERS                         # Per-provider owners (auto-assign reviewers)
│   ├── pull_request_template.md
│   ├── ISSUE_TEMPLATE/
│   │   ├── new_model.md
│   │   ├── price_correction.md
│   │   └── new_provider.md
│   └── workflows/
│       ├── validate.yml                   # PR gate
│       ├── diff.yml                       # PR comment with rendered diff
│       ├── build.yml                      # main → dist/ + CDN
│       ├── scrape-weekly.yml              # cron, opens auto-PRs
│       ├── live-probe.yml                 # nightly, deprecation detection
│       ├── prune-monthly.yml              # cron, removes sunset entries
│       ├── release.yml                    # tag → signed release
│       └── codeql.yml                     # security scan on Go tools
│
├── go.mod
├── go.sum
└── Makefile                               # Convenience: make build/test/lint
```

## File-naming conventions

### Provider IDs
- Lowercase, snake_case, alphanumeric: `openai`, `vertex_ai`, `azure_foundry`
- Match the value used in `Provider.Name` constants in the gateway repo (`providers/names.go`)
- One provider = one folder; **no aliasing folders** like `vertex_ai-deepseek_models` (use `extends` instead)

### Model IDs (file names)
- Match the **canonical provider model ID** exactly: `gpt-5.yaml`, `claude-sonnet-4-5.yaml`
- Replace `/` with `__` if the model ID contains slashes (rare): `meta-llama__Meta-Llama-3.1-70B-Instruct.yaml`
- Replace `:` with `_` for Bedrock-style ARNs: `anthropic.claude-sonnet-4-5-v1_0.yaml`

### Override files
- `overrides/YYYY-MM-DD-<short-description>.yaml`
- Examples: `overrides/2026-04-15-anthropic-price-cut.yaml`, `overrides/2026-05-01-openai-gpt-5-launch.yaml`
- Auto-applied by build, ordered by date (newer wins)

## What lives where, decision matrix

| If you're updating… | Edit… | Don't edit… |
|---|---|---|
| Existing model price | `providers/<id>/models/<model>.yaml` | `dist/`, `overrides/` |
| New model | Add `providers/<id>/models/<model>.yaml` | — |
| New provider | Add `providers/<id>/provider.yaml` + at least one model | — |
| Bulk price change with effective date | Add `overrides/YYYY-MM-DD-*.yaml` | The base model files |
| Deprecation date / status | The model file's `lifecycle:` section | — |
| Schema field added | `schema/model.schema.json` + bump schema version | — |

## Branch model

- `main` — protected, signed releases originate here
- `pr/*` — short-lived, auto-merged by Mergify when checks pass + CODEOWNER approves
- `snapshots` — orphan branch holding raw scraped HTML/JSON for audit trail
- `release/*` — release candidates if we ever need to test before tagging

## Tagging and releases

- Catalog uses CalVer: `v2026.04.28` plus a daily counter if needed: `v2026.04.28.1`
- Each tag triggers `release.yml` which:
  - Signs `dist/manifest.json` with cosign
  - Publishes a GitHub Release with the JSON artifacts attached
  - Pushes to CDN with cache-purge
  - Sends a webhook to the gateway repo (optional, for hot-reload)

## Why not split tools/ into its own repo?

Considered. Decided against because:
- The tools (validate, build, scrape) have a 1:1 dependency on the schema — versioning them separately doubles complexity
- Single Go module = single `go.mod`, simpler for contributors
- The catalog repo's CI already runs Go anyway (it's how `dist/` is generated)

If `tools/ferrocat` ever becomes useful as a standalone CLI for *other* catalog projects, we can extract it. Not before then.
