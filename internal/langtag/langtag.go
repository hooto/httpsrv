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

// Package langtag implements the small subset of BCP 47 (RFC 5646) needed to
// negotiate an Accept-Language header against a list of supported locales. It
// carries no CLDR data: no registry validation, no likely-subtag inference,
// no script or region preference beyond the rules below. The data tables it
// does keep (aliases, macroFamily) are frozen standards, so the package
// needs no periodic updates.
//
// A Matcher is built with NewMatcher from the supported locale strings and
// answers Match(header) with the best supported locale:
//   - Exact tag: case-insensitive, after ISO 639 alias canonicalization
//     (e.g. "ger" and "deu" -> "de").
//   - Same primary language: an accepted "en-US" matches a supported "en".
//   - Same macrolanguage family: an accepted "nb" matches a supported "no",
//     an accepted "yue" a supported "zh", an accepted "prs" a supported
//     "fa". Two members of one family never match each other (yue does not
//     match cmn).
//
// Header entries are tried by descending quality; the first entry that
// matches at any level wins, so quality order beats match closeness. Among
// supported locales sharing a language, the first registered wins. When
// nothing matches, Match returns the default (the first supported locale).
// The returned string is always the registered spelling, not the header's.
//
// Canonicalization mirrors golang.org/x/text: underscore separators read as
// dashes, legacy and grandfathered tags map to their modern base language,
// ISO 639 alias codes and bare language names canonicalize, and
// macrolanguage families match in both directions. Deliberate divergences,
// in the name of staying small: unknown codes are accepted rather than
// registry-validated; a header entry x/text would reject is skipped instead
// of invalidating the whole header, and a single entry's tag is
// length-bounded where x/text bounds the whole header by dash count; an
// extlang subtag is not promoted to the primary language (zh-yue stays zh,
// where x/text yields yue); bare language names match case-insensitively;
// a q of NaN is rejected as malformed (x/text keeps it, leaning on sort
// behavior that is undefined for NaN); with no CLDR inference, among
// locales sharing a language the first registered wins and an unmatched
// header falls back to the default; and "*" selects the default (x/text
// maps it to "mul", the same outcome in practice).
package langtag

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

// errSyntax reports a malformed language tag.
var errSyntax = errors.New("langtag: invalid language tag")

// tag is a normalized tag: an alias-canonicalized primary language and the
// lowercased full tag.
type tag struct {
	lang string // alias-canonicalized primary language
	norm string // lowercased full tag
}

// aliases maps deprecated or alternate language codes to their modern BCP 47
// canonical form. Three frozen sources, all keyed by lowercase code:
//   - codes withdrawn from ISO 639-1 (in, iw, ji, jw, mo, sh, tl)
//   - divergent ISO 639-2/B bibliographic codes (ger -> de, chi -> zh, ...)
//   - a curated subset of ISO 639-2/T (and matching 639-3) codes for
//     commonly used languages (deu, zho, fra, ...) so a "jpn" or "por"
//     header entry matches a supported "ja" or "pt"
//
// The table is frozen and not exhaustive: a missing code simply behaves
// like any other unsupported language.
var aliases = map[string]string{
	"alb": "sq", "aka": "ak", "amh": "am", "ara": "ar", "arm": "hy",
	"asm": "as", "aze": "az", "bam": "bm", "baq": "eu", "bel": "be",
	"ben": "bn", "bos": "bs", "bul": "bg", "bur": "my",
	"cat": "ca", "ces": "cs", "chi": "zh", "cym": "cy", "cze": "cs",
	"dan": "da", "deu": "de", "dut": "nl", "dzo": "dz",
	"ell": "el", "eng": "en", "epo": "eo", "est": "et",
	"eus": "eu", "fas": "fa", "fij": "fj", "fin": "fi", "fra": "fr",
	"fre": "fr", "ful": "ff", "ger": "de", "geo": "ka",
	"gla": "gd", "gle": "ga", "glg": "gl", "grn": "gn", "gre": "el",
	"guj": "gu", "hat": "ht", "hau": "ha", "ice": "is", "heb": "he",
	"hin": "hi", "hrv": "hr", "hun": "hu", "hye": "hy", "ibo": "ig",
	"in": "id", "ind": "id", "isl": "is", "ita": "it",
	"iw": "he", "jav": "jv", "ji": "yi", "jpn": "ja", "jw": "jv",
	"kan": "kn", "kat": "ka", "kaz": "kk", "khm": "km", "kin": "rw",
	"kir": "ky", "kor": "ko", "kur": "ku", "lao": "lo", "lav": "lv",
	"lin": "ln", "lit": "lt", "ltz": "lb", "mac": "mk", "mal": "ml",
	"mao": "mi", "mar": "mr", "may": "ms", "mkd": "mk", "mlg": "mg",
	"mlt": "mt", "mo": "ro", "mol": "ro", "mon": "mn", "mri": "mi",
	"msa": "ms", "mya": "my", "nep": "ne", "nld": "nl", "nno": "nn",
	"nob": "nb", "nor": "no", "nya": "ny", "ori": "or", "orm": "om",
	"pan": "pa", "per": "fa", "pol": "pl", "por": "pt", "pus": "ps",
	"que": "qu", "ron": "ro", "run": "rn", "rum": "ro", "rus": "ru",
	"sh": "sr-latn", "sin": "si", "slk": "sk",
	"slv": "sl", "slo": "sk", "smo": "sm", "sna": "sn", "snd": "sd",
	"som": "so", "sot": "st", "spa": "es", "sqi": "sq", "srp": "sr",
	"swa": "sw", "swe": "sv", "tam": "ta", "tel": "te", "tgk": "tg",
	"tgl": "fil", "tha": "th", "tib": "bo", "bod": "bo", "tir": "ti",
	"tl":  "fil",
	"ton": "to", "tsn": "tn", "tso": "ts", "tur": "tr", "uig": "ug",
	"ukr": "uk", "urd": "ur", "uzb": "uz", "ven": "ve", "vie": "vi",
	"wel": "cy", "wol": "wo", "xho": "xh", "yid": "yi", "yor": "yo",
	"zho": "zh", "zul": "zu",
}

