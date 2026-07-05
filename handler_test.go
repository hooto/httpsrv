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
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestHandlerInfo(t *testing.T) {
	tests := []struct {
		name     string
		handler  *regHandler
		expected string
	}{
		{
			name: "handler with method and pattern",
			handler: &regHandler{
				method:  "GET",
				pattern: "/test",
			},
			expected: "GET /test",
		},
		{
			name: "handler with pattern only",
			handler: &regHandler{
				pattern: "/test",
			},
			expected: "/test",
		},
		{
			name: "handler with func",
			handler: &regHandler{
				pattern:     "/test",
				handlerFunc: func(w http.ResponseWriter, r *http.Request) {},
			},
			expected: "/test func",
		},
		{
			name: "handler with controller",
			handler: &regHandler{
				pattern: "/test",
				handlerController: &handlerController{
					Name:       "App",
					ActionName: "Index",
				},
			},
			expected: "/test ctrl App/Index",
		},
		{
			name: "handler with file server",
			handler: &regHandler{
				pattern: "/static",
				handlerFileServer: &handlerFileServer{
					filepath: "/path/to/static",
				},
			},
			expected: "/static fs (/path/to/static)",
		},
		{
			name: "handler with bin fs",
			handler: &regHandler{
				pattern: "/static",
				handlerFileServer: &handlerFileServer{
					binFs: http.Dir("/"),
				},
			},
			expected: "/static fs (bin)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.handler.info()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestHandlerHandleFunc(t *testing.T) {
	srv := NewService()

	called := false
	srv.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Write([]byte("test response"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	srv.router.add("/test", srv.handlers[len(srv.handlers)-1])
	h, _, _ := srv.router.find(req)
	h.handle(rec, req, "/test", "/test", time.Now())

	if !called {
		t.Error("handler function was not called")
	}
}

func TestHandlerHandleNotFound(t *testing.T) {
	srv := NewService()

	req := httptest.NewRequest("GET", "/notfound", nil)
	rec := httptest.NewRecorder()

	h, _, _ := srv.router.find(req)
	h.handle(rec, req, "/notfound", "/notfound", time.Now())

	// Default handler returns 200 with "page not found"
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestHandlerModulerFind(t *testing.T) {
	moduler := &handlerModuler{
		actions: map[string]*handlerController{
			"/hello/world": {
				Name:       "Hello",
				ActionName: "World",
			},
		},
	}

	// Create a request with path values set
	req := httptest.NewRequest("GET", "/hello/world", nil)
	req.SetPathValue("controller", "hello")
	req.SetPathValue("action", "world")

	hc := moduler.find(req)
	if hc == nil {
		t.Fatal("handler controller should be found")
	}
	if hc.Name != "Hello" {
		t.Errorf("expected name Hello, got %s", hc.Name)
	}
	if hc.ActionName != "World" {
		t.Errorf("expected action World, got %s", hc.ActionName)
	}
}

func TestHandlerModulerFindNotFound(t *testing.T) {
	moduler := &handlerModuler{
		actions: map[string]*handlerController{},
	}

	req := httptest.NewRequest("GET", "/hello/world", nil)
	req.SetPathValue("controller", "hello")
	req.SetPathValue("action", "world")

	hc := moduler.find(req)
	if hc != nil {
		t.Error("handler controller should be nil for unknown action")
	}
}

func TestHandlerModulerNilActions(t *testing.T) {
	moduler := &handlerModuler{}

	req := httptest.NewRequest("GET", "/test", nil)
	req.SetPathValue("controller", "test")
	req.SetPathValue("action", "index")

	hc := moduler.find(req)
	if hc != nil {
		t.Error("handler controller should be nil when actions is nil")
	}
}

func TestCompressResponseGzip(t *testing.T) {
	srv := NewService()
	srv.Config.CompressResponse = true

	srv.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test response for compression"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	srv.router.add("/test", srv.handlers[len(srv.handlers)-1])
	h, _, _ := srv.router.find(req)
	h.handle(rec, req, "/test", "/test", time.Now())

	// Check if gzip encoding is set
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Error("expected gzip content encoding")
	}
}

func TestCompressResponseBrotli(t *testing.T) {
	srv := NewService()
	srv.Config.CompressResponse = true

	srv.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test response for compression"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "br")
	rec := httptest.NewRecorder()

	srv.router.add("/test", srv.handlers[len(srv.handlers)-1])
	h, _, _ := srv.router.find(req)
	h.handle(rec, req, "/test", "/test", time.Now())

	// Check if brotli encoding is set
	if rec.Header().Get("Content-Encoding") != "br" {
		t.Error("expected brotli content encoding")
	}
}

func TestCompressResponseNoEncoding(t *testing.T) {
	srv := NewService()
	srv.Config.CompressResponse = true

	body := "test response without compression"
	srv.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	srv.router.add("/test", srv.handlers[len(srv.handlers)-1])
	h, _, _ := srv.router.find(req)
	h.handle(rec, req, "/test", "/test", time.Now())

	// No compression encoding should be set
	if rec.Header().Get("Content-Encoding") != "" {
		t.Error("expected no content encoding")
	}
}

func TestCompressResponseDisabled(t *testing.T) {
	srv := NewService()
	srv.Config.CompressResponse = false

	body := "test response"
	srv.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	srv.router.add("/test", srv.handlers[len(srv.handlers)-1])
	h, _, _ := srv.router.find(req)
	h.handle(rec, req, "/test", "/test", time.Now())

	// No compression should be applied when disabled
	if rec.Header().Get("Content-Encoding") != "" {
		t.Error("expected no content encoding when compression is disabled")
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(body)) {
		t.Error("response body should contain uncompressed content")
	}
}

// -- ActionFunc tests --

func sampleActionOk(ctx Ctx) error {
	return ctx.Send([]byte("hello from action"))
}

func sampleActionError(ctx Ctx) error {
	return fmt.Errorf("something went wrong")
}

func sampleActionJson(ctx Ctx) error {
	return ctx.JSON(map[string]string{"status": "ok"})
}

func TestActionFuncDispatch(t *testing.T) {
	srv := NewService()

	srv.regHandler(&regHandler{
		pattern: "/test/action-ok",
		handlerAction: &handlerAction{
			name: "ActionOk",
			fn:   sampleActionOk,
		},
	})

	lastHandler := srv.handlers[len(srv.handlers)-1]
	srv.router.add(lastHandler.pattern, lastHandler)

	req := httptest.NewRequest("GET", "/test/action-ok/", nil)
	rec := httptest.NewRecorder()

	h, urlPath, _ := srv.router.find(req)
	h.handle(rec, req, urlPath, urlPath, time.Now())

	if !bytes.Contains(rec.Body.Bytes(), []byte("hello from action")) {
		t.Errorf("expected body to contain 'hello from action', got %q", rec.Body.String())
	}
}

func TestActionFuncDispatchError(t *testing.T) {
	srv := NewService()

	srv.regHandler(&regHandler{
		pattern: "/test/action-err",
		handlerAction: &handlerAction{
			name: "ActionError",
			fn:   sampleActionError,
		},
	})

	lastHandler := srv.handlers[len(srv.handlers)-1]
	srv.router.add(lastHandler.pattern, lastHandler)

	req := httptest.NewRequest("GET", "/test/action-err/", nil)
	rec := httptest.NewRecorder()

	h, urlPath, _ := srv.router.find(req)
	h.handle(rec, req, urlPath, urlPath, time.Now())

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("something went wrong")) {
		t.Errorf("expected body to contain error message, got %q", rec.Body.String())
	}
}

func TestActionFuncDispatchJson(t *testing.T) {
	srv := NewService()

	srv.regHandler(&regHandler{
		pattern: "/test/action-json",
		handlerAction: &handlerAction{
			name: "ActionJson",
			fn:   sampleActionJson,
		},
	})

	lastHandler := srv.handlers[len(srv.handlers)-1]
	srv.router.add(lastHandler.pattern, lastHandler)

	req := httptest.NewRequest("GET", "/test/action-json/", nil)
	rec := httptest.NewRecorder()

	h, urlPath, _ := srv.router.find(req)
	h.handle(rec, req, urlPath, urlPath, time.Now())

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"ok"`)) {
		t.Errorf("expected json body, got %q", rec.Body.String())
	}
}

func TestActionFuncFiltersRun(t *testing.T) {
	srv := NewService()

	filterCalled := false
	srv.Filters = append(srv.Filters, func(c *Controller) {
		filterCalled = true
	})

	srv.regHandler(&regHandler{
		pattern: "/test/filter",
		handlerAction: &handlerAction{
			name: "Filter",
			fn: func(ctx Ctx) error {
				return ctx.Send([]byte("ok"))
			},
		},
	})

	lastHandler := srv.handlers[len(srv.handlers)-1]
	srv.router.add(lastHandler.pattern, lastHandler)

	req := httptest.NewRequest("GET", "/test/filter/", nil)
	rec := httptest.NewRecorder()

	h, urlPath, _ := srv.router.find(req)
	h.handle(rec, req, urlPath, urlPath, time.Now())

	if !filterCalled {
		t.Error("expected filter to be called for ActionFunc")
	}
}

func TestActionFuncViaModule(t *testing.T) {
	mod := NewModule()
	mod.RegisterAction("/api/action-ok", sampleActionOk)
	mod.RegisterAction("/api/action-json", sampleActionJson)

	srv := NewService()
	srv.HandleModule("/v1", mod)

	for _, h := range srv.handlers {
		srv.router.add(h.pattern, h)
	}

	tests := []struct {
		path     string
		wantBody string
	}{
		{"/v1/api/action-ok/", "hello from action"},
		{"/v1/api/action-json/", `"status":"ok"`},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()

			h, urlPath, _ := srv.router.find(req)
			h.handle(rec, req, urlPath, urlPath, time.Now())

			if !bytes.Contains(rec.Body.Bytes(), []byte(tt.wantBody)) {
				t.Errorf("expected body to contain %q, got %q", tt.wantBody, rec.Body.String())
			}
		})
	}
}

// TestControllerUrlBase verifies that UrlBase produces correct paths
// without double slashes, including when UrlBasePath is set.
func TestControllerUrlBase(t *testing.T) {
	srv := NewService()

	// httptest.NewRequest uses "example.com" as the default Host
	const host = "http://example.com"
	tests := []struct {
		name       string
		basePath   string
		path       string
		wantSuffix string
	}{
		{"no_basepath", "", "/path", host + "/path"},
		{"with_basepath", "app", "/path", host + "/app/path"},
		{"basepath_with_slash", "/app", "/path", host + "/app/path"},
		{"empty_path", "app", "", host + "/app"},
		{"path_no_leading_slash", "app", "sub", host + "/app/sub"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv.Config.UrlBasePath = tt.basePath
			req := httptest.NewRequest("GET", "/", nil)
			c := newController(srv, newRequest(req), newResponse(httptest.NewRecorder()))
			got := c.UrlBase(tt.path)
			if got != tt.wantSuffix {
				t.Errorf("got %q, want %q", got, tt.wantSuffix)
			}
		})
	}
}

// TestControllerRedirectNoDoubleSlash verifies Redirect does not produce
// URLs with double slashes when UrlBasePath is set.
func TestControllerRedirectNoDoubleSlash(t *testing.T) {
	srv := NewService()
	srv.Config.UrlBasePath = "app"

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	c := newController(srv, newRequest(req), newResponse(rec))

	c.Redirect("login")

	loc := rec.Header().Get("Location")
	if loc != "/app/login" {
		t.Errorf("expected /app/login, got %q", loc)
	}
}

// VoidInitController verifies that a controller whose Init() method
// returns no values does not cause a panic.
type VoidInitController struct {
	*Controller
}

func (c *VoidInitController) Init() {
}

func (c *VoidInitController) IndexAction() {
	c.RenderString("inited")
}

func TestControllerVoidInit(t *testing.T) {
	srv := NewService()

	hc := &handlerController{
		Name:        "VoidInit",
		ActionName:  "Index",
		ctrlType:    reflect.TypeOf(&VoidInitController{}).Elem(),
		ctrlIndexes: findControllers(reflect.TypeOf(&VoidInitController{}).Elem()),
	}

	h := &regHandler{
		pattern:           "/void-init/index/",
		handlerController: hc,
		service:           srv,
	}
	srv.router.add(h.pattern, h)

	req := httptest.NewRequest("GET", "/void-init/index/", nil)
	rec := httptest.NewRecorder()

	foundHandler, urlPath, _ := srv.router.find(req)
	if foundHandler == nil {
		t.Fatal("handler not found")
	}
	foundHandler.handle(rec, req, urlPath, urlPath, time.Now())

	if !bytes.Contains(rec.Body.Bytes(), []byte("inited")) {
		t.Errorf("expected body to contain 'inited', got %q", rec.Body.String())
	}
}

func TestHandlerInfoAction(t *testing.T) {
	h := &regHandler{
		pattern: "/test",
		handlerAction: &handlerAction{
			name: "ListUsers",
			fn:   func(ctx Ctx) error { return nil },
		},
	}
	expected := "/test action ListUsers"
	if got := h.info(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
