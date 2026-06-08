# AGENTS.md

## Project Overview

**Ferro Model Catalog** is an open-source, community-maintained database of LLM model pricing, capabilities, and lifecycle metadata. It serves as the single source of truth for the [Ferro Labs AI Gateway](https://github.com/ferro-labs/ai-gateway).

- **Module**: `github.com/ferro-labs/model-catalog`
- **Go version**: 1.24+
- **License**: Apache 2.0
- **Data**: 2,505 models across 83 providers

---

## Build, Test, and Run Commands

```bash
# Build the catalog (YAML → JSON + slices + manifest)
make build

# Run all tests (including round-trip regression)
make test

# Validate structural correctness
make validate

# Lint for junk keys and duplicates
go run ./cmd/ferrocat lint

# Run scrapers against OpenRouter + models.dev
go run ./cmd/ferrocat scrape

# Check catalog freshness against provider APIs (needs API keys)
ANTHROPIC_API_KEY=... OPENAI_API_KEY=... go run ./cmd/ferrocat freshness

# Format Go code
make fmt
```

### Individual commands

```bash
go run ./cmd/ferrocat build --output dist/
go run ./cmd/ferrocat validate
go run ./cmd/ferrocat lint
go run ./cmd/ferrocat scrape
go run ./cmd/ferrocat freshness
go run ./cmd/ferrocat split <input.json> --output providers/
go run ./cmd/ferrocat migrate-extends --wrapper <provider> --base <base-provider>
```

---

## Project Structure

```
model-catalog/
├── cmd/ferrocat/               # CLI entry point (main.go only)
├── catalog/                    # Public Go library (importable)
│   ├── types.go                # Entry, NullFloat64, Pricing, Capabilities, Lifecycle
│   ├── json.go                 # ReadCatalogJSON, WriteCatalogJSON
│   ├── yaml.go                 # ReadModelYAML, WriteModelYAML
│   ├── extends.go              # ResolveExtends (deep-merge inheritance)
│   ├── build.go                # Build() — YAML → JSON + slices + manifest
│   ├── split.go                # Split(), SanitizeFilename()
│   ├── validate.go             # Validate(), ValidationError
│   ├── lint.go                 # Lint(), IsJunkKey(), LintIssue
│   ├── migrate.go              # MigrateExtends(), ReadProviderModels()
│   ├── manifest.go             # Manifest types
│   ├── yamlnode.go             # YAML node helpers for wrapper generation
│   └── *_test.go
├── scrape/                     # Public Go library (importable)
│   ├── types.go                # Observation, Confidence, Scraper interface
│   ├── httputil.go             # FetchJSON() — shared HTTP client with retry
│   ├── reconciler.go           # Cross-check observations against catalog
│   ├── report.go               # Human-readable reports
│   ├── api/
│   │   ├── anthropic.go          # Anthropic /v1/models scraper
│   │   └── openai.go             # OpenAI /v1/models scraper
│   └── oracle/
│       ├── openrouter.go       # OpenRouter /api/v1/models
│       └── models_dev.go       # models.dev /api.json
├── internal/cli/               # Cobra command wiring (thin wrappers, not importable)
├── providers/                  # Source of truth — 2,505 per-model YAML files
├── dist/                       # Generated artifacts (do not edit manually)
│   ├── catalog.json            # Full flat catalog
│   ├── manifest.json           # Version, SHA-256 hashes, stats
│   └── providers/              # 83 per-provider JSON slices
├── docs/architecture.md        # Full technical reference
└── .github/workflows/          # CI: validate.yml (PR gate), build.yml (release)
```

---

## Key Files

| File | Role |
|------|------|
| `catalog/types.go` | Core types: `Entry`, `NullFloat64`, `Pricing`, `Capabilities`, `Lifecycle` |
| `catalog/build.go` | `Build()` — walks providers/, resolves extends, generates dist/ |
| `catalog/extends.go` | `ResolveExtends()` — deep-merge wrapper models onto base entries |
| `catalog/json.go` | `ReadCatalogJSON()`, `WriteCatalogJSON()` — sorted keys, 2-space indent |
| `catalog/validate.go` | `Validate()` — mode/status/tier enum, required fields, provider match |
| `catalog/lint.go` | `Lint()`, `IsJunkKey()` — dimension patterns, duplicate detection |
| `scrape/httputil.go` | `FetchJSON()` — shared HTTP client, User-Agent, 3x retry on 5xx |
| `scrape/reconciler.go` | `Reconcile()` — cross-check scraped data against catalog |
| `cmd/ferrocat/main.go` | CLI entry point — delegates to `internal/cli` |

---

## Architecture & Design Patterns

### Three-layer system

1. **Source of truth**: `providers/<id>/models/*.yaml` — human-edited, schema-validated
2. **Automation**: validate (PR gate) → build (on merge) → scrape (weekly cron)
3. **Distribution**: `dist/catalog.json` + per-provider slices + manifest → GitHub Releases

### Extends inheritance

Wrapper models (e.g., `vertex_ai/gemini-2.0-flash`) use `extends: gemini/gemini-2.0-flash` to inherit from base models. The build resolves inheritance and emits fully-merged entries. Max chain depth = 1. Mode cannot be overridden. 269 wrappers currently use this pattern.

### NullFloat64

Custom type for nullable pricing fields. Preserves `3.0` format in JSON (not `3`). `Valid=false` means null (not applicable), `Valid=true, Value=0` means genuinely free.

### Catalog key

`provider/model_id` (e.g., `openai/gpt-4o`). Used as the map key in `dist/catalog.json`.

### Filename sanitization

Model IDs with `/` become `__`, `:` becomes `_` in filenames. The `SanitizeFilename()` function handles this.

---

## Data Conventions

- **Pricing**: USD per 1,000,000 tokens. `null` = not applicable. `0` = free.
- **Mode**: `chat`, `embedding`, `image`, `audio_in`, `audio_out`
- **Status**: `preview`, `ga`, `deprecated`, `sunset`, `legacy`
- **Tier**: `flagship`, `standard`
- **Provider ID**: lowercase snake_case, must match folder name
- **Versioning**: CalVer `v2026.04.30` with `.N` suffix for same-day releases

---

## Adding or Modifying Models

### Add a new model

Create `providers/<provider>/models/<model-id>.yaml` with all required fields. See any existing model file for the template. Run `make validate` to check.

### Add a wrapper model (extends)

Create a YAML file with `extends: <base-provider>/<model-id>`, plus `provider`, `model_id`, all 12 pricing fields, and all 11 capability fields. The build resolves the rest from the base.

### Run the full pipeline

```bash
make validate && make build && make test
```

---

## Testing Conventions

- All tests in `*_test.go` alongside implementation
- Round-trip regression test in `catalog/roundtrip_test.go` — splits and rebuilds the full catalog
- Scraper tests use `httptest.Server` with fixture data (no real API calls)
- Run with race detector: `go test -race ./...`
- Set `CATALOG_TEST_SOURCE=/path/to/catalog.json` to test against a specific catalog file

---

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `gopkg.in/yaml.v3` | YAML parsing |

Minimal by design — only two direct dependencies.

---

## Do NOT

- Edit files in `dist/` manually — they are generated by `make build`
- Add providers or models to `dist/` — edit `providers/` YAML instead
- Use `*float64` for pricing — use `NullFloat64` (preserves `.0` in JSON)
- Create extends chains deeper than 1 level
- Override `mode` in an extends wrapper
- Add paid infrastructure dependencies — everything must run on free tiers
