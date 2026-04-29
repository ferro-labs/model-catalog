# 03 — Schema Design

## Design principles

1. **Superset of LiteLLM, Portkey, OpenRouter, models.dev fields** — never schema-migrate again when a new modality lands
2. **Per-field provenance** — every price/capability carries `source.url + verified_at`
3. **Money is explicit** — all prices USD, per 1M tokens unless explicitly suffixed (`per_image`, `per_minute`, `per_call`)
4. **`null` ≠ `0`** — null means "not applicable to this mode," 0 means "genuinely free"
5. **Nothing inferred at runtime** — capabilities are declarative, gateway never has to guess

## Provider schema (`provider.yaml`)

```yaml
# providers/openai/provider.yaml
id: openai                                 # required, must match folder name
display_name: OpenAI                       # required, human-readable
website: https://openai.com
docs_url: https://platform.openai.com/docs
status: active                             # active | beta | deprecated

# How the gateway authenticates
auth:
  style: bearer                            # bearer | aws_sigv4 | gcp_oauth2 | basic | api_key_header | custom
  env_var: OPENAI_API_KEY                  # primary env var name (matches gateway code)
  alt_env_vars: []                         # fallback env vars
  header: Authorization                    # only if non-default

# How the gateway reaches the provider
endpoints:
  base_url: https://api.openai.com/v1
  chat_completions: /chat/completions
  completions: /completions
  embeddings: /embeddings
  models: /models                          # null if provider has no live models endpoint
  images: /images/generations
  audio_speech: /audio/speech
  audio_transcriptions: /audio/transcriptions

# Wire format the provider expects
api_compat:
  family: openai                           # openai | anthropic | bedrock | vertex | cohere | ollama | custom
  version: "2024-02-01"                    # provider API version, if applicable
  notes: |
    Supports streaming, function calling, and structured outputs natively.

# Where this provider serves from (informational)
regions:
  - global
  - us-east-1
  - eu-west-1

# Routing hints — used by the gateway's strategy layer
features:
  supports_streaming: true
  supports_batch: true                     # provider has a Batch API
  supports_fine_tuning: true
  supports_files: true
  supports_assistants: true

# Operational notes for gateway operators
notes: |
  Default rate limits are tier-based — see https://platform.openai.com/docs/guides/rate-limits.
  All models are routed through the global endpoint unless `region` is specified per-request.

# Per-field provenance
sources:
  endpoints:
    url: https://platform.openai.com/docs/api-reference
    verified_at: 2026-04-25
  auth:
    url: https://platform.openai.com/docs/api-reference/authentication
    verified_at: 2026-04-25
```

## Model schema (`model.yaml`) — full reference

```yaml
# providers/openai/models/gpt-5.yaml

# ─── Identity ──────────────────────────────────────────────────────────────
id: gpt-5                                  # required, matches provider's canonical ID
provider: openai                           # required, must match folder
display_name: GPT-5                        # required
aliases:                                   # optional, dated/legacy IDs that map here
  - gpt-5-2025-08-07
  - gpt-5-latest
mode: chat                                 # required: chat | embedding | image | audio_in | audio_out | rerank | moderation | search

# ─── Lifecycle ─────────────────────────────────────────────────────────────
status: ga                                 # required: preview | ga | deprecated | sunset
release_date: 2025-08-07                   # required, YYYY-MM-DD
knowledge_cutoff: 2024-09                  # optional, YYYY-MM
deprecation_date: null                     # date provider announced deprecation
sunset_date: null                          # date provider stops serving
successor: null                            # e.g. "openai/gpt-5.1"
last_updated: 2026-04-25                   # auto-bumped by build

# ─── Context window ────────────────────────────────────────────────────────
context:
  input_tokens: 400000                     # max prompt tokens
  output_tokens: 128000                    # max completion tokens
  reasoning_tokens: 272000                 # for reasoning models (o-series, GPT-5 thinking)
  total_tokens: null                       # if provider gives a single combined number

# ─── Capabilities (declarative, gateway never guesses) ─────────────────────
capabilities:
  vision: true
  audio_input: false
  audio_output: false
  function_calling: true
  parallel_tool_calls: true
  json_mode: true
  response_schema: true                    # JSON Schema-constrained outputs
  prompt_caching: true
  reasoning: true                          # exposes reasoning tokens
  reasoning_effort: true                   # supports reasoning_effort param
  streaming: true
  batch: true                              # supported via provider Batch API
  fine_tunable: false
  web_search: true
  computer_use: false
  pdf_input: true
  open_weights: false                      # are weights publicly downloadable

# ─── Pricing ───────────────────────────────────────────────────────────────
# All prices USD per 1,000,000 tokens unless suffix says otherwise.
# null = not applicable. 0 = genuinely free.
pricing:
  input: 1.25
  output: 10.00
  cache_read: 0.13
  cache_write: 1.25
  reasoning: 10.00                         # output tokens consumed by reasoning
  embedding: null                          # only on embedding models

  # Image / audio (only on relevant modes)
  image_per_call: null                     # flat price per image generation
  image_input: null                        # per-image input token surcharge
  audio_input_per_minute: null
  audio_output_per_character: null

  # Web search / tools (OpenAI, Anthropic)
  web_search_per_call: 0.025

  # Batch API (typically 50% PAYG)
  batch_input: 0.625
  batch_output: 5.00

  # Fine-tuning (only when fine_tunable: true)
  finetune_train: null
  finetune_input: null
  finetune_output: null

  # Tiered pricing (Vertex AI PayGo, Bedrock service tiers)
  tiers:
    - name: priority
      trigger: ON_DEMAND_PRIORITY          # response field that activates this tier
      input: 2.50
      output: 20.00
    - name: flex
      trigger: FLEX
      input: 0.625
      output: 5.00

  # Above-context pricing (Gemini 2M, Claude 1M)
  above_threshold:
    threshold_tokens: 200000
    input: 2.50
    output: 20.00
    cache_read: 0.25
    cache_write: 2.50

# ─── Regional availability ─────────────────────────────────────────────────
regions:
  - global

# ─── Provenance (THE missing field in current catalog) ─────────────────────
sources:
  pricing:
    url: https://openai.com/api/pricing
    verified_at: 2026-04-25
    verified_by: scraper-openai-v3         # `manual:<github-username>` for human edits
    snapshot_sha256: 3f2a9c0b1e...         # SHA of the price-bearing element snapshot
    snapshot_branch: snapshots             # which branch the raw HTML lives on
    confidence: high                       # high | medium | low
  capabilities:
    url: https://platform.openai.com/docs/models/gpt-5
    verified_at: 2026-04-25
    verified_by: manual:mitulshah1
  context:
    url: https://platform.openai.com/docs/models/gpt-5
    verified_at: 2026-04-25
  lifecycle:
    url: https://platform.openai.com/docs/deprecations
    verified_at: 2026-04-25
```

