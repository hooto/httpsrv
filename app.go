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
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hooto/httpsrv/v2/internal/radix"
)

// This is the foundation of the package: a standalone, net/http-native
// routing layer built on the radix tree (internal/radix).
//
//   - The typical handler is a Handler (func(Ctx) error, see ctx.go); an
//     http.Handler is also accepted for direct stdlib interop. Both register
//     through the same verbs.
//   - The registration surface offers method-specific routes (Get/Post/...),
//     All, nestable Groups, and Use middleware, with method calls chaining.
//   - Middleware is a Handler (func(Ctx) error); it calls Ctx.Next to continue
//     the chain (fiber v3 style). Use may take a leading path to scope middleware
//     to a prefix (segment-aware, all HTTP methods).
//   - Path parameters use the "{name}" syntax and are exposed via the standard
//     library (*http.Request).PathValue/SetPathValue (Go 1.22+); Handler reads
//     them via Ctx.Params, an http.Handler via r.PathValue directly.
//   - Routes and middleware must be registered before Run. During serving the
//     trees are read-only; an RLock per request only guards against late writes.

// Router is the shared route-registration surface implemented by both the App
// and route Groups. Each method registers a handler for one HTTP method and
// returns the receiver, so calls chain.
type Router interface {
	// Get/Post/.../All accept a Handler (func(Ctx) error, the typical form) or an
	// http.Handler (for direct stdlib interop); a Handler receives a request
	// context (Ctx).
	Get(path string, handler any) Router
	Head(path string, handler any) Router
	Post(path string, handler any) Router
	Put(path string, handler any) Router
	Patch(path string, handler any) Router
	Delete(path string, handler any) Router
	Options(path string, handler any) Router
	All(path string, handler any) Router

	// Use registers middleware. With a leading string argument it scopes the
	// middleware to that path prefix (segment-aware, all methods); without one
	// it runs for every request.
	Use(args ...any) Router

	// Group returns a sub-router whose routes are prefixed with prefix.
	// Groups nest: a group created from another group inherits its prefix.
	Group(prefix string) Router
}

// App is a self-contained HTTP application: a Router plus a Run entry point.
// Create one with New.
type App interface {
	Router

	// Run starts the HTTP server and blocks until it stops. Each optional arg
	// may be:
	//   - a string "host:port" or ":port" listen address (default ":8080"), or
	//   - a pre-bound net.Listener.
	// http.ErrServerClosed from a graceful stop is not reported as an error.
	Run(args ...any) error

	// Shutdown gracefully stops the server (see http.Server.Shutdown), waiting
	// for active connections to finish or ctx to expire. Call it from another
	// goroutine while Run is blocking.
	Shutdown(ctx context.Context) error
}

// New creates a new App. Pass options to configure it, e.g. WithViews or
// WithConfig(Config{Views: ...}).
func New(opts ...Option) App {
	a := &app{
		trees:             make(map[string]*radix.Node[Handler]),
		addr:              ":8080",
		notFound:          http.HandlerFunc(defaultNotFound),
		readTimeout:       60 * time.Second,
		writeTimeout:      60 * time.Second,
		readHeaderTimeout: 10 * time.Second,
		maxHeaderBytes:    1 << 20,
	}
	a.router = &router{app: a} // prefix "" => routes register at the app root
	for _, o := range opts {
		if o != nil {
			o(a)
		}
	}
	return a
}

// Option configures an App at construction.
type Option func(*app)

// WithViews attaches a template engine (a Views implementation, e.g. a
// *Renderer from TemplatesDir/TemplatesFS) so Handler code can call Ctx.Render.
// It is shorthand for WithConfig(Config{Views: v}); a custom Views engine may
// equally be plugged in via Config.Views.
func WithViews(v Views) Option {
	return func(a *app) { a.setViews(v) }
}

// setViews stores a Views implementation behind an atomic pointer so per-request
// reads in Ctx.Render stay lock-free. A nil v (including a typed-nil pointer
// such as (*Renderer)(nil)) clears the engine.
func (a *app) setViews(v Views) {
	if isNilViews(v) {
		a.views.Store(nil)
		return
	}
	a.views.Store(&v)
}

// isNilViews reports whether v is an untyped nil or a typed-nil pointer (e.g.
// (*Renderer)(nil)), either of which should be treated as "no engine".
func isNilViews(v Views) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}

// WithI18n attaches an opt-in i18n store so Handler code can call Ctx.Translate
// and Ctx.Locale. i18n is not loaded by default. For template use, also pass
// the store's Funcs() to TemplatesFS/TemplatesDir.
func WithI18n(i *I18n) Option {
	return func(a *app) { a.i18n.Store(i) }
}

// ErrorHandler handles a non-nil error returned from a Handler. It may write a
// custom response (e.g. a JSON error body). If the Handler already wrote the
// response, further writes are superfluous. Set via WithErrorHandler; the
// default writes 500 + err.Error().
type ErrorHandler func(c Ctx, err error)

