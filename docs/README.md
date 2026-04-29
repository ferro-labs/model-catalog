# Ferro Model Catalog — Plan & Design Docs

This folder is the **single source of truth** for why this repo exists, how it's structured, how data flows in and out, and how it integrates with the FerroGateway ecosystem.

Read in order if you're new:

| # | Doc | What it covers |
|---|---|---|
| 00 | [Vision & OSS Strategy](./00-vision.md) | Why a separate public repo, what's the moat |
| 01 | [Architecture Overview](./01-architecture.md) | Three-layer system: source-of-truth → automation → distribution |
| 02 | [Repo Structure](./02-repo-structure.md) | Directory layout, file conventions |
| 03 | [Schema Design](./03-schema.md) | Provider + model YAML schema with full field reference |
| 04 | [`extends` Inheritance Pattern](./04-extends-pattern.md) | How Bedrock / Vertex / OpenRouter wrappers reuse base models |
| 05 | [Scraper Strategy](./05-scraping.md) | Five-tier scraping with confidence scoring (zero paid services) |
| 06 | [Automation Pipelines](./06-automation.md) | GitHub Actions: validate, build, scrape, prune, release |
| 07 | [Distribution & Versioning](./07-distribution.md) | CDN, signing, manifests, hot-reload protocol |
| 08 | [Gateway Integration](./08-gateway-integration.md) | How `ai-gateway` consumes the catalog (runtime upgrades) |
| 09 | [Migration Plan](./09-migration-plan.md) | Phased rollout from current 111K-line `catalog.json` |
| 10 | [Roadmap & Milestones](./10-roadmap.md) | Week-by-week execution schedule |
| 11 | [Competitor Analysis](./11-competitor-analysis.md) | LiteLLM, Portkey, models.dev, OpenRouter — what they do, what we improve |
| 12 | [Glossary](./12-glossary.md) | Shared vocabulary |

## TL;DR

We are extracting the model catalog from `github.com/ferro-labs/ai-gateway` into this dedicated repo because:

1. **The current 111K-line `catalog.json` is unmaintainable** — diff hell, 2,531 entries with ~30% duplicates and junk keys, no per-field provenance.
2. **OSS leaders all use a separate catalog repo** — Portkey, models.dev, and LiteLLM (effectively, via its dedicated JSON file) prove the pattern.
3. **A public catalog is a flywheel asset, not a moat** — every contributor PR improves accuracy for free, and the gateway runtime + FerroCloud multi-tenancy remain the actual product.
4. **Automation is what fixes consistency** — we need scrapers + cross-validation, not more manual PRs. That belongs in a repo that can release independently from the gateway.

The end-state: per-provider YAML files → schema-validated PRs → automated weekly scrapers → signed JSON artifacts on a CDN → gateway hot-reloads with zero downtime on day-0 model launches.

## Status

This folder holds the plan. No code yet — implementation begins after the plan is approved. See [Roadmap](./10-roadmap.md) for the execution order.
