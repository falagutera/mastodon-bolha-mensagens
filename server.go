package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

type filter string

const (
	filterPosts   filter = ""
	filterReplies filter = "replies"
	filterMedia   filter = "media"
	filterPublic  filter = "public"
)

// Long enough to spare the instance, short enough that a deleted post or a
// removed link goes away quickly.
const (
	cacheTTL   = time.Minute
	cacheLimit = 512
)

var (
	errOtherInstance = errors.New("account from another instance")
	errBadHandle     = errors.New("invalid handle")
	errNoConsent     = errors.New("no link on the profile")
	usernameRe       = regexp.MustCompile(`^[A-Za-z0-9_]+([A-Za-z0-9_.-]*[A-Za-z0-9_])?$`)
	idRe             = regexp.MustCompile(`^[0-9]{1,24}$`)
)

// parseHandle accepts name and name@instance, the form the original site used.
func parseHandle(in, domain string) (string, error) {
	user, host, remote := strings.Cut(strings.TrimPrefix(in, "@"), "@")
	if remote && !strings.EqualFold(host, domain) {
		return "", errOtherInstance
	}
	if len(user) > 64 || !usernameRe.MatchString(user) {
		return "", errBadHandle
	}
	return user, nil
}

type cached struct {
	at   time.Time
	page *listing
	err  error
}

type server struct {
	cfg    config
	tmpl   *template.Template
	client *http.Client
	api    *api

	mu    sync.Mutex
	cache map[string]cached
}

func newServer(cfg config, tmpl *template.Template, client *http.Client) *server {
	return &server{cfg: cfg, tmpl: tmpl, client: client, api: &api{base: cfg.instance, client: client}, cache: map[string]cached{}}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("GET /robots.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "User-agent: *\nDisallow: /\n")
	})
	assets, err := fs.Sub(assetFS, "assets")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServerFS(assets)))
	mux.HandleFunc("GET /login", s.handleLogin)
	mux.HandleFunc("GET /auth/callback", s.handleCallback)
	mux.HandleFunc("POST /logout", s.handleLogout)
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /{handle}", s.handleProfile)
	return securityHeaders(mux)
}

// securityHeaders allows same-origin scripts for the bar the edge injects on
// bolha sites, and images from anywhere for media and emoji. The app itself has
// no JavaScript.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Cache-Control", "no-store")
		h.Set("X-Robots-Tag", "noindex, nofollow")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; connect-src 'self'; style-src 'self'; font-src 'self'; img-src 'self' https:; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

type tab struct {
	Label, Href string
	On          bool
}

// listing is the part of a profile page shared by everyone who sees it.
type listing struct {
	Account *account
	Name    string
	Posts   []post
	Next    string
}

type pageData struct {
	T        map[string]string
	Domain   string
	Message  string
	PageURL  string
	Username string
	Active   bool
	LoggedIn bool
	List     *listing
	Tabs     []tab
}

func (s *server) page(r *http.Request) pageData {
	return pageData{T: strs(language(r, s.cfg.lang)), Domain: s.cfg.domain}
}

func (s *server) render(w http.ResponseWriter, status int, name string, data pageData) {
	var buf bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	w.Write(buf.Bytes())
}

func (s *server) renderError(w http.ResponseWriter, r *http.Request, status int, key string) {
	d := s.page(r)
	d.Message = strings.ReplaceAll(d.T["error_"+key], "%s", s.cfg.domain)
	s.render(w, status, "index.html", d)
}

