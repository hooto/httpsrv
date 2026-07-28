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
// New returns an httpsrv.Handler that serves files from a directory; register it
// on a catch-all route so it serves a whole file tree:
//
//	import "github.com/hooto/httpsrv/v2"
//	"github.com/hooto/httpsrv/v2/middleware/static"
//
//	app.Get("/static/{*path}", static.New("./assets"))
//	// GET /static/css/main.css  ->  ./assets/css/main.css
//
// For an embedded filesystem use FS. The catch-all value (the part of the path
// after the prefix) is set by the httpsrv router under the conventional key "*"
// (fiber's c.Params("*") style), which the handler reads via r.PathValue("*").
// Use app.All instead of app.Get to serve static files for every HTTP method.
package static

import (
	"net/http"

	"github.com/hooto/httpsrv/v2"
)

// fileServer serves static files from an http.FileSystem.
type fileServer struct {
	fs http.FileSystem
}

// New returns a handler that serves files from the directory at root, mirroring
// gofiber v3's static.New(root) -> Handler. Path escape is prevented by
// http.Dir. Register it on a catch-all route, e.g.
// app.Get("/static/{*path}", static.New("./assets")).
func New(root string) httpsrv.Handler {
	return newHandler(http.Dir(root))
}

// FS returns a handler that serves files from fs (e.g. http.FS(anEmbed.FS)) for
// serving embedded content.
func FS(fs http.FileSystem) httpsrv.Handler {
	return newHandler(fs)
}

// newHandler wraps a FileSystem in an httpsrv.Handler. The catch-all value is
// read from r.PathValue("*"), set by the router for a {*name} route.
func newHandler(fs http.FileSystem) httpsrv.Handler {
	s := &fileServer{fs: fs}
	return func(c httpsrv.Ctx) error {
		s.serve(c.Response(), c.Request(), c.Request().PathValue("*"))
		return nil
	}
}

// serve writes the file located at subPath (the URL path with the mount prefix
// removed; the catch-all value has no leading "/"), or 404. Directories are not
// listed (they 404). http.ServeContent handles GET/HEAD, ranges, and
// Last-Modified/ETag.
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
