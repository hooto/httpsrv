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

package static

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/hooto/httpsrv/v2"
)

// mustWriteFile writes content to path, creating parent dirs; fails the test on error.
func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// doRouteRec dispatches a request through the app and records the response.
func doRouteRec(a httpsrv.App, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, nil)
	a.(http.Handler).ServeHTTP(rec, req)
	return rec
}

// ---------------------------------------------------------------------------
// Unit tests: serve directly (no router/handler layer)
// ---------------------------------------------------------------------------

// dirServer builds a fileServer over a temp directory for unit testing serve.
func dirServer(t *testing.T) (*fileServer, string) {
	t.Helper()
	dir := t.TempDir()
	return &fileServer{fs: http.Dir(dir)}, dir
}

// serveRec invokes serve with sub as the catch-all sub-path (the value the
// router would set via r.PathValue("*")).
func serveRec(t *testing.T, s *fileServer, sub string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/"+sub, nil)
	rec := httptest.NewRecorder()
	s.serve(rec, r, sub)
	return rec
}

func TestServeDir(t *testing.T) {
	s, dir := dirServer(t)
	mustWriteFile(t, filepath.Join(dir, "hello.txt"), "hello world")
	mustWriteFile(t, filepath.Join(dir, "css", "main.css"), "body{}")

	if rec := serveRec(t, s, "hello.txt"); rec.Code != 200 || rec.Body.String() != "hello world" {
		t.Fatalf("hello.txt: code=%d body=%q", rec.Code, rec.Body.String())
	}
	if rec := serveRec(t, s, "css/main.css"); rec.Code != 200 || rec.Body.String() != "body{}" {
		t.Fatalf("css/main.css: code=%d body=%q", rec.Code, rec.Body.String())
	}
	if rec := serveRec(t, s, "missing.txt"); rec.Code != 404 {
		t.Fatalf("missing: code=%d, want 404", rec.Code)
	}
	if rec := serveRec(t, s, "css"); rec.Code != 404 { // directory -> no listing
		t.Fatalf("dir: code=%d, want 404", rec.Code)
	}
}

func TestServeFS(t *testing.T) {
	mapFS := fstest.MapFS{
		"assets/a.txt": {Data: []byte("from-fs"), Mode: 0o644},
	}
	s := &fileServer{fs: http.FS(mapFS)}
	if rec := serveRec(t, s, "assets/a.txt"); rec.Code != 200 || rec.Body.String() != "from-fs" {
		t.Fatalf("fs serve: code=%d body=%q", rec.Code, rec.Body.String())
	}
}

// Traversal beyond the root is blocked (http.Dir cleans and confines the path).
func TestServeTraversal(t *testing.T) {
	base := t.TempDir()
	mustWriteFile(t, filepath.Join(base, "secret.txt"), "top-secret")
	mustWriteFile(t, filepath.Join(base, "public", "ok.txt"), "ok")

	s := &fileServer{fs: http.Dir(filepath.Join(base, "public"))}
	if rec := serveRec(t, s, "ok.txt"); rec.Code != 200 || rec.Body.String() != "ok" {
		t.Fatalf("ok.txt: code=%d body=%q", rec.Code, rec.Body.String())
	}
	if rec := serveRec(t, s, "../secret.txt"); rec.Code != 404 {
		t.Fatalf("traversal: code=%d, want 404", rec.Code)
	}
}

// An empty catch-all value (e.g. a request for the bare prefix "/") resolves to
// the root, which is a directory and thus 404.
func TestServeEmptyCatchAll(t *testing.T) {
	s, _ := dirServer(t)
	if rec := serveRec(t, s, ""); rec.Code != 404 {
		t.Fatalf("empty catch-all: code=%d, want 404", rec.Code)
	}
}

// New returns non-nil handlers for both the directory and Config.FS forms.
func TestNew(t *testing.T) {
	if New(".") == nil {
		t.Fatal("New(\".\") returned nil")
	}
	if New(".", Config{FS: fstest.MapFS{}}) == nil {
		t.Fatal("New with Config.FS returned nil")
	}
}

// ---------------------------------------------------------------------------
// Integration tests: end-to-end through the httpsrv router
// ---------------------------------------------------------------------------

