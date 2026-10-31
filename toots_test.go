package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseHandle(t *testing.T) {
	for in, want := range map[string]string{"@guto": "guto", "@guto@bolha.us": "guto", "@guto@BOLHA.US": "guto", "@a.b_c": "a.b_c"} {
		if got, err := parseHandle(in, "bolha.us"); err != nil || got != want {
			t.Errorf("parseHandle(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := parseHandle("@a@mastodon.social", "bolha.us"); !errors.Is(err, errOtherInstance) {
		t.Errorf("other instance: %v", err)
	}
	for _, in := range []string{"@", "@a/b", "@a b", "@../x", "@guto.", "@" + strings.Repeat("a", 65)} {
		if _, err := parseHandle(in, "bolha.us"); !errors.Is(err, errBadHandle) {
			t.Errorf("parseHandle(%q): %v", in, err)
		}
	}
}

func TestConsent(t *testing.T) {
	const host, domain = "justmytoots.bolha.us", "bolha.us"
	link := func(u string) string { return `<a href="` + u + `" rel="me">x</a>` }
	yes := []string{
		"https://justmytoots.bolha.us/@guto", "https://justmytoots.bolha.us/guto", "https://justmytoots.bolha.us/@guto@bolha.us",
		"https://justmytoots.bolha.us/@GUTO", "https://justmytoots.bolha.us/@guto?f=media", "https://justmytoots.bolha.us/@guto/",
	}
	no := []string{
		"https://justmytoots.bolha.us/@guto2", "https://justmytoots.bolha.us/@guto@other.social", "https://justmytoots.bolha.us/@other",
		"https://evil.example/justmytoots.bolha.us/@guto", "https://justmytoots.bolha.us.evil.example/@guto", "https://justmytoots.bolha.us/@guto.bak",
	}
	for _, u := range yes {
		acc := &account{Username: "guto", Note: link(u)}
		if !consented(acc, host, domain) {
			t.Errorf("bio link %q not accepted", u)
		}
	}
	for _, u := range no {
		acc := &account{Username: "guto", Note: link(u)}
		if consented(acc, host, domain) {
			t.Errorf("bio link %q accepted", u)
		}
	}
	acc := &account{Username: "guto"}
	acc.Fields = append(acc.Fields, struct {
		Value string `json:"value"`
	}{link(yes[0])})
	if !consented(acc, host, domain) {
		t.Error("profile field not accepted")
	}
}

func TestSanitize(t *testing.T) {
	got := string(sanitize(`<p onclick="x()">oi <script>alert(1)</script><a href="javascript:alert(1)">a</a><a href="https://ok.example" class="mention evil">b</a><img src="https://x/y" onerror="z()"><iframe src="https://x"></iframe><span class="invisible">https://</span></p>`, nil))
	for _, bad := range []string{"onclick", "<script", "alert(1)", "javascript:", "onerror", "<iframe", "<img", "evil"} {
		if strings.Contains(got, bad) {
			t.Errorf("%q survived: %s", bad, got)
		}
	}
	for _, good := range []string{"<p>", `href="https://ok.example"`, `class="invisible"`, `rel="noreferrer"`} {
		if !strings.Contains(got, good) {
			t.Errorf("%q missing: %s", good, got)
		}
	}
}

func TestEmojiCantInject(t *testing.T) {
	got := string(sanitize("<p>:ok: :bad:</p>", []emoji{
		{Shortcode: "ok", URL: "https://x/o.png"},
		{Shortcode: "bad", URL: `javascript:alert(1)`},
		{Shortcode: `"><script>`, URL: "https://x/s.png"},
	}))
	if !strings.Contains(got, `<img class="emoji" src="https://x/o.png"`) || strings.Contains(got, "javascript") || strings.Contains(got, "<script") {
		t.Fatal(got)
	}
}

func TestCountersHideZero(t *testing.T) {
	tr := strs("pt")
	p := newPost(status{Replies: 0, Boosts: 1, Favourites: 3, CreatedAt: "2026-10-31T05:00:00Z"}, tr)
	if len(p.Counters) != 2 || p.Counters[0].Label != "1 impulso" || p.Counters[1].Label != "3 favoritos" {
		t.Fatalf("%+v", p.Counters)
	}
	if p.Date != "31 out 2026" {
		t.Fatalf("date %q", p.Date)
	}
	if d := formatDate("2026-10-31T05:00:00Z", strs("en")); d != "Oct 31, 2026" {
		t.Fatalf("date %q", d)
	}
}

// fakeInstance answers like Mastodon for one account, guto, who linked the page.
type fakeInstance struct {
	lookups atomic.Int32
	query   atomic.Value
	linked  atomic.Bool
}

func (f *fakeInstance) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/oauth/token":
		fmt.Fprint(w, `{"access_token":"tok"}`)
	case r.URL.Path == "/oauth/revoke":
	case r.URL.Path == "/api/v1/accounts/verify_credentials":
		fmt.Fprint(w, `{"username":"guto"}`)
	case r.URL.Path == "/api/v1/accounts/lookup":
		f.lookups.Add(1)
		switch r.URL.Query().Get("acct") {
		case "guto":
			note := ""
			if f.linked.Load() {
				note = `<p><a href="https://justmytoots.bolha.us/@guto">page</a></p>`
			}
			json.NewEncoder(w).Encode(map[string]any{"id": "7", "username": "guto", "acct": "guto", "display_name": "<b>Guto</b> :blob:", "url": "https://bolha.us/@guto", "avatar_static": "https://m/a.png", "note": note, "statuses_count": 3})
		case "far":
			fmt.Fprint(w, `{"id":"8","username":"far","acct":"far@other.social"}`)
		default:
			w.WriteHeader(404)
		}
	case strings.HasPrefix(r.URL.Path, "/api/v1/accounts/7/statuses"):
		f.query.Store(r.URL.RawQuery)
		list := []map[string]any{
			{"id": "3", "created_at": "2026-10-31T05:00:00Z", "url": "https://bolha.us/@guto/3", "visibility": "public", "content": "<p>olá <script>x</script></p>", "spoiler_text": "", "favourites_count": 2,
				"media_attachments": []map[string]any{{"type": "image", "url": "https://m/full.png", "preview_url": "https://m/p.png", "description": "desc"}}},
			{"id": "2", "created_at": "2026-10-30T05:00:00Z", "url": "https://bolha.us/@guto/2", "visibility": "unlisted", "content": "<p>só listado</p>", "spoiler_text": "aviso", "media_attachments": []any{}},
		}
		json.NewEncoder(w).Encode(list)
	default:
		w.WriteHeader(404)
	}
}

