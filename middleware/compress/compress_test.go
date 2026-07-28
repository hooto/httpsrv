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
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andybalholm/brotli"
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

func Benchmark_brotli_encoding(b *testing.B) {
	for l := brotli.BestSpeed; l <= brotli.BestCompression; l++ {
		b.Run(fmt.Sprintf("level_%d", l), func(b *testing.B) {
			var (
				buf bytes.Buffer
				w   = brotli.NewWriterLevel(&buf, l)
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

// brotli is preferred when both are accepted.
func TestCompressBrotli(t *testing.T) {
	body := bytes.Repeat([]byte("hello httpsrv "), 100)
	a := httpsrv.New()
	a.Use(New())
	a.Get("/", func(c httpsrv.Ctx) error { return c.Send(body) })

	rec := doReq(a, "/", "gzip, br")
	if rec.Header().Get("Content-Encoding") != "br" {
		t.Fatalf("Content-Encoding=%q, want br (preferred)", rec.Header().Get("Content-Encoding"))
	}
	br := brotli.NewReader(rec.Body)
	dec, _ := io.ReadAll(br)
	if !bytes.Equal(dec, body) {
		t.Fatalf("decompressed mismatch: got %d bytes, want %d", len(dec), len(body))
	}
}

// No supported Accept-Encoding -> no compression, body untouched.
func TestCompressSkipped(t *testing.T) {
	body := []byte("plain")
	a := httpsrv.New()
	a.Use(New())
	a.Get("/", func(c httpsrv.Ctx) error { return c.Send(body) })

	rec := doReq(a, "/", "")
	if got := rec.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding=%q, want none", got)
	}
	if !bytes.Equal(rec.Body.Bytes(), body) {
		t.Fatalf("body=%q, want %q", rec.Body.String(), body)
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
