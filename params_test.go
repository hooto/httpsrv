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
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestParamsValue(t *testing.T) {
	// Create request with query params
	req := httptest.NewRequest("GET", "/test?key1=value1&key2=value2", nil)

	p := &Params{
		request: req,
	}

	if v := p.Value("key1"); v != "value1" {
		t.Errorf("expected value1, got %s", v)
	}

	if v := p.Value("key2"); v != "value2" {
		t.Errorf("expected value2, got %s", v)
	}

	if v := p.Value("nonexistent"); v != "" {
		t.Errorf("expected empty string, got %s", v)
	}
}

func TestParamsPathValue(t *testing.T) {
	req := httptest.NewRequest("GET", "/users/123", nil)
	req.SetPathValue("id", "123")

	p := &Params{
		request: req,
	}

	if v := p.Value("id"); v != "123" {
		t.Errorf("expected 123, got %s", v)
	}
}

func TestParamsSetValue(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)

	p := &Params{
		request: req,
	}

	p.SetValue("custom", "value")

	if v := p.Value("custom"); v != "value" {
		t.Errorf("expected value, got %s", v)
	}
}

func TestParamsIntValue(t *testing.T) {
	tests := []struct {
		key      string
		value    string
		expected int64
	}{
		{"num", "42", 42},
		{"negative", "-10", -10},
		{"zero", "0", 0},
		{"invalid", "abc", 0},
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test?"+tt.key+"="+tt.value, nil)
			p := &Params{request: req}

			result := p.IntValue(tt.key)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestParamsFloatValue(t *testing.T) {
	tests := []struct {
		key      string
		value    string
		expected float64
	}{
		{"num", "3.14", 3.14},
		{"negative", "-2.5", -2.5},
		{"zero", "0", 0},
		{"invalid", "abc", 0},
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test?"+tt.key+"="+tt.value, nil)
			p := &Params{request: req}

			result := p.FloatValue(tt.key)
			if result != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestParamsPostForm(t *testing.T) {
	form := url.Values{}
	form.Set("username", "testuser")
	form.Set("password", "testpass")

	req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	p := &Params{request: req}

	if v := p.Value("username"); v != "testuser" {
		t.Errorf("expected testuser, got %s", v)
	}

	if v := p.Value("password"); v != "testpass" {
		t.Errorf("expected testpass, got %s", v)
	}
}

func TestParamsInitOnlyOnce(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?key=value", nil)

	p := &Params{request: req}

	// Call init multiple times
	p.init()
	firstValues := p.values

	p.init()
	secondValues := p.values

	// Should only initialize once
	if firstValues == nil || secondValues == nil {
		t.Error("values should be initialized")
	}
}

func TestParamsNilRequest(t *testing.T) {
	p := &Params{}

	// Should not panic with nil request
	if v := p.Value("any"); v != "" {
		t.Errorf("expected empty string, got %s", v)
	}
}

func TestParamsPriority(t *testing.T) {
	// Path value should have priority over query param
	req := httptest.NewRequest("GET", "/test?id=query", nil)
	req.SetPathValue("id", "path")

	p := &Params{request: req}

	if v := p.Value("id"); v != "path" {
		t.Errorf("expected path value 'path', got %s", v)
	}
}

func TestParamsFilter(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	resp := newResponse(httptest.NewRecorder())

	c := newController(NewService(), newRequest(req), resp)

	// ParamsFilter should initialize Params
	ParamsFilter(c)

	if c.Params == nil {
		t.Error("Params should be initialized")
	}
}
