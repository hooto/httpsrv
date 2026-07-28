// Copyright 2015 Eryx <evorui at gmail dot com>, All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httpsrv

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func doLocale(a App, header string) string {
	req := httptest.NewRequest(http.MethodGet, "/l", nil)
	if header != "" {
		req.Header.Set("Accept-Language", header)
	}
	rec := httptest.NewRecorder()
	a.(http.Handler).ServeHTTP(rec, req)
	return rec.Body.String()
}

// AcceptLanguage picks the best supported locale (BCP-47 via x/text), falling
// back to the default.
func TestAcceptLanguageMatch(t *testing.T) {
	a := New()
	a.Use(AcceptLanguage("en", "zh", "ja"))
	a.Get("/l", func(c Ctx) error { return c.SendString(c.Locale()) })

	cases := []struct {
		header string
		want   string
	}{
		{"en-US,en;q=0.9", "en"},
		{"zh-CN,zh;q=0.9,en;q=0.8", "zh"},
		{"ja", "ja"},
		{"de", "en"},                 // unsupported -> default
		{"", "en"},                   // no header -> default
		{"fr;q=0.9, zh;q=0.8", "zh"}, // fr unsupported, zh supported
	}
	for _, c := range cases {
		if got := doLocale(a, c.header); got != c.want {
			t.Fatalf("Accept-Language %q: locale=%q, want %q", c.header, got, c.want)
		}
	}
}

// i18n is opt-in: without WithI18n, Ctx.Translate returns the key unchanged.
func TestCtxTranslateOptIn(t *testing.T) {
	a := New() // no WithI18n
	a.Get("/", func(c Ctx) error { return c.SendString(c.Translate("en", "hi")) })
	if got := doRouteRec(a, http.MethodGet, "/").Body.String(); got != "hi" {
		t.Fatalf("without i18n: %q (want key)", got)
	}

	i := NewI18n("en")
	i.Add("en", map[string]string{"hi": "Hello"})
	a2 := New(WithI18n(i))
	a2.Get("/", func(c Ctx) error { return c.SendString(c.Translate("en", "hi")) })
	if got := doRouteRec(a2, http.MethodGet, "/").Body.String(); got != "Hello" {
		t.Fatalf("with i18n: %q", got)
	}
}

// Without the AcceptLanguage middleware, Locale() falls back to the i18n default.
func TestCtxLocaleFallbackDefault(t *testing.T) {
	a := New(WithI18n(NewI18n("zh")))
	a.Get("/", func(c Ctx) error { return c.SendString(c.Locale()) })
	if got := doRouteRec(a, http.MethodGet, "/").Body.String(); got != "zh" {
		t.Fatalf("locale fallback: %q, want zh", got)
	}
}

// LoadJSON loads flat {key: text} messages.
func TestI18nLoadJSON(t *testing.T) {
	i := NewI18n("en")
	if err := i.LoadJSON("en", []byte(`{"hi": "Hello", "bye": "Bye"}`)); err != nil {
		t.Fatal(err)
	}
	if got := i.Translate("en", "hi"); got != "Hello" {
		t.Fatalf("hi: %q", got)
	}
	if got := i.Translate("en", "bye"); got != "Bye" {
		t.Fatalf("bye: %q", got)
	}
}

// Translate falls back locale -> default -> key, and formats args via Sprintf.
func TestI18nFallback(t *testing.T) {
	i := NewI18n("en")
	i.Add("en", map[string]string{"hi": "Hello", "greet": "Hi %s"})
	if got := i.Translate("zh", "hi"); got != "Hello" { // falls back to default locale
		t.Fatalf("zh fallback: got %q", got)
	}
	if got := i.Translate("en", "missing"); got != "missing" { // key fallback
		t.Fatalf("missing key: got %q", got)
	}
	if got := i.Translate("en", "greet", "Bob"); got != "Hi Bob" { // args via fmt.Sprintf
		t.Fatalf("args: got %q", got)
	}
}
