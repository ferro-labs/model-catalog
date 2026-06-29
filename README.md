<div align="center">
  <table border="0" cellspacing="0" cellpadding="0"><tr>
    <td rowspan="2"><img src="https://raw.githubusercontent.com/ferro-labs/ai-gateway/refs/heads/main/docs/logo.png" alt="Ferro Labs AI" width="64" /></td>
    <td align="center"><h1>Ferro Labs AI - Model Catalog</h1></td>
  </tr><tr>
    <td align="center"><strong>Open-Source LLM Pricing & Capability Database</strong></td>
  </tr></table>
  <p>
    <a href="https://github.com/ferro-labs/model-catalog/releases/latest"><img src="https://img.shields.io/github/v/release/ferro-labs/model-catalog?label=catalog&color=blue" alt="Latest Release" /></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue.svg" alt="License" /></a>
    <a href="https://github.com/ferro-labs/model-catalog/actions/workflows/validate.yml"><img src="https://github.com/ferro-labs/model-catalog/actions/workflows/validate.yml/badge.svg" alt="Validate" /></a>
    <a href="https://github.com/ferro-labs/model-catalog/actions/workflows/build.yml"><img src="https://github.com/ferro-labs/model-catalog/actions/workflows/build.yml/badge.svg" alt="Build" /></a>
  </p>
  <p>
    <strong>2,479 models</strong> &middot; <strong>82 providers</strong> &middot; <strong>Updated weekly</strong> &middot; <strong>Zero paid infrastructure</strong>
  </p>
</div>

---

Every AI application needs the same data: **what models exist**, **what they cost**, and **what they can do**. This repo is that data — structured, validated, and open.

```bash
curl -sL https://github.com/ferro-labs/model-catalog/releases/latest/download/catalog.json | \
  python3 -c "import json,sys; m=json.load(sys.stdin)['openai/gpt-4o']; print(f'GPT-4o: \${m[\"pricing\"][\"input_per_m_tokens\"]}/M input, \${m[\"pricing\"][\"output_per_m_tokens\"]}/M output, {m[\"context_window\"]:,} ctx')"
```

```
GPT-4o: $2.5/M input, $10.0/M output, 128,000 ctx
```

---

## Who is this for?

| If you're building... | You can use the catalog to... |
|---|---|
| An AI gateway or proxy | Route requests by model capability, calculate costs per request |
| A cost tracker or billing system | Look up per-token pricing for any model across 82 providers |
| A coding agent (like Aider, OpenCode, Cursor) | Know which models support function calling, vision, streaming |
| An LLM comparison tool | Compare pricing and context windows across providers |
| A model selection UI | Display model metadata with accurate, up-to-date pricing |

---

## What's in it?

One YAML file per model. One JSON artifact per provider. Everything cross-checked weekly.

```yaml
# providers/openai/models/gpt-4o.yaml
provider: openai
model_id: gpt-4o
display_name: GPT-4o
mode: chat
context_window: 128000
max_output_tokens: 16384
pricing:
    input_per_m_tokens: 2.5         # USD per 1M tokens
    output_per_m_tokens: 10.0
    cache_read_per_m_tokens: 1.25
    cache_write_per_m_tokens: null   # null = not applicable
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
    status: ga                      # preview | ga | deprecated | sunset
source: https://openai.com/api/pricing
updated_at: "2026-04-30"
tier: flagship                      # flagship | standard
```

### Providers

OpenAI, Anthropic, Google Gemini, AWS Bedrock, Azure, Vertex AI, Groq, Mistral, Cohere, Together AI, Fireworks, DeepInfra, DeepSeek, xAI (Grok), Meta Llama, Replicate, Perplexity, NVIDIA NIM, Hugging Face, Cerebras, SambaNova, and 60+ more.

<details>
<summary><strong>All 82 providers with model counts</strong></summary>

| Provider | Models | | Provider | Models |
|----------|-------:|-|----------|-------:|
| bedrock | 343 | | groq | 14 |
| fireworks | 273 | | gradient_ai | 13 |
| openai | 156 | | oci | 13 |
| vertex_ai | 123 | | ai21 | 12 |
| azure | 115 | | aiml | 12 |
| vercel_ai_gateway | 101 | | anyscale | 12 |
| novita | 83 | | fal_ai | 12 |
| gemini | 78 | | minimax | 9 |
| openrouter | 78 | | publicai | 9 |
| deepinfra | 67 | | zai | 9 |
| mistral | 43 | | volcengine | 8 |
| together | 42 | | cerebras | 7 |
| perplexity | 42 | | deepseek | 6 |
| replicate | 40 | | aleph_alpha | 6 |
| deepgram | 36 | | gigachat | 6 |
| xai | 32 | | runwayml | 6 |
| github_copilot | 31 | | qwen | 5 |
| ollama | 29 | | azure_openai | 5 |
| watsonx | 29 | | cloudflare | 4 |
| databricks | 28 | | amazon_nova | 4 |
| anthropic | 25 | | meta_llama | 4 |
| snowflake | 24 | | ollama_cloud | 4 |
| dashscope | 23 | | azure_foundry | 3 |
| stability | 23 | | hugging_face | 3 |
| moonshot | 21 | | nvidia_nim | 3 |
| lambda_ai | 20 | | featherless_ai | 2 |
| cohere | 17 | | morph | 2 |
| gmi | 17 | | recraft | 2 |
| hyperbolic | 16 | | sarvam | 1 |
| nscale | 16 | | *+ vertex sub-providers* | 27 |
| sambanova | 16 | | | |

