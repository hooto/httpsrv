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
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
)

var (
	DefaultModule = &Module{
		idxHandlers: make(map[string]*regHandler),
		routes:      make(map[string]*regRouter),
	}

	DefaultModules = []*Module{}
)

type Module struct {
	Path        string
	viewpaths   []string
	viewfss     []http.FileSystem
	handlers    []*regHandler
	idxHandlers map[string]*regHandler
	routes      map[string]*regRouter
}

func NewModule() *Module {
	return &Module{
		idxHandlers: make(map[string]*regHandler),
		routes:      make(map[string]*regRouter),
	}
}

func (m *Module) SetTemplatePath(paths ...string) {

	for _, path := range paths {

		path = filepath.Clean(path)

		added := false

		for _, prev := range m.viewpaths {

			if path == prev {
				added = true
				break
			}
		}

		if !added {
			m.viewpaths = append(m.viewpaths, path)
		}
	}
}

func (m *Module) SetTemplateFileSystem(fss ...http.FileSystem) {

	for _, fs := range fss {

		added := false

		for _, prev := range m.viewfss {

			if fs == prev {
				added = true
				break
			}
		}

		if !added {
			m.viewfss = append(m.viewfss, fs)
		}
	}
}

func (m *Module) SetRoute(pattern string, params map[string]string) {
	pattern = filepath.Clean(pattern)
	if params == nil {
		params = make(map[string]string)
	}
	m.routes[pattern] = &regRouter{
		pattern: pattern,
		params:  params,
	}
}

func (m *Module) RegisterFileServer(pattern, path string, fs http.FileSystem) {
	m.handlers = append(m.handlers, &regHandler{
		pattern: pattern,
		handlerFileServer: &handlerFileServer{
			filepath: path,
			binFs:    fs,
		},
	})
}

func (m *Module) RegisterController(args ...interface{}) {
	for _, c := range args {
		m.registerController(c)
	}
}

func (m *Module) registerController(c interface{}) {

	if c == nil {
		return
	}

	cval := reflect.ValueOf(c)
	if !cval.IsValid() {
		return
	}

	var (
		t       = reflect.TypeOf(c)
		elem    = t.Elem()
		indexes = findControllers(elem)
	)

	for i := 0; i < elem.NumMethod(); i++ {

		am := elem.Method(i)

		if len(am.Name) <= 6 || !strings.HasSuffix(am.Name, "Action") {
			continue
		}

		if vm := cval.MethodByName(am.Name); !vm.IsValid() {
			continue
		}

		hc := &handlerController{
			Name:        elem.Name(),
			ActionName:  am.Name[:len(am.Name)-6],
			ctrlType:    elem,
			ctrlIndexes: indexes,
		}

		h := &regHandler{
			pattern:           controllerActionPattern(hc.Name, hc.ActionName),
			handlerController: hc,
		}

		m.handlers = append(m.handlers, h)

		m.idxHandlers[h.pattern] = h

		if am.Name == "IndexAction" {

			m.handlers = append(m.handlers, &regHandler{
				pattern:           strings.ToLower(fmt.Sprintf("/%s", elem.Name())),
				handlerController: hc,
			})

			if elem.Name() == "Index" {
				m.handlers = append(m.handlers, &regHandler{
					pattern:           "/",
					handlerController: hc,
				})
			}
		}
	}
}
