package main

import (
	"strings"
	"testing"
)

func TestEmojiOnlyInTextNodes(t *testing.T) {
	in := `<p><a href="https://example.com/:blob:">x</a> :blob:</p>`
	emojis := []emoji{{Shortcode: "blob", URL: "https://m.example/b.png"}}
	out := string(sanitize(in, emojis))
	if !strings.Contains(out, `href="https://example.com/:blob:"`) {
		t.Errorf("shortcode in href was altered: %s", out)
	}
	if strings.Count(out, "<img") != 1 {
		t.Errorf("expected exactly one emoji image, got: %s", out)
	}
}
