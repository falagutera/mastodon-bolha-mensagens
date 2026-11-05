package main

import (
	"html"
	"html/template"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	xhtml "golang.org/x/net/html"
)

// The instance already sanitizes posts. This second pass means a bug there or a
// strange post from a fork can't reach the page.
var postPolicy = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("p", "br", "strong", "b", "em", "i", "u", "s", "del", "code", "pre",
		"blockquote", "ul", "ol", "li", "h1", "h2", "h3", "h4", "h5", "h6")
	p.AllowAttrs("href").OnElements("a")
	p.AllowURLSchemes("http", "https", "mailto")
	p.RequireParseableURLs(true)
	p.AddTargetBlankToFullyQualifiedLinks(false)
	p.RequireNoReferrerOnLinks(true)
	p.AllowAttrs("class").Matching(regexp.MustCompile(`^(invisible|ellipsis|h-card|hashtag|mention|u-url)( (invisible|ellipsis|h-card|hashtag|mention|u-url))*$`)).OnElements("a", "span")
	return p
}()

var (
	shortcodeRe = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	emojiNameRe = regexp.MustCompile(`:[a-zA-Z0-9_]+:`)
)

// sanitize cleans a post's HTML and turns custom emoji shortcodes into images.
func sanitize(content string, emojis []emoji) template.HTML {
	clean := postPolicy.Sanitize(content)
	repl := make(map[string]string, len(emojis))
	for _, e := range emojis {
		if !shortcodeRe.MatchString(e.Shortcode) || !strings.HasPrefix(e.URL, "https://") {
			continue
		}
		repl[e.Shortcode] = `<img class="emoji" src="` + html.EscapeString(e.URL) + `" alt=":` + e.Shortcode + `:" width="20" height="20">`
	}
	return template.HTML(replaceEmoji(clean, repl)) // sanitized above
}

// replaceEmoji swaps :shortcode: for its <img> tag, but only inside text nodes
// of the already-sanitized HTML. Replacing on the raw string let a shortcode in
// an attribute value (e.g. a link href) break out of the quotes and corrupt the
// markup, so the substitution is now tokenizer-aware.
func replaceEmoji(h string, repl map[string]string) string {
	if len(repl) == 0 {
		return h
	}
	z := xhtml.NewTokenizer(strings.NewReader(h))
	var b strings.Builder
	for {
		switch z.Next() {
		case xhtml.ErrorToken:
			return b.String()
		case xhtml.TextToken:
			b.WriteString(substituteEmoji(string(z.Text()), repl))
		default:
			b.Write(z.Raw())
		}
	}
}

// substituteEmoji replaces every known :shortcode: in a plain-text run with its
// image tag, HTML-escaping the surrounding text and leaving unknown codes as-is.
func substituteEmoji(text string, repl map[string]string) string {
	var b strings.Builder
	last := 0
	for _, m := range emojiNameRe.FindAllStringIndex(text, -1) {
		tag, ok := repl[text[m[0]+1:m[1]-1]]
		if !ok {
			continue
		}
		b.WriteString(html.EscapeString(text[last:m[0]]))
		b.WriteString(tag)
		last = m[1]
	}
	b.WriteString(html.EscapeString(text[last:]))
	return b.String()
}

// displayName drops custom emoji shortcodes, which are not worth an image here.
func displayName(a *account) string {
	if n := strings.TrimSpace(emojiNameRe.ReplaceAllString(a.DisplayName, "")); n != "" {
		return n
	}
	return a.Username
}

type picture struct {
	Src, Alt, Link string
}

type counter struct{ Label string }

// post is a status ready for the template.
type post struct {
	Date     string
	Stamp    string
	URL      string
	Warning  string
	Body     template.HTML
	Pictures []picture
	Hidden   bool // media marked sensitive, without a content warning
	Counters []counter
}

func formatDate(stamp string, t map[string]string) string {
	d, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		return ""
	}
	months := strings.Split(t["months"], " ")
	return strings.NewReplacer("{d}", strconv.Itoa(d.Day()), "{m}", months[d.Month()-1], "{y}", strconv.Itoa(d.Year())).Replace(t["date"])
}

func count(n int, key string, t map[string]string) []counter {
	switch {
	case n == 0:
		return nil
	case n == 1:
		return []counter{{"1 " + t[key+"_one"]}}
	}
	return []counter{{strconv.Itoa(n) + " " + t[key+"_many"]}}
}

func newPost(s status, t map[string]string) post {
	p := post{
		Date:    formatDate(s.CreatedAt, t),
		Stamp:   s.CreatedAt,
		URL:     s.URL,
		Warning: s.Spoiler,
		Body:    sanitize(s.Content, s.Emojis),
		Hidden:  s.Sensitive && s.Spoiler == "",
	}
	for _, m := range s.Media {
		if !strings.HasPrefix(m.Preview, "https://") {
			continue
		}
		link := m.URL
		if m.Type != "image" || !strings.HasPrefix(link, "https://") {
			link = s.URL // video and audio play on the instance
		}
		p.Pictures = append(p.Pictures, picture{Src: m.Preview, Alt: m.Description, Link: link})
	}
	p.Counters = append(p.Counters, count(s.Replies, "replies", t)...)
	p.Counters = append(p.Counters, count(s.Boosts, "boosts", t)...)
	p.Counters = append(p.Counters, count(s.Favourites, "favourites", t)...)
	return p
}