// An explicit route takes priority over a static catch-all at the same prefix.
func TestStaticExplicitWins(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "x.txt"), "from-file")
	mustWriteFile(t, filepath.Join(dir, "y.txt"), "from-file-2")

	a := httpsrv.New()
	a.Get("/static/x.txt", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "from-route") }))
	a.All("/static/{*path}", New(dir))

	if rec := doRouteRec(a, http.MethodGet, "/static/x.txt"); rec.Body.String() != "from-route" {
		t.Fatalf("explicit route should win: %q", rec.Body.String())
	}
	if rec := doRouteRec(a, http.MethodGet, "/static/y.txt"); rec.Body.String() != "from-file-2" {
		t.Fatalf("other files still served: %q", rec.Body.String())
	}
}

// Prefix matching is segment-aware: /static does not catch /staticfoo.
func TestStaticPrefixBoundary(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "a.txt"), "A")

	a := httpsrv.New()
	a.All("/static/{*path}", New(dir))

	if rec := doRouteRec(a, http.MethodGet, "/static/a.txt"); rec.Code != 200 {
		t.Fatalf("/static/a.txt: code=%d", rec.Code)
	}
	if rec := doRouteRec(a, http.MethodGet, "/staticfoo/a.txt"); rec.Code != 404 {
		t.Fatalf("/staticfoo/a.txt: code=%d, want 404 (segment boundary)", rec.Code)
	}
}

// embedStaticFS is a real embed.FS fixture for the Config.FS tests.
//
//go:embed testdata/embed
var embedStaticFS embed.FS

// embedRoot returns the embedded test fixture as an fs.FS.
func embedRoot(t *testing.T) fs.FS {
	t.Helper()
	root, err := fs.Sub(embedStaticFS, "testdata/embed")
	if err != nil {
		t.Fatalf("fs.Sub: %v", err)
	}
	return root
}

// New serves files from a Config.FS (io/fs), e.g. an embed.FS. root is a
// sub-path within the FS; "." or "" means the whole FS.
func TestStaticNewConfigFS(t *testing.T) {
	a := httpsrv.New()
	a.All("/e/{*path}", New(".", Config{FS: embedRoot(t)}))

	cases := []struct {
		target string
		body   string
	}{
		{"/e/hello.txt", "embed-hello\n"},
		{"/e/sub/nested.txt", "nested\n"},
	}
	for _, c := range cases {
		rec := doRouteRec(a, http.MethodGet, c.target)
		if rec.Code != 200 || rec.Body.String() != c.body {
			t.Fatalf("%s: code=%d body=%q, want 200/%q", c.target, rec.Code, rec.Body.String(), c.body)
		}
	}
	if rec := doRouteRec(a, http.MethodGet, "/e/missing.txt"); rec.Code != 404 {
		t.Fatalf("missing: code=%d, want 404", rec.Code)
	}
}

// With Config.FS set, root selects a sub-path within the filesystem.
func TestStaticNewConfigFSSubPath(t *testing.T) {
	a := httpsrv.New()
	a.All("/e/{*path}", New("sub", Config{FS: embedRoot(t)})) // serve the "sub" subtree

	// "sub/nested.txt" is served at the mount root as "nested.txt"
	rec := doRouteRec(a, http.MethodGet, "/e/nested.txt")
	if rec.Code != 200 || rec.Body.String() != "nested\n" {
		t.Fatalf("sub-path root: code=%d body=%q", rec.Code, rec.Body.String())
	}
	// "hello.txt" lives at the FS root, not under "sub" -> not served
	if rec := doRouteRec(a, http.MethodGet, "/e/hello.txt"); rec.Code != 404 {
		t.Fatalf("root sub-path should hide top-level file: code=%d, want 404", rec.Code)
	}
}

// An empty root with no Config.FS does not panic: it logs the error and falls
// back to an empty filesystem that 404s every request.
func TestNewEmptyRootFallback(t *testing.T) {
	a := httpsrv.New()
	a.Get("/*", New("")) // misconfiguration degrades gracefully

	rec := doRouteRec(a, http.MethodGet, "/anything.txt")
	if rec.Code != 404 {
		t.Fatalf("empty-root fallback: code=%d, want 404", rec.Code)
	}
}

