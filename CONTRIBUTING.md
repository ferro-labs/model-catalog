# Contributing to Ferro Model Catalog

Thank you for helping keep the LLM model catalog accurate! This guide covers the most common contributions.

## Quick Start: Add or Update a Model (5 minutes)

### 1. Fork and clone

```bash
git clone https://github.com/<your-username>/model-catalog.git
cd model-catalog
```

### 2. Find or create the provider directory

Models live in `providers/<provider>/models/`. Provider IDs are lowercase snake_case: `openai`, `anthropic`, `vertex_ai`, `groq`, etc.

### 3. Create or edit the model YAML

```yaml
# providers/openai/models/gpt-4o.yaml
provider: openai
model_id: gpt-4o
display_name: GPT-4o
mode: chat
context_window: 128000
max_output_tokens: 16384
pricing:
    input_per_m_tokens: 2.5
    output_per_m_tokens: 10.0
    cache_read_per_m_tokens: 1.25
    cache_write_per_m_tokens: null
    reasoning_per_m_tokens: null
    image_per_tile: null
    audio_input_per_minute: null
    audio_output_per_character: null
    embedding_per_m_tokens: null
    finetune_train_per_m_tokens: null
    finetune_input_per_m_tokens: null
    finetune_output_per_m_tokens: null
capabilities:
    vision: true
    audio_input: false
    audio_output: false
    function_calling: true
    parallel_tool_calls: true
    json_mode: true
    response_schema: true
    prompt_caching: true
    reasoning: false
    streaming: true
    finetuneable: false
lifecycle:
    status: ga
    deprecation_date: null
    sunset_date: null
    successor: null
source: https://openai.com/api/pricing
updated_at: "2026-04-30"
tier: flagship
```

### 4. Validate locally (optional)

```bash
go run ./cmd/ferrocat validate
go run ./cmd/ferrocat build --output /tmp/dist
```

### 5. Open a PR

CI will run `validate`, `lint`, and a dry-run `build` automatically.

## Field Reference

| Field | Required | Values |
|-------|----------|--------|
| `provider` | Yes | Must match folder name |
| `model_id` | Yes | Provider's canonical model ID |
| `display_name` | Yes | Human-readable name |
| `mode` | Yes | `chat`, `embedding`, `image`, `audio_in`, `audio_out` |
| `pricing.*` | Yes | USD per 1M tokens. `null` = not applicable. `0` = free. |
| `capabilities.*` | Yes | Boolean flags |
| `lifecycle.status` | Yes | `preview`, `ga`, `deprecated`, `sunset`, `legacy` |
| `tier` | Yes | `flagship` or `standard` |
| `source` | Recommended | URL where you verified the data |
| `updated_at` | Recommended | `YYYY-MM-DD` |

## Wrapper Models (extends)

If a provider hosts another provider's model (e.g., Vertex AI hosting Gemini), use `extends`:

```yaml
# providers/vertex_ai/models/gemini-2.0-flash.yaml
extends: gemini/gemini-2.0-flash
provider: vertex_ai
model_id: gemini-2.0-flash
display_name: gemini-2.0-flash
pricing:
    # Only include fields that differ from the base
    input_per_m_tokens: 0.1
    output_per_m_tokens: 0.4
    # ... all 12 pricing fields required
capabilities:
    # All capability fields required (wrapper overrides all)
    vision: true
    # ...
tier: standard
```

## What NOT to Contribute

- Model quality rankings or benchmarks
- Latency measurements
- Model weights or tokenizers
- Negotiated/private pricing
- Pre-release models under NDA

## Code of Conduct

Be respectful. See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

## License

By contributing, you agree that your contributions will be licensed under the Apache-2.0 license.
