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

package compress

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hooto/httpsrv/v2"
)

var compressTestText = []byte(`<!DOCTYPE html>
<html>
<head>
<title>Error</title>
<style>
html { color-scheme: light dark; }
body { width: 35em; margin: 0 auto;
font-family: Tahoma, Verdana, Arial, sans-serif; }
</style>
</head>
<body>
<h1>An error occurred.</h1>
<p>Sorry, the page you are looking for is currently unavailable.<br/>
Please try again later.</p>
<p>If you are the system administrator of this resource then you should check
the error log for details.</p>
<p><em>Faithfully yours, nginx.</em></p>
</body>
</html>`)

func Benchmark_gzip_encoding(b *testing.B) {
	for l := gzip.BestSpeed; l <= gzip.BestCompression; l++ {
		b.Run(fmt.Sprintf("level_%d", l), func(b *testing.B) {
			var (
				buf  bytes.Buffer
				w, _ = gzip.NewWriterLevel(&buf, l)
			)
			defer w.Close()

			for i := 0; i < b.N; i++ {
				buf.Reset()
				w.Reset(&buf)

				w.Write(compressTestText)
				w.Flush()
			}
		})
	}
}

// The built-in encoder recycles pooled writers: steady-state compression
// should allocate almost nothing per response.
func Benchmark_gzip_encoder_pooled(b *testing.B) {
	enc := gzipEncoder(LevelDefault)
	for b.Loop() {
		cw := enc(io.Discard)
		cw.Write(compressTestText)
		cw.Close()
	}
}

// doReq dispatches a request through the app with the given Accept-Encoding and
// records the response.
func doReq(a httpsrv.App, target, acceptEnc string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if acceptEnc != "" {
		req.Header.Set("Accept-Encoding", acceptEnc)
	}
	rec := httptest.NewRecorder()
	a.(http.Handler).ServeHTTP(rec, req)
	return rec
}

// resetRegistry clears registered encoders (tests share the global registry).
func resetRegistry() {
	regMu.Lock()
	reg = nil
	regMu.Unlock()
}

// isolateRegistry clears the registry for the current test and restores it
// afterwards.
func isolateRegistry(t *testing.T) {
	resetRegistry()
	t.Cleanup(resetRegistry)
}

// dummy returns an Encoder for negotiation tests, where the writer itself is
// never invoked.
func dummy() Encoder {
	return func(w io.Writer) io.WriteCloser { return nil }
}

// quality parses q-values: explicit entry, wildcard fallback, malformed
// entries, refusals (q=0).
func TestQuality(t *testing.T) {
	tests := []struct {
		accept string
		name   string
		want   float64
	}{
		{"", "gzip", 0},
		{"gzip", "gzip", 1},
		{"GZIP", "gzip", 1}, // tokens are case-insensitive
		{"gzip, deflate", "gzip", 1},
		{"gzip, deflate", "br", 0}, // unlisted coding, no wildcard
		{"gzip;q=0.5", "gzip", 0.5},
		{"gzip; q=0.50", "gzip", 0.5},
		{"gzip;q=0", "gzip", 0},
		{"gzip;q=-1", "gzip", 1}, // negative q is malformed: default 1
		{"gzip;q=abc", "gzip", 1},
		{"gzip;q=NaN", "gzip", 1},
		{"*", "gzip", 1},
		{"*;q=0", "gzip", 0},
		{"deflate, *;q=0.5", "gzip", 0.5},
		{"gzip;q=0.2, gzip;q=0.8", "gzip", 0.2}, // first entry wins
		{"identity", "gzip", 0},
	}
	for _, tt := range tests {
		if got := quality(tt.accept, tt.name); got != tt.want {
			t.Errorf("quality(%q, %q) = %v, want %v", tt.accept, tt.name, got, tt.want)
		}
	}
}

// negotiate honors client weights, registration order, and refusals (q=0).
func TestNegotiate(t *testing.T) {
	offers := []offer{
		{"br", dummy()},
		{"deflate", dummy()},
		{"gzip", gzipEncoder(gzip.DefaultCompression)},
	}

	tests := []struct {
		accept string
		want   string
	}{
		{"", ""}, // no header: no compression
		{"gzip", "gzip"},
		{"GZIP", "gzip"}, // case-insensitive
		{"br", "br"},
		{"gzip, br", "br"},         // tie: earlier offer wins
		{"deflate, br", "br"},      // tie: offer order (br first)
		{"br;q=0.1, gzip", "gzip"}, // client weights beat offer order
		{"br;q=0, gzip", "gzip"},   // refused coding is not offered
		{"br;q=0", ""},             // ...and nothing else is accepted
		{"*", "br"},                // wildcard: first offer
		{"zstd", ""},               // unknown coding, no wildcard
	}
	for _, tt := range tests {
		name, enc := negotiate(tt.accept, offers)
		if name != tt.want {
			t.Errorf("negotiate(%q) = %q, want %q", tt.accept, name, tt.want)
		}
		if (enc == nil) != (tt.want == "") {
			t.Errorf("negotiate(%q) encoder = %v, want nil iff name is empty", tt.accept, enc)
		}
	}
}

