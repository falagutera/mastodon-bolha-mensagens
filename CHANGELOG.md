# Changelog

All notable changes to justmytoots. Versions follow [SemVer](https://semver.org/); images are
published as `registry.gutocarvalho.net/images/justmytoots:<version>-<build>`.

## 0.3.0 — 2026-11-13

- **New address and name: mensagens.bolha.us, "Mensagens"** (same in PT and EN). The old address justmytoots.bolha.us redirects there.
- **`JUSTMYTOOTS_LEGACY_HOSTS`**: old hosts, comma separated, whose profile links still count. Pages made with a link to justmytoots.bolha.us keep working; the "your link" box suggests the new address.

## 0.2.0 — 2026-11-08

- **The bolha.io look**, from `open/gcn/ui` (`assets/style.css`, a synced copy, and `VISUAL.md`), like numeros, atualiza, id and encaminha: light and dark from the system, system fonts, a header with the bubble, cards on the home page, the filters and buttons as pills, and the footer "version · source code · part of bolha.io · based on JustMyToots". The IBM Plex webfonts and the artefatos stylesheet are gone.

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
