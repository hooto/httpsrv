// Copyright 2015 Eryx <evorui аt gmаil dοt cοm>, All rights reserved.
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
		controllers: make(map[string]interface{}),
		viewpaths:   []string{},
	}

	DefaultModules = []*Module{}
)

type Module struct {
	Path        string
	controllers map[string]interface{}
	viewpaths   []string
	viewfss     []http.FileSystem
	handlers    []*regHandler
}

func NewModule() *Module {
	return &Module{
		controllers: make(map[string]interface{}),
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

func (m *Module) RegisterStaticFileSystem(pattern string, fs http.FileSystem) {
	m.handlers = append(m.handlers, &regHandler{
		pattern: pattern,
		handlerStatic: &handlerStaticFile{
			binFs: fs,
		},
	})
}

func (m *Module) RegisterStaticFilepath(pattern, path string) {
	m.handlers = append(m.handlers, &regHandler{
		pattern: pattern,
		handlerStatic: &handlerStaticFile{
			filepath: path,
		},
	})
}

func (m *Module) RegisterController(c interface{}) {

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

		m.handlers = append(m.handlers, &regHandler{
			pattern: strings.ToLower(fmt.Sprintf("/%s/%s", elem.Name(), am.Name[:len(am.Name)-6])),
			handlerController: &handlerController{
				Name:        elem.Name(),
				ActionName:  am.Name[:len(am.Name)-6],
				ctrlType:    elem,
				ctrlIndexes: indexes,
			},
		})
	}
}
