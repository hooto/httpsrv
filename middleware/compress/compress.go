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
// Register it with Use; it compresses responses with gzip or brotli, chosen
// from the request's Accept-Encoding (brotli preferred, then gzip):
//
//	import "github.com/hooto/httpsrv/v2"
//	"github.com/hooto/httpsrv/v2/middleware/compress"
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
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/hooto/httpsrv/v2"
)

// Compression levels. They are an enum (not raw gzip/brotli levels); New maps
// each to the appropriate encoder level.
const (
	LevelDefault         = iota // 0: gzip DefaultCompression, brotli quality 5
	LevelBestSpeed              // 1: gzip BestSpeed, brotli BestSpeed
	LevelBestCompression        // 2: gzip BestCompression, brotli BestCompression
)

// Config configures the compression middleware.
type Config struct {
	// Next optionally skips this middleware when it returns true.
	Next func(c httpsrv.Ctx) bool

	// Level is the compression level: LevelDefault (default), LevelBestSpeed,
	// or LevelBestCompression. Defaults to LevelDefault.
	Level int
}

// ConfigDefault is the default configuration (LevelDefault, no Next).
var ConfigDefault = Config{
	Level: LevelDefault,
}

// New returns compression middleware. It compresses responses with brotli
// (preferred) or gzip based on Accept-Encoding.
func New(config ...Config) httpsrv.Handler {
	cfg := ConfigDefault
	if len(config) > 0 {
		cfg = config[0]
	}
	return func(c httpsrv.Ctx) (err error) {
		if cfg.Next != nil && cfg.Next(c) {
			return c.Next()
		}
		enc := pickEncoding(c.Request().Header.Get("Accept-Encoding"))
		if enc == "" {
			return c.Next()
		}
		// Wrap the response writer for the rest of the chain so downstream
		// writes are compressed. Restore the original writer and flush the
		// encoder on the way out, even if Next panics. The encoder writes to
		// the original writer captured above (not c.w), so restore-then-Close
		// is order-independent.
		orig := c.Response()
		cw := newCompressResponseWriter(orig, enc, cfg.Level)
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

// pickEncoding returns the best supported encoding from an Accept-Encoding
// header: "br" or "gzip", preferring brotli. It is a simple matcher and does
// not parse q-values (so an explicit "gzip;q=0" is not honored). Returns "" if
// neither encoding is present.
func pickEncoding(ae string) string {
	if ae == "" {
		return ""
	}
	if strings.Contains(ae, "br") {
		return "br"
	}
	if strings.Contains(ae, "gzip") {
		return "gzip"
	}
	return ""
}

// gzipLevel maps a Config level to a compress/gzip level.
func gzipLevel(level int) int {
	switch level {
	case LevelBestSpeed:
		return gzip.BestSpeed
	case LevelBestCompression:
		return gzip.BestCompression
	default:
		return gzip.DefaultCompression
	}
}

// brotliLevel maps a Config level to a brotli quality level.
func brotliLevel(level int) int {
	switch level {
	case LevelBestSpeed:
		return brotli.BestSpeed
	case LevelBestCompression:
		return brotli.BestCompression
	default:
		return 5 // quality 5: good ratio, fast
	}
}

// compressResponseWriter wraps an http.ResponseWriter, compressing written bytes
// with a gzip or brotli writer. Content-Encoding must be set (in
// newCompressResponseWriter) before the first Write/WriteHeader, which the
// middleware guarantees.
type compressResponseWriter struct {
	http.ResponseWriter
	enc  string
	encW io.WriteCloser
}

func newCompressResponseWriter(w http.ResponseWriter, enc string, level int) *compressResponseWriter {
	cw := &compressResponseWriter{ResponseWriter: w, enc: enc}
	switch enc {
	case "gzip":
		cw.encW, _ = gzip.NewWriterLevel(w, gzipLevel(level))
	case "br":
		cw.encW = brotli.NewWriterLevel(w, brotliLevel(level))
	}
	w.Header().Set("Content-Encoding", enc)
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
