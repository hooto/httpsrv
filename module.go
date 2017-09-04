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
	"path/filepath"
	"reflect"
	"strings"
)

var (
	DefaultModule = Module{
		name:        "default",
		routes:      []Route{},
		controllers: make(map[string]*controllerType),
		viewpaths:   []string{},
	}
)

type Module struct {
	name        string
	baseuri     string
	routes      []Route
	controllers map[string]*controllerType
	viewpaths   []string
}

func NewModule(name string) Module {

	return Module{
		name:        name,
		routes:      []Route{},
		controllers: make(map[string]*controllerType),
		viewpaths:   []string{},
	}
}

func NewStaticModule(name, path string) Module {

	m := NewModule(name)

	m.RouteSet(Route{
		Type:       RouteTypeStatic,
		Path:       "",
		StaticPath: path,
	})

	return m
}

func (m *Module) RouteSet(r Route) {

	if r.Type == "" {
		r.Type = RouteTypeBasic
	}

	if r.Type != RouteTypeBasic && r.Type != RouteTypeStatic {
		return
	}

	if r.Type == RouteTypeStatic && r.StaticPath == "" {
		return
	}

	r.Path = strings.Trim(r.Path, "/")

	for i, route := range m.routes {

		if route.Path == r.Path {
			m.routes[i] = r
			return
		}
	}

	m.routes = append(m.routes, r)
}

func (m *Module) TemplatePathSet(paths ...string) {

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

func (m *Module) ControllerRegister(c interface{}) {

	cval := reflect.ValueOf(c)
	if !cval.IsValid() {
		return
	}

	var (
		t       = reflect.TypeOf(c)
		elem    = t.Elem()
		methods = []string{}
	)

	for i := 0; i < elem.NumMethod(); i++ {

		m := elem.Method(i)

		if len(m.Name) > 6 && m.Name[len(m.Name)-6:] == "Action" {
			methods = append(methods, m.Name)
		}
	}

	cm := &controllerType{
		Type:              elem,
		Methods:           []string{},
		ControllerIndexes: findControllers(elem),
	}

	for _, method := range methods {

		if m := cval.MethodByName(method); m.IsValid() {
			cm.Methods = append(cm.Methods, method)
		}
	}

	m.controllers[elem.Name()] = cm
}
