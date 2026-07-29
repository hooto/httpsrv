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

// Package static provides a static-file handler for httpsrv.
//
// New returns an httpsrv.Handler that serves a whole file tree when registered
// on a catch-all route — the "/*" wildcard or a named {*name}.
// root is a directory path, or — with Config.FS set — a sub-path within that
// fs.FS:
//
//	app.Get("/*", static.New("./public"))
//	// GET /css/main.css  ->  ./public/css/main.css
//
//	app.Get("/static/*", static.New("./assets"))
//	// GET /static/css/main.css  ->  ./assets/css/main.css
//
//	// embedded filesystem
//	app.Get("/*", static.New(".", static.Config{FS: embedFS}))
//
// The path remainder is read via r.PathValue("*") (the router exposes it under
// "*" for both /* and {*name}).
package static

import (
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/hooto/httpsrv/v2"
)

// Config configures a static handler.
type Config struct {
	// FS is the filesystem (io/fs) to serve files from, e.g. an embed.FS or
	// os.DirFS(root). When nil, root is served from the real filesystem via
	// http.Dir. When set, root is interpreted as a sub-path within FS (so
	// New("dist", Config{FS: embedFS}) serves the "dist" subtree).
	FS fs.FS
}

// New returns a handler that serves static files. root is a directory path
// served via http.Dir (which prevents path escape); when Config.FS is set,
// root is instead a sub-path within that filesystem.
//
// Any setup failure — an empty root, an invalid Config.FS, or an unexpected
// panic — is logged via slog and degrades to an empty filesystem that responds
// 404 to every request, instead of crashing the process.
//
// Register it on a catch-all route to serve a file tree — the unnamed "/*"
// wildcard or a named {*name} — e.g. app.Get("/*", static.New("./public"))
// or app.Get("/static/*", static.New("./assets")). The path remainder is read
// via r.PathValue("*").
func New(root string, config ...Config) (h httpsrv.Handler) {
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	}

	fsys, err := resolveFS(root, cfg)
	if err != nil {
		slog.Error("static: setup failed, serving empty filesystem (all requests 404)",
			"root", root, "error", err)
		return newHandler(emptyFS)
	}
	return newHandler(fsys)
}

// resolveFS resolves root against an optional Config.FS into an http.FileSystem.
// With Config.FS set, root is a sub-path within it;
// otherwise root is a real directory (http.Dir).
func resolveFS(root string, cfg Config) (http.FileSystem, error) {
	if cfg.FS != nil {
		root = strings.TrimLeft(root, "/")
		if root == "" || root == "." {
			return http.FS(cfg.FS), nil
		}
		sub, err := fs.Sub(cfg.FS, root)
		if err != nil {
			return nil, fmt.Errorf("invalid root %q for Config.FS: %w", root, err)
		}
		return http.FS(sub), nil
	}
	if root == "" {
		return nil, fmt.Errorf("empty root (pass a directory path, or set Config.FS)")
	}
	return http.Dir(root), nil
}

// emptyFS is the fallback filesystem used when New cannot build a real one: it
// holds no files, so a handler backed by it responds 404 to every request.
var emptyFS http.FileSystem = emptyFileSystem{}

// emptyFileSystem is an http.FileSystem that contains no files.
type emptyFileSystem struct{}

func (emptyFileSystem) Open(string) (http.File, error) { return nil, fs.ErrNotExist }

// fileServer serves static files from an http.FileSystem.
type fileServer struct {
	fs http.FileSystem
}

// newHandler wraps a FileSystem in an httpsrv.Handler. The path remainder is
// read from r.PathValue("*"), set by the router for a catch-all route — either a
// named {*name} or the unnamed /* wildcard.
func newHandler(fs http.FileSystem) httpsrv.Handler {
	s := &fileServer{fs: fs}
	return func(c httpsrv.Ctx) error {
		s.serve(c.Response(), c.Request(), c.Request().PathValue("*"))
		return nil
	}
}

// serve writes the file located at subPath (the catch-all remainder, no leading
// "/"), or 404. Directories are not listed (they 404). http.ServeContent handles
// GET/HEAD, ranges, and Last-Modified/ETag.
func (s *fileServer) serve(w http.ResponseWriter, r *http.Request, subPath string) {
	if subPath == "" {
		subPath = "/"
	}
	f, err := s.fs.Open(subPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, st.Name(), st.ModTime(), f)
}