// Wildcard mode: the unnamed "/*" wildcard serves a whole file tree without a
// named {*path} parameter.
func TestStaticRootWildcard(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "hello.txt"), "hello world")
	mustWriteFile(t, filepath.Join(dir, "css", "main.css"), "body{}")

	a := httpsrv.New()
	a.Get("/*", New(dir))

	cases := []struct {
		target string
		code   int
		body   string // checked only when non-empty
	}{
		{"/hello.txt", 200, "hello world"},
		{"/css/main.css", 200, "body{}"},
		{"/missing.txt", 404, ""},
		{"/css", 404, ""}, // directory -> 404
	}
	for _, c := range cases {
		rec := doRouteRec(a, http.MethodGet, c.target)
		if rec.Code != c.code || (c.body != "" && rec.Body.String() != c.body) {
			t.Fatalf("%s: code=%d body=%q, want code=%d body=%q", c.target, rec.Code, rec.Body.String(), c.code, c.body)
		}
	}
}

// A prefix wildcard "/static/*" serves under that prefix.
func TestStaticPrefixWildcard(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "logo.png"), "PNGDATA")

	a := httpsrv.New()
	a.Get("/static/*", New(dir))

	rec := doRouteRec(a, http.MethodGet, "/static/logo.png")
	if rec.Code != 200 || rec.Body.String() != "PNGDATA" {
		t.Fatalf("/static/logo.png: code=%d body=%q", rec.Code, rec.Body.String())
	}
	// outside the prefix
	if rec := doRouteRec(a, http.MethodGet, "/logo.png"); rec.Code != 404 {
		t.Fatalf("/logo.png: code=%d, want 404 (outside /static)", rec.Code)
	}
}

// Directory traversal beyond the root is blocked.
func TestStaticTraversal(t *testing.T) {
	base := t.TempDir()
	mustWriteFile(t, filepath.Join(base, "secret.txt"), "top-secret")
	mustWriteFile(t, filepath.Join(base, "public", "ok.txt"), "ok")

	a := httpsrv.New()
	a.All("/files/{*path}", New(filepath.Join(base, "public")))

	if rec := doRouteRec(a, http.MethodGet, "/files/ok.txt"); rec.Code != 200 || rec.Body.String() != "ok" {
		t.Fatalf("ok.txt: code=%d body=%q", rec.Code, rec.Body.String())
	}
	if rec := doRouteRec(a, http.MethodGet, "/files/../secret.txt"); rec.Code != 404 {
		t.Fatalf("traversal: code=%d, want 404", rec.Code)
	}
}

// A static mount on a Group is scoped to the group's prefix.
func TestGroupStatic(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "logo.png"), "PNGDATA")

	a := httpsrv.New()
	assets := a.Group("/assets")
	assets.All("/img/{*path}", New(dir))

	rec := doRouteRec(a, http.MethodGet, "/assets/img/logo.png")
	if rec.Code != 200 || rec.Body.String() != "PNGDATA" {
		t.Fatalf("group static: code=%d body=%q", rec.Code, rec.Body.String())
	}
}

// Static-file requests still pass through the middleware chain (dispatch is the
// chain's terminal handler).
func TestStaticWithMiddleware(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "a.txt"), "A")

	a := httpsrv.New()
	a.Use(func(c httpsrv.Ctx) error {
		c.SetHeader("X-Mw", "1")
		return c.Next()
	})
	a.All("/s/{*path}", New(dir))

	rec := doRouteRec(a, http.MethodGet, "/s/a.txt")
	if rec.Header().Get("X-Mw") != "1" || rec.Body.String() != "A" {
		t.Fatalf("mw+static: X-Mw=%q body=%q", rec.Header().Get("X-Mw"), rec.Body.String())
	}
}

// The catch-all value is exposed under both the declared name and the
// conventional "*" key.
func TestStaticCatchAllParam(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "a.txt"), "A")

	var gotName, gotStar string
	a := httpsrv.New()
	a.Get("/files/{*path}", func(c httpsrv.Ctx) error {
		gotName = c.Params("path")
		gotStar = c.Params("*")
		return c.SendString("ok")
	})
	a.Get("/static/{*path}", New(dir))

	doRouteRec(a, http.MethodGet, "/files/css/main.css")
	if gotName != "css/main.css" || gotStar != "css/main.css" {
		t.Fatalf("catch-all params: name=%q star=%q", gotName, gotStar)
	}
	if rec := doRouteRec(a, http.MethodGet, "/static/a.txt"); rec.Code != 200 || rec.Body.String() != "A" {
		t.Fatalf("static via catch-all: code=%d body=%q", rec.Code, rec.Body.String())
	}
}
