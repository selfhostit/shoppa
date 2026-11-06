# Changelog

What changed in each release of Shoppa. Installs update to the newest release; see
[docs/deploy.md](docs/deploy.md#updates-and-versions).

## Unreleased

- Tap a product's photo to see it larger. The server keeps a 400-pixel copy the first
  time someone looks.
- Typed items get an icon and a section from the words in their name: "Yogurt lactose
  free" is a yoghurt, "chicken breast small" a chicken breast. The most specific match
  wins.
- Pick any item's section and icon by hand. The choice is kept by the item's name, for
  the whole household, and wins over the automatic one.
- A light, dark or like-the-phone look, in Me, kept on each phone.
- golang.org/x/net 0.62.0, which fixes GO-2026-6860.
- SECURITY.md: how to report a problem privately, and how quickly fixes ship.
- Security scans on every pull request and every Monday: CodeQL, govulncheck, Grype
  (web and worker dependencies, released images), dependency review and gitleaks.
- The containers' filesystems are read-only apart from `pb_data` and `/tmp`.
- The compose file follows the major version (`:0`) instead of `:latest`, and the
  installer fetches the compose files of the newest release instead of `main`.

## 0.1.0 (2026-11-04)

- Releases: a `vX.Y.Z` tag publishes `:X.Y.Z`, `:X.Y`, `:X` and `:latest`, and installs follow
  `:latest`, so they only update to a release. `main` publishes `:edge`.
- Every published image has a signed provenance attestation (`gh attestation verify`).
- Security and maintenance section in the README.
- A fresh install asks for a one-time setup code on the first-run screen.
- Sign-ins and first-run setup are rate limited; the web app sends a Content Security Policy.
- The containers drop every Linux capability and can't gain new privileges.
- GitHub Actions are pinned to commit hashes, and workflow tokens are read-only by default.
