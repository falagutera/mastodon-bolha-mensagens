# justmytoots

A clean, public page with the posts of an account of a Mastodon instance: **mensagens.bolha.us** (Mensagens; the old address justmytoots.bolha.us redirects there, and profile links to it still count).

It shows a person's own posts, without boosts, timelines or the rest of the interface, so they're easy to read, follow and share. Think jokes, drawings, photos, or anything worth looking back through. Only accounts of the instance have a page. To offer it on your instance, run your own.

It's a rewrite in Go of [JustMyToots](https://github.com/cookiemumbles/justmytoots.com) by cookiemumbles, see `NOTICE`.

## Contents

- [Screenshot](#screenshot)
- [Testing and security](#testing-and-security)
- [Architecture](#architecture)
- [Install](#install)
- [How it works](#how-it-works)
- [Development](#development)

## Screenshot

[![justmytoots rendering an account's posts](docs/screenshot.png)](https://mensagens.bolha.us)

*A live instance at [mensagens.bolha.us](https://mensagens.bolha.us): an account's own posts, no boosts, PT/EN.*

## Testing and security

justmytoots shows posts written by other people on a public page, so it is tested against what would hurt: a page for someone who didn't ask for one, a post that isn't public, and post HTML that could inject markup or script. What is done today (2026-11-08):

**Every pipeline.** GitLab CI ([`.gitlab-ci.yml`](.gitlab-ci.yml), plus the shared security stage from `open/gcn/ci-templates`, file `security.yml`). The test job runs on every pipeline; the security stage runs on every merge request and every push to `main`; the image is built and published only from `main`.

- Blocking (the pipeline fails and no image is published):
  - `go vet`, a `gofmt` check, and the full test suite (no services needed: the tests use a fake instance);
  - `govulncheck`: known vulnerabilities in Go code paths justmytoots actually reaches (in the test job and in the security stage);
  - `gitleaks`: secrets committed to the repository;
  - `trivy`: critical vulnerabilities that have a fix, secrets and misconfigurations in the repository.
- Informative (reported, not blocking): `semgrep`, `gosec`, `hadolint` (Dockerfile), `yamllint`, and `poutine` (security of the CI pipeline itself).

**The test suite.** 19 tests (`go test -list`), running the real HTTP handlers against a fake Mastodon instance. They cover:

- opt-in: no profile link, no page (posts aren't even fetched), and look-alike hosts or names in the link don't count;
- only what is public: posts are read without a token, boosts are left out, the filters send the right query, and an unknown filter or a malformed `max_id` is a 404 that never reaches the instance;
- local accounts only: other domains and odd names are refused;
- untrusted HTML: sanitizing, custom emoji that can't inject markup and are replaced only in text, never inside links;
- sign-in shows only the person's own page;
- the cache: one lookup per minute per page, bounded by size, separate per language;
- both languages, clean error texts, required settings, and the release notes.

**Reviews.** A [Strix](https://github.com/usestrix/strix) review found that a custom emoji shortcode inside a link could break out of the link's markup; fixed in 0.1.7 (emoji are now replaced only in text) with a regression test. In 0.1.9 the findings of the updated linters (`gosec`, `semgrep`, `hadolint`) were fixed. These are what we checked; they are not an outside audit and don't certify anything.

**Reporting a vulnerability.** Write to contato@gutocarvalho.net. Please don't open a public issue for it.

## Architecture

**Why it exists.** A Mastodon profile mixes a person's own posts with boosts, replies and the rest of the interface. Some people want a plain page with just what they wrote (jokes, drawings, photos) to share or look back through. This one is run by the instance itself, serves only its accounts, and only those that ask for it.

**In plain words.**

1. a person adds a link to `justmytoots.example.social/@name` in their bio or a profile field;
2. when someone opens that page, justmytoots looks up the account on the instance's public API and checks that the link is there; without it, there is no page;
3. it reads the account's posts from the public API as an anonymous visitor, sanitizes their HTML again and shows them;
4. signing in is optional and only shows a person whether their page is on and which link to add.

**Components.**

```
          browser
             |
      justmytoots (this app, no JavaScript)
             |                         media and avatars load in the browser
             |                         straight from the instance (https)
             v
      the Mastodon instance (one only)
        - public API, no token: account lookup and posts
        - OAuth sign-in, scope "profile" (token revoked right away)
```

- Nothing is stored: no database and no disk. API answers are cached in memory for a minute (at most 512 pages and 32 MiB); the session is a signed cookie with just the username, valid for 4 hours.
- Because posts are read without a token, followers-only and direct posts never come back.

## Install

You need Docker. There is no database.

1. Register the OAuth application:

   ```sh
   curl -F client_name=justmytoots -F scopes=profile \
     -F redirect_uris=https://justmytoots.example.social/auth/callback \
     https://example.social/api/v1/apps
   ```

2. Create a folder with these two files. The image is ready to pull, so you don't need this repository.

   `.env`:

   ```ini
   JUSTMYTOOTS_INSTANCE=https://example.social
   JUSTMYTOOTS_BASE_URL=https://justmytoots.example.social
   JUSTMYTOOTS_CLIENT_ID=the client_id from step 1
   JUSTMYTOOTS_CLIENT_SECRET=the client_secret from step 1
   JUSTMYTOOTS_SECRET=the output of: openssl rand -hex 32
   ```

   `docker-compose.yml`:

   ```yaml
   services:
     justmytoots:
       image: registry.gutocarvalho.net/images/justmytoots:latest
       container_name: justmytoots
       restart: unless-stopped
       env_file: .env
       read_only: true
       cap_drop: [ALL]
       security_opt: ["no-new-privileges:true"]
       mem_limit: 128m
       ports:
         - "127.0.0.1:8080:8080"
   ```

3. Start it: `docker compose up -d`.

4. Point your reverse proxy at `127.0.0.1:8080`. It must serve HTTPS: the session cookie needs it.

To update: `docker compose pull && docker compose up -d`. For a fixed version, swap `latest` for a tag such as `0.1.5-850`.

To build it yourself instead, clone this repository, copy `.env.example` to `.env` and run `docker compose up -d --build`.

### Versions

Versions look like `X.Y.Z-BUILD`. `X.Y.Z` is in the `VERSION` file and follows SemVer, and `BUILD` is the GitLab pipeline number. A local build shows `X.Y.Z-dev`. The page footer shows the version and links to the [container registry](https://gitlab.gutocarvalho.net/images/justmytoots/container_registry).

What changed in each version is in [CHANGELOG.md](CHANGELOG.md), which the app also serves as its release notes page (`/release-notes`, linked at the top of the page) and as `/release-notes.json`. Add an entry when you bump `VERSION`: a test fails without it.

| Image tag | Moves? |
|---|---|
| `X.Y.Z-BUILD` | Never. Pin this one in production. |
| `X.Y.Z` | On each build of that version. |
| `latest` | On each build. |

### Settings

| Variable | Required | Default | What it is |
|---|---|---|---|
| `JUSTMYTOOTS_INSTANCE` | yes |  | Your instance. Only its accounts have a page. |
| `JUSTMYTOOTS_BASE_URL` | yes |  | Where this app is served, always https. |
| `JUSTMYTOOTS_LEGACY_HOSTS` | no |  | Old hosts of this app, comma separated. Profile links to them still count, so a rename doesn't take pages down. |
| `JUSTMYTOOTS_CLIENT_ID` | yes |  | OAuth application. |
| `JUSTMYTOOTS_CLIENT_SECRET` | yes |  | OAuth application. |
| `JUSTMYTOOTS_SECRET` | yes |  | Signs the session cookie. |
| `JUSTMYTOOTS_LANG` | no | `en` | `en` or `pt`, when the browser asks for neither. |
| `JUSTMYTOOTS_LISTEN` | no | `:8080` | Address inside the container. |

## How it works

- Pages are public, at `/@name`, but opt-in: one only exists while the account links to it from its bio or a profile field. Remove the link and the page goes away. The original `/@name@instance` links keep working.
- Posts come from the instance's public API, as an anonymous visitor would see them, so private and followers-only posts never show up. Filters: pinned, replies, media only, public only.
- The HTML of each post is sanitized again here, and the page has a strict CSP.
- Answers are cached in memory for a minute, to spare the instance while keeping deleted posts from lingering.
- Signing in is only for seeing your own page and link: OAuth with PKCE and the minimal `profile` scope. The token is used once to read the username, then revoked. The session is a signed cookie that lasts 4 hours.
- English and Brazilian Portuguese. It follows the browser, with a selector at the top to switch (kept in a cookie).
- No JavaScript, no storage, one small container.

## Development

    go test ./...

Changing the code, with or without a coding agent: [`AGENTS.md`](AGENTS.md) has the commands,
a map of the files and the rules a change must keep.

AGPL-3.0, see `LICENSE` and `NOTICE`.
