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
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// A Handler (func(Ctx) error) registers through the same verbs as an http.Handler.
func TestCtxHandlerRoute(t *testing.T) {
	a := New()
	a.Get("/hi", func(c Ctx) error { return c.SendString("hi") })

	rec := doRouteRec(a, http.MethodGet, "/hi")
	if rec.Code != 200 || rec.Body.String() != "hi" {
		t.Fatalf("code=%d body=%q", rec.Code, rec.Body.String())
	}
}

// An http.Handler and a Handler coexist on the same app.
func TestCtxAndBasicCoexist(t *testing.T) {
	a := New()
	a.Get("/basic", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("basic")) }))
	a.Get("/ctx", func(c Ctx) error { return c.SendString("ctx") })

	if rec := doRouteRec(a, http.MethodGet, "/basic"); rec.Body.String() != "basic" {
		t.Fatalf("basic: %q", rec.Body.String())
	}
	if rec := doRouteRec(a, http.MethodGet, "/ctx"); rec.Body.String() != "ctx" {
		t.Fatalf("ctx: %q", rec.Body.String())
	}
}

func TestCtxParams(t *testing.T) {
	a := New()
	a.Get("/users/{id}", func(c Ctx) error {
		return c.SendString("id=" + c.Params("id"))
	})
	if rec := doRouteRec(a, http.MethodGet, "/users/42"); rec.Body.String() != "id=42" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestCtxQueryDefault(t *testing.T) {
	a := New()
	a.Get("/q", func(c Ctx) error { return c.SendString(c.Query("name", "default")) })
	if rec := doRouteRec(a, http.MethodGet, "/q?name=hi"); rec.Body.String() != "hi" {
		t.Fatalf("body=%q", rec.Body.String())
	}
	if rec := doRouteRec(a, http.MethodGet, "/q"); rec.Body.String() != "default" {
		t.Fatalf("default body=%q", rec.Body.String())
	}
}

func TestCtxJSON(t *testing.T) {
	a := New()
	a.Get("/j", func(c Ctx) error { return c.JSON(map[string]int{"a": 1}) })

	rec := doRouteRec(a, http.MethodGet, "/j")
	if rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("content-type=%q", rec.Header().Get("Content-Type"))
	}
	var m map[string]int
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil || m["a"] != 1 {
		t.Fatalf("body=%q err=%v", rec.Body.String(), err)
	}
}

// TestCtxJSONCustomContentType verifies that a handler-set Content-Type is
// preserved and not overwritten by JSON's default.
func TestCtxJSONCustomContentType(t *testing.T) {
	a := New()
	a.Get("/j", func(c Ctx) error {
		c.SetHeader("Content-Type", "application/problem+json")
		return c.JSON(map[string]int{"a": 1})
	})

	rec := doRouteRec(a, http.MethodGet, "/j")
	if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("content-type=%q, want application/problem+json", got)
	}
}

// TestCtxJSONRepeatedContentType verifies that calling SetHeader for
// Content-Type multiple times (e.g. from middleware then the handler) before
// JSON does not produce a duplicate Content-Type header and still yields a
// single, correct value with a valid body.
func TestCtxJSONRepeatedContentType(t *testing.T) {
	a := New()
	a.Use(func(c Ctx) error { // middleware sets it once
		c.SetHeader("Content-Type", "application/json; charset=utf-8")
		return c.Next()
	})
	a.Get("/j", func(c Ctx) error { // handler sets it again, then JSON
		c.SetHeader("Content-Type", "application/json; charset=utf-8")
		return c.JSON(map[string]int{"a": 1})
	})

	rec := doRouteRec(a, http.MethodGet, "/j")
	if vals := rec.Header().Values("Content-Type"); len(vals) != 1 {
		t.Fatalf("Content-Type values=%v, want exactly 1", vals)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("content-type=%q", got)
	}
	var m map[string]int
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil || m["a"] != 1 {
		t.Fatalf("body=%q err=%v", rec.Body.String(), err)
	}
}

