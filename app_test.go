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
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// doRoute dispatches a request through app (as an http.Handler) and returns
// the status code and response body.
func doRoute(a App, method, target string) (int, string) {
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	a.(http.Handler).ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func TestAppBasicRoute(t *testing.T) {
	a := New()
	a.Get("/hello", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hi")
	}))
	if code, body := doRoute(a, http.MethodGet, "/hello"); code != 200 || body != "hi" {
		t.Fatalf("GET /hello: code=%d body=%q", code, body)
	}
}

func TestAppRootRoute(t *testing.T) {
	a := New()
	a.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "root")
	}))
	// A request path is at least "/"; the empty-string case is exercised at the
	// tree level by TestRouteTreeRootAndEmpty.
	if code, body := doRoute(a, http.MethodGet, "/"); code != 200 || body != "root" {
		t.Fatalf("GET /: code=%d body=%q", code, body)
	}
}

func TestAppMethodRouting(t *testing.T) {
	a := New()
	a.Get("/x", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "get") }))
	a.Post("/x", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "post") }))

	if _, body := doRoute(a, http.MethodGet, "/x"); body != "get" {
		t.Fatalf("GET /x: body=%q", body)
	}
	if _, body := doRoute(a, http.MethodPost, "/x"); body != "post" {
		t.Fatalf("POST /x: body=%q", body)
	}
	// A method with no route falls through to 404.
	if code, _ := doRoute(a, http.MethodDelete, "/x"); code != 404 {
		t.Fatalf("DELETE /x: code=%d, want 404", code)
	}
}

func TestAppParams(t *testing.T) {
	a := New()
	a.Get("/users/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "id="+r.PathValue("id"))
	}))
	a.Get("/posts/{post_id}/comments/{comment_id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, r.PathValue("post_id")+":"+r.PathValue("comment_id"))
	}))

	if _, body := doRoute(a, http.MethodGet, "/users/42"); body != "id=42" {
		t.Fatalf("/users/42: body=%q", body)
	}
	// A dotted value is captured whole (matches [^/]+).
	if _, body := doRoute(a, http.MethodGet, "/users/a.b"); body != "id=a.b" {
		t.Fatalf("/users/a.b: body=%q", body)
	}
	if _, body := doRoute(a, http.MethodGet, "/posts/5/comments/22"); body != "5:22" {
		t.Fatalf("/posts/5/comments/22: body=%q", body)
	}
}

// A static route must beat a param route at the same position, order-independent.
func TestAppStaticBeatsParam(t *testing.T) {
	for _, order := range [2][2]string{
		{"/users/profile", "/users/{id}"},
		{"/users/{id}", "/users/profile"},
	} {
		t.Run(order[0]+"_first", func(t *testing.T) {
			a := New()
			for _, p := range order {
				pp := p
				a.Get(pp, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, pp) }))
			}
			cases := []struct{ path, want string }{
				{"/users/profile", "/users/profile"},
				{"/users/7", "/users/{id}"},
			}
			for _, c := range cases {
				if _, body := doRoute(a, http.MethodGet, c.path); body != c.want {
					t.Fatalf("GET %q: body=%q, want %q", c.path, body, c.want)
				}
			}
		})
	}
}

func TestAppGroup(t *testing.T) {
	a := New()
	api := a.Group("/api")
	api.Get("/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "list") }))
	api.Get("/users/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "user "+r.PathValue("id"))
	}))

	// Nested group inherits the parent prefix.
	v1 := api.Group("/v1")
	v1.Get("/items", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "items") }))
	v1.Get("/items/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "item "+r.PathValue("id"))
	}))

	cases := []struct {
		path string
		want string
	}{
		{"/api/users", "list"},
		{"/api/users/9", "user 9"},
		{"/api/v1/items", "items"},
		{"/api/v1/items/3", "item 3"},
	}
	for _, c := range cases {
		if code, body := doRoute(a, http.MethodGet, c.path); code != 200 || body != c.want {
			t.Fatalf("GET %q: code=%d body=%q, want %q", c.path, code, body, c.want)
		}
	}
}

func TestAppNotFound(t *testing.T) {
	a := New()
	a.Get("/here", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	if code, _ := doRoute(a, http.MethodGet, "/missing"); code != 404 {
		t.Fatalf("GET /missing: code=%d, want 404", code)
	}
}

func TestAppAll(t *testing.T) {
	a := New()
	a.All("/any", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "any") }))
	for _, m := range []string{
		http.MethodGet, http.MethodPost, http.MethodPut,
		http.MethodDelete, http.MethodPatch, http.MethodHead, http.MethodOptions,
	} {
		if code, body := doRoute(a, m, "/any"); code != 200 || body != "any" {
			t.Fatalf("%s /any: code=%d body=%q", m, code, body)
		}
	}
}

