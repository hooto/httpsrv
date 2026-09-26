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

// Package compress provides response-compression middleware for httpsrv
// (New(config) -> Handler).
//
// Register it with Use; it compresses responses according to the request's
// Accept-Encoding. Only gzip is built in (from the standard library); further
// encodings are opt-in via Register, so their implementations stay out of this
// module's dependencies. Brotli, for example, is wired up by the application:
//
//	import "github.com/andybalholm/brotli"
//	"github.com/hooto/httpsrv/v2/middleware/compress"
//
//	func init() {
//		compress.Register("br", func(w io.Writer) io.WriteCloser {
//			return brotli.NewWriterLevel(w, 5) // quality 5: good ratio, fast
//		})
//	}
//
//	app.Use(compress.New())
//	// or: app.Use(compress.New(compress.Config{Level: compress.LevelBestSpeed}))
//
// It wraps the response writer, sets Content-Encoding (and Vary), and removes
// any Content-Length the handler may have set. All response bodies are
// compressed; there is no content-type or minimum-size filter.
package compress

import (
	"compress/gzip"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/hooto/httpsrv/v2"
)

// Compression levels. They are an enum (not raw gzip levels); New maps each
// to the built-in gzip encoder's level. Encoders added via Register choose
// their own settings.
const (
	LevelDefault         = iota // 0: gzip DefaultCompression
	LevelBestSpeed              // 1: gzip BestSpeed
	LevelBestCompression        // 2: gzip BestCompression
)

// Encoder builds the compression writer for one Content-Encoding token: it
// wraps w and returns a writer that compresses everything written through it.
// The returned writer must be non-nil and must be Closed after use to flush
// any trailer; for streaming responses it should also implement
// Flush() error (flushing is best-effort, see compressResponseWriter).
type Encoder func(w io.Writer) io.WriteCloser

// Config configures the compression middleware.
type Config struct {
	// Next optionally skips this middleware when it returns true.
	Next func(c httpsrv.Ctx) bool

	// Level is the compression level for the built-in gzip encoder:
	// LevelDefault (default), LevelBestSpeed, or LevelBestCompression.
	// It does not apply to encoders added via Register; those carry their
	// own settings in their closures.
	Level int
}

// ConfigDefault is the default configuration (LevelDefault, no Next).
var ConfigDefault = Config{
	Level: LevelDefault,
}

// offer pairs a Content-Encoding token with its Encoder.
type offer struct {
	name string
	enc  Encoder
}

// Registered encoders. regMu guards reg against concurrent Register calls
// (setup) and the per-request negotiation in pickEncoder (serving).
var (
	regMu sync.RWMutex
	reg   []offer // registration order = preference order
)

// Register adds enc as the encoder for a Content-Encoding token, e.g. "br"
// or "zstd", replacing any encoder previously registered for that token.
// The token is lowercased (Content-Encoding tokens are case-insensitive).
// It panics on an empty token or a nil encoder, mirroring the fail-fast
// route registration.
//
// Encoders are offered in registration order, ahead of the built-in gzip;
// the client's Accept-Encoding weights pick the winner, ties going to the
// earlier offer. Registering "gzip" replaces the built-in encoder, and
// Config.Level then no longer applies. Register must run before New: each
// middleware fixes its offer list at construction.
func Register(encoding string, enc Encoder) {
	encoding = strings.ToLower(strings.TrimSpace(encoding))
	if encoding == "" || enc == nil {
		panic("compress: Register requires a non-empty encoding and a non-nil encoder")
	}
	regMu.Lock()
	defer regMu.Unlock()
	for i, o := range reg {
		if o.name == encoding {
			reg[i].enc = enc
			return
		}
	}
	reg = append(reg, offer{encoding, enc})
}

// New returns compression middleware. It compresses responses with the
// encoder the client weighs highest in Accept-Encoding among the registered
// encoders (registration order) and the built-in gzip. Without Register it
// supports gzip only.
func New(config ...Config) httpsrv.Handler {
	cfg := ConfigDefault
	if len(config) > 0 {
		cfg = config[0]
	}
	// The offer list is fixed here: Register must run before New. Keeping it
	// per-instance moves all registry work off the request path.
	offers := registeredOffers(gzipEncoder(cfg.Level))
	return func(c httpsrv.Ctx) (err error) {
		if cfg.Next != nil && cfg.Next(c) {
			return c.Next()
		}
		name, enc := negotiate(c.Request().Header.Get("Accept-Encoding"), offers)
		if enc == nil {
			return c.Next()
		}
		// Wrap the response writer for the rest of the chain so downstream
		// writes are compressed. Restore the original writer and flush the
		// encoder on the way out, even if Next panics. The encoder writes to
		// the original writer captured above (not c.w), so restore-then-Close
		// is order-independent.
		orig := c.Response()
		cw := newCompressResponseWriter(orig, name, enc)
		c.SetResponse(cw)
		defer func() {
			c.SetResponse(orig)
			if cerr := cw.Close(); err == nil {
				err = cerr
			}
		}()
		return c.Next()
	}
}

