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

2. Configure and start it:

   ```sh
   cp .env.example .env   # URLs, OAuth application and secret
   docker compose up -d
   ```

3. Point your reverse proxy at `127.0.0.1:8080`. It must serve HTTPS: the session cookie needs it.

## How it works

- Pages are public, at `/@name`, but opt-in: one only exists while the account links to it from its bio or a profile field. Remove the link and the page goes away. The original `/@name@instance` links keep working.
- Posts come from the instance's public API, as an anonymous visitor would see them, so private and followers-only posts never show up. Filters: replies, media only, public only.
- The HTML of each post is sanitized again here, and the page has a strict CSP.
- Answers are cached in memory for a minute, to spare the instance while keeping deleted posts from lingering.
- Signing in is only for seeing your own page and link: OAuth with PKCE and the minimal `profile` scope. The token is used once to read the username, then revoked. The session is a signed cookie that lasts 4 hours.
- English and Brazilian Portuguese. It follows the browser, with a selector at the top to switch (kept in a cookie).
- No JavaScript, no storage, one small container.

## Development

    go test ./...

AGPL-3.0, see `LICENSE` and `NOTICE`.
