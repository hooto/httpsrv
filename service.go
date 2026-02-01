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
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Service struct {
	mu sync.RWMutex

	Config  Config
	Filters []Filter

	router *rootRouter

	server *http.Server

	modules  []*Module
	handlers []*regHandler

	TemplateLoader *TemplateLoader
}

var (
	DefaultService = NewService()
)

func NewService() *Service {

	return &Service{

		Config: DefaultConfig,

		Filters: DefaultFilters,

		modules: DefaultModules,

		handlers: defaultHandlers,

		router: &rootRouter{},

		TemplateLoader: newTemplateLoader(),
	}
}

func (s *Service) regHandler(h *regHandler) {

	h.service = s
	h.pattern = filepath.Clean("/" + h.pattern)

	if !strings.HasSuffix(h.pattern, "/") {
		h.pattern += "/"
	}

	for i, v := range s.handlers {
		if v.pattern == h.pattern {
			s.handlers[i] = h
			slog.Info("route reset", "pattern", h.pattern)
			return
		}
	}
	s.handlers = append(s.handlers, h)
}

/**
func (s *Service) HandleHttp(method, pattern string, fn func(ctx *Context) error) {
	if fn == nil {
		return
	}
	method = strings.ToUpper(method)
	switch method {
	case "GET", "POST", "PUT", "DELETE":
		//
	default:
		method = ""
	}
	s.regHandler(&regHandler{
		method:         method,
		pattern:        pattern,
		handlerContext: fn,
	})
}
*/

func (s *Service) HandleFunc(pattern string, h func(w http.ResponseWriter, r *http.Request)) {
	s.regHandler(&regHandler{
		pattern:     pattern,
		handlerFunc: h,
	})
}

func (s *Service) HandleModule(pattern string, mod *Module) {

	mod1 := &Module{
		Path:      filepath.Clean(pattern),
		viewpaths: mod.viewpaths,
		viewfss:   mod.viewfss,
	}

	modr := &handlerModuler{
		actions: map[string]*handlerController{},
	}

	for _, h := range mod.handlers {

		if h.handlerController != nil {
			//
			h.handlerController.ModPath = mod1.Path
			s.regHandler(&regHandler{
				pattern:           mod1.Path + "/" + h.pattern,
				handlerController: h.handlerController,
			})
			//
			modr.actions[h.pattern] = h.handlerController

		} else if h.handlerFileServer != nil {
			//
			h.handlerFileServer.filepath = filepath.Clean(h.handlerFileServer.filepath)
			s.regHandler(&regHandler{
				pattern:           filepath.Clean(mod1.Path + "/" + h.pattern),
				handlerFileServer: h.handlerFileServer,
			})
		}
	}

	for _, r := range mod.routes {
		//
		if strings.Contains(r.pattern, "/{controller}/{action}") {
			s.regHandler(&regHandler{
				pattern:        filepath.Clean(mod1.Path + "/" + r.pattern),
				handlerModuler: modr,
			})
			continue
		}
		//
		ctrl, ok := r.params["controller"]
		if !ok {
			ctrl = "Index"
		}
		action, ok := r.params["action"]
		if !ok {
			action = "Index"
		}
		k := controllerActionPattern(ctrl, action)
		h, ok := mod.idxHandlers[k]
		if !ok || h.handlerController == nil {
			continue
		}
		h.handlerController.ModPath = mod1.Path
		s.regHandler(&regHandler{
			pattern:           filepath.Clean(mod1.Path + "/" + r.pattern),
			handlerController: h.handlerController,
		})
	}

	s.TemplateLoader.Set(mod1.Path, mod1.viewpaths, mod1.viewfss)

	for i, pmod := range s.modules {
		if pmod.Path == mod1.Path {
			s.modules[i] = mod1
			mod1 = nil
			break
		}
	}
	if mod1 != nil {
		s.modules = append(s.modules, mod1)
	}

	sort.Slice(s.modules, func(i, j int) bool {
		return strings.Compare(s.modules[i].Path, s.modules[j].Path) > 0
	})
}

func (s *Service) Start(args ...interface{}) error {

	//
	if s.Config.UrlBasePath != "" {
		s.Config.UrlBasePath = strings.TrimRight(filepath.Clean("/"+s.Config.UrlBasePath), "/")
	}

	//
	network, localAddr := "tcp", s.Config.HttpAddr

	// If the port is zero, treat the address as a fully qualified local address.
	// This address must be prefixed with the network type followed by a colon,
	// e.g. unix:/tmp/app.socket or tcp6:::1 (equivalent to tcp6:0:0:0:0:0:0:0:1)
	if s.Config.HttpPort == 0 || strings.HasPrefix(s.Config.HttpAddr, "unix:") {
		parts := strings.SplitN(s.Config.HttpAddr, ":", 2)
		if len(parts) > 0 {
			network = parts[0]
		}
		if len(parts) > 1 {
			localAddr = parts[1]
		}
	} else {
		localAddr += fmt.Sprintf(":%d", s.Config.HttpPort)
	}

	if len(args) > 0 {
		for _, arg := range args {
			switch arg := arg.(type) {
			case string:
				if host, port, err := net.SplitHostPort(arg); err == nil {
					localAddr = host + ":" + port
				}
			}
		}
	}

	if network != "unix" && network != "tcp" {
		slog.Error("httpsrv unknown network", "network", network)
		return errors.New("invalid network " + network)
	}

	//
	if network == "unix" {
		// TODO already in use
		os.Remove(localAddr)
	}

	//
	if s.Config.HttpTimeout == 0 {
		s.Config.HttpTimeout = 10
	} else if s.Config.HttpTimeout < 1 {
		s.Config.HttpTimeout = 1
	} else if s.Config.HttpTimeout > 600 {
		s.Config.HttpTimeout = 600
	}

	//
	sort.Slice(s.handlers, func(i, j int) bool {
		return strings.Compare(s.handlers[i].pattern, s.handlers[j].pattern) < 0
	})
	for _, h := range s.handlers {
		if s.Config.UrlBasePath != "" {
			h.pattern = s.Config.UrlBasePath + h.pattern
		}
		// s.logger.Infof("httpsrv: reg handler #%02d, path %s", i, h.pattern)
		s.router.add(h.pattern, h)
	}

	//

	s.server = &http.Server{
		Addr:           localAddr,
		ReadTimeout:    time.Duration(s.Config.HttpTimeout) * time.Second,
		WriteTimeout:   time.Duration(s.Config.HttpTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
		Handler:        &rootHandler{s},
	}

	//
	listener, err := net.Listen(network, localAddr)
	if err != nil {
		slog.Error("httpsrv net listen error", "err", err)
		return err
	}
	slog.Info("httpsrv listening", "network", network, "address", localAddr)

	if network == "unix" {
		os.Chmod(localAddr, 0770)
	}

	//
	if err = s.server.Serve(listener); err != nil {
		slog.Error("httpsrv start server fail", "err", err)
	}

	return err
}

func (s *Service) Stop() error {
	return nil
}