</details>

---

## Quick Start

### Fetch the catalog

```bash
# Full catalog (~3 MB)
curl -sLO https://github.com/ferro-labs/model-catalog/releases/latest/download/catalog.json

# Just one provider (~50 KB each)
curl -sLO https://github.com/ferro-labs/model-catalog/releases/latest/download/providers/openai.json

# CDN mirror for the latest published dist/
curl -sLO https://catalog.ferrolabs.ai/catalog.json
curl -sLO https://catalog.ferrolabs.ai/providers/openai.json
```

### Use it in Python

```python
import json

with open("catalog.json") as f:
    catalog = json.load(f)

# Look up any model
model = catalog["anthropic/claude-sonnet-4-5"]
print(f"Input:   ${model['pricing']['input_per_m_tokens']}/M tokens")
print(f"Output:  ${model['pricing']['output_per_m_tokens']}/M tokens")
print(f"Context: {model['context_window']:,} tokens")
print(f"Vision:  {model['capabilities']['vision']}")

# Find all models with function calling under $1/M input
cheap_tool_models = {
    k: v for k, v in catalog.items()
    if v["capabilities"]["function_calling"]
    and v["pricing"]["input_per_m_tokens"] is not None
    and v["pricing"]["input_per_m_tokens"] < 1.0
}
print(f"\n{len(cheap_tool_models)} models with tool use under $1/M input")
```

### Use it in Go

```go
import "github.com/ferro-labs/model-catalog/catalog"

data, _ := os.ReadFile("catalog.json")
entries, _ := catalog.ReadCatalogJSON(data)

model := entries["openai/gpt-4o"]
fmt.Printf("Input: $%.2f/M tokens\n", model.Pricing.InputPerMTokens.Value)
```

### Use it in JavaScript/TypeScript

```javascript
const catalog = await fetch(
  "https://github.com/ferro-labs/model-catalog/releases/latest/download/catalog.json"
).then(r => r.json());

const model = catalog["openai/gpt-4o"];
console.log(`Input: $${model.pricing.input_per_m_tokens}/M tokens`);
```

---

## How it stays accurate

### Automated cross-checking

Every week, scrapers fetch pricing data from independent oracle sources and live provider model APIs, then compare against the catalog.

Oracle scrapers:

