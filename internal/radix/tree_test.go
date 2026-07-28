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

package radix

import (
	"testing"
)

// paramsEqual reports whether two param slices are element-wise equal
// (nil and a zero-length slice are treated as equal).
func paramsEqual(a, b Params) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Key != b[i].Key || a[i].Value != b[i].Value || a[i].CatchAll != b[i].CatchAll {
			return false
		}
	}
	return true
}

// mustInsert registers a route, failing the test on error.
func mustInsert(t *testing.T, root *Node[string], pattern, handler string) {
	t.Helper()
	if err := root.Insert(pattern, handler); err != nil {
		t.Fatalf("Insert(%q): %v", pattern, err)
	}
}

// searchCase is one Search expectation: want is the expected handler, or "" to
// expect not-found.
type searchCase struct {
	path   string
	want   string
	params Params
}

// checkSearch asserts root.Search(c.path) matches c (handler + params), or is
// not-found when c.want == "".
func checkSearch(t *testing.T, root *Node[string], c searchCase) {
	t.Helper()
	t.Run(c.path, func(t *testing.T) {
		h, params, found := root.Search(c.path, nil)
		if found != (c.want != "") {
			t.Fatalf("Search(%q): found=%v params=%v", c.path, found, params)
		}
		if !found {
			return
		}
		if h != c.want {
			t.Errorf("handler=%q, want %q", h, c.want)
		}
		if !paramsEqual(params, c.params) {
			t.Errorf("params=%v, want %v", params, c.params)
		}
	})
}

// runSearchTests runs checkSearch over a slice of cases.
func runSearchTests(t *testing.T, root *Node[string], cases []searchCase) {
	t.Helper()
	for _, c := range cases {
		checkSearch(t, root, c)
	}
}

func TestRouterTree(t *testing.T) {
	root := &Node[string]{}
	mustInsert(t, root, "/doc/name", "StaticNameHandler")
	mustInsert(t, root, "/doc/{id}", "ParamIdHandler")
	mustInsert(t, root, "/doc/{id}/profile", "ProfileHandler")

	runSearchTests(t, root, []searchCase{
		{"/doc/name", "StaticNameHandler", nil},
		{"/doc/12345", "ParamIdHandler", Params{{"id", "12345", false}}},
		// A param value keeps its '.' (matches [^/]+, not dot-delimited).
		{"/doc/news.html", "ParamIdHandler", Params{{"id", "news.html", false}}},
		{"/doc/999/profile", "ProfileHandler", Params{{"id", "999", false}}},
		{"/doc/abc", "ParamIdHandler", Params{{"id", "abc", false}}},
		{"/doc/", "", nil},
		{"/doc/123/edit", "", nil},
		{"/missing", "", nil},
	})
}

// Regression: a {param} matches the full [^/]+ segment. Registering
// /user/{username}.json must not stop /user/{username} matching a dotted value.
func TestRouteTreeParamFullSegment(t *testing.T) {
	root := &Node[string]{}
	mustInsert(t, root, "/user/{username}", "UserHandler")
	mustInsert(t, root, "/user/{username}.json", "UserJsonHandler")
	checkSearch(t, root, searchCase{"/user/john.doe", "UserHandler", Params{{"username", "john.doe", false}}})
}

// A static route beats a param route at the same position, independent of order.
func TestRouteTreeStaticPriority(t *testing.T) {
	want := []struct {
		path    string
		handler string
		params  Params
	}{
		{"/users/profile", "UserProfileHandler", nil},
		{"/users/42", "UserParamHandler", Params{{"id", "42", false}}},
	}
	for _, order := range [2][2]string{
		{"/users/{id}", "/users/profile"},
		{"/users/profile", "/users/{id}"},
	} {
		root := &Node[string]{}
		for _, p := range order {
			h := "UserParamHandler"
			if p == "/users/profile" {
				h = "UserProfileHandler"
			}
			mustInsert(t, root, p, h)
		}
		for _, w := range want {
			h, params, found := root.Search(w.path, nil)
			if !found || h != w.handler || !paramsEqual(params, w.params) {
				t.Errorf("order=%v Search(%q): h=%q params=%v found=%v", order, w.path, h, params, found)
			}
		}
	}
}

