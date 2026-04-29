# 05 — Scraper Strategy

## Goal

Replace manual catalog edits as the primary source of truth with **automated weekly scrapers** that open PRs with detected diffs. Each scraper ships with a snapshot, a source URL, and a confidence rating. Humans review and merge — they don't author from scratch.

The existing `scripts/catalog-check` in `ai-gateway` only proves URLs are alive. **It does not verify pricing accuracy.** This is the gap we close.

## Five-tier source hierarchy

We never trust a single source. Each model is reconciled across up to five tiers:

| Tier | Source type | Example providers | Reliability | Coverage |
|---|---|---|---|---|
| 1 | **Live `/v1/models` API** | OpenAI, Anthropic, OpenRouter, Together, Fireworks, Groq, Mistral, Cohere, DeepSeek, Cerebras, SambaNova, HuggingFace, Replicate, Hyperbolic, xAI | High (existence/deprecation) | ~40% of providers |
| 2 | **Provider pricing API / data file** | AWS Bedrock (Pricing API), Azure (`prices.azure.com`), Vertex AI (Cloud Billing Catalog), Vercel AI Gateway, Replicate | High (machine-readable) | ~10% of providers |
| 3 | **HTML scrape (server-rendered)** | Mistral, Cohere, Groq, DeepSeek, AWS docs | Medium | ~25% of providers |
| 4 | **Markdown docs in provider repos** | `openai/openai-openapi`, `anthropics/anthropic-sdk-python`, `google/generative-ai-docs`, `groq/groq-python` | High (versioned) | ~15% of providers |
| 5 | **Headless browser (SPA)** | OpenAI pricing page, Anthropic pricing page, Gemini pricing page | Low (DOM brittle) | ~10% of providers |

Plus two **oracle sources** that cross-check everything:

- **OpenRouter `/api/v1/models`** — 300+ models with structured pricing in one HTTP call
- **models.dev `api.json`** — community catalog, updated by SST + contributors

## Scraper architecture

```
                         ┌───────────────────────────┐
                         │  ferrocat scrape --all    │
                         └─────────────┬─────────────┘
                                       │ fan-out
        ┌──────────────┬───────────────┼──────────────┬──────────────┐
        ▼              ▼               ▼              ▼              ▼
   Tier 1: API   Tier 2: PriceAPI  Tier 3: HTML  Tier 4: Docs   Tier 5: Browser
        │              │               │              │              │
        └──────────────┴───────┬───────┴──────────────┴──────────────┘
                               │
                               ▼
                  ┌────────────────────────┐
                  │   Normalize → ModelObs │  (one observation per source)
                  └────────────┬───────────┘
                               │
                               ▼
                  ┌────────────────────────┐
                  │  Reconcile by model_id │  (group across sources)
                  └────────────┬───────────┘
                               │
                               ▼
                  ┌────────────────────────┐
                  │  Cross-check + score   │
                  └────────────┬───────────┘
                               │
              ┌────────────────┼────────────────┐
              ▼                ▼                ▼
        agree (≥2)      single source     disagree
              │                │                │
              ▼                ▼                ▼
        auto-PR         auto-PR           open issue
        "high"          "needs-review"    "scraper-conflict"
```

## Scraper interface

Single Go interface, multiple backends. Lives in `tools/ferrocat/scrape/interface.go`:

```go
package scrape

import (
    "context"
    "time"
)

// Scraper produces zero or more model observations from a single source.
type Scraper interface {
    Name() string                                          // e.g. "openai-html-pricing"
    Provider() string                                      // e.g. "openai"
    Tier() Tier                                            // 1-5 from doc above
    Scrape(ctx context.Context) ([]Observation, error)
}

type Observation struct {
    Source      string                                     // scraper name
    SourceURL   string                                     // page URL or API endpoint
    SourceSHA   string                                     // SHA of fetched content
    ScrapedAt   time.Time
    Confidence  Confidence                                 // High | Medium | Low
    Provider    string
    ModelID     string
    Fields      map[string]any                             // partial; only what this source knows
    Snapshot    []byte                                     // raw HTML/JSON, stored on snapshots branch
}
```

A reconciler combines observations and emits a `ModelDelta` per model — this is what the auto-PR contains.

## Backend abstraction

For HTML/SPA pages, the actual fetch happens through a pluggable backend:

```go
type Browser interface {
    Render(ctx context.Context, url string, opts RenderOpts) (Rendered, error)
}

// Implementations:
//   • httpBackend     — plain http.Get + goquery (free, server-rendered pages)
//   • chromedpBackend — headless Chromium in CI (free, GitHub Actions, default for SPAs)
//   • cloudflareBackend — Cloudflare Browser Rendering /markdown (paid, optional)
```

Selection priority:
1. `--backend=auto` (default): try `http` → fall back to `chromedp` if page is SPA
2. `--backend=chromedp`: force browser
3. `--backend=cloudflare`: only if `CF_API_TOKEN` env var is set; **never required**

Contributors and CI run with **zero paid services** by default. Cloudflare backend is opt-in.

## Cross-check & confidence scoring

For every (provider, model_id) pair, after reconciliation:

