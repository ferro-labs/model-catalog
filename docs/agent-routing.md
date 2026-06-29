# Agent-routing metadata

Optional, catalog-owned metadata that helps coding agents and local routing
clients choose models without hard-coding private tier tables. Introduced in
[#23](https://github.com/ferro-labs/model-catalog/issues/23).

All fields are **optional**. Models without any `agent_routing`, `aliases`, or
`benchmarks` block validate cleanly and emit unchanged JSON artifacts. The
warnings only fire on **non-empty malformed** enum values — missing metadata
is always valid.

## Schema

### `agent_routing`

Per-model recommendation signals.

| Field                    | Type       | Enum                                                                 |
|--------------------------|------------|----------------------------------------------------------------------|
| `coding_quality_tier`    | string     | `frontier` `strong` `balanced` `fast` `experimental` `unknown`      |
| `reasoning_quality_tier`  | string     | same as `coding_quality_tier`                                        |
| `tool_use_quality_tier`   | string     | `strong` `balanced` `weak` `unknown`                               |
| `latency_tier`            | string     | `low` `medium` `high` `unknown`                                    |
| `local_suitability`       | string     | `excellent` `good` `poor` `unknown`                                 |
| `recommended_roles`       | string[]   | free-form; suggested: `planning`, `code-review`, `implementation`, `search-summary`, `synthesis` |

### `aliases`

Map of routing surface → resolved model IDs. Provider-agnostic; the catalog
records the mapping, not which gateway instance owns which alias.

```yaml
aliases:
  ferro:
    - qwen3.5:397b-cloud
    - deepseek-v4-flash:cloud
```

### `benchmarks`

Local or 3rd-party benchmark artifacts so routing decisions can be catalog-owned.

```yaml
benchmarks:
  coding:
    source: swe-bench        # swe-bench | local | other
    score: 0.42
    updated_at: "2026-06-27"
  local_runtime:
    quantization: Q4_K_M
    backend: llama.cpp
    tokens_per_second: 71.2
    hardware: "Apple M-series"
```

## Extends inheritance

- `agent_routing`: per-field deep merge — a wrapper inherits the base block and
  overrides individual tiers without restating the block.
- `aliases`, `benchmarks`: full replacement — the wrapper must restate the full
  block to override (mirrors `Pricing` / `Capabilities` semantics, since YAML
  can't distinguish "unset" from "explicitly empty" for map/struct fields).

## JSON artifact

Resolved entries expose the new top-level keys, omitted when unset
(`omitempty`):

```json
{
  "provider": "openai",
  "model_id": "gpt-5-pro",
  "agent_routing": {
    "coding_quality_tier": "frontier",
    "recommended_roles": ["planning", "implementation"]
  },
  "aliases": { "ferro": ["openai/gpt-5-pro"] },
  "benchmarks": {
    "coding": { "source": "swe-bench", "score": 0.42 }
  }
}
```

A model with no routing metadata omits all three keys entirely.

## Coding-agent routing example (TypeScript)

```ts
type AgentRouting = {
  coding_quality_tier?: "frontier" | "strong" | "balanced" | "fast" | "experimental" | "unknown";
  reasoning_quality_tier?: string;
  tool_use_quality_tier?: "strong" | "balanced" | "weak" | "unknown";
  latency_tier?: "low" | "medium" | "high" | "unknown";
  local_suitability?: "excellent" | "good" | "poor" | "unknown";
  recommended_roles?: string[];
};

// Pick a strong, low-latency coding model from the catalog artifact.
function pickCodingModel(catalog: Record<string, { agent_routing?: AgentRouting; lifecycle?: { status?: string } }>): string | null {
  const tierRank: Record<string, number> = { frontier: 0, strong: 1, balanced: 2, fast: 3, experimental: 4, unknown: 5 };
  let best: { key: string; rank: number } | null = null;
  for (const [key, entry] of Object.entries(catalog)) {
    if (entry.lifecycle?.status === "deprecated" || entry.lifecycle?.status === "sunset") continue;
    const tier = entry.agent_routing?.coding_quality_tier;
    if (!tier) continue;
    const rank = tierRank[tier] ?? 99;
    if (!best || rank < best.rank) best = { key, rank };
  }
  return best?.key ?? null;
}
```

## Validation

`make validate` rejects malformed enums:

```
benchmarks.coding.source: invalid value "hearsay"; must be one of: swe-bench, local, other
agent_routing.latency_tier: invalid value "instant"; must be one of: low, medium, high, unknown
```

Missing or empty values never trigger a validation error.

## Non-goals

- Pi-specific routing policy does **not** live in the catalog — the catalog
  only records reusable metadata, not the policy that consumes it.
- Private/local benchmark data is **not** mandatory for public models.
