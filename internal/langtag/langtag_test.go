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

package langtag

import (
	"slices"
	"strings"
	"testing"
)

// newMatcher builds a Matcher or fails the test.
func newMatcher(t testing.TB, def string, others ...string) *Matcher {
	t.Helper()
	m, err := NewMatcher(def, others...)
	if err != nil {
		t.Fatalf("NewMatcher(%q, %v): %v", def, others, err)
	}
	return m
}

func norms(tags []tag) []string {
	ss := make([]string, len(tags))
	for i, t := range tags {
		ss[i] = t.norm
	}
	return ss
}

func TestParse(t *testing.T) {
	cases := []struct {
		in         string
		lang, norm string
	}{
		{"en", "en", "en"},
		{"EN", "en", "en"},
		{"zh-Hans-CN", "zh", "zh-hans-cn"},
		{"EN-us", "en", "en-us"},
		{"pt-BR", "pt", "pt-br"},
		{"es-419", "es", "es-419"},
		{"de-CH-1996", "de", "de-ch-1996"},
		{"en-x-private", "en", "en-x-private"},
		{"fil", "fil", "fil"},
		// Underscore separators are tolerated as dashes.
		{"en_US", "en", "en-us"},
		{"zh_Hans_CN", "zh", "zh-hans-cn"},
		// Unknown codes parse as-is (no registry, unlike x/text); bare
		// language names are handled only in header entries (acceptFallback).
		{"english", "english", "english"},
		{"scc", "scc", "scc"},
		// Macrolanguage bases are kept as-is; matching expands the family.
		{"bh", "bh", "bh"},
		// Legacy aliases canonicalize to their modern code.
		{"iw", "he", "he"},
		{"in-ID", "id", "id-id"},
		{"sh", "sr", "sr-latn"},
		{"sh-RS", "sr", "sr-latn-rs"},
		{"tl-PH", "fil", "fil-ph"},
		// ISO 639-2/B bibliographic codes.
		{"ger-DE", "de", "de-de"},
		{"chi", "zh", "zh"},
		{"fre", "fr", "fr"},
		{"ice-IS", "is", "is-is"},
		{"mao", "mi", "mi"},
		// ISO 639-2/T codes of top languages.
		{"zho-CN", "zh", "zh-cn"},
		{"jpn", "ja", "ja"},
		{"tgl", "fil", "fil"},
		{"tib", "bo", "bo"},
		{"bod", "bo", "bo"},
		{"nob", "nb", "nb"},
		{"nno-NO", "nn", "nn-no"},
		// Grandfathered and legacy whole tags canonicalize.
		{"i-lux", "lb", "lb"},
		{"i-klingon", "tlh", "tlh"},
		{"i-default", "en", "en"},
		{"i-mingo", "see", "see"},
		{"cel-gaulish", "xtg", "xtg"},
		{"root", "und", "und"},
		{"zh-min-nan", "nan", "nan"},
		{"zh-min", "nan", "nan"},
		{"zh-guoyu", "cmn", "cmn"},
		{"no-bok", "nb", "nb"},
		{"SGN-BE-FR", "sfb", "sfb"},
		// Grandfathered/private-use without a mapping resolve to "und".
		{"i-enochian", "und", "i-enochian"},
		{"x-klingon", "und", "x-klingon"},
		// Single-letter prefixes are case-insensitive (RFC 5646 2.1.1).
		{"I-LUX", "lb", "lb"},
		{"I-DEFAULT", "en", "en"},
		{"X-KLINGON", "und", "x-klingon"},
		{"i-LUX", "lb", "lb"},
	}
	for _, c := range cases {
		tag, err := parse(c.in)
		if err != nil {
			t.Fatalf("parse(%q): unexpected error %v", c.in, err)
		}
		if tag.lang != c.lang || tag.norm != c.norm {
			t.Fatalf("parse(%q) = (%q, %q), want (%q, %q)", c.in, tag.lang, tag.norm, c.lang, c.norm)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, in := range []string{
		"", " ", "e", "toolonglang", "123", "en--us", "-en", "en-",
		"e n", "zh-Hans-CN@", "e1-", "*", "en;-q=1", "en-us-",
		"i", "x", "x-", "I", "X", "I-", "X-",
		// Non-ASCII must not fold into a valid tag (U+212A lowercases to k
		// under strings.ToLower; toLowerASCII keeps it non-ASCII so the
		// isAlpha/isAlnum checks reject it).
		"K", "KK", "en-K", "i-Klingon",
	} {
		if _, err := parse(in); err == nil {
			t.Fatalf("parse(%q): expected error", in)
		}
	}
}

// NewMatcher validates every supported locale.
func TestNewMatcherInvalid(t *testing.T) {
	for _, supported := range [][]string{{""}, {"en", "e"}, {"en@US"}} {
		if _, err := NewMatcher(supported[0], supported[1:]...); err == nil {
			t.Fatalf("NewMatcher(%v): expected error", supported)
		}
	}
}

func TestParseAcceptLanguage(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"single", "en", []string{"en"}},
		{"case and spaces", " EN-US ; q=0.5 , zh ", []string{"zh", "en-us"}},
		{"q order", "fr;q=0.9, zh;q=0.8, en;q=0.7", []string{"fr", "zh", "en"}},
		{"ties keep header order", "de, en, es", []string{"de", "en", "es"}},
		{"q=0 dropped", "en;q=0, zh", []string{"zh"}},
		{"wildcard dropped", "*, en", []string{"en"}},
		{"malformed tag skipped", "!!-, en;q=0.5", []string{"en"}},
		{"malformed q skipped", "en;q=zz, zh", []string{"zh"}},
		{"empty q value skipped", "en;q=, zh", []string{"zh"}},
		{"bare q skipped", "en;q, zh", []string{"zh"}},
		{"q NaN skipped", "en;q=NaN, zh", []string{"zh"}},
		{"q with spaces", "en;Q = 0.9, zh;q=0.5", []string{"en", "zh"}},
		{"q-like other param ignored", "en;qlang=x, zh;q=0.5", []string{"en", "zh"}},
		{"language name", "english, de;q=0.5", []string{"en", "de"}},
		{"language name with space before q", "english ; q=0.5, de;q=0.2", []string{"en", "de"}},
		{"language name with tab before q", "english\t;q=0.5, de;q=0.2", []string{"en", "de"}},
		{"language name with trailing semicolon", "english ;", []string{"en"}},
		{"q out of range kept", "en;q=1.5, zh", []string{"en", "zh"}},
		{"other params ignored", "en;foo=bar, de;q=0.5", []string{"en", "de"}},
		{"empty entries skipped", ",,en,,", []string{"en"}},
		{"tag within length bound kept", "en" + strings.Repeat("-us", 50), []string{"en" + strings.Repeat("-us", 50)}},
		{"oversized tag skipped", "en" + strings.Repeat("-us", 120) + ", fr", []string{"fr"}},
	}
	for _, c := range cases {
		got := norms(parseAcceptLanguage(c.in))
		if !slices.Equal(got, c.want) {
			t.Fatalf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// Hostile headers are bounded: at most maxAcceptEntries entries are kept.
func TestParseAcceptLanguageEntryCap(t *testing.T) {
	s := strings.Repeat("en,", maxAcceptEntries) + "zh"
	if got := parseAcceptLanguage(s); len(got) != maxAcceptEntries {
		t.Fatalf("got %d entries, want %d", len(got), maxAcceptEntries)
	}
	// The cap counts iterations, not kept entries: malformed entries beyond
	// the cap are not scanned at all.
	if got := parseAcceptLanguage(strings.Repeat("!!,", 150) + "en"); len(got) != 0 {
		t.Fatalf("beyond-cap entries: got %d, want 0", len(got))
	}
}

func TestMatch(t *testing.T) {
	// supported: en (default), zh, zh-TW, nb
	m := newMatcher(t, "en", "zh", "zh-TW", "nb")
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"no header", "", "en"},
		{"unsupported", "de, fr", "en"},
		{"exact", "zh-TW", "zh-TW"},
		{"region falls back to language", "en-US", "en"},
		{"script falls back to language", "zh-Hans", "zh"},
		{"first registered wins same language", "zh-HK", "zh"},
		{"macro member accepted", "cmn-CN", "zh"},
		{"macro member yue", "yue", "zh"},
		{"macro accepted, member supported", "no", "nb"},
		{"grandfathered no-bok reaches member", "no-bok", "nb"},
		{"grandfathered zh-guoyu reaches macro", "zh-guoyu", "zh"},
		{"sibling members never match", "nn", "en"},
		{"quality order beats match closeness", "en, zh-TW", "en"},
		{"later accepted tag wins only if earlier miss", "fr, zh-TW", "zh-TW"},
	}
	for _, c := range cases {
		if got := m.Match(c.header); got != c.want {
			t.Fatalf("%s: Match(%q) = %q, want %q", c.name, c.header, got, c.want)
		}
	}
}

// Match always returns the registered spelling, never the header's.
func TestMatchReturnsRegisteredSpelling(t *testing.T) {
	if got := newMatcher(t, "he").Match("iw"); got != "he" {
		t.Fatalf("iw vs supported he: got %q, want he", got)
	}
	if got := newMatcher(t, "iw").Match("he-IL"); got != "iw" {
		t.Fatalf("he-IL vs supported iw: got %q, want iw", got)
	}
}

// ISO 639-2/B and /T codes match their modern canonical language.
func TestMatchISO6392Codes(t *testing.T) {
	m := newMatcher(t, "en", "de", "zh", "pt", "ja")
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"B code ger", "ger-DE", "de"},
		{"B code chi", "chi", "zh"},
		{"T code deu", "deu-AT", "de"},
		{"T code zho", "zho-TW", "zh"},
		{"T code por", "por-BR", "pt"},
		{"T code jpn", "jpn", "ja"},
		{"T code nob falls to default", "nob-NO", "en"}, // no "no"/"nb" supported
	}
	for _, c := range cases {
		if got := m.Match(c.header); got != c.want {
			t.Fatalf("%s: Match(%q) = %q, want %q", c.name, c.header, got, c.want)
		}
	}
	// The Tibetan /B and /T codes both reach a supported "bo".
	if got := newMatcher(t, "en", "bo").Match("bod"); got != "bo" {
		t.Fatalf("T code bod: got %q, want bo", got)
	}
}