| Source | Models | What it provides |
|--------|--------|-----------------|
| [OpenRouter API](https://openrouter.ai/api/v1/models) | 368 | Real-time pricing (includes their margin — we adjust) |
| [models.dev](https://models.dev) | 4,362 | Community-curated pricing and capabilities |

When both sources agree on a price that differs from ours, it's flagged as **high confidence** and auto-PRd. When only one source reports a diff, it's marked **needs review**.

Freshness checks also query provider model-list APIs when CI secrets are configured: Anthropic, OpenAI, Groq, Mistral, Together, Fireworks, DeepSeek, Cohere, xAI, and Cerebras.

### Community contributions

Found a wrong price? A missing model? A new provider? Open a PR — it's one YAML file:

1. Fork the repo
2. Add or edit a file in `providers/<provider>/models/`
3. Open a PR — CI validates automatically

See [CONTRIBUTING.md](CONTRIBUTING.md) for the 5-minute walkthrough, or use the issue templates:
- [Report a wrong price](https://github.com/ferro-labs/model-catalog/issues/new?template=price_correction.md)
- [Request a new model](https://github.com/ferro-labs/model-catalog/issues/new?template=new_model.md)
- [Request a new provider](https://github.com/ferro-labs/model-catalog/issues/new?template=new_provider.md)

---

## Key features

### Per-provider slices

Most apps use 3-5 providers, not all 82. Download only what you need:

```bash
# Just OpenAI + Anthropic (~100 KB total instead of 3 MB)
curl -sLO https://github.com/ferro-labs/model-catalog/releases/latest/download/providers/openai.json
curl -sLO https://github.com/ferro-labs/model-catalog/releases/latest/download/providers/anthropic.json
```

### Extends inheritance

When Vertex AI hosts Gemini or Azure hosts OpenAI, the wrapper model inherits from the base and overrides only what differs. A single price update to GPT-4o propagates to azure/gpt-4o, azure_openai/gpt-4o, and github_copilot/gpt-4o automatically.

193 wrapper models currently use this pattern.

### Agent-routing metadata

Optional, catalog-owned metadata that lets coding agents and local routing clients pick a model without hard-coding private tier tables. All fields are optional; models without the block keep validating unchanged.

```yaml
# providers/openai/models/gpt-5-pro.yaml
agent_routing:
  coding_quality_tier: frontier    # frontier|strong|balanced|fast|experimental|unknown
  reasoning_quality_tier: frontier # same enum
  tool_use_quality_tier: strong     # strong|balanced|weak|unknown
  latency_tier: low                 # low|medium|high|unknown
  local_suitability: excellent      # excellent|good|poor|unknown
  recommended_roles: [planning, code-review, implementation, search-summary, synthesis]
aliases:
  ferro:                            # routing surface → resolved model ids
    - qwen3.5:397b-cloud
    - deepseek-v4-flash:cloud
benchmarks:
  coding:
    source: swe-bench               # swe-bench|local|other
    score: 0.42
    updated_at: "2026-06-27"
  local_runtime:
    quantization: Q4_K_M
    backend: llama.cpp
    tokens_per_second: 71.2
    hardware: "Apple M-series"
```

Wrapper models inherit the base's `agent_routing` block and can override individual tiers without restating the whole block; `aliases` and `benchmarks` follow full-replacement semantics (the wrapper must restate the full block to override). Validation rejects malformed enum values but allows missing metadata. See [`docs/agent-routing.md`](docs/agent-routing.md) for the full schema and a coding-agent routing example.

### Manifest with integrity

Every release includes a `manifest.json` with SHA-256 hashes for the full catalog and each provider slice. Verify what you downloaded matches what was published.

### CalVer releases

Tagged `v2026.04.30` — you always know when the data was published. Pin a version or follow `latest`.

---

## CLI tool

The `ferrocat` CLI manages the catalog locally:

```bash
ferrocat build                              # YAML → JSON (catalog + slices + manifest)
ferrocat validate                           # Check structural correctness
ferrocat lint                               # Detect junk keys and duplicates
ferrocat scrape                             # Cross-check pricing against external sources
ferrocat freshness                          # Check live provider model APIs for missing catalog entries
ferrocat prune --days 90                    # Remove models sunset more than 90 days ago
ferrocat split catalog.json                 # Legacy JSON → per-model YAML (one-time migration)
ferrocat migrate-extends --wrapper azure --base openai   # Convert wrappers to extends
```

### Build locally

```bash
git clone https://github.com/ferro-labs/model-catalog
cd model-catalog
make build        # Generate dist/ from YAML source files
make test         # Run all tests including round-trip regression
make validate     # Check structural correctness
```

Requires Go 1.24+.

---

## How it compares

| Feature | This repo | [LiteLLM](https://github.com/BerriAI/litellm) | [models.dev](https://github.com/sst/models.dev) | [Portkey](https://github.com/Portkey-AI/models) |
|---------|-----------|---------|------------|---------|
| Open source | Yes | Yes (one JSON file in main repo) | Yes (separate repo) | Yes (separate repo) |
| Per-model files | Yes (YAML) | No (single 111K-line JSON) | Yes (TOML) | No (per-provider JSON) |
| `extends` inheritance | Yes (193 wrappers) | No | Yes | No |
| Automated cross-check scrapers | Yes (OpenRouter + models.dev) | No | No | No |
| Per-provider slices | Yes (82 files) | No | No | Yes |
| Integrity verification (SHA-256) | Yes | No | No | No |
| Auto-prune deprecated models | Yes | No ([open issue](https://github.com/BerriAI/litellm/issues/21240)) | No | No |
| Raw provider pricing (no margin) | Yes | Yes | Yes | Yes |
| Community PR contribution path | Yes (YAML + CI) | Yes (JSON, conflicts) | Yes (TOML + CI) | Yes (JSON) |

---

## Architecture

For the full technical deep-dive — repo structure, data model, extends resolution, build pipeline, scraper design, CI/CD, and Go package design — see **[docs/architecture.md](docs/architecture.md)**.

---

## Related

- [ferro-labs/ai-gateway](https://github.com/ferro-labs/ai-gateway) — Open-source AI gateway (30 providers, 8 routing strategies, plugin middleware) that consumes this catalog for pricing and capability lookups

---

## License

Apache-2.0 — see [LICENSE](LICENSE).
