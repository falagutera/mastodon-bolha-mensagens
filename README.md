# justmytoots

A clean, public page with the posts of an account of a Mastodon instance: **justmytoots.bolha.us**.

It shows a person's own posts, without boosts, timelines or the rest of the interface, so they're easy to read, follow and share. Think jokes, drawings, photos, or anything worth looking back through. Only accounts of the instance have a page. To offer it on your instance, run your own.

It's a rewrite in Go of [JustMyToots](https://github.com/cookiemumbles/justmytoots.com) by cookiemumbles, see `NOTICE`.

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

AGPL-3.0, see `LICENSE` and `NOTICE`.