func TestRouteTreeMultiParams(t *testing.T) {
	root := &Node[string]{}
	mustInsert(t, root, "/posts/{post_id}/comments/{comment_id}", "CommentHandler")

	runSearchTests(t, root, []searchCase{
		{"/posts/5/comments/22", "CommentHandler", Params{{"post_id", "5", false}, {"comment_id", "22", false}}},
		// {post_id} is not terminal (has a child), so a bare id must not match.
		{"/posts/5", "", nil},
	})
}

// Re-registering a pattern replaces its handler (last-write-wins).
func TestRouteTreeOverwrite(t *testing.T) {
	t.Run("static", func(t *testing.T) {
		root := &Node[string]{}
		mustInsert(t, root, "/page", "Old")
		mustInsert(t, root, "/page", "New")
		checkSearch(t, root, searchCase{"/page", "New", nil})
	})
	t.Run("param", func(t *testing.T) {
		root := &Node[string]{}
		mustInsert(t, root, "/item/{id}", "Old")
		mustInsert(t, root, "/item/{id}", "New")
		checkSearch(t, root, searchCase{"/item/7", "New", Params{{"id", "7", false}}})
	})
}

// Insert and Search normalize identically (path.Clean): repeated slashes
// collapse, '.'/'..' resolve, trailing slash equals none.
func TestRouteTreePathClean(t *testing.T) {
	root := &Node[string]{}
	mustInsert(t, root, "//a//b//", "H") // registered dirty; normalizes to "/a/b"
	for _, p := range []string{"//a//b", "/a/b/", "/a/./b", "/x/../a/b"} {
		checkSearch(t, root, searchCase{p, "H", nil})
	}
}

// Malformed patterns are rejected by Insert and leave the tree untouched.
func TestRouteTreeInvalidPattern(t *testing.T) {
	for _, p := range []string{
		"/doc/{id", "/doc/{}", "/doc/{a{b}}", "/doc/id}",
		"/doc/{a/b}", "/doc/{a}{b}", "/doc/{a}x{b}",
	} {
		t.Run(p, func(t *testing.T) {
			root := &Node[string]{}
			if err := root.Insert(p, "H"); err == nil {
				t.Errorf("Insert(%q): expected error", p)
			}
			if root.path != "" || len(root.children) != 0 {
				t.Errorf("Insert(%q) mutated the tree", p)
			}
		})
	}
}

// "" and "/" both normalize to "/", so a root route matches the empty request.
func TestRouteTreeRootAndEmpty(t *testing.T) {
	root := &Node[string]{}
	mustInsert(t, root, "/", "RootHandler")
	for _, p := range []string{"/", ""} {
		checkSearch(t, root, searchCase{p, "RootHandler", nil})
	}
}

// Two static routes sharing a prefix split the shared node; both stay reachable.
func TestRouteTreeCommonPrefixSplit(t *testing.T) {
	root := &Node[string]{}
	mustInsert(t, root, "/api/users", "UsersHandler")
	mustInsert(t, root, "/api/posts", "PostsHandler")
	mustInsert(t, root, "/api/users/{id}", "UserHandler")

	runSearchTests(t, root, []searchCase{
		{"/api/users", "UsersHandler", nil},
		{"/api/posts", "PostsHandler", nil},
		{"/api/users/7", "UserHandler", Params{{"id", "7", false}}},
		{"/api/orders", "", nil}, // shares the prefix, matches no branch
	})
}

// A param node can branch into several static children; an unmatched suffix
// after the param falls through to not-found.
func TestRouteTreeParamChildren(t *testing.T) {
	root := &Node[string]{}
	mustInsert(t, root, "/files/{id}/raw", "RawHandler")
	mustInsert(t, root, "/files/{id}/meta", "MetaHandler")

	runSearchTests(t, root, []searchCase{
		{"/files/abc/raw", "RawHandler", Params{{"id", "abc", false}}},
		{"/files/abc/meta", "MetaHandler", Params{{"id", "abc", false}}},
		{"/files/abc/other", "", nil}, // no static child matches "other"
	})
}

// Static segments match byte-for-byte (case-sensitive).
func TestRouteTreeCaseSensitive(t *testing.T) {
	root := &Node[string]{}
	mustInsert(t, root, "/Users", "UsersHandler")
	checkSearch(t, root, searchCase{"/Users", "UsersHandler", nil})
	if _, _, found := root.Search("/users", nil); found {
		t.Errorf(`Search("/users"): expected not found (case-sensitive)`)
	}
}