// registeredOffers snapshots the registry (registration order) and appends
// the built-in gzip encoder unless "gzip" was registered to replace it.
func registeredOffers(gz Encoder) []offer {
	regMu.RLock()
	offers := slices.Clone(reg)
	regMu.RUnlock()
	if !slices.ContainsFunc(offers, func(o offer) bool { return o.name == "gzip" }) {
		offers = append(offers, offer{name: "gzip", enc: gz})
	}
	return offers
}

// negotiate returns the offer the client weighs highest in an Accept-Encoding
// header ("", nil when the response should not be compressed): the client's
// weights pick the winner, ties going to the earlier offer.
func negotiate(ae string, offers []offer) (string, Encoder) {
	if ae == "" {
		return "", nil
	}
	name := ""
	var enc Encoder
	bestQ := 0.0
	for _, o := range offers {
		if q := quality(ae, o.name); q > bestQ {
			name, enc, bestQ = o.name, o.enc, q
		}
	}
	return name, enc
}

// quality returns the weight an Accept-Encoding header gives one coding: its
// own entry, else the wildcard's, else 0 (RFC 9110 section 12.5.3). Entries
// are comma-separated; the first entry for a coding wins, and a missing or
// malformed q-value leaves the default weight of 1.
func quality(ae, name string) float64 {
	wildcard := 0.0
	for item := range strings.SplitSeq(ae, ",") {
		token, params, _ := strings.Cut(item, ";")
		token = strings.ToLower(strings.TrimSpace(token))
		w := 1.0
		for param := range strings.SplitSeq(params, ";") {
			if v, ok := strings.CutPrefix(strings.ToLower(strings.TrimSpace(param)), "q="); ok {
				if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil && f >= 0 {
					w = f
				}
			}
		}
		switch token {
		case name:
			return w
		case "*":
			wildcard = w
		}
	}
	return wildcard
}

// gzipEncoder returns the built-in Encoder using compress/gzip, mapping a
// Config level (or any unknown level) to DefaultCompression. A gzip writer
// costs about 1 MB, so writers are pooled and recycled on Close.
func gzipEncoder(level int) Encoder {
	lvl := gzip.DefaultCompression
	switch level {
	case LevelBestSpeed:
		lvl = gzip.BestSpeed
	case LevelBestCompression:
		lvl = gzip.BestCompression
	}
	pool := sync.Pool{New: func() any {
		zw, _ := gzip.NewWriterLevel(io.Discard, lvl)
		return zw
	}}
	return func(w io.Writer) io.WriteCloser {
		zw := pool.Get().(*gzip.Writer)
		zw.Reset(w)
		return &pooledGzipWriter{Writer: zw, pool: &pool}
	}
}

// pooledGzipWriter returns its gzip.Writer to the pool on Close. The pool
// guarantees a single owner per checkout, and the middleware guarantees a
// single Close per response.
type pooledGzipWriter struct {
	*gzip.Writer
	pool *sync.Pool
}

func (w *pooledGzipWriter) Close() error {
	err := w.Writer.Close()
	w.Writer.Reset(io.Discard) // drop the response writer reference
	w.pool.Put(w.Writer)
	return err
}

// compressResponseWriter wraps an http.ResponseWriter, compressing written
// bytes with the Encoder chosen in newCompressResponseWriter. Content-Encoding
// is set there before the first Write/WriteHeader, which the middleware
// guarantees.
type compressResponseWriter struct {
	http.ResponseWriter
	encW io.WriteCloser
}

func newCompressResponseWriter(w http.ResponseWriter, name string, enc Encoder) *compressResponseWriter {
	cw := &compressResponseWriter{ResponseWriter: w, encW: enc(w)}
	w.Header().Set("Content-Encoding", name)
	w.Header().Add("Vary", "Accept-Encoding")
	w.Header().Del("Content-Length") // compressed length differs
	return cw
}

func (cw *compressResponseWriter) Write(b []byte) (int, error) {
	return cw.encW.Write(b)
}

// Flush flushes the encoder then the underlying writer, for streaming
// responses (SSE, chunked).
func (cw *compressResponseWriter) Flush() {
	if f, ok := cw.encW.(interface{ Flush() error }); ok {
		_ = f.Flush()
	}
	if f, ok := cw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (cw *compressResponseWriter) Close() error {
	return cw.encW.Close()
}
