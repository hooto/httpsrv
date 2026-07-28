// Copyright 2015 Eryx <evoruni at gmail dot com>, All rights reserved.
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

// Package radix implements a generic radix tree for HTTP route matching.
//   - Nodes are static (byte-exact), param ("{name}", matches [^/]+), or
//     catch-all ("{*name}", matches the rest of the path including slashes).
//   - Only brace params "{name}" / "{*name}" are recognized.
//   - Static nodes beat params, which beat catch-alls, at the same position —
//     independent of registration order, so a static route is never shadowed.
//   - A catch-all "{*name}" must be the last segment of a pattern.
//   - Insert and Search both CleanPath-normalize; both must use the same rule.
//
// Matching is case-sensitive. The tree is generic in the handler value T, so it
// stores whatever the caller needs (a handler in httpsrv, a string in tests).
// It never compares two T values, so T may be a func type.
package radix

import (
	"fmt"
	"path"
	"strings"
)

// nodeType is the kind of a tree node.
type nodeType uint8

const (
	nodeStatic nodeType = iota // static segment, e.g. "/doc/"
	nodeParam                  // param segment, e.g. "{id}"
	nodeCatchAll               // catch-all segment, e.g. "{*path}", matches the rest of the path
)

// Param is a single extracted path parameter.
type Param struct {
	Key      string
	Value    string
	CatchAll bool // true for a {*name} catch-all match
}

// Params is a list of path parameters.
type Params []Param

// Node is a radix tree node. The handler value has type T.
type Node[T any] struct {
	kind     nodeType   // node kind
	path     string     // path fragment; for param nodes the full "{name}" declaration
	isWord   bool       // true if this node is a registered route end
	handler  T          // handler stored at this node
	children []*Node[T] // child nodes; statics are searched before params
}

// New returns a new empty radix tree root of value type T.
func New[T any]() *Node[T] { return &Node[T]{} }

// CleanPath normalizes a path: guarantees a leading '/', collapses duplicate
// slashes, and resolves '.'/'..'. Uses path (not path/filepath) for URL
// semantics on every platform. Insert and Search must use the same rule.
func CleanPath(p string) string {
	return path.Clean("/" + p)
}

// validatePattern checks route pattern syntax. A "{name}" param must:
//   - have balanced, non-nested braces;
//   - have a non-empty name containing no '/';
//   - be the only param in its segment (no "{a}{b}").
//
// A "{*name}" catch-all additionally must be the last segment of the pattern
// (nothing may follow its closing '}'). Rejecting these at Insert avoids nodes
// with broken match semantics.
func validatePattern(p string) error {
	inParam := false
	catchAll := false
	paramInSegment := false
	nameStart := 0
	for i := 0; i < len(p); i++ {
		switch p[i] {
		case '{':
			if inParam {
				return fmt.Errorf("invalid route pattern %q: nested '{'", p)
			}
			if paramInSegment {
				return fmt.Errorf("invalid route pattern %q: more than one parameter in a segment", p)
			}
			inParam = true
			if i+1 < len(p) && p[i+1] == '*' {
				catchAll = true
				nameStart = i + 2 // skip "{*"
			} else {
				catchAll = false
				nameStart = i + 1
			}
		case '}':
			if !inParam {
				return fmt.Errorf("invalid route pattern %q: unmatched '}'", p)
			}
			if i == nameStart {
				return fmt.Errorf("invalid route pattern %q: empty parameter name", p)
			}
			if catchAll && i != len(p)-1 {
				return fmt.Errorf("invalid route pattern %q: catch-all must be the last segment", p)
			}
			inParam = false
			paramInSegment = true
		case '/':
			if inParam {
				return fmt.Errorf("invalid route pattern %q: '/' inside parameter name", p)
			}
			paramInSegment = false
		}
	}
	if inParam {
		return fmt.Errorf("invalid route pattern %q: unclosed '{'", p)
	}
	return nil
}

// Insert registers a route. The pattern is CleanPath-normalized then validated;
// on error the tree is left unchanged.
func (n *Node[T]) Insert(path string, handler T) error {
	path = CleanPath(path)
	if err := validatePattern(path); err != nil {
		return err
	}
	// Empty tree: build from scratch, skipping a meaningless prefix compare.
	if n.path == "" && len(n.children) == 0 {
		return n.insertPath(path, handler)
	}
	return n.insert(path, handler)
}

// insertPath builds a fresh subtree for a path that may contain "{name}" params
// (matched as [^/]+ in Search). On a malformed fragment it returns an error
// without mutating n: the child subtree is built first, then n is filled in.
func (n *Node[T]) insertPath(path string, handler T) error {
	pos := strings.IndexByte(path, '{')

	// Pure static path.
	if pos == -1 {
		n.path = path
		n.kind = nodeStatic
		n.isWord = true
		n.handler = handler
		return nil
	}

	// 1. Static part before '{'.
	if pos > 0 {
		child := &Node[T]{}
		if err := child.insertPath(path[pos:], handler); err != nil {
			return err
		}
		n.path = path[:pos]
		n.kind = nodeStatic
		n.children = append(n.children, child)
		return nil
	}

	// 2. Path starts with '{': read the param name up to '}'.
	end := strings.IndexByte(path, '}')
	if end == -1 {
		return fmt.Errorf("invalid route pattern %q: unclosed '{'", path)
	}
	if end == 1 { // "{}": empty name
		return fmt.Errorf("invalid route pattern %q: empty parameter name", path)
	}

	// Catch-all "{*name}": matches the rest of the path. Validation guarantees
	// it is terminal (nothing follows '}'), so it is always a leaf.
	if path[1] == '*' {
		n.path = path[:end+1] // "{*name}", braces included
		n.kind = nodeCatchAll
		n.isWord = true
		n.handler = handler
		return nil
	}

	if end+1 < len(path) {
		// More path follows (e.g. "/profile"): build the child subtree first, then fill n.
		child := &Node[T]{}
		if err := child.insertPath(path[end+1:], handler); err != nil {
			return err
		}
		n.path = path[:end+1] // "{id}", braces included
		n.kind = nodeParam
		n.isWord = false
		n.children = append(n.children, child)
	} else {
		// Param ends the path.
		n.path = path[:end+1]
		n.kind = nodeParam
		n.isWord = true
		n.handler = handler
	}
	return nil
}