// WithErrorHandler sets a custom Handler-error handler.
func WithErrorHandler(h ErrorHandler) Option {
	return func(a *app) { a.errorHandler = h }
}

// app implements App and http.Handler.
type app struct {
	*router // shared registration logic; verbs and Use are defined once, on router

	mu                sync.RWMutex
	trees             map[string]*radix.Node[Handler] // one radix tree per HTTP method
	middleware        []useEntry                      // Use-registered middleware, in registration order
	views             atomic.Pointer[Views]           // template engine (Views) for Ctx.Render (nil = none)
	i18n              atomic.Pointer[I18n]            // opt-in i18n store for Ctx.Translate/Ctx.Locale
	errorHandler      ErrorHandler                    // custom Handler-error handler (nil = default 500)
	addr              string
	notFound          http.Handler
	readTimeout       time.Duration // server config (defaults applied in New; override via WithConfig)
	writeTimeout      time.Duration
	readHeaderTimeout time.Duration
	maxHeaderBytes    int
	server            *http.Server
}

// useEntry is one Use registration: middleware to run for paths under prefix.
type useEntry struct {
	prefix string // normalized, no trailing slash; "" => every path
	mws    []Handler
}

// addRoute registers a handler under method at path (already prefix-joined and
// brace-validated by the radix tree). A nil handler or malformed pattern is a
// programmer error, so it panics to fail fast at startup.
func (a *app) addRoute(method, path string, handler Handler) {
	if handler == nil {
		panic("httpsrv: nil handler for " + method + " " + path)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	t := a.trees[method]
	if t == nil {
		t = radix.New[Handler]()
		a.trees[method] = t
	}
	if err := t.Insert(path, handler); err != nil {
		panic("httpsrv: invalid route " + method + " " + path + ": " + err.Error())
	}
}

// addUse appends middleware scoped to prefix. The dispatch chain is built per
// request (see chainFor), so registration only records the entry.
func (a *app) addUse(prefix string, mws []Handler) {
	if len(mws) == 0 {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.middleware = append(a.middleware, useEntry{prefix: prefix, mws: mws})
}

// ServeHTTP runs the request through the middleware chain into the dispatcher.
// app implements http.Handler so it is used directly as the *http.Server
// handler. The chain is built per request: the prefix-matching Use middleware
// (registration order) followed by dispatch; middleware continue it via Ctx.Next.
func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := newCtx(w, r)
	c.app = a
	c.chain = a.chainFor(r.URL.Path)
	c.index = -1
	if err := c.Next(); err != nil {
		a.handleError(c, err)
	}
}

// dispatch is the terminal handler of the chain: match the method's tree,
// extract path params, and invoke the matched Handler (its error propagates up
// to ServeHTTP). A miss falls back to notFound. Static files are served by a
// FileServer (github.com/hooto/httpsrv/v2/middleware/static) registered on a
// catch-all route, so they flow through the same tree as explicit routes —
// explicit routes win because static nodes beat the catch-all.
func (a *app) dispatch(c Ctx) error {
	r := c.Request()
	a.mu.RLock()
	t := a.trees[r.Method]
	var (
		h      Handler
		params radix.Params
		ok     bool
	)
	if t != nil {
		h, params, ok = t.Search(r.URL.Path, nil)
	}
	a.mu.RUnlock()

	if ok {
		// Expose params via r.SetPathValue so both Handler (Ctx.Params, which
		// reads r.PathValue) and an http.Handler (r.PathValue) see them. A
		// catch-all is also exposed under "*" (fiber's c.Params("*") convention)
		// so a handler such as FileServer can read it without knowing the
		// declared param name.
		for _, p := range params {
			r.SetPathValue(p.Key, p.Value)
			if p.CatchAll {
				r.SetPathValue("*", p.Value)
			}
		}
		return h(c)
	}

	a.notFound.ServeHTTP(c.Response(), r)
	return nil
}

// chainFor builds the per-request handler chain: the prefix-matching Use
// middleware (registration order) followed by the dispatcher.
func (a *app) chainFor(urlPath string) []Handler {
	a.mu.RLock()
	defer a.mu.RUnlock()
	chain := make([]Handler, 0, len(a.middleware)+1)
	for _, e := range a.middleware {
		if matchPathPrefix(e.prefix, urlPath) {
			chain = append(chain, e.mws...)
		}
	}
	chain = append(chain, a.dispatch)
	return chain
}

// handleError routes a non-nil error from a Handler/middleware to the custom
// ErrorHandler, or the default 500.
func (a *app) handleError(c Ctx, err error) {
	if a.errorHandler != nil {
		a.errorHandler(c, err)
	} else {
		defaultCtxError(c.Response(), err)
	}
}

// Run starts the HTTP server and blocks until it stops.
func (a *app) Run(args ...any) error {
	addr := a.addr
	var listener net.Listener
	for _, arg := range args {
		if arg == nil {
			continue
		}
		switch v := arg.(type) {
		case string:
			if v != "" {
				addr = v
			}
		case net.Listener:
			listener = v
		}
	}

	a.server = &http.Server{
		Addr:              addr,
		Handler:           a,
		ReadTimeout:       a.readTimeout,
		WriteTimeout:      a.writeTimeout,
		ReadHeaderTimeout: a.readHeaderTimeout,
		MaxHeaderBytes:    a.maxHeaderBytes,
	}

	if listener != nil {
		slog.Info("httpsrv serving", "network", "listener")
		return filterServeErr(a.server.Serve(listener))
	}

	slog.Info("httpsrv listening", "address", addr)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return filterServeErr(a.server.Serve(ln))
}

// Shutdown gracefully shuts down the server without interrupting active
// connections (see http.Server.Shutdown). It is a no-op if Run has not been
// called yet.
func (a *app) Shutdown(ctx context.Context) error {
	if a.server == nil {
		return nil
	}
	return a.server.Shutdown(ctx)
}

// filterServeErr drops http.ErrServerClosed so a graceful stop is not an error.
func filterServeErr(err error) error {
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// defaultNotFound is the fallback handler when no route matches.
func defaultNotFound(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintln(w, "404 page not found")
}

// router holds the shared registration logic for the App and its Groups.
// Both the App (prefix "") and Groups (a non-empty prefix) use it, so the verb
// methods and Use are defined once and chain correctly for either.
type router struct {
	app    *app
	prefix string // normalized, no trailing slash; "" for the App root
}

// add joins the group prefix onto path, normalizes the handler (a Handler or an
// http.Handler) to a Handler, then registers on the app.
func (c *router) add(method, path string, handler any) {
	c.app.addRoute(method, joinRoutes(c.prefix, path), toHandler(method, path, handler))
}

func (c *router) Get(path string, handler any) Router {
	c.add(http.MethodGet, path, handler)
	return c
}

func (c *router) Head(path string, handler any) Router {
	c.add(http.MethodHead, path, handler)
	return c
}

func (c *router) Post(path string, handler any) Router {
	c.add(http.MethodPost, path, handler)
	return c
}

func (c *router) Put(path string, handler any) Router {
	c.add(http.MethodPut, path, handler)
	return c
}

func (c *router) Patch(path string, handler any) Router {
	c.add(http.MethodPatch, path, handler)
	return c
}

func (c *router) Delete(path string, handler any) Router {
	c.add(http.MethodDelete, path, handler)
	return c
}

func (c *router) Options(path string, handler any) Router {
	c.add(http.MethodOptions, path, handler)
	return c
}

// allMethods is the set of methods registered by All.
var allMethods = []string{
	http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
	http.MethodPatch, http.MethodDelete, http.MethodConnect,
	http.MethodOptions, http.MethodTrace,
}

// All registers the handler for every supported HTTP method.
func (c *router) All(path string, handler any) Router {
	for _, m := range allMethods {
		c.add(m, path, handler)
	}
	return c
}

// Use registers middleware scoped to this router's prefix. A leading string
// argument is a sub-path joined onto the prefix; the rest are Middleware.
func (c *router) Use(args ...any) Router {
	sub, mws := parseUseArgs(args)
	c.app.addUse(joinPrefix(c.prefix, sub), mws)
	return c
}

// Group returns a sub-router whose prefix is this prefix joined with prefix.
func (c *router) Group(prefix string) Router {
	return &router{app: c.app, prefix: joinPrefix(c.prefix, prefix)}
}

// joinRoutes returns prefix + "/" + rel, slash-normalized (the radix tree
// cleans again on Insert, but joining here keeps the prefix meaningful).
func joinRoutes(prefix, rel string) string {
	return radix.CleanPath(prefix + "/" + rel)
}

// joinPrefix is joinRoutes for group/Use prefixes, additionally mapping a bare
// root to "" so Group("/") and app-wide Use behave like the app root.
func joinPrefix(parent, sub string) string {
	p := radix.CleanPath(parent + "/" + sub)
	if p == "/" {
		return ""
	}
	return p
}

// parseUseArgs splits Use arguments: if the first is a string it is the path
// prefix; the remaining arguments must be middleware (Handler / func(Ctx) error).
// Any other type panics (programmer error, surfaced at startup).
func parseUseArgs(args []any) (prefix string, mws []Handler) {
	i := 0
	if len(args) > 0 {
		if s, ok := args[0].(string); ok {
			prefix = s
			i = 1
		}
	}
	for ; i < len(args); i++ {
		switch fn := args[i].(type) {
		case Handler:
			mws = append(mws, fn)
		case func(Ctx) error:
			mws = append(mws, fn)
		default:
			panic("httpsrv: Use argument #" + strconv.Itoa(i) +
				" is not a Handler (func(Ctx) error)")
		}
	}
	return prefix, mws
}

// matchPathPrefix reports whether path equals prefix or is a sub-path of it,
// comparing on "/" boundaries so "/api" does not match "/apifoo".
func matchPathPrefix(prefix, path string) bool {
	if prefix == "" {
		return true
	}
	if path == prefix {
		return true
	}
	return strings.HasPrefix(path, prefix+"/")
}
