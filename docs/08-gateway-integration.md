# 08 — Gateway Integration

How `github.com/ferro-labs/ai-gateway` consumes this repo's published artifacts. This is the "consumer side" of the architecture in [01-architecture.md](./01-architecture.md).

## What changes in `ai-gateway/models/`

Today: a single `catalog.go` that fetches one URL with a 1-second timeout and silently falls back to embedded JSON. After integration:

```
ai-gateway/models/
├── catalog.go              # public API surface (mostly unchanged)
├── load.go                 # NEW — multi-source loader (CDN → GH releases → embedded)
├── verify.go               # NEW — cosign signature verification
├── manifest.go             # NEW — fetch + parse manifest.json
├── slice.go                # NEW — per-provider lazy loading
├── reload.go               # NEW — atomic.Value swap, ticker, admin endpoint hook
├── catalog_backup.json     # only used as last-resort embedded fallback
└── manifest_pin.json       # NEW — committed pin of last-known-good manifest sha
```

The public `Load()`, `Get()`, and `Catalog` types **stay the same** so the rest of the gateway compiles unchanged.

## Lifecycle inside the gateway

```
                     ┌──────────────────────┐
   gateway start ───▶│   models.Load()      │
                     └──────────┬───────────┘
                                │
                  ┌─────────────┼─────────────┐
                  ▼             ▼             ▼
              CDN fetch    GH Release    embedded
              (manifest)   (fallback)    (last resort)
                  │             │             │
                  └─────────────┼─────────────┘
                                ▼
                        verify signature
                                │
                                ▼
                  determine which providers
                  this gateway needs (config)
                                │
                                ▼
                  fetch only those slices
                                │
                                ▼
                       atomic.Value swap
                                │
                                ▼
                   ┌────────────────────────┐
                   │  every 1h: reload tick │◀─── webhook poke (optional)
                   └────────────────────────┘
```

## Replacement for `defaultCatalogURL`

The current code in `ai-gateway/models/catalog.go`:

```go
const defaultCatalogURL = "https://raw.githubusercontent.com/ferro-labs/ai-gateway/main/models/catalog.json"
```

becomes:

```go
const (
    defaultManifestURL    = "https://catalog.ferrolabs.ai/v1/manifest.json"
    defaultBaseURL        = "https://catalog.ferrolabs.ai/v1"
    fallbackReleaseBase   = "https://github.com/ferro-labs/model-catalog/releases/latest/download"

    CatalogManifestURLEnv = "FERRO_MODEL_CATALOG_MANIFEST_URL"  // override
    CatalogBaseURLEnv     = "FERRO_MODEL_CATALOG_BASE_URL"
    CatalogPinEnv         = "FERRO_MODEL_CATALOG_PIN"           // exact sha
)
```

The old `FERRO_MODEL_CATALOG_URL` env var continues to work for one major version (it points at a single full-catalog JSON, which we still publish for backwards compatibility) and emits a deprecation warning in logs.

## Signature verification

```go
// ai-gateway/models/verify.go
package models

import (
    _ "embed"
    "github.com/sigstore/cosign/v2/pkg/cosign"
)

//go:embed cosign-trust-policy.yaml
var trustPolicy []byte

// VerifyManifest checks that manifest bytes were signed by the expected
// GitHub Actions workflow on the model-catalog repo.
func VerifyManifest(manifest, signature []byte) error {
    return cosign.VerifyBlob(manifest, signature, cosign.VerifyBlobOptions{
        CertIdentityRegexp: `^https://github\.com/ferro-labs/model-catalog/`,
        OIDCIssuer:         "https://token.actions.githubusercontent.com",
    })
}
```

Verification fails closed: the gateway never swaps in an unsigned or wrong-signed catalog. Failure paths:

| Result | Action |
|---|---|
| Signature valid | Swap, emit `catalog.updated` event |
| Signature invalid | Keep current catalog, emit `catalog.signature_invalid` event with severity=error, alert via gateway's existing observability pipeline |
| No signature available (offline) | Use last-known-good pin if it matches; refuse otherwise |

## Per-provider lazy loading

The gateway already knows what providers an operator configured (from `config.yaml`'s `targets:` list). Use that to slice the catalog at fetch time:

```go
// load.go
func loadFromCDN(ctx context.Context, cfg Config) (Catalog, error) {
    manifest, err := fetchManifest(ctx, cfg.ManifestURL)
    if err != nil {
        return nil, err
    }
    if err := VerifyManifest(manifest.Raw, manifest.Signature); err != nil {
        return nil, err
    }

    // Determine providers we need: configured + their extends bases
    needed := resolveProviderSet(cfg.ConfiguredProviders, manifest)

    out := make(Catalog)
    for _, p := range manifest.Providers {
        if !needed[p.ID] {
            continue
        }
        slice, err := fetchProviderSlice(ctx, p.URL, p.SHA256)
        if err != nil {
            return nil, fmt.Errorf("fetch %s: %w", p.ID, err)
        }
        for k, v := range slice {
            out[k] = v
        }
    }
    return out, nil
}
```

For typical 3–5-provider gateway deployments this drops in-memory catalog size from ~30 MB to ~1–2 MB and **shaves seconds off cold starts**.

## Hot-reload — admin endpoint

Add to `ai-gateway/internal/admin/handlers.go`:

```go
// POST /admin/catalog/reload
//
// Forces an immediate fetch of the latest manifest and verifies + swaps if newer.
// Returns 200 with the new sha, or 304 if already up-to-date.
func (h *Admin) ReloadCatalog(w http.ResponseWriter, r *http.Request) {
    result, err := h.catalogManager.ReloadNow(r.Context())
    if err != nil {
        respondError(w, http.StatusInternalServerError, err)
        return
    }
    respondJSON(w, http.StatusOK, result)
}
```

This is exactly LiteLLM's `/reload/model_cost_map` pattern. Behind the same admin auth as the rest of the admin API. FerroCloud uses this from its control plane fan-out; OSS operators can poke it manually.

## Background ticker

```go
// reload.go
func (m *Manager) Run(ctx context.Context) {
    t := time.NewTicker(m.cfg.ReloadInterval) // default 1h
    defer t.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-t.C:
            if _, err := m.ReloadNow(ctx); err != nil {
                m.logger.Warn("catalog reload failed", "err", err)
            }
        }
    }
}
```

Atomic swap inside `ReloadNow`:

```go
func (m *Manager) swap(c Catalog) {
    m.current.Store(&c)
    m.events.Emit("catalog.updated", c.ManifestSHA())
}

