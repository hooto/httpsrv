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
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

// Ctx carries the request/response state for a Handler. It is implemented over
// plain net/http and built per request; path params come from the standard
// library (r.PathValue, set by the router), so Ctx.Params needs no custom
// context wiring.
type Ctx interface {
	// Request / Response
	Request() *http.Request
	Response() http.ResponseWriter
	// SetResponse replaces the response writer for the remainder of the chain.
	// Intended for middleware that wraps the writer (e.g. compression); route
	// handlers normally do not need it.
	SetResponse(w http.ResponseWriter)

	// Identity
	Method() string
	Path() string
	IP() string
	// BaseURL returns (protocol + host + base path).
	BaseURL() string
	// Locale returns the request locale (set by the AcceptLanguage middleware),
	// falling back to the i18n default, then "".
	Locale() string

	// Input
	Params(key string) string
	Query(key string, def ...string) string
	Header(key string, def ...string) string // request header
	Body() []byte
	FormValue(key string, def ...string) string // body form field (urlencoded or multipart)
	Bind(out any) error                         // JSON body -> out (json.Unmarshal)

	// Output (Status/SetHeader chain before the body is written)
	Status(code int) Ctx
	SetHeader(key, value string) // response header
	JSON(v any) error
	Send(b []byte) error
	SendString(s string) error
	Redirect(status int, url string) error
	Render(name string, bind any, layouts ...string) error // html template render
	Translate(locale, key string, args ...any) string      // i18n lookup (requires WithI18n)

	// Next runs the next handler in the middleware chain (fiber v3 style).
	// Middleware call it to continue; route handlers are terminal and ignore it.
	Next() error
}

// Handler is the typical handler signature: it receives a Ctx and returns an
// error (nil on success). Route registrars (Get/Post/.../All) accept a Handler
// or, for direct stdlib interop, an http.Handler.
type Handler func(ctx Ctx) error

// ctxImpl is the concrete Ctx. Cheap to allocate per request.
type ctxImpl struct {
	app      *app // back-reference for app-level services (views); nil-safe
	w        http.ResponseWriter
	r        *http.Request
	body     []byte
	bodyRead bool
	chain    []Handler // per-request handler chain (middleware + dispatch)
	index    int       // current position in chain; Next advances it
}

func newCtx(w http.ResponseWriter, r *http.Request) *ctxImpl {
	return &ctxImpl{w: w, r: r}
}

// Next runs the next handler in the chain (fiber v3 style). Middleware use it to
// continue; the terminal dispatch handler does not call it. Returns the next
// handler's error (nil past the end of the chain).
func (c *ctxImpl) Next() error {
	c.index++
	if c.index >= len(c.chain) {
		return nil
	}
	return c.chain[c.index](c)
}

func (c *ctxImpl) Request() *http.Request            { return c.r }
func (c *ctxImpl) Response() http.ResponseWriter     { return c.w }
func (c *ctxImpl) SetResponse(w http.ResponseWriter) { c.w = w }
func (c *ctxImpl) Method() string                    { return c.r.Method }
func (c *ctxImpl) Path() string                      { return c.r.URL.Path }
func (c *ctxImpl) IP() string                        { return clientIP(c.r) }

// BaseURL returns (protocol + host + base path). Protocol honors
// X-Forwarded-Proto and TLS; host honors X-Forwarded-Host then r.Host. There is
// no base-path config yet, so the base path is currently empty.
func (c *ctxImpl) BaseURL() string {
	scheme := "http"
	if c.r.TLS != nil {
		scheme = "https"
	}
	if p := c.r.Header.Get("X-Forwarded-Proto"); p != "" {
		scheme = p
	}
	host := c.r.Host
	if h := c.r.Header.Get("X-Forwarded-Host"); h != "" {
		host = h
	}
	return scheme + "://" + host
}

func (c *ctxImpl) Locale() string {
	if loc := localeFromRequest(c.r); loc != "" {
		return loc
	}
	if i := c.i18n(); i != nil {
		return i.Default()
	}
	return ""
}

func (c *ctxImpl) Params(key string) string {
	return c.r.PathValue(key)
}

func (c *ctxImpl) Query(key string, def ...string) string {
	if v := c.r.URL.Query().Get(key); v != "" {
		return v
	}
	if len(def) > 0 {
		return def[0]
	}
	return ""
}

func (c *ctxImpl) Header(key string, def ...string) string {
	if v := c.r.Header.Get(key); v != "" {
		return v
	}
	if len(def) > 0 {
		return def[0]
	}
	return ""
}

func (c *ctxImpl) Body() []byte {
	c.ensureBody()
	return c.body
}