## Override schema (`override.yaml`)

Used for dated patches without losing history.

```yaml
# overrides/2026-04-15-anthropic-price-cut.yaml
effective_date: 2026-04-15                 # required
description: |                             # required, shows in PR + release notes
  Anthropic announced 20% price cut on all Claude Sonnet 4.5 models.
  Source: https://www.anthropic.com/news/pricing-update-april-2026
applies_to:                                # required
  - anthropic/claude-sonnet-4-5
  - bedrock/anthropic.claude-sonnet-4-5    # extends-resolved bedrock entry
patch:                                     # JSON Pointer paths, applied as deep merge
  pricing:
    input: 2.40                            # was 3.00
    output: 12.00                          # was 15.00
    cache_read: 0.24                       # was 0.30
sources:
  pricing:
    url: https://www.anthropic.com/news/pricing-update-april-2026
    verified_at: 2026-04-15
    verified_by: manual:mitulshah1
```

## Manifest schema (`dist/manifest.json`)

The pointer file gateways fetch most often. Tiny, cache-friendly.

```json
{
  "version": "v2026.04.28",
  "schema_version": 1,
  "generated_at": "2026-04-28T09:13:42Z",
  "git_sha": "a3f9c1b...",
  "catalog_sha256": "b7d0e2f...",
  "providers": [
    {
      "id": "openai",
      "model_count": 47,
      "sha256": "1a2b3c...",
      "url": "https://catalog.ferrolabs.ai/v1/providers/openai.json"
    }
  ],
  "stats": {
    "total_models": 1287,
    "total_providers": 30,
    "deprecated_models": 142
  },
  "signature": {
    "algorithm": "cosign-keyless",
    "certificate": "https://catalog.ferrolabs.ai/v1/v2026.04.28.crt",
    "transparency_log": "rekor.sigstore.dev/api/v1/log/entries/abc123"
  }
}
```

## JSON output shape (backwards-compatible)

The build pipeline materializes YAML into a flat JSON map matching the **current `ai-gateway` Catalog type** so existing gateway code works unchanged on day 1:

```json
{
  "openai/gpt-5": {
    "provider": "openai",
    "model_id": "gpt-5",
    "display_name": "GPT-5",
    "mode": "chat",
    "context_window": 400000,
    "max_output_tokens": 128000,
    "pricing": { },
    "capabilities": { },
    "lifecycle": { },
    "source": "https://openai.com/api/pricing",
    "updated_at": "2026-04-25"
  }
}
```

After the gateway is upgraded to consume the new richer schema, we add fields without breaking old consumers (additive only, no field removals for at least one major version).

## Field validation rules (enforced by `validate.yml`)

- **`id`**: must match `^[a-z0-9][a-z0-9._:/-]*$`
- **`provider`**: must equal containing folder name
- **Key shape**: catalog key (`provider/model_id`) must match `^[a-z][a-z0-9_]*\/[a-zA-Z0-9._:/@-]+$` and **no segment** may match a known parameter name (`steps`, `width`, `height`, `quality`)
- **Mode**: enum
- **Status**: enum, with `deprecated` requiring either `successor` or a justification comment
- **Pricing**: all numbers ≥ 0; if `mode == chat`, both `input` and `output` must be non-null
- **`verified_at`**: must be ≤ today; on update, must be ≥ previous value
- **`extends`**: target must exist in repo and not itself extend (no chains > depth 1)
- **`deprecation_date < sunset_date`** if both set
- **No duplicate aliases** within a provider
- **Currency**: USD only (multi-currency deferred — providers publish USD anyway)

## Schema versioning

The schema itself is versioned in `schema/manifest.schema.json`:

```json
{ "schema_version": 1, "...": "..." }
```

- Bumping `schema_version` is a coordinated change across schema files + `tools/ferrocat`
- Build always emits both old and new schemas during a deprecation window (≥1 release)
- Gateways detect schema version and warn loudly if they're behind