// Register fails fast on an empty token or a nil encoder.
func TestRegisterPanics(t *testing.T) {
	for _, tt := range []struct {
		desc     string
		encoding string
		enc      Encoder
	}{
		{"empty encoding", "", dummy()},
		{"blank encoding", "  ", dummy()},
		{"nil encoder", "br", nil},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: Register did not panic", tt.desc)
				}
			}()
			Register(tt.encoding, tt.enc)
		}()
	}
}

// Registering "gzip" replaces the built-in encoder in the offer list
// (Config.Level no longer applies to it).
func TestRegisterGzipOverride(t *testing.T) {
	isolateRegistry(t)

	calls := 0
	Register("gzip", func(w io.Writer) io.WriteCloser {
		calls++
		zw, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
		return zw
	})

	offers := registeredOffers(gzipEncoder(gzip.DefaultCompression))
	if len(offers) != 1 || offers[0].name != "gzip" {
		t.Fatalf("offers = %v, want exactly one gzip offer", offers)
	}
	_ = offers[0].enc(io.Discard)
	if calls != 1 {
		t.Fatalf("custom gzip encoder calls = %d, want 1", calls)
	}
}

// gzip round-trips through the middleware end-to-end.
func TestCompressGzip(t *testing.T) {
	body := bytes.Repeat([]byte("hello httpsrv "), 100)
	a := httpsrv.New()
	a.Use(New())
	a.Get("/", func(c httpsrv.Ctx) error { return c.Send(body) })

	rec := doReq(a, "/", "gzip")
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Content-Encoding=%q, want gzip", rec.Header().Get("Content-Encoding"))
	}
	if rec.Header().Get("Vary") != "Accept-Encoding" {
		t.Fatalf("Vary=%q, want Accept-Encoding", rec.Header().Get("Vary"))
	}
	gr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	dec, _ := io.ReadAll(gr)
	if !bytes.Equal(dec, body) {
		t.Fatalf("decompressed mismatch: got %d bytes, want %d", len(dec), len(body))
	}
}

// An encoder added via Register is preferred over the built-in gzip and
// round-trips end-to-end (flate stands in for an external codec such as
// brotli, which is no longer a dependency of this module).
func TestCompressRegistered(t *testing.T) {
	isolateRegistry(t)
	Register("deflate", func(w io.Writer) io.WriteCloser {
		fw, _ := flate.NewWriter(w, flate.DefaultCompression)
		return fw
	})

	body := bytes.Repeat([]byte("hello httpsrv "), 100)
	a := httpsrv.New()
	a.Use(New())
	a.Get("/", func(c httpsrv.Ctx) error { return c.Send(body) })

	rec := doReq(a, "/", "gzip, deflate")
	if rec.Header().Get("Content-Encoding") != "deflate" {
		t.Fatalf("Content-Encoding=%q, want deflate (registered preferred)", rec.Header().Get("Content-Encoding"))
	}
	fr := flate.NewReader(rec.Body)
	defer fr.Close()
	dec, _ := io.ReadAll(fr)
	if !bytes.Equal(dec, body) {
		t.Fatalf("decompressed mismatch: got %d bytes, want %d", len(dec), len(body))
	}
}

// No acceptable Accept-Encoding -> no compression, body untouched: no header
// at all, or the coding explicitly refused (q=0).
func TestCompressNotNegotiable(t *testing.T) {
	body := []byte("plain")
	a := httpsrv.New()
	a.Use(New())
	a.Get("/", func(c httpsrv.Ctx) error { return c.Send(body) })

	for _, ae := range []string{"", "gzip;q=0"} {
		rec := doReq(a, "/", ae)
		if got := rec.Header().Get("Content-Encoding"); got != "" {
			t.Errorf("Accept-Encoding %q: Content-Encoding=%q, want none", ae, got)
		}
		if !bytes.Equal(rec.Body.Bytes(), body) {
			t.Errorf("Accept-Encoding %q: body=%q, want %q", ae, rec.Body.String(), body)
		}
	}
}

// The Level config is honored (best-compression still round-trips correctly).
func TestCompressLevel(t *testing.T) {
	body := bytes.Repeat([]byte("hello httpsrv "), 100)
	a := httpsrv.New()
	a.Use(New(Config{Level: LevelBestCompression}))
	a.Get("/", func(c httpsrv.Ctx) error { return c.Send(body) })

	rec := doReq(a, "/", "gzip")
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Content-Encoding=%q, want gzip", rec.Header().Get("Content-Encoding"))
	}
	gr, _ := gzip.NewReader(rec.Body)
	dec, _ := io.ReadAll(gr)
	if !bytes.Equal(dec, body) {
		t.Fatalf("decompressed mismatch: got %d bytes, want %d", len(dec), len(body))
	}
}

// Next skips compression for selected requests.
func TestCompressNext(t *testing.T) {
	body := []byte("plain")
	a := httpsrv.New()
	a.Use(New(Config{
		Next: func(c httpsrv.Ctx) bool { return c.Path() == "/skip" },
	}))
	a.Get("/{*p}", func(c httpsrv.Ctx) error { return c.Send(body) })

	if rec := doReq(a, "/skip", "gzip"); rec.Header().Get("Content-Encoding") != "" {
		t.Fatalf("/skip should not be compressed: Content-Encoding=%q", rec.Header().Get("Content-Encoding"))
	}
	if rec := doReq(a, "/ok", "gzip"); rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("/ok should be compressed: Content-Encoding=%q", rec.Header().Get("Content-Encoding"))
	}
}
