# 06 — Automation Pipelines

Five GitHub Actions workflows. Each does one thing well.

## Cost & infrastructure

| Service | Tier | Why |
|---|---|---|
| GitHub Actions | Free for public repo | All CI |
| GitHub Pages or Cloudflare Pages | Free | CDN for `dist/` |
| cosign keyless | Free | Signing (OIDC-bound to repo) |
| Sigstore Rekor | Free | Transparency log |
| Cloudflare Browser Rendering | **NOT used by default** — only as opt-in scraper backend | Avoid paid dependency |

## 1. `validate.yml` — PR gate

```yaml
# .github/workflows/validate.yml
name: Validate
on:
  pull_request:
    paths:
      - 'providers/**'
      - 'overrides/**'
      - 'schema/**'
      - 'tools/**'

permissions:
  contents: read
  pull-requests: write

jobs:
  schema:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
        with: { go-version-file: go.mod }
      - run: go run ./tools/ferrocat validate --strict
      # Schema, key shape, dedupe, monotonic verified_at, extends-target-exists,
      # deprecation/successor consistency.

  build-dryrun:
    runs-on: ubuntu-latest
    needs: schema
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
        with: { go-version-file: go.mod }
      - run: go run ./tools/ferrocat build --dry-run --output /tmp/dist
      - name: Compare against committed dist/
        run: diff -ru dist/ /tmp/dist/ || true   # informational only on PRs
```

**Failure modes that block merge:**
- JSON Schema violation
- Catalog key doesn't match `^[a-z][a-z0-9_]*\/[a-zA-Z0-9._:/@-]+$`
- Key contains a known parameter segment (`steps`, `width`, `height`, `quality`)
- `extends:` target doesn't exist
- `verified_at` decreased on an existing field
- Duplicate model IDs across providers (must use `extends`)
- `chat` mode without `pricing.input` and `pricing.output`

## 2. `diff.yml` — Render PR diff comment

Reviewers see the **materialized JSON impact**, not just the YAML diff.

```yaml
# .github/workflows/diff.yml
name: Diff
on:
  pull_request:
    paths: ['providers/**', 'overrides/**']

permissions:
  contents: read
  pull-requests: write

jobs:
  render-diff:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v6
        with: { go-version-file: go.mod }
      - name: Build base ref
        run: |
          git checkout ${{ github.base_ref }}
          go run ./tools/ferrocat build --output /tmp/base
      - name: Build PR ref
        run: |
          git checkout ${{ github.head_ref }}
          go run ./tools/ferrocat build --output /tmp/head
      - name: Render diff
        run: go run ./tools/ferrocat diff /tmp/base /tmp/head > /tmp/diff.md
      - uses: marocchino/sticky-pull-request-comment@v2
        with:
          path: /tmp/diff.md
          header: catalog-diff
```

Diff format includes:
- Added / removed models (counts + names)
- Per-model changed fields with before → after
- **Wrapper blast radius** — when a base model changes, every `extends`-er is listed
- Provider-level summary

## 3. `build.yml` — Publish on merge

```yaml
# .github/workflows/build.yml
name: Build
on:
  push:
    branches: [main]
    paths: ['providers/**', 'overrides/**', 'schema/**', 'tools/**']
  workflow_dispatch:

permissions:
  id-token: write       # cosign keyless
  contents: write       # commit dist/ back to main
  pages: write          # if using GitHub Pages
  deployments: write

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
        with: { token: ${{ secrets.GITHUB_TOKEN }} }
      - uses: actions/setup-go@v6
      - run: go run ./tools/ferrocat build --output dist/
      - run: go run ./tools/ferrocat manifest --output dist/manifest.json

      - uses: sigstore/cosign-installer@v3
      - name: Sign manifest
        run: cosign sign-blob --yes dist/manifest.json --output-signature dist/manifest.json.sig

      - name: Commit dist
        run: |
          git config user.email "ferrocat-bot@ferrolabs.ai"
          git config user.name "ferrocat-bot"
          git add dist/
          git commit -m "build: regenerate dist [skip ci]" || echo "no changes"
          git push

      - name: Deploy to CDN
        env:
          CF_API_TOKEN: ${{ secrets.CF_API_TOKEN }}
        run: ./scripts/deploy-cdn.sh dist/

      - name: Notify gateway
        run: |
          curl -X POST https://api.ferrolabs.ai/internal/catalog-updated \
               -H "Authorization: Bearer ${{ secrets.GATEWAY_WEBHOOK }}" \
               -d "{\"sha\":\"${{ github.sha }}\"}"
```

The CDN deploy is idempotent and cache-purges versioned URLs.