func TestCtxStatusSetHeader(t *testing.T) {
	a := New()
	a.Get("/teapot", func(c Ctx) error {
		c.SetHeader("X-Test", "1")
		return c.Status(418).SendString("teapot")
	})
	rec := doRouteRec(a, http.MethodGet, "/teapot")
	if rec.Code != 418 || rec.Body.String() != "teapot" || rec.Header().Get("X-Test") != "1" {
		t.Fatalf("code=%d body=%q X-Test=%q", rec.Code, rec.Body.String(), rec.Header().Get("X-Test"))
	}
}

// TestCtxSetHeaderChain verifies SetHeader returns a Ctx so header/status/body
// can be chained.
func TestCtxSetHeaderChain(t *testing.T) {
	a := New()
	a.Get("/c", func(c Ctx) error {
		return c.SetHeader("X-A", "1").SetHeader("X-B", "2").Status(201).SendString("ok")
	})
	rec := doRouteRec(a, http.MethodGet, "/c")
	if rec.Code != 201 || rec.Body.String() != "ok" ||
		rec.Header().Get("X-A") != "1" || rec.Header().Get("X-B") != "2" {
		t.Fatalf("code=%d body=%q X-A=%q X-B=%q",
			rec.Code, rec.Body.String(), rec.Header().Get("X-A"), rec.Header().Get("X-B"))
	}
}

