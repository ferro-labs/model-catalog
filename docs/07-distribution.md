# 07 — Distribution & Versioning

## What gateways actually fetch

Three URL shapes, all on the CDN, all immutable except `manifest.json` and `latest`:

```
https://catalog.ferrolabs.ai/v1/manifest.json                 # ← pointer (small, frequent)
https://catalog.ferrolabs.ai/v1/manifest.json.sig
https://catalog.ferrolabs.ai/v1/<sha>.json                    # ← immutable, content-hashed
https://catalog.ferrolabs.ai/v1/<sha>.json.sig
https://catalog.ferrolabs.ai/v1/latest.json                   # ← rolling, points at newest sha
https://catalog.ferrolabs.ai/v1/providers/<id>.json           # ← per-provider slice (latest)
https://catalog.ferrolabs.ai/v1/providers/<id>/<sha>.json     # ← per-provider, immutable
https://catalog.ferrolabs.ai/v1/diff/<from>/<to>.json         # ← optional, hot-reload patch
```

## Manifest contract

`manifest.json` is the only file gateways need to fetch frequently. It's small (~5 KB), aggressively cacheable, and tells the gateway which `<sha>` to fetch.

Schema in [03-schema.md](./03-schema.md#manifest-schema-distmanifestjson).

| Field | Why |
|---|---|
| `version` | Human-readable (`v2026.04.28`) |
| `git_sha` | Pin from this repo's commit |
| `catalog_sha256` | Integrity of the full catalog |
| `providers[*].sha256` | Integrity of per-provider slice |
| `providers[*].url` | Exact URL to fetch this slice |
| `signature.certificate` | cosign keyless cert |
| `signature.transparency_log` | Rekor entry |

Gateways verify: `cosign verify-blob --certificate-identity-regexp '...github.com/ferro-labs/model-catalog...' manifest.json`.

## CDN topology

**Primary**: Cloudflare Pages (or Render's static site service — `render.yaml` already exists in the gateway repo, infrastructure familiar).

**Failover**: GitHub Releases as immutable backup. Every tag publishes the full `dist/` as release assets at `github.com/ferro-labs/model-catalog/releases/download/v2026.04.28/catalog.json`. Gateways can fall back here if the CDN is unreachable.

**Mirror** (optional, future): jsDelivr automatically mirrors GitHub Releases — `cdn.jsdelivr.net/gh/ferro-labs/model-catalog@v2026.04.28/dist/catalog.json` works for free with no setup.

This gives three layers of availability without adding paid infrastructure.

## Cache headers

| URL pattern | Cache-Control | Reason |
|---|---|---|
| `manifest.json`, `latest.json` | `public, max-age=300, stale-while-revalidate=3600` | Cheap-to-fetch pointer; tolerable to be 5 min stale |
| `<sha>.json`, `providers/<id>/<sha>.json` | `public, max-age=31536000, immutable` | Content-addressed; never changes |
| `*.sig`, `*.crt` | Same as the file they sign | Always paired with content |

## Version semantics

CalVer plus daily counter: `v2026.04.28` → `v2026.04.28.1` if a second release ships the same day.

| Change type | Version bump | Backwards compatibility |
|---|---|---|
| New model added | Daily release | Always backwards-compatible |
| Price update | Daily release | Backwards-compatible |
| New schema field added | Daily release | Backwards-compatible (additive) |
| Schema field renamed/removed | Major version (`v2/`) | **Breaking** — old `v1/` URLs continue serving for 6 months minimum |
| Bug-fix re-publish of same date | `.N` suffix | Same content shape |

Old major versions remain on the CDN; gateways pin a major version in their manifest URL (`/v1/manifest.json`), so an upgrade is opt-in.

## Signature verification

Every published manifest is signed with **cosign keyless** — OIDC-bound to the GitHub Actions workflow on this repo's `main` branch. No long-lived keys to rotate, no secrets to leak.

```bash
cosign verify-blob \
  --certificate-identity-regexp '^https://github\.com/ferro-labs/model-catalog/' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --signature manifest.json.sig \
  manifest.json
```

Gateway-side verification logic (Go) lives in `ai-gateway/models/verify.go` (added during gateway integration phase). See [08-gateway-integration.md](./08-gateway-integration.md).

## Hot-reload protocol

Day-0 model launches reflect without restart. Two options gateways implement:

### Option 1: Pull-based (default)

Gateway has a background ticker (default 1 hour). Each tick:
1. `GET /v1/manifest.json` (small, cached)
2. If `git_sha` changed:
   - `GET /v1/<sha>.json` (or per-provider slices)
   - Verify signature
   - Atomic swap of in-memory catalog (single `atomic.Value`)
3. Emit a `catalog.updated` event for observability

### Option 2: Push-based (FerroCloud only)

Build pipeline POSTs to a control-plane webhook on release. The control plane fans out to tenant gateways via existing event bus. Gateways still verify the signature locally — webhook is just a "check now" hint, not a trust source.

### Diff endpoint (optional optimization)

For very large catalogs, gateways can request a patch:

```
GET /v1/diff/<from-sha>/<to-sha>.json
```

Returns JSON-Patch (RFC 6902) for in-place updates. Falls back to full fetch if the diff is missing (e.g., gateway is too far behind).

## Per-provider lazy loading

Most gateway deployments use 3–5 providers, not all 30. The current behavior of loading the entire 111K-line catalog into memory wastes ~95% of it.

New runtime behavior:
1. At startup, gateway reads its own `config.yaml` to determine active providers
2. Fetches only `providers/<id>.json` for those providers (plus their `extends` bases — manifest declares the dependency graph)
3. Memory drops from ~30 MB to ~1–2 MB for typical configs

This is invisible to users — `Catalog.Get(key)` works the same way.

## Failure modes

| Failure | Behavior |
|---|---|
| CDN unreachable | Try GitHub Releases; if both fail, use embedded fallback in gateway binary |
| Manifest fetch returns 5xx | Use cached version up to `stale-while-revalidate` window |
| Signature verification fails | **Refuse to swap.** Log loudly. Keep current catalog. Open an alert. |
| Schema version newer than gateway supports | Log warning; load fields the gateway understands; ignore unknown fields |
| Schema version older than gateway requires | Log warning; gateway still starts (it has the embedded fallback) |
| Per-provider slice fetch fails after manifest succeeded | Skip that provider's update, retry next tick, alert after 3 misses |

## Privacy considerations

Public CDN, public catalog, public traffic. Nothing sensitive should ever land here. Specifically:

- **No tenant IDs**, no FerroCloud customer references
- **No internal model names** (pre-release / under NDA)
- **No private endpoint URLs** (use the public docs URL only)

The build pipeline lints for these — a regex-based scrubber rejects PRs containing `ferrocloud-`, `tenant-`, or any string matching internal naming patterns.

## Bandwidth & cost projection

Even pessimistically:

| Scenario | Daily bandwidth |
|---|---|
| 10,000 active gateway instances, 1 manifest fetch / hour, 5 KB each | 1.2 GB |
| 1% of fetches discover an update and pull a 1 MB slice | 1.2 GB |
| Per-provider slices average 50 KB; 5 providers per gateway | Negligible |

**Total**: ~3 GB/day, well within Cloudflare Pages free tier (unlimited bandwidth on the free plan as of writing). GitHub Releases bandwidth is also free for public repos.

This is why **distribution costs zero dollars** at the relevant scale.