// The ":name" syntax is not recognized; ':' is literal here.
func TestRouteTreeColonIsLiteral(t *testing.T) {
	root := &Node[string]{}
	mustInsert(t, root, "/items/:id", "ColonHandler")
	checkSearch(t, root, searchCase{"/items/:id", "ColonHandler", nil})
	if _, _, found := root.Search("/items/42", nil); found {
		t.Errorf(`Search("/items/42"): expected not found (':' is literal)`)
	}
}

// A {*name} catch-all matches the rest of the path (including slashes) and is
// exposed as a param with CatchAll=true. The leading "/" is stripped from the
// value, matching fiber's c.Params("*") convention.
func TestRouteTreeCatchAll(t *testing.T) {
	root := &Node[string]{}
	mustInsert(t, root, "/files/{*path}", "FileHandler")

	runSearchTests(t, root, []searchCase{
		{"/files/a.txt", "FileHandler", Params{{"path", "a.txt", true}}},
		{"/files/css/main.css", "FileHandler", Params{{"path", "css/main.css", true}}},
		{"/files/deep/nested/dir/x", "FileHandler", Params{{"path", "deep/nested/dir/x", true}}},
		{"/files", "", nil},   // bare prefix (no trailing segment) does not match
		{"/other/a.txt", "", nil},
	})
}

// Statics and single-segment params both beat a catch-all at the same position,
// independent of registration order.
func TestRouteTreeCatchAllPriority(t *testing.T) {
	root := &Node[string]{}
	mustInsert(t, root, "/files/{*path}", "CatchAllHandler")
	mustInsert(t, root, "/files/{id}", "ParamHandler")
	mustInsert(t, root, "/files/index.html", "StaticHandler")

	runSearchTests(t, root, []searchCase{
		{"/files/index.html", "StaticHandler", nil},
		{"/files/abc", "ParamHandler", Params{{"id", "abc", false}}},
		{"/files/abc/def", "CatchAllHandler", Params{{"path", "abc/def", true}}},
		{"/files/css/main.css", "CatchAllHandler", Params{{"path", "css/main.css", true}}},
	})
}

// A catch-all must be the last segment; patterns with trailing content are
// rejected at Insert and leave the tree untouched.
func TestRouteTreeCatchAllInvalid(t *testing.T) {
	for _, p := range []string{
		"/files/{*path}/extra", // trailing segment after catch-all
		"/files/{*}",           // empty catch-all name
		"/files/{*a}{b}",       // catch-all not alone in segment
	} {
		t.Run(p, func(t *testing.T) {
			root := &Node[string]{}
			if err := root.Insert(p, "H"); err == nil {
				t.Errorf("Insert(%q): expected error", p)
			}
			if root.path != "" || len(root.children) != 0 {
				t.Errorf("Insert(%q) mutated the tree", p)
			}
		})
	}
}

// Two catch-all declarations at the same position conflict; the second Insert
// errors and the tree is unchanged.
func TestRouteTreeCatchAllConflict(t *testing.T) {
	root := &Node[string]{}
	mustInsert(t, root, "/files/{*path}", "First")
	if err := root.Insert("/files/{*other}", "Second"); err == nil {
		t.Fatalf("Insert conflicting catch-all: expected error")
	}
	// Re-registering the same declaration overwrites the handler (no conflict).
	if err := root.Insert("/files/{*path}", "Overwritten"); err != nil {
		t.Fatalf("Insert same catch-all: %v", err)
	}
	h, params, found := root.Search("/files/a/b", nil)
	if !found || h != "Overwritten" || !paramsEqual(params, Params{{"path", "a/b", true}}) {
		t.Fatalf("Search: h=%q params=%v found=%v", h, params, found)
	}
}

// ---------------------------------------------------------------------------
// Benchmarks (typical website route table)
// ---------------------------------------------------------------------------

// benchSink prevents the compiler from eliminating the benchmark loop's work.
var benchSink string

// routeSpec is a (pattern, handler) pair used to build benchmark trees.
type routeSpec struct {
	pattern string
	handler string
}