// acceptFallback maps bare language names occasionally sent by misconfigured
// clients to their language code - the same set x/text applies, and likewise
// consulted for header entries only (a registered locale "english" stays an
// unknown code).
var acceptFallback = map[string]string{
	"deutsch": "de", "english": "en", "french": "fr", "italian": "it",
}

// macroFamily maps a macrolanguage to its member languages, the practically
// relevant frozen subset; "bh" is the Bihari collection of ISO 639-2/3, not
// a CLDR macrolanguage. Matching expands both ways: a member matches its
// macrolanguage and vice versa. Locales with their own CLDR data (e.g. ckb
// Sorani) are deliberately not folded in.
var macroFamily = map[string][]string{
	"ar": {"arb", "apc", "arq", "ary", "arz"},                      // Arabic
	"az": {"azj", "azb"},                                           // Azerbaijani
	"bh": {"bho", "mag", "mai"},                                    // Bihari
	"fa": {"pes", "prs"},                                           // Persian
	"iu": {"ike", "ikt"},                                           // Inuktitut
	"ku": {"kmr", "sdh"},                                           // Kurdish
	"mg": {"plt"},                                                  // Malagasy
	"mn": {"khk"},                                                  // Mongolian
	"ms": {"zsm"},                                                  // Malay
	"no": {"nb", "nn"},                                             // Norwegian
	"ps": {"pbu"},                                                  // Pashto
	"sw": {"swh"},                                                  // Swahili
	"uz": {"uzn", "uzs"},                                           // Uzbek
	"zh": {"cmn", "yue", "wuu", "hak", "nan", "gan", "hsn", "lzh"}, // Chinese
}

// grandfathered maps complete grandfathered and legacy tags (RFC 5646
// section 2.2.8, plus the retired i- prefix registry) to their modern base
// language - every x/text entry that canonicalizes. The rest already parse
// to the right base (en-GB-oed, en-US-POSIX) or resolve to "und" (other
// i-/x- tags), so none are listed.
var grandfathered = map[string]string{
	"art-lojban":  "jbo",
	"cel-gaulish": "xtg",
	"i-ami":       "ami",
	"i-bnn":       "bnn",
	"i-default":   "en",
	"i-hak":       "hak",
	"i-klingon":   "tlh",
	"i-lux":       "lb",
	"i-mingo":     "see",
	"i-navajo":    "nv",
	"i-pwn":       "pwn",
	"i-tao":       "tao",
	"i-tay":       "tay",
	"i-tsu":       "tsu",
	"no-bok":      "nb",
	"no-nyn":      "nn",
	"root":        "und",
	"sgn-be-fr":   "sfb",
	"sgn-be-nl":   "vgt",
	"sgn-ch-de":   "sgg",
	"zh-guoyu":    "cmn",
	"zh-hakka":    "hak",
	"zh-min":      "nan",
	"zh-min-nan":  "nan",
	"zh-xiang":    "hsn",
}

