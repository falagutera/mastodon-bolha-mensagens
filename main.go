// justmytoots shows the posts of an account of a Mastodon instance on a clean
// page, without boosts. See README.md.
package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed assets
var assetFS embed.FS

type config struct {
	listen       string
	baseURL      string
	host         string
	instance     string
	domain       string
	clientID     string
	clientSecret string
	secret       []byte
	lang         string
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func loadConfig() (config, error) {
	c := config{
		listen:       env("JUSTMYTOOTS_LISTEN", ":8080"),
		baseURL:      strings.TrimRight(os.Getenv("JUSTMYTOOTS_BASE_URL"), "/"),
		instance:     strings.TrimRight(os.Getenv("JUSTMYTOOTS_INSTANCE"), "/"),
		clientID:     os.Getenv("JUSTMYTOOTS_CLIENT_ID"),
		clientSecret: os.Getenv("JUSTMYTOOTS_CLIENT_SECRET"),
		secret:       []byte(os.Getenv("JUSTMYTOOTS_SECRET")),
		lang:         env("JUSTMYTOOTS_LANG", "en"),
	}
	inst, err := url.Parse(c.instance)
	if err != nil || inst.Host == "" {
		return c, fmt.Errorf("JUSTMYTOOTS_INSTANCE must be a URL like https://example.social")
	}
	c.domain = inst.Host
	base, err := url.Parse(c.baseURL)
	if err != nil || base.Host == "" {
		return c, fmt.Errorf("JUSTMYTOOTS_BASE_URL must be a URL like https://justmytoots.example.social")
	}
	c.host = base.Host
	switch {
	case c.clientID == "" || c.clientSecret == "":
		return c, fmt.Errorf("JUSTMYTOOTS_CLIENT_ID and JUSTMYTOOTS_CLIENT_SECRET are required")
	case len(c.secret) < 32:
		return c, fmt.Errorf("JUSTMYTOOTS_SECRET must have at least 32 characters")
	}
	return c, nil
}

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe the local server and exit")
	flag.Parse()
	if *healthcheck {
		resp, err := http.Get("http://127.0.0.1" + env("JUSTMYTOOTS_LISTEN", ":8080") + "/healthz")
		if err != nil || resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		return
	}
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Fatal(err)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	s := newServer(cfg, tmpl, client)
	log.Printf("listening on %s for %s", cfg.listen, cfg.domain)
	srv := &http.Server{
		Addr:              cfg.listen,
		Handler:           s.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}