// Every macrolanguage family expands in both directions. The Bihari members
// are bho/mag/mai; mwr (Marwari) belongs to the Rajasthani group, not Bihari.
func TestMatchMacroFamilies(t *testing.T) {
	family := []string{"en", "ar", "az", "fa", "ku", "ms", "sw", "uz"}
	cases := []struct {
		supported []string
		header    string
		want      string
	}{
		{family, "arb", "ar"},    // Standard Arabic
		{family, "arz-EG", "ar"}, // Egyptian Arabic
		{family, "azj", "az"},    // North Azerbaijani
		{family, "prs", "fa"},    // Dari
		{family, "kmr-TR", "ku"}, // Kurmanji
		{family, "zsm", "ms"},    // Standard Malay
		{family, "swh-KE", "sw"}, // Swahili
		{family, "uzn", "uz"},    // Northern Uzbek
		{family, "ar", "ar"},     // macro itself
		{[]string{"en", "bho"}, "bh", "bho"},
		{[]string{"en", "bh"}, "bho", "bh"},
		{[]string{"en", "mai"}, "bh", "mai"},
		{[]string{"en", "mwr"}, "bh", "en"}, // mwr is not a member
	}
	for _, c := range cases {
		m := newMatcher(t, c.supported[0], c.supported[1:]...)
		if got := m.Match(c.header); got != c.want {
			t.Fatalf("supported %v: Match(%q) = %q, want %q", c.supported, c.header, got, c.want)
		}
	}
}