// typicalSiteRoutes returns a route table representative of a typical website:
// static pages, auth, profiles, a blog, a nested REST API, admin, and static
// assets — mixing static, single-param, multi-param, and long shared prefixes.
func typicalSiteRoutes() []routeSpec {
	return []routeSpec{
		{"/", "Home"},
		{"/about", "About"},
		{"/contact", "Contact"},
		{"/pricing", "Pricing"},
		{"/login", "Login"},
		{"/logout", "Logout"},
		{"/signup", "Signup"},
		{"/reset-password", "ResetPassword"},
		{"/blog", "BlogIndex"},
		{"/blog/page/{page}", "BlogPage"},
		{"/blog/category/{category}", "BlogCategory"},
		{"/blog/{slug}", "BlogPost"},
		{"/blog/{slug}/comments", "BlogComments"},
		{"/u/{username}", "UserProfile"},
		{"/u/{username}/followers", "UserFollowers"},
		{"/u/{username}/settings", "UserSettings"},
		{"/api/v1/users", "ApiUsers"},
		{"/api/v1/search", "ApiSearch"},
		{"/api/v1/users/{id}", "ApiUser"},
		{"/api/v1/users/{id}/posts", "ApiUserPosts"},
		{"/api/v1/posts", "ApiPosts"},
		{"/api/v1/posts/{id}", "ApiPost"},
		{"/api/v1/posts/{id}/comments", "ApiPostComments"},
		{"/api/v1/comments/{id}", "ApiComment"},
		{"/api/v1/tags/{tag}/posts", "ApiTagPosts"},
		{"/api/v1/upload/{bucket}", "ApiUpload"},
		{"/admin", "Admin"},
		{"/admin/users", "AdminUsers"},
		{"/admin/users/{id}", "AdminUser"},
		{"/admin/users/{id}/edit", "AdminUserEdit"},
		{"/static/{path}", "StaticFile"},
		{"/favicon.ico", "Favicon"},
	}
}

// newTypicalSiteTree builds a fresh tree from typicalSiteRoutes.
func newTypicalSiteTree(tb testing.TB) *Node[string] {
	tb.Helper()
	root := &Node[string]{}
	for _, r := range typicalSiteRoutes() {
		if err := root.Insert(r.pattern, r.handler); err != nil {
			tb.Fatalf("Insert(%q): %v", r.pattern, err)
		}
	}
	return root
}

// BenchmarkRadixTreeBuild measures constructing the full tree from scratch.
func BenchmarkRadixTreeBuild(b *testing.B) {
	routes := typicalSiteRoutes()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root := &Node[string]{}
		for _, r := range routes {
			if err := root.Insert(r.pattern, r.handler); err != nil {
				b.Fatal(err)
			}
		}
		benchSink = root.handler
	}
}

// BenchmarkRadixTreeSearch measures lookup of static, param, and miss paths.
func BenchmarkRadixTreeSearch(b *testing.B) {
	root := newTypicalSiteTree(b)
	cases := []struct {
		name    string
		queries []string
	}{
		{"Static", []string{"/", "/about", "/blog", "/api/v1/search", "/admin/users", "/favicon.ico"}},
		{"Param", []string{
			"/blog/hello-world", "/blog/page/3", "/u/john.doe",
			"/api/v1/posts/42/comments", "/api/v1/tags/go/posts",
			"/admin/users/7/edit", "/static/css/main.css",
		}},
		{"NotFound", []string{
			"/nonexistent", "/api/v2/users", "/u/john/unknown",
			"/blog/hello/extra/deep", "/admin/dashboard",
		}},
	}
	for _, c := range cases {
		queries := c.queries
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchSink, _, _ = root.Search(queries[i%len(queries)], nil)
			}
		})
	}
}

// BenchmarkRadixTreeSearchParallel measures concurrent lookup throughput.
// Search is read-only, so parallel access is safe and lock-free.
func BenchmarkRadixTreeSearchParallel(b *testing.B) {
	root := newTypicalSiteTree(b)
	queries := []string{
		"/", "/about", "/blog/hello-world", "/api/v1/posts/42/comments",
		"/u/john.doe", "/admin/users/7/edit", "/static/css/main.css",
		"/nonexistent", "/api/v2/users",
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var sink string // local: avoids racing the package-level benchSink
		i := 0
		for pb.Next() {
			sink, _, _ = root.Search(queries[i%len(queries)], nil)
			i++
		}
		_ = sink
	})
}
