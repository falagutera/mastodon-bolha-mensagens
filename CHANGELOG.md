# Changelog

All notable changes to justmytoots. Versions follow [SemVer](https://semver.org/); images are
published as `registry.gutocarvalho.net/images/justmytoots:<version>-<build>`.

## 0.1.9 — 2026-11-07

Hardening from the updated linters.

- The image runs as the numeric distroless "nonroot" user (`USER 65532:65532`), so scanners and runtimes see a non-root user.

## 0.1.8 — 2026-11-06

- Release notes page, linked from the top of every page, and the same data at `/release-notes.json`.

## 0.1.7 — 2026-11-05

- Custom emoji are replaced only in the text of a post, never inside links or markup.

## 0.1.6 — 2026-10-31

First tagged version on justmytoots.bolha.us.

- A clean public page with an account's own posts, for accounts that opt in from their profile.
- Pinned-posts filter.
- Bounded cache.
- Version in the footer.
