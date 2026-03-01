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
	"testing"
)

func TestNewService(t *testing.T) {
	srv := NewService()

	if srv == nil {
		t.Fatal("service should not be nil")
	}
	if srv.router == nil {
		t.Error("router should not be nil")
	}
	if srv.TemplateLoader == nil {
		t.Error("template loader should not be nil")
	}
	if srv.Config.HttpAddr != DefaultConfig.HttpAddr {
		t.Errorf("expected HttpAddr %s, got %s", DefaultConfig.HttpAddr, srv.Config.HttpAddr)
	}
	if srv.Config.HttpPort != DefaultConfig.HttpPort {
		t.Errorf("expected HttpPort %d, got %d", DefaultConfig.HttpPort, srv.Config.HttpPort)
	}
}

func TestDefaultService(t *testing.T) {
	if DefaultService == nil {
		t.Fatal("DefaultService should not be nil")
	}
}

func TestServiceHandleFunc(t *testing.T) {
	srv := NewService()

	srv.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test"))
	})

	// Check if handler was added
	found := false
	for _, h := range srv.handlers {
		if h.pattern == "/test/" {
			found = true
			break
		}
	}
	if !found {
		t.Error("handler should be added to service")
	}
}

func TestServiceRegHandler(t *testing.T) {
	srv := NewService()

	h := &regHandler{
		pattern: "/api",
	}

	srv.regHandler(h)

	// Pattern should have trailing slash
	if h.pattern != "/api/" {
		t.Errorf("expected pattern /api/, got %s", h.pattern)
	}

	// Handler should be in the list
	found := false
	for _, sh := range srv.handlers {
		if sh.pattern == "/api/" {
			found = true
			break
		}
	}
	if !found {
		t.Error("handler should be registered")
	}
}

func TestServiceRegHandlerReplace(t *testing.T) {
	srv := NewService()

	// Register first handler
	h1 := &regHandler{pattern: "/test"}
	srv.regHandler(h1)

	initialCount := len(srv.handlers)

	// Register handler with same pattern - should replace
	h2 := &regHandler{pattern: "/test"}
	srv.regHandler(h2)

	if len(srv.handlers) != initialCount {
		t.Errorf("handler count should remain same after replace, got %d", len(srv.handlers))
	}
}

func TestServiceStop(t *testing.T) {
	srv := NewService()

	// Stop should return nil when server is not started
	err := srv.Stop()
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestServiceHandleModule(t *testing.T) {
	srv := NewService()

	mod := NewModule()
	mod.RegisterController(&TestController{})

	srv.HandleModule("/test", mod)

	// Module should be added
	if len(srv.modules) == 0 {
		t.Error("module should be added to service")
	}
}

func TestServiceHandleModuleReplace(t *testing.T) {
	srv := NewService()

	mod1 := NewModule()
	mod1.RegisterController(&TestController{})

	mod2 := NewModule()
	mod2.RegisterController(&TestController{})

	srv.HandleModule("/test", mod1)
	initialCount := len(srv.modules)

	srv.HandleModule("/test", mod2)

	// Should replace, not add
	if len(srv.modules) != initialCount {
		t.Errorf("module count should remain same after replace, got %d", len(srv.modules))
	}
}

// Test controller for testing
type TestController struct {
	*Controller
}

func (c *TestController) IndexAction() {
	c.RenderString("test index")
}