// familyIndex inverts macroFamily for lookup from either side: the family of
// a member lists its macrolanguage, the family of a macrolanguage lists its
// members.
var familyIndex = func() map[string][]string {
	idx := make(map[string][]string)
	for macro, members := range macroFamily {
		idx[macro] = append(idx[macro], members...)
		for _, mem := range members {
			idx[mem] = append(idx[mem], macro)
		}
	}
	return idx
}()

// parse parses a single language tag such as "en" or "zh-Hans-CN". Only the
// RFC 5646 shape is checked: a 2-8 letter primary language followed by
// alphanumeric subtags of 1-8 characters each. Underscores are treated as
// subtag separators ("en_US" reads as "en-US"), tolerating a common client
// quirk. Alternate and deprecated ISO 639 codes canonicalize via aliases;
// unknown codes are accepted as-is, since no registry lookup is performed.
func parse(s string) (tag, error) {
	// Lowercased once, up front: every comparison below is case-insensitive
	// by construction.
	norm := toLowerASCII(strings.ReplaceAll(strings.TrimSpace(s), "_", "-"))
	// Grandfathered and legacy whole tags map to their modern base language;
	// replacements are plain codes, so one hop suffices (TestTableInvariants
	// keeps values from pointing back into the table).
	if repl, ok := grandfathered[norm]; ok {
		norm = repl
	}
	if strings.HasSuffix(norm, "-") {
		return tag{}, errSyntax // trailing dash: empty subtag
	}
	lang, rest, _ := strings.Cut(norm, "-")
	// A single "i" or "x" also parses as the grandfathered / private-use
	// prefix when subtags follow (i-lux, x-klingon), as in x/text.
	switch {
	case len(lang) >= 2 && len(lang) <= 8 && isAlpha(lang):
	case (lang == "i" || lang == "x") && rest != "":
	default:
		return tag{}, errSyntax
	}
	for r := rest; r != ""; {
		var sub string
		sub, r, _ = strings.Cut(r, "-")
		if len(sub) > 8 || !isAlnum(sub) {
			return tag{}, errSyntax
		}
	}
	if lang == "i" || lang == "x" {
		lang = "und" // grandfathered/private-use prefix without a mapping
	}
	if repl, ok := aliases[lang]; ok {
		// Alias values are lowercase and may carry a script ("sh" -> "sr-latn").
		if rest != "" {
			norm = repl + "-" + rest
		} else {
			norm = repl
		}
		lang, _, _ = strings.Cut(repl, "-")
	}
	return tag{lang: lang, norm: norm}, nil
}

// toLowerASCII lowercases ASCII letters only. Unlike strings.ToLower, it
// cannot fold a non-ASCII rune into an ASCII one (U+212A KELVIN SIGN -> k),
// so a non-ASCII tag never slips past the isAlpha/isAlnum checks.
func toLowerASCII(s string) string {
	for i := 0; i < len(s); i++ {
		if b := s[i]; b >= 'A' && b <= 'Z' {
			buf := []byte(s)
			for ; i < len(buf); i++ {
				if c := buf[i]; c >= 'A' && c <= 'Z' {
					buf[i] = c + ('a' - 'A')
				}
			}
			return string(buf)
		}
	}
	return s
}

func isAlpha(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if b := s[i]; (b < 'a' || b > 'z') && (b < 'A' || b > 'Z') {
			return false
		}
	}
	return true
}

func isAlnum(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch b := s[i]; {
		case b >= '0' && b <= '9', b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z':
		default:
			return false
		}
	}
	return true
}

// maxAcceptEntries bounds how many header entries are considered, capping
// work on hostile headers. Realistic headers hold well under 50 entries.
const maxAcceptEntries = 100

// maxTagLen bounds a single entry's tag length (before any ';'), capping
// parse's per-entry copy cost; realistic tags are well under 40 bytes.
const maxTagLen = 256

// entry is one Accept-Language header entry with its quality value.
type entry struct {
	tag tag
	q   float32
}

// parseAcceptLanguage parses an Accept-Language header value (RFC 9110,
// section 10.3.4) into tags ordered by descending quality; equal quality
// keeps header order. An entry is skipped when its tag or q is malformed or
// over maxTagLen, and dropped at q=0. "*" is dropped too: it could only
// select the default, which Match already returns when nothing matches.
func parseAcceptLanguage(s string) []tag {
	entries := parseEntries(s)
	tags := make([]tag, len(entries))
	for i, e := range entries {
		tags[i] = e.tag
	}
	return tags
}