// Chaining returns a Router, so registrations can be fluent.
func TestAppChaining(t *testing.T) {
	a := New()
	a.Get("/a", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "a") })).
		Get("/b", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "b") })).
		Post("/c", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "c") }))

	for _, c := range []struct {
		method, path, want string
	}{
		{http.MethodGet, "/a", "a"},
		{http.MethodGet, "/b", "b"},
		{http.MethodPost, "/c", "c"},
	} {
		if _, body := doRoute(a, c.method, c.path); body != c.want {
			t.Fatalf("%s %q: body=%q, want %q", c.method, c.path, body, c.want)
		}
	}
}

// A malformed route pattern panics at registration (fail-fast), leaving
// previously registered routes intact.
func TestAppInvalidRoutePanics(t *testing.T) {
	a := New()
	a.Get("/ok", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "ok") }))

	for _, bad := range []string{"/users/{id", "/users/{}", "/users/{a}{b}"} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatalf("expected panic registering %q", bad)
				}
			}()
			a.Get(bad, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		}()
	}

	// The good route still works.
	if code, body := doRoute(a, http.MethodGet, "/ok"); code != 200 || body != "ok" {
		t.Fatalf("GET /ok after bad registrations: code=%d body=%q", code, body)
	}
}

// Run wires the app into a real *http.Server. A pre-bound listener lets us
// stop the server by closing the listener.
func TestAppRunListener(t *testing.T) {
	a := New()
	a.Get("/ping", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "pong") }))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	done := make(chan struct{})
	go func() {
		_ = a.Run(ln)
		close(done)
	}()
	t.Cleanup(func() {
		_ = ln.Close()
		<-done
	})

	// Poll until the server is accepting connections.
	var resp *http.Response
	for range 100 {
		resp, err = http.Get("http://" + ln.Addr().String() + "/ping")
		if err == nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("server never served: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "pong" {
		t.Fatalf("/ping body=%q, want pong", body)
	}
}

// doRouteRec is like doRoute but returns the recorder so tests can inspect
// response headers (used by the Use/middleware tests).
func doRouteRec(a App, method, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	a.(http.Handler).ServeHTTP(rec, req)
	return rec
}

// Global Use middleware runs for every request, before the matched handler.
func TestAppUseGlobal(t *testing.T) {
	a := New()
	a.Use(func(c Ctx) error {
		c.SetHeader("X-Mw", "ran")
		return c.Next()
	})
	a.Get("/x", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "ok") }))

	rec := doRouteRec(a, http.MethodGet, "/x")
	if rec.Code != 200 || rec.Body.String() != "ok" {
		t.Fatalf("code=%d body=%q", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Mw") != "ran" {
		t.Fatalf("middleware did not run: X-Mw=%q", rec.Header().Get("X-Mw"))
	}
}

// Middleware runs in registration order, all before the handler.
func TestAppUseOrder(t *testing.T) {
	mw := func(tag string) Handler {
		return func(c Ctx) error {
			c.Response().Header().Add("X-Order", tag)
			return c.Next()
		}
	}
	a := New()
	a.Use(mw("a"))
	a.Use(mw("b"))
	a.Get("/x", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Order", "h")
		io.WriteString(w, "ok")
	}))

	rec := doRouteRec(a, http.MethodGet, "/x")
	got := rec.Header().Values("X-Order")
	want := []string{"a", "b", "h"}
	if len(got) != len(want) {
		t.Fatalf("X-Order=%v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("X-Order[%d]=%q, want %q (full=%v)", i, got[i], want[i], got)
		}
	}
}

// A middleware that does not call next short-circuits: the handler never runs.
func TestAppUseShortCircuit(t *testing.T) {
	a := New()
	handlerRan := false
	a.Use(func(c Ctx) error {
		c.Status(http.StatusForbidden)
		return c.SendString("blocked") // no c.Next() -> short-circuit
	})
	a.Get("/x", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { handlerRan = true }))

	rec := doRouteRec(a, http.MethodGet, "/x")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("code=%d, want 403", rec.Code)
	}
	if rec.Body.String() != "blocked" {
		t.Fatalf("body=%q, want blocked", rec.Body.String())
	}
	if handlerRan {
		t.Fatalf("handler ran; middleware should have short-circuited")
	}
}

