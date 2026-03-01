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
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterAdd(t *testing.T) {
	router := &rootRouter{}

	// Test adding a simple route
	h := &regHandler{
		pattern: "/test",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("test"))
		},
	}
	router.add("/test", h)

	if router.node == nil {
		t.Fatal("router node should not be nil after adding route")
	}
}

func TestRouterAddWithParams(t *testing.T) {
	router := &rootRouter{}

	h := &regHandler{
		pattern: "/users/:id",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("user"))
		},
	}
	router.add("/users/:id", h)

	if router.node == nil {
		t.Fatal("router node should not be nil")
	}
}

func TestRouterAddWithCurlyBraceParams(t *testing.T) {
	router := &rootRouter{}

	h := &regHandler{
		pattern: "/items/{id}",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("item"))
		},
	}
	router.add("/items/{id}", h)

	if router.node == nil {
		t.Fatal("router node should not be nil")
	}
}

func TestRouterFind(t *testing.T) {
	router := &rootRouter{}

	h := &regHandler{
		pattern: "/hello",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello"))
		},
	}
	router.add("/hello", h)

	req := httptest.NewRequest("GET", "/hello", nil)
	foundHandler, urlPath, urlRoutePath := router.find(req)

	if foundHandler == nil {
		t.Fatal("handler should be found")
	}
	if urlPath != "/hello" {
		t.Errorf("expected urlPath /hello, got %s", urlPath)
	}
	if urlRoutePath != "/hello" {
		t.Errorf("expected urlRoutePath /hello, got %s", urlRoutePath)
	}
}

func TestRouterFindWithParams(t *testing.T) {
	router := &rootRouter{}

	h := &regHandler{
		pattern: "/users/:id",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("user"))
		},
	}
	router.add("/users/:id", h)

	req := httptest.NewRequest("GET", "/users/123", nil)
	foundHandler, _, _ := router.find(req)

	if foundHandler == nil {
		t.Fatal("handler should be found")
	}

	// Check if path value is set
	id := req.PathValue("id")
	if id != "123" {
		t.Errorf("expected path value id=123, got %s", id)
	}
}

func TestRouterFindNotFound(t *testing.T) {
	router := &rootRouter{}

	h := &regHandler{
		pattern: "/exists",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("exists"))
		},
	}
	router.add("/exists", h)

	req := httptest.NewRequest("GET", "/notexists", nil)
	foundHandler, _, _ := router.find(req)

	// Should return default handler
	if foundHandler == nil {
		t.Fatal("handler should not be nil (should return default)")
	}
}

func TestRouterFindWithTrailingSlash(t *testing.T) {
	router := &rootRouter{}

	h := &regHandler{
		pattern: "/path",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("path"))
		},
	}
	router.add("/path", h)

	req := httptest.NewRequest("GET", "/path/", nil)
	foundHandler, _, _ := router.find(req)

	if foundHandler == nil {
		t.Fatal("handler should be found")
	}
}

func TestRouterMultipleRoutes(t *testing.T) {
	router := &rootRouter{}

	routes := []struct {
		pattern string
		handler *regHandler
	}{
		{"/", &regHandler{pattern: "/", handlerFunc: func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("root")) }}},
		{"/users", &regHandler{pattern: "/users", handlerFunc: func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("users")) }}},
		{"/users/:id", &regHandler{pattern: "/users/:id", handlerFunc: func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("user")) }}},
		{"/posts/:post_id/comments/:comment_id", &regHandler{pattern: "/posts/:post_id/comments/:comment_id", handlerFunc: func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("comment")) }}},
	}

	for _, route := range routes {
		router.add(route.pattern, route.handler)
	}

	tests := []struct {
		path  string
		found bool
	}{
		{"/", true},
		{"/users", true},
		{"/users/123", true},
		{"/posts/1/comments/2", true},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		foundHandler, _, _ := router.find(req)
		if (foundHandler != nil) != tt.found {
			t.Errorf("path %s: expected found=%v", tt.path, tt.found)
		}
	}
}

func TestRouterCleanPath(t *testing.T) {
	router := &rootRouter{}

	h := &regHandler{
		pattern: "/test",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("test"))
		},
	}
	router.add("/test", h)

	// Test path with double slashes
	req := httptest.NewRequest("GET", "//test//", nil)
	foundHandler, urlPath, _ := router.find(req)

	if foundHandler == nil {
		t.Fatal("handler should be found")
	}
	if urlPath != "/test" {
		t.Errorf("expected cleaned urlPath /test, got %s", urlPath)
	}
}

func TestRouterCaseInsensitive(t *testing.T) {
	router := &rootRouter{}

	h := &regHandler{
		pattern: "/TestPath",
		handlerFunc: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("test"))
		},
	}
	router.add("/TestPath", h)

	// Pattern is lowercased internally
	req := httptest.NewRequest("GET", "/testpath", nil)
	foundHandler, _, _ := router.find(req)

	if foundHandler == nil {
		t.Fatal("handler should be found (case insensitive)")
	}
}
