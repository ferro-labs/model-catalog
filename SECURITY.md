# Security Policy

## Scope

This repository contains model metadata (pricing, capabilities, lifecycle) as YAML and JSON files. It does not contain executable code that runs in production environments, API keys, or user data.

The primary security concern is **data integrity** — ensuring that published catalog artifacts accurately reflect the source YAML files and have not been tampered with.

## Reporting a Vulnerability

If you discover a security issue (e.g., a way to inject malicious data through the build pipeline, bypass CI validation, or poison published artifacts), please report it responsibly:

1. **Do not open a public issue.**
2. Email **security@ferrolabs.ai** with a description of the issue.
3. We will acknowledge receipt within 48 hours and provide a timeline for a fix.

## What We Consider Security Issues

- Bypass of CI validation that would allow invalid data into `dist/`
- Injection of malicious content through YAML parsing
- Tampering with published GitHub Release artifacts
- Exposure of secrets in CI logs or artifacts

## What We Do NOT Consider Security Issues

- Incorrect pricing data (this is a data quality issue — open a regular issue or PR)
- Missing models or providers (open a regular issue)
- Stale data (the scrapers handle this)

## Integrity Measures

- Every PR runs `ferrocat validate` + `ferrocat lint` + full test suite
- Published `dist/manifest.json` includes SHA-256 hashes for all artifacts
- Only the `build.yml` workflow on `main` can publish releases
- `CODEOWNERS` requires maintainer review for all changes
