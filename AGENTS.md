# AGENTS.md

Notes for coding agents (and people) changing justmytoots. The README says what the app does and
how to install it; this file says how to change it without breaking the promises the README makes.

## What it is

One Go binary, no JavaScript, no database, no disk. `/@name` shows the posts of a local account,
read from the instance's **public API as an anonymous visitor**, sanitized again and cached in
memory for a minute. A page exists only while the account links to it from its profile. Signing
in (OAuth) only shows a person whether their page is on.

## Commands

```sh
gofmt -l .            # must print nothing
go vet ./...
go test ./...         # everything runs against a fake instance (fakeInstance in toots_test.go)
```

There is no database and no test variable. To look at a rendered profile page:
`D=$(mktemp -d) && JUSTMYTOOTS_DUMP=$D go test -run TestProfilePage ./... && echo $D`
writes `profile.html` and `assets/` there. To run the app for real you need an instance and an
OAuth app (README, Install); `JUSTMYTOOTS_BASE_URL` (or one of `JUSTMYTOOTS_LEGACY_HOSTS`) must be the address in the profile link.

## Map

| File | What |
|---|---|
| `main.go` | config from `JUSTMYTOOTS_*` env, HTTP server timeouts, `-healthcheck` |
| `server.go` | routes, `securityHeaders` (CSP), `parseHandle`, `/@name` handler, filters, the cache (`listing`, `remember`) |
| `mastodon.go` | the public API client (`lookup`, `posts`) and `consented` (the opt-in check) |
| `render.go` | `postPolicy` (bluemonday), `sanitize`, emoji in text nodes only, `newPost` (CW, sensitive media, counters) |
| `auth.go` | OAuth (state + PKCE), signed cookies (`seal`/`open`), token revoke |
| `i18n.go` | every text, in `en` and `pt`, month names and date format |
| `templates/` | `layout.html` (head/foot), `index.html`, `profile.html`, `pictures.html`, `notes.html` |
| `assets/` | `style.css` (the look, a copy from open/gcn/ui: never edit it here) and `toots.css` (the app's own CSS, using the ui tokens) |
| `internal/relnotes` | copy of the shared release-notes package (synced from its own repo, never edited here) |

Tests: `toots_test.go` (`fakeInstance` answers like Mastodon for `guto`, whose bio links the page
while `linked` is true; `testServer`, `get`), `emoji_test.go`, `i18n_test.go`, `changelog_test.go`,
`dump_test.go` (the `JUSTMYTOOTS_DUMP` helper).

## Rules that must hold

1. **Opt-in.** A page is served only while the account's bio or a profile field links to it
   (`consented`: this host, `/@name` or `/name`, optional `@domain`, nothing after the name but
   `/ ? " <` or space). Posts are not even fetched without it, and errors are cached too, so
   removing the link takes the page down within `cacheTTL` (1 minute).
   `TestConsent` (look-alike hosts and names must fail), `TestNoLinkNoPage`.
2. **Only what is public.** Posts come from the API **without a token** (`api` has none), so
   followers-only and direct posts never come back; never add authentication to `api.get`.
   Boosts are always excluded (`exclude_reblogs=true`); the "public only" filter also drops
   unlisted posts. `TestProfilePage`, `TestFilters`.
3. **Local accounts only.** `parseHandle` refuses another domain and odd names; `lookup` refuses
   an account whose `acct` has a domain. `TestParseHandle`, `TestOtherAccounts`.
4. **Post HTML is untrusted.** It goes through `postPolicy` (no images, scripts, styles or
   event attributes, http/https/mailto links with `rel=noreferrer`), and custom emoji become
   `<img>` only inside text nodes, with an https URL and a plain shortcode. Everything else in
   the templates is escaped by `html/template`. `TestSanitize`, `TestEmojiCantInject`,
   `TestEmojiOnlyInTextNodes`. Content warnings and sensitive media stay behind `<details>`.
5. **Inputs are allowlisted.** `f` is one of the `filter` constants and `max_id` matches `idRe`;
   anything else is a 404 and never reaches the instance (`TestFilters`).
6. **The sign-in token never outlives the request.** Scope `profile` only, read the username,
   revoke (`defer s.revokeToken`). The session (4 hours) holds just the username.
   `TestLoginShowsOwnPage`. Logout is a POST refused cross-site (`handleLogout`).
7. **No JavaScript, no inline style.** CSP in `securityHeaders`: `default-src 'none'`, `'self'`
   for script/style/font, `img-src 'self' https:` because media and avatars load straight from
   the instance. CSS goes in `assets/toots.css`.
8. **Both languages.** Every key in `translations` exists in `en` and `pt`, with 12 month names
   (`TestTextsInBothLanguages`). Error texts may hold one `%s` (the domain) and no other `%`
   (`TestErrorMessagesAreClean`). The cache key includes the language (`TestLanguage`).
9. **Bounded memory and load.** 40 posts a page, 8 MiB per API answer, at most 512 cached pages
   and 32 MiB (`cacheLimit`, `cacheBytes`; `TestCacheIsBoundedByBytes`), one lookup per minute
   per page (`TestCache`).

## Adding a filter or a post detail (the usual change)

0. **Check it is not there already.** Filters: the `filter` constants and the tab list in
   `handleProfile` (posts, pinned, with replies, media only, public only). Post details:
   `newPost` and `templates/profile.html` (date, CW, pictures, replies/boosts/favourites).
1. Filter: a constant, the allowlist check in `handleProfile`, the query in `api.posts`, a tab
   with a `tab_*` text. Detail: the field in `status`, then `post`, then the template; HTML from
   the instance only through `sanitize`, URLs only if `https://`; CSS for a new element in
   `assets/toots.css`. Validate or parse any API field you branch on (timestamps, enums) and test
   it with a junk value.
2. Texts in both languages in `i18n.go`.
3. A case in `TestFilters` (the query sent, and that a bad value is still a 404) or in
   `TestProfilePage` (the fake post list is an inline literal in `fakeInstance`'s `ServeHTTP`;
   add the field there), plus what must not show.
4. `gofmt`, `go vet`, `go test`, and look at the dump.

## Releasing

- `VERSION` holds `X.Y.Z`; CI adds the build number (`X.Y.Z-BUILD`). Which part to bump: a fix or a
  small visual tweak is a patch; a new feature, option or detail is a minor; a change that needs a
  new grant or setting at install is a minor whose changelog entry starts with that step in **bold**;
  a breaking change is a major; docs, tests or CI only need no bump. Work not released yet can sit
  under `## Unreleased` in `CHANGELOG.md` (`internal/relnotes` skips that section).
- Every version needs its `CHANGELOG.md` entry (`TestChangelogHasVersion`); it is also the public
  `/release-notes` page and `/release-notes.json`, so write it for the people who use the site.
- Code, comments, commits and the README in English. Keep commits small with a message that says why.

## Security checks

CI runs `go vet`, `gofmt -l`, the tests, govulncheck and the shared security stage: gitleaks, govulncheck, trivy, gosec,
semgrep, poutine, hadolint, shellcheck, yamllint. Before a release that touches sanitizing, the
opt-in check or sign-in, also review what a stranger could make the page show — the part scanners
do not see.