// Get always reads from current pointer (lock-free, fast path)
func (m *Manager) Get(key string) (Model, bool) {
    c := *m.current.Load().(*Catalog)
    return c.Get(key)
}
```

## Pin file (build-time guarantee)

Each gateway release embeds `manifest_pin.json` — the exact manifest SHA that the gateway was tested against:

```json
{
  "pinned_sha": "a3f9c1b...",
  "pinned_at": "2026-04-28T09:13:42Z",
  "compatibility": ">= v2026.01.01"
}
```

At runtime, the gateway will refuse to roll **backward** of its pin (defends against an attacker forcing it to use ancient pricing). Forward updates are allowed as long as schema version is compatible.

To opt out (e.g., gateway is far behind and needs to catch up): set `FERRO_MODEL_CATALOG_PIN=accept-any` env var. Logged as a warning.

## Backwards compatibility for existing operators

We must not break any current `ai-gateway` installation when this lands. The migration plan:

1. **Phase 1**: ship the new loader behind a feature flag (`FERRO_USE_REMOTE_CATALOG=true`). Default off. Old single-URL fetch remains the default.
2. **Phase 2**: flip the default to on. The new loader still understands the old single-JSON URL via the `FERRO_MODEL_CATALOG_URL` env var (backwards-compatible mode).
3. **Phase 3** (1 release later): emit deprecation warning when old env var is used.
4. **Phase 4** (next major): remove old code path.

This is documented in [09-migration-plan.md](./09-migration-plan.md).

## Observability hooks

New events the gateway emits, consumed by FerroCloud's observability layer:

| Event | When |
|---|---|
| `catalog.loaded` | Gateway successfully loaded a catalog (CDN or fallback) |
| `catalog.updated` | Hot-reload swapped to a newer SHA |
| `catalog.signature_invalid` | Refused to swap due to bad signature |
| `catalog.fallback_to_release` | CDN unreachable, used GitHub Release |
| `catalog.fallback_to_embedded` | Both CDN and Release unreachable |
| `catalog.schema_version_mismatch` | Loaded a catalog with newer schema than gateway expects |

Each event includes `manifest_sha`, `provider_count`, `model_count`, and load duration. These feed the SLO dashboard ("day-0 model launches reflected in production within 24 hours" — see [00-vision.md](./00-vision.md)).

## Testing

In `ai-gateway/models/`, alongside existing `gateway_test.go`:

- `load_test.go` — exercises CDN fetch, GH-release fallback, embedded fallback paths with `httptest.Server`
- `verify_test.go` — known-good and tampered manifests, ensure tamper is rejected
- `slice_test.go` — provider set resolution including `extends` dependency graph
- `reload_test.go` — atomic swap correctness under concurrent `Get()`s, race detector clean

Plus a smoke test in CI that pulls the **real** production manifest weekly to make sure the gateway↔catalog contract hasn't drifted.

## Operator-visible UX

Three things change for an operator:

1. **Faster startup** — small per-provider fetches instead of one 11 MB blob
2. **Day-0 launches work without redeploy** — new models appear within 1h (or instantly via webhook in FerroCloud)
3. **Catalog source is auditable** — every model now has `sources.pricing.url + verified_at` in the response of `/admin/catalog/inspect/<key>`

No config migration required for the default case. Operators with custom `FERRO_MODEL_CATALOG_URL` settings keep working until they choose to migrate.
