# 04 — `extends` Inheritance Pattern

## The problem this solves

In the current `ai-gateway` catalog, the same Anthropic Claude Sonnet 4.5 appears as **at least four** entries:

- `anthropic/claude-sonnet-4-5`
- `bedrock/anthropic.claude-sonnet-4-5-v1:0`
- `vertex_ai/claude-sonnet-4-5`
- `vertex_ai-anthropic_models/claude-sonnet-4-5`

Each is hand-edited. Each can drift independently. Each price update touches multiple files. Multiply by Llama, Mistral, DeepSeek across Bedrock + Vertex + OpenRouter + Together + Fireworks + Groq → ~600 of the current 2,531 rows are wrapper duplicates.

This is the same wall models.dev hit and solved with `extends`.

## How it works

Every wrapper file declares its base and overrides only what differs:

```yaml
# providers/bedrock/models/anthropic.claude-sonnet-4-5-v1_0.yaml
extends: anthropic/claude-sonnet-4-5       # ← canonical base model
provider: bedrock                          # required, overrides base
id: anthropic.claude-sonnet-4-5-v1:0       # required, Bedrock-specific ARN

# Override only fields that differ from the base. Everything else is inherited.
pricing:
  cache_write: 3.75                        # Bedrock charges differently for cache writes
regions:                                   # full list, replaces base
  - us-east-1
  - us-west-2
  - eu-central-1
sources:
  pricing:
    url: https://aws.amazon.com/bedrock/pricing/
    verified_at: 2026-04-25
```

## Resolution semantics

When `tools/ferrocat build` materializes a wrapper into the final JSON:

1. Load the base model (`anthropic/claude-sonnet-4-5.yaml`)
2. Load the wrapper file
3. **Deep-merge** wrapper *over* base, with these rules:
   - Scalar fields → wrapper wins
   - Maps → recursively merged
   - Arrays → wrapper **replaces** base (not concatenated) — this is intentional, prevents accidental region leakage
   - `extends` → stripped from final output
4. Re-validate the merged result against the model schema
5. Emit under the wrapper's catalog key (`bedrock/anthropic.claude-sonnet-4-5-v1:0`)

## Rules and constraints

| Rule | Why |
|---|---|
| Max chain depth = 1 (no `extends` of an `extends`) | Keeps merge logic O(1), avoids debugging multi-hop inheritance |
| Base must exist in the same repo | No remote references — guarantees reproducible builds |
| Base must not itself be a wrapper | Same reason as max-depth |
| `provider` and `id` are *always* required in the wrapper, even when same as base | Catalog keys must be explicit, not inferred |
| `mode` cannot be overridden | A chat model can't become an embedding model via extends |
| `aliases` are inherited but extendable | Wrapper can add provider-specific aliases |

## Special cases

### Custom-priced wrappers (DeepInfra, Together hosting Llama)

```yaml
# providers/deepinfra/models/meta-llama__Llama-3.3-70B-Instruct.yaml
extends: meta_llama/Llama-3.3-70B-Instruct
provider: deepinfra
id: meta-llama/Llama-3.3-70B-Instruct
display_name: "Llama 3.3 70B Instruct (via DeepInfra)"

pricing:
  input: 0.23                              # DeepInfra's hosting price
  output: 0.40
  # cache_read: not supported on DeepInfra → null
  cache_read: null
  cache_write: null

capabilities:
  prompt_caching: false                    # explicitly turned off (base says true)

sources:
  pricing:
    url: https://deepinfra.com/pricing
    verified_at: 2026-04-25
```

### Cross-provider wrappers (OpenRouter, Vercel AI Gateway, LLMGateway)

These extend whatever the *origin* provider is, not the actual hosted endpoint:

```yaml
# providers/openrouter/models/anthropic__claude-sonnet-4-5.yaml
extends: anthropic/claude-sonnet-4-5
provider: openrouter
id: anthropic/claude-sonnet-4-5

# OpenRouter takes 5.5% margin on most providers — captured here, not invented at runtime
pricing:
  input: 3.165                             # 3.00 * 1.055
  output: 15.825                           # 15.00 * 1.055

sources:
  pricing:
    url: https://openrouter.ai/api/v1/models
    verified_at: 2026-04-25
    verified_by: scraper-openrouter-oracle
```

### When NOT to use `extends`

Don't use it when the model is genuinely different:
- Bedrock's `amazon.nova-pro-v1:0` is **Amazon's own model**, not a wrapped third-party. Lives under `providers/bedrock/models/amazon.nova-pro.yaml` with no `extends`.
- A fine-tuned variant (`anthropic/claude-haiku-4-5-customer-finetune-abc123`) — different weights, different model.

Rule of thumb: **`extends` if the underlying weights are identical to the base**. Different weights = standalone file.

## Migration impact

Estimated reductions on the existing 2,531-row catalog:

| Provider | Current rows | After `extends` collapse | Reduction |
|---|---|---|---|
| `bedrock` | ~180 | ~40 base + ~140 wrappers | Source files: −60% |
| `vertex_ai` + `vertex_ai-*_models` | ~220 | ~80 base + ~140 wrappers | Source files: −55% |
| `openrouter` | ~300 | 0 base + 300 wrappers | Source files: 0% (all wrappers) |
| `together`, `fireworks`, `groq`, `deepinfra`, `novita` | ~250 | 0 base + 250 wrappers | Source files: 0% |
| `bedrock` junk keys (`bedrock/1024-x-1024/...`) | ~80 | 0 (deleted, not models) | Source files: −100% |

**Net effect**: ~1,200 source YAML files producing the same ~2,400 generated catalog entries (after junk removal). Roughly half the editing surface, with **single-source price updates** propagating automatically to wrappers.

## Build artifact correctness

The generated `dist/catalog.json` continues to have one row per catalog key (no inheritance metadata leaks through). External consumers (existing gateways, third-party tools) see the same shape they always have.

The validator runs after merge to ensure no wrapper produces an invalid model — e.g., a wrapper that overrides `pricing.input` to `null` while the mode is `chat` will fail CI with a clear error pointing at the wrapper file.

## Diffing behavior

`ferrocat diff` shows the *materialized* JSON diff (what consumers will see), not the YAML diff. This means a contributor editing a base model file sees the impact on every wrapper inheriting from it — surfacing accidentally large blast radii before merge.