// insert adds path into an existing tree, handling common prefixes and node
// splits. The returned error only propagates insertPath validation failures
// (already checked at the Insert entry; not hit on the normal path).
func (n *Node[T]) insert(path string, handler T) error {
	cp := 0
	switch n.kind {
	case nodeStatic:
		// Static node: find the longest common prefix (stop at '{', the param start).
		for cp < len(n.path) && cp < len(path) && n.path[cp] == path[cp] && path[cp] != '{' {
			cp++
		}
	case nodeParam:
		// Param node: braces are self-delimiting, so {id} can't prefix another
		// param; consume the declaration if it matches, avoiding a duplicate
		// {id} nested under {id}.
		if strings.HasPrefix(path, n.path) {
			cp = len(n.path)
		}
	case nodeCatchAll:
		// A catch-all is terminal and matches the entire remainder, so two
		// distinct catch-all declarations at the same position conflict. The
		// same declaration re-registered just overwrites the handler.
		if path == n.path {
			cp = len(n.path)
		} else {
			return fmt.Errorf("invalid route pattern: conflicting catch-all %q vs %q", path, n.path)
		}
	}

	// Split the static node.
	if cp < len(n.path) && n.kind == nodeStatic {
		child := &Node[T]{
			path:     n.path[cp:],
			kind:     n.kind,
			isWord:   n.isWord,
			handler:  n.handler,
			children: n.children,
		}
		n.path = n.path[:cp]
		n.isWord = false
		var zero T
		n.handler = zero // zero value of T (node is no longer a route end)
		n.children = []*Node[T]{child}
	}

	// Insert the remaining path.
	if cp < len(path) {
		rem := path[cp:]

		for _, child := range n.children {
			// Match by node kind so a param, a catch-all, and a static at the
			// same position never collide on the shared '{' first byte.
			if rem[0] == '{' {
				if len(rem) > 1 && rem[1] == '*' {
					if child.kind == nodeCatchAll {
						return child.insert(rem, handler)
					}
					continue
				}
				if child.kind == nodeParam {
					return child.insert(rem, handler)
				}
				continue
			}
			if child.kind == nodeStatic && child.path[0] == rem[0] {
				return child.insert(rem, handler)
			}
		}

		// No matching child: start a new branch.
		child := &Node[T]{}
		if err := child.insertPath(rem, handler); err != nil {
			return err
		}
		n.children = append(n.children, child)
	} else if cp == len(path) {
		// Exact match at this node.
		n.isWord = true
		n.handler = handler
	}
	return nil
}

// Search looks up a route, returning the matching handler and any extracted
// params. The path is normalized once here; the recursion in search avoids
// re-cleaning already-consumed fragments (which would wrongly re-add a '/').
func (n *Node[T]) Search(path string, params Params) (T, Params, bool) {
	return n.search(CleanPath(path), params)
}

// search recurses on path, the normalized remainder after each matched prefix.
func (n *Node[T]) search(path string, params Params) (T, Params, bool) {
	var zero T
	if n.kind == nodeStatic {
		// Static node: must be a full prefix match.
		if !strings.HasPrefix(path, n.path) {
			return zero, nil, false
		}
		path = path[len(n.path):]
	} else if n.kind == nodeParam {
		// Param matches [^/]+: consume up to the next '/'.
		end := 0
		for end < len(path) && path[end] != '/' {
			end++
		}

		// Param name is the brace contents.
		key := n.path
		if len(key) >= 2 && key[0] == '{' && key[len(key)-1] == '}' {
			key = key[1 : len(key)-1]
		}
		params = append(params, Param{Key: key, Value: path[:end]})
		path = path[end:]
	}

	// Request path fully consumed.
	if len(path) == 0 {
		if n.isWord {
			return n.handler, params, true
		}
		return zero, nil, false
	}

	// Try children: statics first, then params, then catch-alls — so a static
	// route is never shadowed by a param, and neither by a catch-all
	// (order-independent).
	for _, child := range n.children {
		if child.kind == nodeStatic && len(path) > 0 && child.path[0] == path[0] {
			if h, p, found := child.search(path, params); found {
				return h, p, true
			}
		}
	}
	for _, child := range n.children {
		if child.kind == nodeParam {
			if h, p, found := child.search(path, params); found {
				return h, p, true
			}
		}
	}
	for _, child := range n.children {
		if child.kind == nodeCatchAll {
			// Catch-all consumes the rest of the path (drop the leading "/").
			rest := path
			if len(rest) > 0 && rest[0] == '/' {
				rest = rest[1:]
			}
			key := child.path
			if len(key) >= 3 && key[0] == '{' && key[1] == '*' && key[len(key)-1] == '}' {
				key = key[2 : len(key)-1] // strip "{*" and "}"
			}
			if child.isWord {
				return child.handler, append(params, Param{Key: key, Value: rest, CatchAll: true}), true
			}
			return zero, nil, false
		}
	}

	return *new(T), nil, false
}