// Family expansion resolves to the first registered family member, and a
// directly registered member is never shadowed by an earlier macro.
func TestMatchFamilyOrder(t *testing.T) {
	cases := []struct {
		supported []string
		header    string
		want      string
	}{
		{[]string{"en", "nn", "nb"}, "no", "nn"},
		{[]string{"en", "yue", "cmn"}, "zh", "yue"},
		{[]string{"zh", "yue"}, "yue-HK", "yue"},
	}
	for _, c := range cases {
		m := newMatcher(t, c.supported[0], c.supported[1:]...)
		if got := m.Match(c.header); got != c.want {
			t.Fatalf("supported %v: Match(%q) = %q, want %q", c.supported, c.header, got, c.want)
		}
	}
}

// Frozen tables stay internally consistent: values parse and are lowercase,
// family members round-trip through familyIndex, no alias key shadows a
// family code, and no value points back into the alias or grandfathered
// table (parse follows single hops).
func TestTableInvariants(t *testing.T) {
	for key, val := range aliases {
		if _, err := parse(val); err != nil {
			t.Fatalf("alias %q -> %q: value does not parse: %v", key, val, err)
		}
		if _, ok := familyIndex[key]; ok {
			t.Fatalf("alias key %q is also a macrolanguage or member", key)
		}
		if val != strings.ToLower(val) {
			t.Fatalf("alias %q -> %q: value is not lowercase", key, val)
		}
		if _, ok := aliases[val]; ok {
			t.Fatalf("alias %q -> %q: value is also an alias key (chain)", key, val)
		}
		if _, ok := grandfathered[val]; ok {
			t.Fatalf("alias %q -> %q: value is also a grandfathered key", key, val)
		}
	}
	for macro, members := range macroFamily {
		if _, err := parse(macro); err != nil {
			t.Fatalf("macro %q does not parse: %v", macro, err)
		}
		for _, m := range members {
			if _, err := parse(m); err != nil {
				t.Fatalf("member %q of %q does not parse: %v", m, macro, err)
			}
			if !slices.Contains(familyIndex[macro], m) {
				t.Fatalf("familyIndex[%q] missing %q", macro, m)
			}
			if !slices.Contains(familyIndex[m], macro) {
				t.Fatalf("familyIndex[%q] missing %q", m, macro)
			}
		}
	}
	for key, val := range grandfathered {
		if _, ok := grandfathered[val]; ok {
			t.Fatalf("grandfathered %q -> %q: value is also a grandfathered key (cycle)", key, val)
		}
		if val != strings.ToLower(val) {
			t.Fatalf("grandfathered %q -> %q: value is not lowercase", key, val)
		}
		want, err := parse(val)
		if err != nil {
			t.Fatalf("grandfathered %q -> %q: value does not parse: %v", key, val, err)
		}
		got, err := parse(key)
		if err != nil {
			t.Fatalf("grandfathered %q does not parse: %v", key, err)
		}
		if got != want {
			t.Fatalf("grandfathered %q: got %+v, want %+v", key, got, want)
		}
	}
}

// Duplicate supported locales keep the first registration.
func TestMatchDuplicateSupported(t *testing.T) {
	if got := newMatcher(t, "en", "en", "zh").Match("en"); got != "en" {
		t.Fatalf("duplicate supported: got %q, want en", got)
	}
}

func BenchmarkParseAcceptLanguage(b *testing.B) {
	const header = "fr-CH, fr;q=0.9, en;q=0.8, de;q=0.7, *;q=0.5"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		parseAcceptLanguage(header)
	}
}

func BenchmarkMatch(b *testing.B) {
	m := newMatcher(b, "en", "zh", "ja", "pt-BR")
	const header = "fr, zh-Hans-CN, en-US"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.Match(header)
	}
}
