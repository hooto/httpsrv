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
	"net/http"
	"net/http/httptest"
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
