# Contributing

This project is library-first: site-specific behavior lives under `site/<name>/`, while `cmd/msd/` is a thin CLI wrapper around the shared engine.

## Development

Run the default checks before sending changes:

```bash
make test
make lint
```

For a smaller loop:

```bash
go test ./site/<name> ./engine
```

Live integration tests use the `integration` build tag:

```bash
make smoke-one SITE=<name>
```

Do not make unit tests depend on the network. Use `httptest` and fixtures for normal tests.

## Adding a Site Handler

1. Create `site/<name>/<name>.go`.
2. Implement `site.Site`.
3. Register the handler with `func init() { site.Register(&YourSite{}) }`.
4. Add a blank import in `cmd/msd/main.go`.
5. Add unit tests using `httptest`.
6. Add an integration test behind `//go:build integration` when a stable public sample exists.
7. Document site-specific credentials, limits, or flags in `README.md`.

The handler should return typed errors from `site/site.go` whenever possible:

| Error | Use when |
|---|---|
| `site.ErrNotFound` | URL, album, folder, or file does not exist. |
| `site.ErrAuthRequired` | Password, token, premium account, or other credential is required. |
| `site.ErrRateLimited` | The remote site returns rate-limit or anti-abuse throttling. |
| `site.ErrSiteChanged` | Parsing failed because the site structure changed. |

## Handler Guidelines

- Prefer structured APIs over scraping when available.
- Keep expiring CDN URLs out of `Album.Files`; resolve them in `DownloadRequest`.
- Store per-resolve download links on the handler when the site requires it.
- Set realistic `DefaultConcurrency`, `DefaultResolveDelay`, and `DefaultDownloadDelay` values.
- Preserve original filenames when possible; let the engine sanitize filesystem names.
- Keep auth tokens out of logs and tests.
- Put network-dependent tests behind the `integration` build tag.

## CLI Guidelines

- The CLI should translate typed site errors into actionable messages.
- Keep site-specific flags rare. Prefer config/env vars for credentials.
- `--dry-run` must resolve metadata only and never create downloads.
- Debug logging goes to stderr and should not interfere with stdout usage.

## Cutting a release

Releases are tagged with `vMAJOR.MINOR.PATCH`. Pushing the tag triggers `.github/workflows/release.yml`, which calls the org's reusable `go-release.yml` to run the tests, build the binaries and `.deb`/`.rpm` packages, and then **pauses for manual approval** in the `manual-smoke-gate` environment before publishing.

### Steps

```bash
git tag -a v0.5.0 -m "v0.5.0"
git push origin v0.5.0
```

Then go to the **Actions → Release** run on GitHub, open the pending `release` job, click *Review deployments*, tick `manual-smoke-gate`, and approve. Once `release` finishes, two jobs run automatically: `aur` publishes to the AUR (one retry on transient SSH failure), and `docker` builds and pushes the image to `ghcr.io`. Both are gated behind the same approval — `docker` depends on `release` via `needs`, and `aur` runs from the same job.

### What the release produces

| Artifact | Platforms |
|---|---|
| `.tar.gz` binaries | linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 |
| `.zip` binary | windows/amd64 |
| `.deb` package | linux/amd64, linux/arm64 |
| `.rpm` package | linux/amd64, linux/arm64 |
| AUR `msd` package | auto-published after release |
| Docker image (`ghcr.io/anastylosis/msd`) | linux/amd64, linux/arm64 |

The `.deb`/`.rpm` packages are built by [nfpm](https://nfpm.goreleaser.com/) from `nfpm.yaml`. The AUR package is built from `packaging/aur/PKGBUILD`, with `pkgver` stamped from the tag before publishing.

The release does not currently open a linked GitHub Discussion — no `discussion-category` is configured on the reusable workflow call.

### Approver checklist

Before ticking `manual-smoke-gate` and approving:

- [ ] `go test -tags=integration ./site/...` (equivalently `make smoke`) passed locally against **every** handler with a live-site suite — currently bunkr, filester, gofile, instagram, kemono, pixeldrain, turbo. CI cannot run these: they fetch live sites and are excluded from CI (`ci.yml` only vets the `integration` build tag, it does not run it).
- [ ] Any handler whose site changed markup this cycle was re-checked by hand.
- [ ] Release notes accurately describe the user-visible changes.

The gate is a **trust-me** check — nothing verifies that you actually ran the suite. Its only job is to force a pause-and-think before a release goes public.