// parseEntries is the core of parseAcceptLanguage, kept separate so Match
// can consume the sorted entries without copying them into a []tag.
func parseEntries(s string) []entry {
	entries := make([]entry, 0, min(strings.Count(s, ",")+1, maxAcceptEntries))
	for n := 0; s != "" && n < maxAcceptEntries; n++ {
		var item string
		item, s, _ = strings.Cut(s, ",")
		item = strings.TrimSpace(item)
		if item == "" || item == "*" {
			continue
		}
		item, params, _ := strings.Cut(item, ";")
		item = strings.TrimSpace(item)
		if len(item) > maxTagLen {
			continue
		}
		q, valid := quality(params)
		if !valid {
			continue
		}
		// Lowercased once for both the bare-name lookup and parse (then
		// zero-alloc). Bare names apply to header entries only.
		item = toLowerASCII(item)
		if repl, ok := acceptFallback[item]; ok {
			item = repl
		}
		t, err := parse(item)
		if err != nil {
			continue
		}
		entries = append(entries, entry{t, q})
	}
	slices.SortStableFunc(entries, func(a, b entry) int {
		return cmp.Compare(b.q, a.q) // descending; ties keep header order
	})
	return entries
}

// quality extracts the "q" parameter from the ';'-separated parameters
// following a tag: a missing q means 1, other parameters are ignored, and
// an unparsable q reports ok=false so the caller skips the entry. NaN is
// rejected too (x/text keeps it, leaning on sort behavior undefined for
// NaN); other out-of-range values are kept and sorted as-is, as in x/text.
func quality(params string) (float32, bool) {
	for params != "" {
		var seg string
		seg, params, _ = strings.Cut(params, ";")
		seg = strings.TrimSpace(seg)
		name, val, hasVal := strings.Cut(seg, "=")
		if name = strings.TrimSpace(name); !strings.EqualFold(name, "q") {
			continue // not a q parameter: ignored
		}
		if !hasVal {
			return 0, false // bare "q" without a value
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(val), 32)
		if err != nil || v <= 0 || math.IsNaN(v) {
			return 0, false
		}
		return float32(v), true
	}
	return 1, true
}

// A Matcher selects the best supported locale for Accept-Language headers.
// It is immutable after construction and safe for concurrent use.
type Matcher struct {
	supported []string       // registered locale strings, [0] is the default
	byExact   map[string]int // normalized tag -> first supporting index
	byLang    map[string]int // language -> first supporting index
}

// NewMatcher builds a Matcher over the supported locales; def is the default
// returned when nothing matches. Every string must be a well-formed language
// tag (parse reports an error otherwise); among locales sharing a language,
// the first registered wins.
func NewMatcher(def string, others ...string) (*Matcher, error) {
	m := &Matcher{
		supported: append([]string{def}, others...),
		byExact:   make(map[string]int, len(others)+1),
		byLang:    make(map[string]int, len(others)+1),
	}
	tags := make([]tag, len(m.supported))
	for i, s := range m.supported {
		t, err := parse(s)
		if err != nil {
			return nil, fmt.Errorf("invalid locale %q: %w", s, err)
		}
		tags[i] = t
		if _, ok := m.byExact[t.norm]; !ok {
			m.byExact[t.norm] = i
		}
		if _, ok := m.byLang[t.lang]; !ok {
			m.byLang[t.lang] = i
		}
	}
	// Fold macrolanguage families into byLang after the direct entries: a
	// family code resolves to the first registered locale of that family,
	// and a directly registered member is never shadowed by an earlier
	// macro.
	for i, t := range tags {
		for _, f := range familyIndex[t.lang] {
			if _, ok := m.byLang[f]; !ok {
				m.byLang[f] = i
			}
		}
	}
	return m, nil
}

// Match returns the supported locale best matching the Accept-Language
// header value, or the default when nothing matches (including an empty or
// fully malformed header).
func (m *Matcher) Match(header string) string {
	for _, e := range parseEntries(header) {
		if i, ok := m.byExact[e.tag.norm]; ok {
			return m.supported[i]
		}
		if i, ok := m.byLang[e.tag.lang]; ok {
			return m.supported[i]
		}
	}
	return m.supported[0]
}