func testServer(t *testing.T) (*server, *httptest.Server, *fakeInstance) {
	t.Helper()
	inst := &fakeInstance{}
	inst.linked.Store(true)
	is := httptest.NewServer(inst)
	t.Cleanup(is.Close)
	cfg := config{domain: "bolha.us", host: "justmytoots.bolha.us", lang: "pt", instance: is.URL, baseURL: "https://justmytoots.bolha.us",
		clientID: "id", clientSecret: "secret", secret: []byte(strings.Repeat("k", 32))}
	tmpl := template.Must(template.ParseFS(templateFS, "templates/*.html"))
	s := newServer(cfg, tmpl, &http.Client{Timeout: time.Second})
	ts := httptest.NewServer(s.routes())
	t.Cleanup(ts.Close)
	return s, ts, inst
}

func get(t *testing.T, url string, cookies ...*http.Cookie) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	c := http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var b bytes.Buffer
	b.ReadFrom(resp.Body)
	return resp, b.String()
}

func TestProfilePage(t *testing.T) {
	_, ts, inst := testServer(t)
	resp, body := get(t, ts.URL+"/@guto")
	if resp.StatusCode != 200 {
		t.Fatalf("profile: %d", resp.StatusCode)
	}
	for _, want := range []string{"olá", "@guto@bolha.us", "2 favoritos", "<details><summary>aviso</summary>", `href="https://m/full.png"`, "Abrir na bolha.us", "&lt;b&gt;Guto&lt;/b&gt;"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(body, "<script>x") {
		t.Fatal("script reached the page")
	}
	q := inst.query.Load().(string)
	if !strings.Contains(q, "exclude_reblogs=true") || !strings.Contains(q, "exclude_replies=true") {
		t.Fatalf("query %s", q)
	}
	if !strings.Contains(resp.Header.Get("Content-Security-Policy"), "default-src 'none'") {
		t.Fatal("missing CSP")
	}
}

func TestFilters(t *testing.T) {
	_, ts, inst := testServer(t)
	get(t, ts.URL+"/@guto?f=replies")
	if q := inst.query.Load().(string); strings.Contains(q, "exclude_replies") {
		t.Fatalf("replies: %s", q)
	}
	get(t, ts.URL+"/@guto?f=media")
	if q := inst.query.Load().(string); !strings.Contains(q, "only_media=true") {
		t.Fatalf("media: %s", q)
	}
	if _, body := get(t, ts.URL+"/@guto?f=public"); strings.Contains(body, "só listado") {
		t.Fatal("unlisted post on the public filter")
	}
	for _, u := range []string{"/@guto?f=bad", "/@guto?max_id=1%26limit%3D999"} {
		if resp, _ := get(t, ts.URL+u); resp.StatusCode != 404 {
			t.Errorf("%s: %d", u, resp.StatusCode)
		}
	}
}

func TestNoLinkNoPage(t *testing.T) {
	_, ts, inst := testServer(t)
	inst.linked.Store(false)
	resp, body := get(t, ts.URL+"/@guto")
	if resp.StatusCode != 404 || !strings.Contains(body, "não ativou") || strings.Contains(body, "olá") || strings.Contains(body, "%!") {
		t.Fatalf("%d", resp.StatusCode)
	}
	if inst.query.Load() != nil {
		t.Fatal("posts fetched without consent")
	}
}

func TestOtherAccounts(t *testing.T) {
	_, ts, _ := testServer(t)
	if resp, body := get(t, ts.URL+"/@guto@other.social"); resp.StatusCode != 422 || !strings.Contains(body, "Só contas da bolha.us") {
		t.Fatalf("other instance: %d", resp.StatusCode)
	}
	if resp, _ := get(t, ts.URL+"/@far"); resp.StatusCode != 404 {
		t.Fatalf("remote account: %d", resp.StatusCode)
	}
	if resp, _ := get(t, ts.URL+"/@nobody"); resp.StatusCode != 404 {
		t.Fatalf("missing: %d", resp.StatusCode)
	}
	resp, _ := get(t, ts.URL+"/@guto@bolha.us")
	if resp.StatusCode != 301 || resp.Header.Get("Location") != "/@guto" {
		t.Fatalf("old link: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestCache(t *testing.T) {
	_, ts, inst := testServer(t)
	get(t, ts.URL+"/@guto")
	get(t, ts.URL+"/@guto")
	get(t, ts.URL+"/@GUTO")
	if n := inst.lookups.Load(); n != 1 {
		t.Fatalf("%d lookups", n)
	}
	get(t, ts.URL+"/@nobody")
	get(t, ts.URL+"/@nobody")
	if n := inst.lookups.Load(); n != 2 {
		t.Fatalf("errors not cached: %d lookups", n)
	}
}

func TestLoginShowsOwnPage(t *testing.T) {
	s, ts, inst := testServer(t)
	resp, body := get(t, ts.URL+"/")
	if resp.StatusCode != 200 || !strings.Contains(body, "/login") {
		t.Fatalf("index: %d", resp.StatusCode)
	}
	resp, _ = get(t, ts.URL+"/login")
	loc, _ := url.Parse(resp.Header.Get("Location"))
	if resp.StatusCode != 302 || loc.Query().Get("scope") != "profile" || loc.Query().Get("code_challenge") == "" {
		t.Fatalf("login: %d %s", resp.StatusCode, loc)
	}
	var state, sess *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == stateCookie {
			state = &http.Cookie{Name: c.Name, Value: c.Value}
		}
	}
	if resp, _ := get(t, ts.URL+"/auth/callback?code=c&state=wrong", state); resp.StatusCode != 400 {
		t.Fatalf("wrong state: %d", resp.StatusCode)
	}
	resp, _ = get(t, ts.URL+"/auth/callback?code=c&state="+url.QueryEscape(loc.Query().Get("state")), state)
	for _, c := range resp.Cookies() {
		if c.Name == sessionCookie {
			sess = &http.Cookie{Name: c.Name, Value: c.Value}
		}
	}
	if sess == nil {
		t.Fatalf("no session, status %d", resp.StatusCode)
	}
	if _, body := get(t, ts.URL+"/", sess); !strings.Contains(body, "está ativa") || !strings.Contains(body, "https://justmytoots.bolha.us/@guto") {
		t.Fatal(body)
	}
	inst.linked.Store(false)
	if _, body := get(t, ts.URL+"/", sess); !strings.Contains(body, "ainda não está ativa") {
		t.Fatal(body)
	}
	expired, _ := s.seal("session", session{Account: "guto", Expires: time.Now().Add(-time.Minute)})
	if _, body := get(t, ts.URL+"/", &http.Cookie{Name: sessionCookie, Value: expired}); !strings.Contains(body, "/login") {
		t.Fatal("expired session accepted")
	}
}

func TestErrorMessagesAreClean(t *testing.T) {
	_, ts, _ := testServer(t)
	for _, lang := range []string{"pt", "en"} {
		c := &http.Cookie{Name: langCookie, Value: lang}
		for _, u := range []string{"/@nobody", "/guto", "/@a@other.social", "/@guto?f=bad", "/auth/callback"} {
			if _, body := get(t, ts.URL+u, c); strings.Contains(body, "%!") || strings.Contains(body, "%s") {
				t.Errorf("%s %s: broken message", lang, u)
			}
		}
	}
	for lang, tr := range translations {
		for k, v := range tr {
			if strings.HasPrefix(k, "error_") && strings.Contains(v, "%") && !strings.Contains(v, "%s") {
				t.Errorf("%s %s: stray %%", lang, k)
			}
		}
	}
}

func TestLanguage(t *testing.T) {
	_, ts, _ := testServer(t)
	en := &http.Cookie{Name: langCookie, Value: "en"}
	if _, body := get(t, ts.URL+"/@guto", en); !strings.Contains(body, `<html lang="en">`) || !strings.Contains(body, "2 favourites") {
		t.Fatal("english page")
	}
	if _, body := get(t, ts.URL+"/@guto"); !strings.Contains(body, "2 favoritos") {
		t.Fatal("cache shared between languages")
	}
}