## 4. `scrape-weekly.yml` — Auto-PR new data

```yaml
# .github/workflows/scrape-weekly.yml
name: Weekly Scrape
on:
  schedule:
    - cron: "0 6 * * 1"        # Mon 06:00 UTC
  workflow_dispatch:

permissions:
  contents: write
  pull-requests: write
  issues: write

jobs:
  scrape:
    runs-on: ubuntu-latest
    timeout-minutes: 30
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6

      - name: Restore scrape history
        uses: actions/cache/restore@v5
        with:
          path: .scrape-history.json
          key: scrape-history-${{ github.run_id }}
          restore-keys: scrape-history-

      - name: Run scrapers
        env:
          OPENAI_API_KEY: ${{ secrets.SCRAPE_OPENAI_KEY }}            # listing-only
          ANTHROPIC_API_KEY: ${{ secrets.SCRAPE_ANTHROPIC_KEY }}      # listing-only
        run: |
          go run ./tools/ferrocat scrape --all \
            --backend=auto \
            --history .scrape-history.json \
            --report scrape-report.md \
            --output-changes /tmp/proposed-changes/

      - name: Save scrape history
        if: always()
        uses: actions/cache/save@v5
        with:
          path: .scrape-history.json
          key: scrape-history-${{ github.run_id }}

      - name: Commit snapshots to orphan branch
        run: ./scripts/push-snapshots.sh

      - name: Open PR per provider with changes
        run: ./scripts/open-scrape-prs.sh /tmp/proposed-changes/

      - name: Open issue for broken scrapers
        if: failure()
        uses: actions/github-script@v8
        with:
          script: |
            // Same dedupe pattern as existing catalog-check.yml in ai-gateway
```

One PR per provider keeps reviews scoped. Each PR is labeled `auto-pr` + confidence label (`high-confidence`, `needs-review`).

## 5. `prune-monthly.yml` — Remove deprecated entries

```yaml
# .github/workflows/prune-monthly.yml
name: Monthly Prune
on:
  schedule:
    - cron: "0 8 1 * *"        # 1st of month, 08:00 UTC
  workflow_dispatch:

permissions:
  contents: write
  pull-requests: write

jobs:
  prune:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
      - name: Prune sunset models
        run: |
          go run ./tools/ferrocat prune \
            --grace-period=90d \
            --output-pr proposed-prune.diff
      - uses: peter-evans/create-pull-request@v6
        with:
          title: "prune: remove sunset models (90d grace)"
          body-path: proposed-prune.diff
          branch: prune/${{ github.run_id }}
```

Removes entries where `sunset_date < today - 90d` AND `successor` is set. If no successor, opens an issue instead — humans must decide.

This solves the open LiteLLM issue #21240 on day one.

## Cross-cutting concerns

### Secrets

| Secret | Used by | Scope |
|---|---|---|
| `SCRAPE_OPENAI_KEY` | scrape-weekly | listing-only key (no completions) |
| `SCRAPE_ANTHROPIC_KEY` | scrape-weekly | listing-only key |
| `CF_API_TOKEN` | build | CDN deploys; scoped to `model-catalog` zone |
| `GATEWAY_WEBHOOK` | build | optional; tells `ai-gateway` infra to refresh CDN cache |
| `CF_BROWSER_TOKEN` | scrape-weekly (optional) | only if Cloudflare Browser Rendering opt-in is enabled |

All secrets configured at repo level; no environment-scoped secrets needed for a single-environment public repo.

### Branch protection

`main` is protected with:
- Require `validate` and `diff` checks to pass
- Require 1 CODEOWNER review for `providers/<id>/**` changes
- Require linear history (no merge commits)
- Auto-merge enabled via Mergify when checks + review pass on `auto-pr`-labeled PRs

### CODEOWNERS

```
# .github/CODEOWNERS
/providers/openai/        @mitulshah1
/providers/anthropic/     @mitulshah1
/providers/bedrock/       @mitulshah1
/providers/vertex_ai/     @mitulshah1
/schema/                  @mitulshah1
/tools/                   @mitulshah1
*                         @mitulshah1
```

Add per-provider community maintainers as the project grows — pattern proven by Portkey-AI/models.

## Observability

Every workflow run emits structured logs and:

- A run summary in the PR comment (or issue, on failure)
- Metrics scraped into a small JSON in the orphan `metrics` branch:
  - Total scrape runs
  - Per-scraper success rate
  - Average PRs opened per week
  - Time from scrape → merge → CDN
  - Schema validation failure rate

Useful for retros and for setting later SLOs (e.g. "day-0 model launches reflected in production within 24 hours" from [00-vision.md](./00-vision.md)).
