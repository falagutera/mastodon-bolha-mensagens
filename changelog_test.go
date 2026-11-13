package main

import (
	"strings"
	"testing"

	"gitlab.gutocarvalho.net/open/bolha/mensagens/internal/relnotes"
)

// Every VERSION needs its CHANGELOG.md entry: the release notes page is built from it.
func TestChangelogHasVersion(t *testing.T) {
	if err := relnotes.Check(changelog, versionFile); err != nil {
		t.Fatal(err)
	}
}

// The release notes page is public; the header links to it.
func TestReleaseNotesPage(t *testing.T) {
	_, ts, _ := testServer(t)
	if res, body := get(t, ts.URL+"/release-notes"); res.StatusCode != 200 || !strings.Contains(body, `id="v`+releases[0].Version+`"`) || !strings.Contains(body, `aria-current="page">Novidades`) {
		t.Fatalf("/release-notes: %d", res.StatusCode)
	}
	if res, body := get(t, ts.URL+"/release-notes.json"); res.StatusCode != 200 || !strings.Contains(body, `"app": "mensagens"`) {
		t.Fatalf("/release-notes.json: %d", res.StatusCode)
	}
	if _, body := get(t, ts.URL+"/"); !strings.Contains(body, `href="/release-notes"`) {
		t.Fatal("no header link")
	}
}
