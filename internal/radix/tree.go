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
//   - A bare "*" segment is an unnamed catch-all (the "/*" wildcard); "/*" and
//     "/static/*" are equivalent to "/{*}" and "/static/{*}" and expose the
//     remainder under the key "*".
//   - Static nodes beat params, which beat catch-alls, at the same position —
//     independent of registration order, so a static route is never shadowed.
//   - A catch-all matches the rest of the path, including an empty remainder:
//     /files/{*path} matches /files (path="") and /* matches / (*=""). A
//     single-segment param still requires a non-empty value.
//   - A catch-all must be the last segment of a pattern.
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
	nodeStatic   nodeType = iota // static segment, e.g. "/doc/"
	nodeParam                    // param segment, e.g. "{id}"
	nodeCatchAll                 // catch-all segment, e.g. "{*path}", matches the rest of the path
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
//
// The common case — a path that is already canonical (which net/http guarantees
// for r.URL.Path, and which holds for every normalized route) — is returned
// as-is without allocating; only genuinely dirty input falls back to path.Clean.
func CleanPath(p string) string {
	if isClean(p) {
		return p
	}
	return path.Clean("/" + p)
}

// isClean reports whether p is already in CleanPath's canonical form, so that
// CleanPath("/"+p) == p and the result can be returned without allocating. A
// canonical path is "/", or a non-empty path starting with '/' whose segments
// are none of "//", ".", or ".." and which carries no trailing slash.
func isClean(p string) bool {
	if p == "/" {
		return true
	}
	if len(p) == 0 || p[0] != '/' {
		return false
	}
	for i := 0; i < len(p); i++ {
		if p[i] != '/' {
			continue
		}
		// A '/' here. Reject a trailing slash, "//", and the "."/".." segments.
		if i+1 >= len(p) {
			return false // trailing slash (p != "/", handled above)
		}
		if p[i+1] == '/' {
			return false // "//"
		}
		if p[i+1] == '.' {
			if i+2 >= len(p) || p[i+2] == '/' {
				return false // "/." segment
			}
			if p[i+2] == '.' && (i+3 >= len(p) || p[i+3] == '/') {
				return false // "/.." segment
			}
		}
	}
	return true
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
			// An empty name is allowed only for a catch-all ("{*}", the unnamed
			// wildcard); a regular "{}" param still needs a name.
			if i == nameStart && !catchAll {
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

// normalizeWildcards rewrites each segment that is exactly "*" into the unnamed
// catch-all "{*}", so the "/*" wildcard ("/*" or "/static/*") behaves like
// "/{*}" / "/static/{*}". Embedded '*' (e.g. "/a*b") and brace params are left
// untouched. Called only on patterns (Insert), not request paths (Search).
func normalizeWildcards(p string) string {
	if !strings.ContainsRune(p, '*') {
		return p
	}
	segs := strings.Split(p, "/")
	changed := false
	for i, s := range segs {
		if s == "*" {
			segs[i] = "{*}"
			changed = true
		}
	}
	if !changed {
		return p
	}
	return strings.Join(segs, "/")
}

// Insert registers a route. The pattern is CleanPath-normalized, "*" wildcard
// segments are rewritten to "{*}", then validated; on error the tree is left
// unchanged.
func (n *Node[T]) Insert(path string, handler T) error {
	path = normalizeWildcards(CleanPath(path))
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
		if strings.HasPrefix(path, n.path) {
			path = path[len(n.path):]
		} else {
			// A catch-all segment is stored under a static node whose path ends
			// in the separator "/" (e.g. the "/files/" node holds the {*path}
			// child of /files/{*path}). A request for the prefix without that
			// trailing slash ("/files") still reaches the catch-all with an empty
			// remainder; anything else is a miss.
			if len(n.path) > 1 && n.path[len(n.path)-1] == '/' && path == n.path[:len(n.path)-1] {
				path = ""
			} else {
				return zero, nil, false
			}
		}
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
		// A catch-all matches an empty remainder too (e.g. /files/{*path} on
		// /files, or /* on /). A param node is never tried here, so it keeps
		// requiring a non-empty value.
		if h, p, found := n.matchEmptyCatchAll(params); found {
			return h, p, true
		}
		return zero, nil, false
	}

	// Try children in priority order — statics before params before catch-alls —
	// so a static route is never shadowed by a param, and neither by a catch-all
	// (independent of registration order). A node has at most one param and one
	// catch-all child, and its static children have distinct first bytes, so a
	// single pass suffices: try the matching static child first, then the param
	// child, then the catch-all. (len(path) > 0 here; the empty case returned.)
	var paramChild, catchChild *Node[T]
	for _, child := range n.children {
		switch child.kind {
		case nodeStatic:
			if child.path[0] == path[0] {
				if h, p, found := child.search(path, params); found {
					return h, p, true
				}
			}
		case nodeParam:
			paramChild = child
		case nodeCatchAll:
			catchChild = child
		}
	}
	if paramChild != nil {
		if h, p, found := paramChild.search(path, params); found {
			return h, p, true
		}
	}
	if catchChild != nil {
		// Catch-all consumes the rest of the path (drop a leading "/").
		rest := path
		if rest[0] == '/' {
			rest = rest[1:]
		}
		if catchChild.isWord {
			return catchChild.handler, append(params, Param{Key: catchAllKey(catchChild.path), Value: rest, CatchAll: true}), true
		}
		return zero, nil, false
	}

	return zero, nil, false
}

// catchAllKey extracts the parameter key from a catch-all node's path:
// "{*name}" -> "name", and the unnamed "{*}" (the "/*" wildcard) -> "*".
func catchAllKey(p string) string {
	// Catch-all node paths are always "{*name}" or "{*}".
	if len(p) >= 3 && p[0] == '{' && p[1] == '*' && p[len(p)-1] == '}' {
		if k := p[2 : len(p)-1]; k != "" {
			return k
		}
	}
	return "*" // unnamed "{*}" (the "/*" wildcard) uses the conventional "*" key
}

// matchEmptyCatchAll matches a catch-all reached with an empty remainder. A
// catch-all matches the rest of the path — including an empty one — so
// /files/{*path} matches /files and /* matches /. The catch-all is reachable
// either as a direct child of n, or behind a lone "/" separator child of n
// (a "/" node only ever precedes a param/catch-all segment, since CleanPath
// collapses duplicate slashes).
func (n *Node[T]) matchEmptyCatchAll(params Params) (T, Params, bool) {
	var zero T
	for _, child := range n.children {
		if child.kind == nodeCatchAll && child.isWord {
			return child.handler, append(params, Param{Key: catchAllKey(child.path), Value: "", CatchAll: true}), true
		}
		if child.kind == nodeStatic && child.path == "/" {
			for _, gc := range child.children {
				if gc.kind == nodeCatchAll && gc.isWord {
					return gc.handler, append(params, Param{Key: catchAllKey(gc.path), Value: "", CatchAll: true}), true
				}
			}
		}
	}
	return zero, nil, false
}
