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

// i18n — an opt-in module. Not loaded unless attached via WithI18n and/or its
// Funcs() passed to a template engine. The message structure is the simplest common
// shape: locale -> key -> text (no singular/plural distinction).

package httpsrv

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/text/language"
)

// I18n is a locale message store: locale -> key -> text. Construct with
// NewI18n, add messages with Add/Set/LoadJSON, and read with Translate. For use
// in templates, pass i.Funcs() to TemplatesFS/TemplatesDir; for use in
// handlers, attach to the App with WithI18n (then Ctx.Translate / Ctx.Locale).
//
// Locale keys are matched case-insensitively. Messages are flat key -> text
// (no plural forms). No date/number/currency/timezone formatting is included.
type I18n struct {
	def string
	mu  sync.RWMutex
	msg map[string]map[string]string // locale -> key -> text
}

// NewI18n creates an i18n store; defaultLocale is the fallback for lookups whose
// locale is empty or has no entry (defaults to "en").
func NewI18n(defaultLocale string) *I18n {
	if defaultLocale == "" {
		defaultLocale = "en"
	}
	return &I18n{def: defaultLocale, msg: map[string]map[string]string{}}
}

// Default returns the configured default locale.
func (i *I18n) Default() string { return i.def }

// Add merges flat {key: text} entries into locale.
func (i *I18n) Add(locale string, msgs map[string]string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	for k, v := range msgs {
		i.setLocked(locale, k, v)
	}
}

// Set sets a single key's text for locale.
func (i *I18n) Set(locale, key, text string) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.setLocked(locale, key, text)
}

// LoadJSON loads flat {"key": "text"} messages from JSON into locale.
func (i *I18n) LoadJSON(locale string, data []byte) error {
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	i.Add(locale, m)
	return nil
}

func (i *I18n) setLocked(locale, key, text string) {
	locale = strings.ToLower(locale)
	if i.msg[locale] == nil {
		i.msg[locale] = map[string]string{}
	}
	i.msg[locale][key] = text
}

// Translate returns the text for key in locale (locale -> default -> key
// fallback), formatted via fmt.Sprintf when args are given.
func (i *I18n) Translate(locale, key string, args ...any) string {
	text := i.lookup(locale, key)
	if len(args) == 0 {
		return text
	}
	return fmt.Sprintf(text, args...)
}

// lookup finds text for key in locale, falling back to the default locale, then
// to the key itself.
func (i *I18n) lookup(locale, key string) string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	if v := i.msg[strings.ToLower(locale)][key]; v != "" {
		return v
	}
	if lc := strings.ToLower(locale); lc != strings.ToLower(i.def) {
		if v := i.msg[strings.ToLower(i.def)][key]; v != "" {
			return v
		}
	}
	return key
}

// Funcs returns the template function (T) bound to this store, for passing to
// TemplatesFS/TemplatesDir as extraFuncs when i18n is wanted in templates.
func (i *I18n) Funcs() template.FuncMap {
	return template.FuncMap{
		"T": func(locale, key string, args ...any) string { return i.Translate(locale, key, args...) },
	}
}

// localeCtxKey is the request-context key for the locale set by AcceptLanguage.
type localeCtxKey struct{}

// AcceptLanguage returns middleware that detects the request locale from the
// Accept-Language header and stores it in the request context (readable via
// Ctx.Locale). def is the default/fallback; others are additional supported
// locales. Matching uses golang.org/x/text/language (BCP-47), so an "en-US"
// request matches a supported "en", and "zh-Hans" matches "zh".
//
// Register it with Use, e.g. app.Use(httpsrv.AcceptLanguage("en", "zh", "ja")).
func AcceptLanguage(def string, others ...string) Handler {
	supported := append([]string{def}, others...)
	tags := make([]language.Tag, len(supported))
	for i, s := range supported {
		t, err := language.Parse(s)
		if err != nil {
			panic("httpsrv: invalid locale " + s + ": " + err.Error())
		}
		tags[i] = t
	}
	matcher := language.NewMatcher(tags) // tags[0] (def) is the default
	return func(c Ctx) error {
		r := c.Request()
		accepted, _, _ := language.ParseAcceptLanguage(r.Header.Get("Accept-Language"))
		_, idx, _ := matcher.Match(accepted...)
		loc := supported[idx]
		ctx := context.WithValue(r.Context(), localeCtxKey{}, loc)
		c.(*ctxImpl).r = r.WithContext(ctx)
		return c.Next()
	}
}

// localeFromRequest returns the locale stored by AcceptLanguage, or "".
func localeFromRequest(r *http.Request) string {
	if v, ok := r.Context().Value(localeCtxKey{}).(string); ok {
		return v
	}
	return ""
}