func TestCtxRedirect(t *testing.T) {
	a := New()
	a.Get("/old", func(c Ctx) error { return c.Redirect(http.StatusFound, "/new") })
	rec := doRouteRec(a, http.MethodGet, "/old")
	if rec.Code != 302 || rec.Header().Get("Location") != "/new" {
		t.Fatalf("code=%d Location=%q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestCtxBodyHeader(t *testing.T) {
	a := New()
	a.Post("/echo", func(c Ctx) error {
		return c.SendString("hdr=" + c.Header("X-Token") + " body=" + string(c.Body()))
	})
	req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader("payload"))
	req.Header.Set("X-Token", "tok")
	rec := httptest.NewRecorder()
	a.(http.Handler).ServeHTTP(rec, req)

	if rec.Body.String() != "hdr=tok body=payload" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

// httptest defaults: Host=example.com, RemoteAddr=192.0.2.1:1234.
func TestCtxMetaBaseURLIP(t *testing.T) {
	a := New()
	a.Get("/meta", func(c Ctx) error {
		return c.SendString(c.BaseURL() + "|" + c.IP() + "|" + c.Method() + "|" + c.Path())
	})
	rec := doRouteRec(a, http.MethodGet, "/meta")
	if want := "http://example.com|192.0.2.1|GET|/meta"; rec.Body.String() != want {
		t.Fatalf("body=%q, want %q", rec.Body.String(), want)
	}
}

// A Handler returning a non-nil error yields 500.
func TestCtxHandlerError(t *testing.T) {
	a := New()
	a.Get("/err", func(c Ctx) error { return errSentinel })
	rec := doRouteRec(a, http.MethodGet, "/err")
	if rec.Code != 500 {
		t.Fatalf("code=%d, want 500", rec.Code)
	}
}

// The default error response must not leak the error text (it may contain
// template paths, expression snippets, or internal types), must be served as
// text/plain, and must carry X-Content-Type-Options: nosniff.
func TestCtxDefaultErrorIsSafe(t *testing.T) {
	a := New()
	a.Get("/err", func(c Ctx) error { return errSentinel })
	rec := doRouteRec(a, http.MethodGet, "/err")

	if rec.Code != 500 {
		t.Fatalf("code=%d, want 500", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("content-type=%q, want text/plain", rec.Header().Get("Content-Type"))
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("X-Content-Type-Options=%q, want nosniff", rec.Header().Get("X-Content-Type-Options"))
	}
	if strings.Contains(rec.Body.String(), errSentinel.Error()) {
		t.Fatalf("response body leaked the error: %q", rec.Body.String())
	}
	if rec.Body.String() != "Internal Server Error\n" {
		t.Fatalf("body=%q, want generic Internal Server Error", rec.Body.String())
	}
}

var errSentinel = errors.New("boom")

func TestCtxUnsupportedHandlerPanics(t *testing.T) {
	a := New()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for unsupported handler type")
		}
	}()
	a.Get("/bad", 123)
}

// urlencoded form fields.
func TestCtxFormURLEncoded(t *testing.T) {
	a := New()
	a.Post("/login", func(c Ctx) error {
		return c.SendString(c.FormValue("user") + ":" + c.FormValue("pass"))
	})
	form := url.Values{"user": {"alice"}, "pass": {"secret"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	a.(http.Handler).ServeHTTP(rec, req)
	if rec.Body.String() != "alice:secret" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

// Default value when the field is absent.
func TestCtxFormDefault(t *testing.T) {
	a := New()
	a.Post("/f", func(c Ctx) error { return c.SendString(c.FormValue("missing", "def")) })

	req := httptest.NewRequest(http.MethodPost, "/f", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	a.(http.Handler).ServeHTTP(rec, req)
	if rec.Body.String() != "def" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

// multipart/form-data text fields.
func TestCtxFormMultipart(t *testing.T) {
	a := New()
	a.Post("/upload", func(c Ctx) error {
		return c.SendString("title=" + c.FormValue("title"))
	})
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	if err := w.WriteField("title", "hello"); err != nil {
		t.Fatal(err)
	}
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	a.(http.Handler).ServeHTTP(rec, req)
	if rec.Body.String() != "title=hello" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

// Body() and FormValue() are independent of call order: calling Body() first
// (which drains the stream) must not break subsequent FormValue().
func TestCtxBodyThenFormValue(t *testing.T) {
	a := New()
	a.Post("/b", func(c Ctx) error {
		_ = c.Body() // drains the request body stream first
		return c.SendString("body=" + string(c.Body()) + " form=" + c.FormValue("user"))
	})
	form := url.Values{"user": {"bob"}}
	req := httptest.NewRequest(http.MethodPost, "/b", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	a.(http.Handler).ServeHTTP(rec, req)
	if rec.Body.String() != "body=user=bob form=bob" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestCtxBind(t *testing.T) {
	type login struct {
		User string `json:"user"`
		Pass string `json:"pass"`
	}
	a := New()
	a.Post("/login", func(c Ctx) error {
		var in login
		if err := c.Bind(&in); err != nil {
			return c.Status(400).SendString("bad json")
		}
		return c.SendString(in.User + ":" + in.Pass)
	})

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"user":"alice","pass":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	a.(http.Handler).ServeHTTP(rec, req)

	if rec.Body.String() != "alice:secret" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestCtxBindBadJSON(t *testing.T) {
	a := New()
	a.Post("/", func(c Ctx) error {
		var m map[string]any
		if err := c.Bind(&m); err != nil {
			return c.Status(400).SendString("err")
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{bad`))
	rec := httptest.NewRecorder()
	a.(http.Handler).ServeHTTP(rec, req)

	if rec.Code != 400 {
		t.Fatalf("code=%d, want 400", rec.Code)
	}
}

// Empty body is a no-op (Bind returns nil).
func TestCtxBindEmpty(t *testing.T) {
	a := New()
	a.Post("/", func(c Ctx) error {
		var m map[string]any
		if err := c.Bind(&m); err != nil {
			return err
		}
		return c.SendString("empty-ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	rec := httptest.NewRecorder()
	a.(http.Handler).ServeHTTP(rec, req)

	if rec.Body.String() != "empty-ok" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

// Bind then Body: order-independent (Bind consumes the cached body, Body still
// returns the full bytes).
func TestCtxBindAndBody(t *testing.T) {
	a := New()
	a.Post("/", func(c Ctx) error {
		var m map[string]string
		_ = c.Bind(&m) // reads/caches the body first
		return c.SendString(m["x"] + ":" + string(c.Body()))
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"x":"hi"}`))
	rec := httptest.NewRecorder()
	a.(http.Handler).ServeHTTP(rec, req)

	if want := `hi:{"x":"hi"}`; rec.Body.String() != want {
		t.Fatalf("body=%q, want %q", rec.Body.String(), want)
	}
}
