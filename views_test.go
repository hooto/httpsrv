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
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
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

func mapFS(files map[string]string) fstest.MapFS {
	m := fstest.MapFS{}
	for name, data := range files {
		m[name] = &fstest.MapFile{Data: []byte(data), Mode: 0o644}
	}
	return m
}

func TestRenderBasic(t *testing.T) {
	r, err := TemplatesFS(mapFS(map[string]string{
		"hello.html": `<h1>Hello {{.Name}}</h1>`,
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	app := New(WithViews(r))
	app.Get("/h", func(c Ctx) error {
		return c.Render("hello.html", map[string]string{"Name": "World"})
	})

	rec := doRouteRec(app, http.MethodGet, "/h")
	if rec.Code != 200 || rec.Body.String() != "<h1>Hello World</h1>" {
		t.Fatalf("code=%d body=%q", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("content-type=%q", rec.Header().Get("Content-Type"))
	}
}

// TestRenderCustomContentType verifies that a handler-set Content-Type is
// preserved and not overwritten by Render's default.
func TestRenderCustomContentType(t *testing.T) {
	r, err := TemplatesFS(mapFS(map[string]string{
		"hello.html": `<h1>Hello {{.Name}}</h1>`,
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	app := New(WithViews(r))
	app.Get("/h", func(c Ctx) error {
		c.SetHeader("Content-Type", "application/xhtml+xml; charset=utf-8")
		return c.Render("hello.html", map[string]string{"Name": "World"})
	})

	rec := doRouteRec(app, http.MethodGet, "/h")
	if got := rec.Header().Get("Content-Type"); got != "application/xhtml+xml; charset=utf-8" {
		t.Fatalf("content-type=%q, want application/xhtml+xml; charset=utf-8", got)
	}
}

func TestRenderLayout(t *testing.T) {
	r, err := TemplatesFS(mapFS(map[string]string{
		"page.html":   `<p>{{.Name}}</p>`,
		"layout.html": `<html><body>{{.Content}}</body></html>`,
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	app := New(WithViews(r))
	app.Get("/", func(c Ctx) error {
		return c.Render("page.html", map[string]string{"Name": "ok"}, "layout.html")
	})

	rec := doRouteRec(app, http.MethodGet, "/")
	if want := `<html><body><p>ok</p></body></html>`; rec.Body.String() != want {
		t.Fatalf("body=%q, want %q", rec.Body.String(), want)
	}
}

// Layout may also use fields from the bind map (merged), and content is not
// double-escaped.
func TestRenderLayoutBind(t *testing.T) {
	r, err := TemplatesFS(mapFS(map[string]string{
		"page.html":   `<p>{{.Msg}}</p>`,
		"layout.html": `<title>{{.Title}}</title>{{.Content}}`,
	}), nil)
	if err != nil {
		t.Fatal(err)
	}
	app := New(WithViews(r))
	app.Get("/", func(c Ctx) error {
		return c.Render("page.html", map[string]any{"Msg": "hi", "Title": "T"}, "layout.html")
	})

	rec := doRouteRec(app, http.MethodGet, "/")
	if want := `<title>T</title><p>hi</p>`; rec.Body.String() != want {
		t.Fatalf("body=%q, want %q", rec.Body.String(), want)
	}
}

func TestRenderFuncsAndI18n(t *testing.T) {
	i := NewI18n("en")
	i.Add("en", map[string]string{"hi": "Hello"})
	i.Add("zh", map[string]string{"hi": "你好"})

	r, err := TemplatesFS(mapFS(map[string]string{
		"f.html": `{{upper .Name}}|{{T .Lang "hi"}}`,
	}), i.Funcs())
	if err != nil {
		t.Fatal(err)
	}

	app := New(WithViews(r))
	app.Get("/", func(c Ctx) error {
		return c.Render("f.html", map[string]string{"Name": "abc", "Lang": "zh"})
	})

	rec := doRouteRec(app, http.MethodGet, "/")
	if want := "ABC|你好"; rec.Body.String() != want {
		t.Fatalf("body=%q, want %q", rec.Body.String(), want)
	}
}

func TestRenderExtraFuncs(t *testing.T) {
	r, err := TemplatesFS(mapFS(map[string]string{
		"f.html": `{{exclaim .N}}`,
	}), template.FuncMap{
		"exclaim": func(s string) string { return s + "!" },
	})
	if err != nil {
		t.Fatal(err)
	}
	app := New(WithViews(r))
	app.Get("/", func(c Ctx) error {
		return c.Render("f.html", map[string]string{"N": "hi"})
	})

	rec := doRouteRec(app, http.MethodGet, "/")
	if rec.Body.String() != "hi!" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

// TemplatesDir loads from a real filesystem directory.
func TestRenderDir(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "t.html"), "Hi {{.N}}")

	r, err := TemplatesDir(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	app := New(WithViews(r))
	app.Get("/", func(c Ctx) error {
		return c.Render("t.html", map[string]string{"N": "x"})
	})

	rec := doRouteRec(app, http.MethodGet, "/")
	if rec.Body.String() != "Hi x" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

// Render without a configured engine yields 500.
func TestRenderNoRenderer(t *testing.T) {
	app := New() // no WithViews
	app.Get("/", func(c Ctx) error { return c.Render("x.html", nil) })

	rec := doRouteRec(app, http.MethodGet, "/")
	if rec.Code != 500 {
		t.Fatalf("code=%d, want 500", rec.Code)
	}
}

// Render of a missing template name yields 500.
func TestRenderMissingTemplate(t *testing.T) {
	r, err := TemplatesFS(mapFS(map[string]string{"a.html": "a"}), nil)
	if err != nil {
		t.Fatal(err)
	}
	app := New(WithViews(r))
	app.Get("/", func(c Ctx) error { return c.Render("nope.html", nil) })

	rec := doRouteRec(app, http.MethodGet, "/")
	if rec.Code != 500 {
		t.Fatalf("code=%d, want 500", rec.Code)
	}
}

// Parse error at construction surfaces immediately.
func TestRenderParseError(t *testing.T) {
	_, err := TemplatesFS(mapFS(map[string]string{
		"bad.html": `{{ .Name }`, // unclosed action
	}), nil)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

// A custom Views implementation (not the built-in engine) can be plugged in via WithViews.
type fakeViews struct{}

func (fakeViews) Load() error { return nil }
func (fakeViews) Render(w io.Writer, name string, bind any, layout ...string) error {
	_, err := fmt.Fprintf(w, "fake:%s:%v", name, bind)
	return err
}

func TestRenderCustomViews(t *testing.T) {
	app := New(WithViews(fakeViews{}))
	app.Get("/", func(c Ctx) error {
		return c.Render("x.html", "data")
	})

	rec := doRouteRec(app, http.MethodGet, "/")
	if rec.Code != 200 || rec.Body.String() != "fake:x.html:data" {
		t.Fatalf("code=%d body=%q", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("content-type=%q", rec.Header().Get("Content-Type"))
	}
}

// WithViews(nil) must not leave a typed-nil engine that panics on Render.
func TestRenderWithViewsNil(t *testing.T) {
	app := New(WithViews(nil))
	app.Get("/", func(c Ctx) error { return c.Render("x.html", nil) })

	rec := doRouteRec(app, http.MethodGet, "/")
	if rec.Code != 500 {
		t.Fatalf("code=%d, want 500", rec.Code)
	}
}

// WithViews((*renderer)(nil)) (typed nil) is treated as no engine, not a panic.
func TestRenderWithViewsTypedNil(t *testing.T) {
	app := New(WithViews((*renderer)(nil)))
	app.Get("/", func(c Ctx) error { return c.Render("x.html", nil) })

	rec := doRouteRec(app, http.MethodGet, "/")
	if rec.Code != 500 {
		t.Fatalf("code=%d, want 500", rec.Code)
	}
}