// Use middleware applies to all HTTP methods.
func TestAppUseAllMethods(t *testing.T) {
	a := New()
	a.Use(func(c Ctx) error {
		c.SetHeader("X-Mw", "1")
		return c.Next()
	})
	a.Get("/x", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	a.Post("/x", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	for _, m := range []string{http.MethodGet, http.MethodPost} {
		rec := doRouteRec(a, m, "/x")
		if rec.Code != 200 || rec.Header().Get("X-Mw") != "1" {
			t.Fatalf("%s /x: code=%d X-Mw=%q", m, rec.Code, rec.Header().Get("X-Mw"))
		}
	}
}

// A path argument scopes Use to that prefix, matching on "/" boundaries so
// "/api" does not catch "/apifoo".
func TestAppUsePrefix(t *testing.T) {
	a := New()
	a.Use("/api", func(c Ctx) error {
		c.SetHeader("X-Api", "1")
		return c.Next()
	})
	a.Get("/api/x", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "api") }))
	a.Get("/apifoo", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "foo") }))
	a.Get("/other", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "other") }))

	if rec := doRouteRec(a, http.MethodGet, "/api/x"); rec.Header().Get("X-Api") != "1" {
		t.Fatalf("/api/x: X-Api=%q, want 1", rec.Header().Get("X-Api"))
	}
	if rec := doRouteRec(a, http.MethodGet, "/apifoo"); rec.Header().Get("X-Api") != "" {
		t.Fatalf("/apifoo: X-Api=%q, want empty (segment boundary)", rec.Header().Get("X-Api"))
	}
	if rec := doRouteRec(a, http.MethodGet, "/other"); rec.Header().Get("X-Api") != "" {
		t.Fatalf("/other: X-Api=%q, want empty", rec.Header().Get("X-Api"))
	}
}

// Use on a Group is scoped to the group's prefix.
func TestAppGroupUse(t *testing.T) {
	a := New()
	api := a.Group("/api")
	api.Use(func(c Ctx) error {
		c.SetHeader("X-Api", "1")
		return c.Next()
	})
	api.Get("/x", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "api") }))
	a.Get("/other", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "other") }))

	if rec := doRouteRec(a, http.MethodGet, "/api/x"); rec.Header().Get("X-Api") != "1" {
		t.Fatalf("/api/x: X-Api=%q, want 1", rec.Header().Get("X-Api"))
	}
	if rec := doRouteRec(a, http.MethodGet, "/other"); rec.Header().Get("X-Api") != "" {
		t.Fatalf("/other: X-Api=%q, want empty", rec.Header().Get("X-Api"))
	}
}

// Use accepts both a named Handler value and an inline func(Ctx) error.
func TestAppUseArgTypes(t *testing.T) {
	named := Handler(func(c Ctx) error {
		c.SetHeader("X-Named", "1")
		return c.Next()
	})
	a := New()
	a.Use(
		named,
		func(c Ctx) error {
			c.SetHeader("X-Inline", "1")
			return c.Next()
		},
	)
	a.Get("/x", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	rec := doRouteRec(a, http.MethodGet, "/x")
	if rec.Header().Get("X-Named") != "1" || rec.Header().Get("X-Inline") != "1" {
		t.Fatalf("named=%q inline=%q", rec.Header().Get("X-Named"), rec.Header().Get("X-Inline"))
	}
}

// waitServe GETs url until it succeeds or attempts run out.
func waitServe(t *testing.T, url string) *http.Response {
	t.Helper()
	var (
		resp *http.Response
		err  error
	)
	for range 100 {
		resp, err = http.Get(url)
		if err == nil {
			return resp
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("server never served %s: %v", url, err)
	return nil
}

func TestAppErrorHandler(t *testing.T) {
	a := New(WithErrorHandler(func(c Ctx, err error) {
		_ = c.Status(422).JSON(map[string]string{"error": err.Error()})
	}))
	a.Get("/x", func(c Ctx) error { return errors.New("bad input") })

	rec := doRouteRec(a, http.MethodGet, "/x")
	if rec.Code != 422 {
		t.Fatalf("code=%d, want 422", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "bad input") {
		t.Fatalf("body=%q, want it to contain the error", rec.Body.String())
	}
}

// Shutdown gracefully stops a Run-ing server; Run then returns nil.
func TestAppShutdown(t *testing.T) {
	a := New()
	a.Get("/ping", func(c Ctx) error { return c.SendString("pong") })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- a.Run(ln) }()

	resp := waitServe(t, "http://"+ln.Addr().String()+"/ping")
	resp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := a.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("Run returned %v, want nil (ErrServerClosed filtered)", err)
	}
}

// WithConfig(Config{Addr: ...}) makes Run listen on that address.
func TestAppConfigAddr(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close() // free the port; Go sets SO_REUSEADDR so immediate reuse is safe

	a := New(WithConfig(Config{Addr: addr, ReadTimeout: 5 * time.Second}))
	a.Get("/", func(c Ctx) error { return c.SendString("ok") })

	done := make(chan struct{})
	go func() { _ = a.Run(); close(done) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = a.Shutdown(ctx)
		<-done
	})

	resp := waitServe(t, "http://"+addr+"/")
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(b) != "ok" {
		t.Fatalf("body=%q, want ok", b)
	}
}