// handleIndex presents the site. Signed in, it shows the person's own page.
func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if lang := r.URL.Query().Get("lang"); lang != "" {
		if _, ok := translations[lang]; ok {
			s.setCookie(w, langCookie, lang, 365*24*time.Hour)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	d := s.page(r)
	if sess, ok := s.currentSession(r); ok {
		acc, err := s.api.lookup(r.Context(), sess.Account)
		if err != nil {
			s.fail(w, r, err)
			return
		}
		d.LoggedIn = true
		d.Username = acc.Username
		d.PageURL = s.cfg.baseURL + "/@" + acc.Username
		d.Active = consented(acc, s.cfg.host, s.cfg.domain)
	}
	s.render(w, http.StatusOK, "index.html", d)
}

func (s *server) handleProfile(w http.ResponseWriter, r *http.Request) {
	handle := r.PathValue("handle")
	if !strings.HasPrefix(handle, "@") {
		s.fail(w, r, errBadHandle)
		return
	}
	user, err := parseHandle(handle, s.cfg.domain)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	q := r.URL.Query()
	f := filter(q.Get("f"))
	maxID := q.Get("max_id")
	if handle != "@"+user {
		http.Redirect(w, r, "/@"+url.PathEscape(user), http.StatusMovedPermanently) // links from the original site
		return
	}
	if (maxID != "" && !idRe.MatchString(maxID)) || (f != filterPosts && f != filterReplies && f != filterMedia && f != filterPublic) {
		s.fail(w, r, errBadHandle)
		return
	}
	d := s.page(r)
	list, err := s.listing(r.Context(), user, f, maxID, d.T)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	d.List = list
	d.Username = list.Account.Username
	base := "/@" + url.PathEscape(list.Account.Username)
	for _, t := range []struct {
		f     filter
		label string
	}{{filterPosts, "tab_posts"}, {filterReplies, "tab_replies"}, {filterMedia, "tab_media"}, {filterPublic, "tab_public"}} {
		href := base
		if t.f != filterPosts {
			href += "?f=" + string(t.f)
		}
		d.Tabs = append(d.Tabs, tab{Label: d.T[t.label], Href: href, On: t.f == f})
	}
	s.render(w, http.StatusOK, "profile.html", d)
}

// listing returns a page of posts through a short cache. Errors are cached too,
// so a missing account can't be used to hammer the instance.
func (s *server) listing(ctx context.Context, user string, f filter, maxID string, t map[string]string) (*listing, error) {
	key := strings.ToLower(user) + "|" + string(f) + "|" + maxID + "|" + t["lang"]
	s.mu.Lock()
	c, ok := s.cache[key]
	s.mu.Unlock()
	if ok && time.Since(c.at) < cacheTTL {
		return c.page, c.err
	}
	page, err := s.fetch(ctx, user, f, maxID, t)
	if errors.Is(err, context.Canceled) {
		return nil, err
	}
	s.mu.Lock()
	if len(s.cache) >= cacheLimit {
		clear(s.cache)
	}
	s.cache[key] = cached{at: time.Now(), page: page, err: err}
	s.mu.Unlock()
	return page, err
}

func (s *server) fetch(ctx context.Context, user string, f filter, maxID string, t map[string]string) (*listing, error) {
	acc, err := s.api.lookup(ctx, user)
	if err != nil {
		return nil, err
	}
	if !consented(acc, s.cfg.host, s.cfg.domain) {
		return nil, errNoConsent
	}
	statuses, err := s.api.posts(ctx, acc.ID, f, maxID)
	if err != nil {
		return nil, err
	}
	l := &listing{Account: acc, Name: displayName(acc)}
	for _, st := range statuses {
		if f == filterPublic && st.Visibility != "public" {
			continue
		}
		l.Posts = append(l.Posts, newPost(st, t))
	}
	if len(statuses) == postsPerPage {
		q := url.Values{"max_id": {statuses[len(statuses)-1].ID}}
		if f != filterPosts {
			q.Set("f", string(f))
		}
		l.Next = "/@" + url.PathEscape(acc.Username) + "?" + q.Encode()
	}
	return l, nil
}

func (s *server) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, context.Canceled):
	case errors.Is(err, errNoAccount):
		s.renderError(w, r, http.StatusNotFound, "account")
	case errors.Is(err, errNoConsent):
		s.renderError(w, r, http.StatusNotFound, "consent")
	case errors.Is(err, errOtherInstance):
		s.renderError(w, r, http.StatusUnprocessableEntity, "other")
	case errors.Is(err, errBadHandle):
		s.renderError(w, r, http.StatusNotFound, "notfound")
	default:
		log.Printf("page %s: %v", r.URL.Path, err)
		s.renderError(w, r, http.StatusBadGateway, "unavailable")
	}
}
