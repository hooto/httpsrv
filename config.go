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

import "time"

// Version is the library version.
const Version = "2.0.0-beta.1"

// Config holds server settings, applied at construction via WithConfig. Zero
// fields keep the defaults set in New.
type Config struct {
	// Addr is the listen address (default ":8080"; a string passed to Run
	// overrides it).
	Addr string

	// Timeouts; defaults are Read/Write 60s, ReadHeader 10s. Set WriteTimeout
	// to a large value (or use streaming) for long responses.
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	ReadHeaderTimeout time.Duration

	// MaxHeaderBytes caps request header size (default 1 MiB).
	MaxHeaderBytes int

	// Views is the interface that wraps the Render function. Set it to a
	// template engine (e.g. a *Renderer from TemplatesDir/TemplatesFS) so
	// Handler code can call Ctx.Render. A custom engine implementing Views may
	// be plugged in here directly. Equivalent to WithViews.
	//
	// Default: nil
	Views Views `json:"-"`
}

// WithConfig applies server settings from cfg. Only non-zero fields override the
// defaults. Example:
//
//	app := httpsrv.New(httpsrv.WithConfig(httpsrv.Config{
//	    Addr:        ":3000",
//	    ReadTimeout: 30 * time.Second,
//	}))
func WithConfig(cfg Config) Option {
	return func(a *app) {
		if cfg.Addr != "" {
			a.addr = cfg.Addr
		}
		if cfg.ReadTimeout != 0 {
			a.readTimeout = cfg.ReadTimeout
		}
		if cfg.WriteTimeout != 0 {
			a.writeTimeout = cfg.WriteTimeout
		}
		if cfg.ReadHeaderTimeout != 0 {
			a.readHeaderTimeout = cfg.ReadHeaderTimeout
		}
		if cfg.MaxHeaderBytes != 0 {
			a.maxHeaderBytes = cfg.MaxHeaderBytes
		}
		// Views is an interface; a typed-nil (e.g. (*Renderer)(nil)) is treated
		// as "no engine" by setViews, so passing one through is safe.
		if cfg.Views != nil {
			a.setViews(cfg.Views)
		}
	}
}
