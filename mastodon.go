package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const postsPerPage = 40

var errNoAccount = errors.New("account not found")

// api reads the instance's public API, as an anonymous visitor would: it only
// ever returns what is public, and already renders each post to HTML.
type api struct {
	base   string
	client *http.Client
}

type account struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Acct        string `json:"acct"`
	DisplayName string `json:"display_name"`
	Avatar      string `json:"avatar_static"`
	URL         string `json:"url"`
	Note        string `json:"note"`
	Posts       int    `json:"statuses_count"`
	Fields      []struct {
		Value string `json:"value"`
	} `json:"fields"`
}

type emoji struct {
	Shortcode string `json:"shortcode"`
	URL       string `json:"static_url"`
}

type media struct {
	Type        string `json:"type"`
	URL         string `json:"url"`
	Preview     string `json:"preview_url"`
	Description string `json:"description"`
}

type status struct {
	ID         string  `json:"id"`
	CreatedAt  string  `json:"created_at"`
	URL        string  `json:"url"`
	Visibility string  `json:"visibility"`
	Content    string  `json:"content"`
	Spoiler    string  `json:"spoiler_text"`
	Sensitive  bool    `json:"sensitive"`
	Replies    int     `json:"replies_count"`
	Boosts     int     `json:"reblogs_count"`
	Favourites int     `json:"favourites_count"`
	Media      []media `json:"media_attachments"`
	Emojis     []emoji `json:"emojis"`
}

func (a *api) get(ctx context.Context, path string, q url.Values, out any) error {
	u := a.base + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		return errNoAccount
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("instance answered %s", resp.Status)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(out)
}

// lookup finds a local account.
func (a *api) lookup(ctx context.Context, username string) (*account, error) {
	var acc account
	if err := a.get(ctx, "/api/v1/accounts/lookup", url.Values{"acct": {username}}, &acc); err != nil {
		return nil, err
	}
	if strings.Contains(acc.Acct, "@") {
		return nil, errNoAccount
	}
	return &acc, nil
}

// posts returns one page of an account's posts, never boosts.
func (a *api) posts(ctx context.Context, id string, f filter, maxID string) ([]status, error) {
	q := url.Values{"exclude_reblogs": {"true"}, "limit": {strconv.Itoa(postsPerPage)}}
	if f == filterPinned {
		q.Set("pinned", "true")
	} else if f != filterReplies {
		q.Set("exclude_replies", "true")
	}
	if f == filterMedia {
		q.Set("only_media", "true")
	}
	if maxID != "" {
		q.Set("max_id", maxID)
	}
	var out []status
	err := a.get(ctx, "/api/v1/accounts/"+url.PathEscape(id)+"/statuses", q, &out)
	return out, err
}

// consented tells if the account links to its page here, on the bio or on a
// profile field. A page only exists because its owner asked for it, and removing
// the link takes it down. Both /@name and the original /@name@instance count,
// on the current host or on a legacy one (links made before a rename still work).
func consented(acc *account, hosts []string, domain string) bool {
	quoted := make([]string, len(hosts))
	for i, h := range hosts {
		quoted[i] = regexp.QuoteMeta(h)
	}
	re := regexp.MustCompile(`(?i)https?://(www\.)?(` + strings.Join(quoted, "|") + `)/@?` + regexp.QuoteMeta(acc.Username) +
		`(@` + regexp.QuoteMeta(domain) + `)?([/?"<\s]|$)`)
	if re.MatchString(acc.Note) {
		return true
	}
	for _, f := range acc.Fields {
		if re.MatchString(f.Value) {
			return true
		}
	}
	return false
}
