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
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type rootRouter struct {
	mu   sync.RWMutex
	node *routeNode
}

type routeNode struct {
	name string

	stdNodes map[string]*routeNode
	varNodes []*routeNode

	patFields []string
	patParams []int
	patFieldN int

	rawFields []string

	handler *regHandler
}

type routeContext struct {
	hits []*routeNode
}

type regRouter struct {
	method     string
	pattern    string
	params     map[string]string
	controller string
	action     string
}

func (it *rootRouter) add(pattern string, h *regHandler) {

	var (
		rawPath   = strings.Trim(filepath.Clean("/"+pattern), "/")
		rawFields = strings.Split(rawPath, "/")
		patFields = append([]string{}, rawFields...)
		patParams = make([]bool, len(rawFields))
	)

	for i, name := range patFields {

		if len(name) > 1 &&
			name[0] == ':' {

			patFields[i] = name[1:len(name)]
			patParams[i] = true

		} else if len(name) > 2 &&
			name[0] == '{' &&
			name[len(name)-1] == '}' {

			patFields[i] = name[1 : len(name)-1]
			patParams[i] = true

		} else {
			patParams[i] = false
		}

		patFields[i] = strings.ToLower(patFields[i])
	}

	it.mu.Lock()
	defer it.mu.Unlock()

	if it.node == nil {
		it.node = &routeNode{}
	}

	it.node.add(0, patFields, patParams, rawFields, h)
}

func (it *rootRouter) find(r *http.Request) (*regHandler, string, string) {

	var (
		ctx = &routeContext{
			hits: make([]*routeNode, 0, 4),
		}
		urlPath      = filepath.Clean("/" + r.URL.Path)
		urlRoutePath = urlPath
		rawPath      = strings.Trim(urlPath, "/")
		patFields    = strings.Split(strings.ToLower(rawPath), "/")
	)

	it.mu.RLock()
	it.node.find(ctx, patFields)
	it.mu.RUnlock()

	if len(ctx.hits) > 0 {

		sort.Slice(ctx.hits, func(i, j int) bool {
			return ctx.hits[i].patFieldN > ctx.hits[j].patFieldN
		})

		if len(ctx.hits[0].patParams) > 0 {
			for _, i := range ctx.hits[0].patParams {
				r.SetPathValue(ctx.hits[0].patFields[i], patFields[i])
			}
		}

		if ctx.hits[0].patFieldN <= len(patFields) {
			urlRoutePath = "/" + strings.Join(patFields[:ctx.hits[0].patFieldN], "/")
		}

		return ctx.hits[0].handler, urlPath, urlRoutePath
	}

	if n, ok := it.node.stdNodes[""]; ok &&
		n.handler != nil &&
		n.handler.handlerController != nil {
		return n.handler, urlPath, urlRoutePath
	}

	return defaultHandlers[0], urlPath, urlRoutePath
}

func (it *routeNode) add(index int, patFields []string, patParams []bool,
	rawFields []string, h *regHandler) *routeNode {

	var (
		name = patFields[index]
		node *routeNode
	)

	if !patParams[index] {

		if it.stdNodes == nil {
			it.stdNodes = map[string]*routeNode{}
		}

		if p, ok := it.stdNodes[name]; !ok {
			node = &routeNode{
				name: name,
			}
			it.stdNodes[name] = node
		} else {
			node = p
		}

	} else {

		for _, p := range it.varNodes {
			if name == p.name {
				node = p
				break
			}
		}

		if node == nil {
			node = &routeNode{
				name: name,
			}
			it.varNodes = append(it.varNodes, node)
		}
	}

	if index+1 < len(patFields) {
		return node.add(index+1, patFields, patParams, rawFields, h)
	} else {

		node.handler = h
		node.patFields = patFields
		node.patFieldN = len(patFields)
		node.rawFields = rawFields
		node.patParams = []int{}

		for i, b := range patParams {
			if b {
				node.patParams = append(node.patParams, i)
			}
		}

		defaultLogger.Infof("httpsrv: route depth %d, handler %s", node.patFieldN, h.info())
	}

	return node
}

func (it *routeNode) find(ctx *routeContext, netxFields []string) bool {

	for _, n := range it.varNodes {

		if len(netxFields) == 1 {
			if n.handler != nil {
				ctx.hits = append(ctx.hits, n)
				return true
			}
		} else {
			if n.find(ctx, netxFields[1:]) {
				return true
			}
		}
	}

	if it.stdNodes != nil {

		if n, ok := it.stdNodes[netxFields[0]]; ok {

			if n.handler != nil {
				ctx.hits = append(ctx.hits, n)
			}

			if len(netxFields) > 1 {
				if n.find(ctx, netxFields[1:]) {
					return true
				}
			} else if n.handler != nil {
				return true
			}
		}

		if false && netxFields[0] != "index" {

			if n, ok := it.stdNodes["index"]; ok &&
				n.handler.handlerController != nil {

				if n.handler != nil {
					ctx.hits = append(ctx.hits, n)
				}

				if len(netxFields) > 1 {
					if n.find(ctx, netxFields[1:]) {
						return true
					}
				} else if n.handler != nil {
					return true
				}
			}
		}
	}

	return false
}