// FormValue returns the first value of a body form field. It supports both
// application/x-www-form-urlencoded and multipart/form-data (delegated to
// net/http's FormValue, body taking precedence over query). An empty result
// falls back to def, then "".
func (c *ctxImpl) FormValue(key string, def ...string) string {
	c.ensureBody()
	if v := c.r.FormValue(key); v != "" {
		return v
	}
	if len(def) > 0 {
		return def[0]
	}
	return ""
}

// Bind parses the JSON request body into out via json.Unmarshal. It reads the
// cached body (see ensureBody), so it composes with Body/FormValue regardless
// of call order. An empty body is a no-op (returns nil).
func (c *ctxImpl) Bind(out any) error {
	c.ensureBody()
	if len(c.body) == 0 {
		return nil
	}
	return json.Unmarshal(c.body, out)
}

// ensureBody reads r.Body once (caching it) and restores r.Body to a reader
// over the cache. This makes Body() and FormValue() independent of call order:
// without it the first consumer would drain the request body stream and leave
// nothing for the other.
func (c *ctxImpl) ensureBody() {
	if !c.bodyRead {
		c.body, _ = io.ReadAll(c.r.Body)
		c.bodyRead = true
	}
	c.r.Body = io.NopCloser(bytes.NewReader(c.body))
}

func (c *ctxImpl) Status(code int) Ctx {
	c.w.WriteHeader(code)
	return c
}

func (c *ctxImpl) SetHeader(key, value string) {
	c.w.Header().Set(key, value)
}

func (c *ctxImpl) JSON(v any) error {
	c.w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(c.w).Encode(v)
}

func (c *ctxImpl) Send(b []byte) error {
	_, err := c.w.Write(b)
	return err
}

func (c *ctxImpl) SendString(s string) error {
	return c.Send([]byte(s))
}

func (c *ctxImpl) Redirect(status int, url string) error {
	c.w.Header().Set("Location", url)
	c.w.WriteHeader(status)
	return nil
}

// Render renders the named template (with optional layouts) and writes it as
// text/html. Requires a Views engine configured via WithViews or
// WithConfig(Config{Views: ...}).
func (c *ctxImpl) Render(name string, bind any, layouts ...string) error {
	v := c.views()
	if v == nil {
		return errors.New("httpsrv: no views configured (use WithViews or WithConfig(Config{Views: ...}))")
	}
	c.w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return v.Render(c.w, name, bind, layouts...)
}

func (c *ctxImpl) views() Views {
	if c.app != nil {
		if p := c.app.views.Load(); p != nil {
			return *p
		}
	}
	return nil
}

// Translate looks up key in locale via the App's i18n store; without one
// (WithI18n not used) it returns the key unchanged.
func (c *ctxImpl) Translate(locale, key string, args ...any) string {
	if i := c.i18n(); i != nil {
		return i.Translate(locale, key, args...)
	}
	return key
}

func (c *ctxImpl) i18n() *I18n {
	if c.app != nil {
		if v := c.app.i18n.Load(); v != nil {
			return v
		}
	}
	return nil
}

// toHandler normalizes a registrar argument to a Handler. It accepts a Handler
// (or func(Ctx) error), or an http.Handler wrapped via basicToHandler; anything
// else panics at registration (fail-fast). For a plain stdlib handler function,
// wrap it with http.HandlerFunc explicitly.
func toHandler(method, path string, h any) Handler {
	switch fn := h.(type) {
	case Handler:
		return fn
	case func(Ctx) error:
		return fn
	case http.Handler:
		return basicToHandler(fn)
	default:
		panic("httpsrv: unsupported handler type for " + method + " " + path)
	}
}

// basicToHandler adapts a stdlib http.Handler to a Handler: it dispatches to the
// underlying ServeHTTP and returns nil (http.Handlers have no error result).
func basicToHandler(h http.Handler) Handler {
	return func(c Ctx) error {
		h.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

// defaultCtxError is the fallback when a Handler returns an error: respond 500.
// If the Handler already wrote the response, this superfluous header/body is
// logged by net/http — a configurable ErrorHandler can replace this.
func defaultCtxError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintln(w, err.Error())
}

// clientIP extracts the client address, honoring X-Forwarded-For (first hop)
// then X-Real-Ip, falling back to the RemoteAddr host.
func clientIP(r *http.Request) string {
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		if i := strings.IndexByte(v, ','); i >= 0 {
			v = v[:i]
		}
		return strings.TrimSpace(v)
	}
	if v := r.Header.Get("X-Real-Ip"); v != "" {
		return strings.TrimSpace(v)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