```go
func score(observations []Observation, field string) Confidence {
    values := map[any]int{}
    for _, o := range observations {
        if v, ok := o.Fields[field]; ok {
            values[v]++
        }
    }

    switch {
    case len(values) == 0:
        return Unknown
    case len(values) == 1 && len(observations) >= 2:
        return High        // multiple sources agree
    case len(values) == 1:
        return Medium      // one source only
    default:
        return Conflict    // sources disagree → no auto-PR
    }
}
```

Tier weights are **also** applied — Tier 1+2 sources have higher weight than Tier 3+5. A Tier-1 + Tier-4 agreement on price beats a Tier-3 disagreement.

The reconciler emits a `confidence` field per scraped value, which is written into the YAML's `sources.<field>.confidence`.

## Snapshot & audit trail

Every scrape run writes:

1. **Raw snapshot** to a separate `snapshots` orphan branch:
   ```
   snapshots/openai/2026-04-28T060000Z/pricing.html.gz
   snapshots/openai/2026-04-28T060000Z/pricing.png      # screenshot via chromedp
   snapshots/anthropic/2026-04-28T060000Z/pricing.json
   ```
2. **SHA-256** of the price-bearing element written into the model YAML
3. **Diff PR body** includes:
   - Before/after table for every changed field
   - Direct link to the snapshot (PNG screenshot of the actual provider page)
   - Source URL and timestamp
   - Per-field confidence

This makes pricing accuracy auditable forever — anyone can answer "what did anthropic.com say on April 28?" by checking out the `snapshots` branch.

## Failure handling

Reuses the streak pattern from `scripts/catalog-check/main.go`:

```go
type FailureHistory struct {
    ConsecutiveFailures map[string]int  // scraper-name → count
    LastSuccess         map[string]time.Time
}
```

| State | Action |
|---|---|
| Scraper succeeded | reset count to 0 |
| Scraper failed once | warning in run log, no PR/issue |
| Scraper failed 3× consecutively | open `scraper-broken` issue in repo, ping CODEOWNER |
| Scraper failed 6× consecutively | mark scraper as `disabled` in `tools/ferrocat/scrape/registry.yaml`; manual re-enable required |

Disabled scrapers do not block the weekly run; the run continues with the remaining scrapers and emits a warning summary.

## Anti-abuse rules

We are well-behaved scraper citizens:

- **`robots.txt` respected** — every scraper checks before fetching
- **Real `User-Agent`** — `ferro-catalog-scraper/1.0 (+https://github.com/ferro-labs/model-catalog)`
- **Per-domain rate limit** — max 1 request per 2 seconds across the run
- **Weekly cadence** — `0 6 * * 1` (Mon 06:00 UTC), not hourly
- **Cache + ETag** — refuse to re-fetch if `If-None-Match` says no change
- **No login walls** — if a provider requires login to see pricing, we don't scrape; we add a Tier-1 API path or wait for them to publish a JSON

API-key-using scrapers (OpenAI `/v1/models` etc.) only ever **list** — never call completions, never spend money. The CI runner's keys can be limited tokens.

## Phased scraper rollout

Reflected in [10-roadmap.md](./10-roadmap.md), but high-level:

**Phase A (Week 1)** — Oracle scrapers, immediate cross-check value
- `openrouter` (Tier 1, no auth needed for `/api/v1/models`)
- `models_dev` (Tier 1, `models.dev/api.json`)

**Phase B (Week 2–3)** — High-leverage Tier 1 APIs
- `openai`, `anthropic`, `groq`, `mistral`, `cohere`, `together`, `fireworks`, `deepseek`, `cerebras`, `xai`

**Phase C (Week 4)** — Tier 2 cloud pricing APIs
- `aws_bedrock` (AWS Pricing API), `azure` (`prices.azure.com`), `vertex_ai` (Cloud Billing Catalog)

**Phase D (Week 5–6)** — Tier 3 HTML scrapers
- `mistral`, `cohere`, `groq`, `deepseek` (server-rendered pricing pages)

**Phase E (Week 7+)** — Tier 5 headless for the stubborn last mile
- `openai` pricing page, `anthropic` pricing page, `gemini` pricing page

## What "good output" looks like

A successful weekly scrape run produces something like:

```
Scrape run 2026-04-28T06:00:00Z
─────────────────────────────────────────────────────────────
Scrapers run:        14 / 14
Success:             13
Failures:            1 (openai-html: selector miss, streak 1/3)

Models observed:     1284
  • new:             3   (anthropic/claude-sonnet-4-6, …)
  • removed:         2   (openai/gpt-4-vision-preview deprecated)
  • price changed:   17
  • capability change: 4

PRs opened:          2
  • #142 [auto-verified] anthropic price refresh (high confidence, 17 fields)
  • #143 [needs-review]  3 new models detected from openrouter oracle

Issues opened:       1
  • #144 scraper-conflict: vertex_ai claude pricing disagrees between
        Tier 2 (Cloud Billing) and Tier 5 (Vertex pricing page)

Snapshots committed: 23 to branch `snapshots`
─────────────────────────────────────────────────────────────
```
