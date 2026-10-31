package main

import (
	"html"
	"html/template"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
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
	for _, e := range emojis {
		if !shortcodeRe.MatchString(e.Shortcode) || !strings.HasPrefix(e.URL, "https://") {
			continue
		}
		tag := `<img class="emoji" src="` + html.EscapeString(e.URL) + `" alt=":` + e.Shortcode + `:" width="20" height="20">`
		clean = strings.ReplaceAll(clean, ":"+e.Shortcode+":", tag)
	}
	return template.HTML(clean) // sanitized above
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
