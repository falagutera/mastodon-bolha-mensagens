package main

import (
	"net/http"
	"strings"
)

const langCookie = "__Host-justmytoots-lang"

var translations = map[string]map[string]string{
	"en": {
		"lang": "en", "nav_notes": "Release notes", "notes_title": "Release notes", "notes_lede": "What changed in each version.",
		"language":          "Language",
		"brand":             "mensagens",
		"title":             "Mensagens",
		"lede":              "A clean page with the posts of an account on %s, without boosts or timelines. Easy to read, easy to share.",
		"login_button":      "Sign in with your account",
		"how_title":         "How it works",
		"how_1":             "Pages are public, but only exist for those who ask: add the link to your page to your profile.",
		"how_2":             "Signing in shows your link. Only your username is read, and the access token is revoked right after.",
		"how_3":             "Remove the link from your profile and your page goes away within minutes.",
		"me_title":          "Your page",
		"me_active":         "Your page is active.",
		"me_inactive":       "Your page isn't active yet. Add this link to your profile, in the bio or in a profile field:",
		"me_open":           "Open my page",
		"logout":            "Sign out",
		"posts_word":        "posts",
		"tab_posts":         "Posts",
		"tab_pinned":        "Pinned",
		"tab_replies":       "With replies",
		"tab_media":         "Media only",
		"tab_public":        "Public only",
		"open":              "Open on %s",
		"older":             "Older posts",
		"empty":             "Nothing here yet.",
		"sensitive":         "Sensitive media, click to show",
		"replies_one":       "reply",
		"replies_many":      "replies",
		"boosts_one":        "boost",
		"boosts_many":       "boosts",
		"favourites_one":    "favourite",
		"favourites_many":   "favourites",
		"months":            "Jan Feb Mar Apr May Jun Jul Aug Sep Oct Nov Dec",
		"date":              "{m} {d}, {y}",
		"source":            "source code",
		"kicker":            "bolha.io · mensagens",
		"hero_a":            "Just",
		"hero_b":            "my toots",
		"part_of":           "part of",
		"credit":            "Based on JustMyToots",
		"error_account":     "There is no such account on %s.",
		"error_consent":     "This account hasn't turned on its page. If it's yours, sign in to see how.",
		"error_other":       "Only accounts of %s have a page here.",
		"error_notfound":    "Page not found.",
		"error_expired":     "The sign-in expired. Please try again.",
		"error_denied":      "Sign-in was cancelled on the instance.",
		"error_unavailable": "That didn't work. Please try again in a moment.",
	},
	"pt": {
		"lang": "pt-BR", "nav_notes": "Novidades", "notes_title": "Novidades", "notes_lede": "O que mudou em cada versão.",
		"language":          "Idioma",
		"brand":             "mensagens",
		"title":             "Mensagens",
		"lede":              "Uma página limpa com os posts de uma conta da %s, sem impulsos nem timelines. Fácil de ler e de compartilhar.",
		"login_button":      "Entrar com a sua conta",
		"how_title":         "Como funciona",
		"how_1":             "As páginas são públicas, mas só existem para quem pede: coloque o link da sua página no seu perfil.",
		"how_2":             "Ao entrar, você vê o seu link. Só lemos o seu nome de usuário, e o token de acesso é revogado logo em seguida.",
		"how_3":             "Tirou o link do perfil, sua página sai do ar em alguns minutos.",
		"me_title":          "Sua página",
		"me_active":         "Sua página está ativa.",
		"me_inactive":       "Sua página ainda não está ativa. Coloque este link no seu perfil, na bio ou em um campo do perfil:",
		"me_open":           "Abrir minha página",
		"logout":            "Sair",
		"posts_word":        "posts",
		"tab_posts":         "Posts",
		"tab_pinned":        "Pinados",
		"tab_replies":       "Com respostas",
		"tab_media":         "Só com mídia",
		"tab_public":        "Só públicos",
		"open":              "Abrir na %s",
		"older":             "Posts mais antigos",
		"empty":             "Ainda não há nada aqui.",
		"sensitive":         "Mídia sensível, clique para ver",
		"replies_one":       "resposta",
		"replies_many":      "respostas",
		"boosts_one":        "impulso",
		"boosts_many":       "impulsos",
		"favourites_one":    "favorito",
		"favourites_many":   "favoritos",
		"months":            "jan fev mar abr mai jun jul ago set out nov dez",
		"date":              "{d} {m} {y}",
		"source":            "código",
		"kicker":            "bolha.io · mensagens",
		"hero_a":            "Só os",
		"hero_b":            "meus posts",
		"part_of":           "parte da",
		"credit":            "Baseado no JustMyToots",
		"error_account":     "Essa conta não existe na %s.",
		"error_consent":     "Esta conta não ativou a página. Se ela é sua, entre para ver como.",
		"error_other":       "Só contas da %s têm página aqui.",
		"error_notfound":    "Página não encontrada.",
		"error_expired":     "O login expirou. Tente de novo.",
		"error_denied":      "O login foi cancelado na instância.",
		"error_unavailable": "Não deu certo. Tente de novo em instantes.",
	},
}

// language picks the page language: the visitor's choice, then their browser,
// then the instance default.
func language(r *http.Request, fallback string) string {
	if c, err := r.Cookie(langCookie); err == nil {
		if _, ok := translations[c.Value]; ok {
			return c.Value
		}
	}
	for part := range strings.SplitSeq(r.Header.Get("Accept-Language"), ",") {
		tag, _, _ := strings.Cut(strings.TrimSpace(part), ";")
		base, _, _ := strings.Cut(strings.ToLower(tag), "-")
		if _, ok := translations[base]; ok {
			return base
		}
	}
	return fallback
}

func strs(lang string) map[string]string {
	if t, ok := translations[lang]; ok {
		return t
	}
	return translations["en"]
}
