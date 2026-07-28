# Routing & Path Parameters

## Register per method

```go
app.Get("/items", listItems)
app.Post("/items", createItem)
app.Put("/items/{id}", updateItem)
app.Delete("/items/{id}", deleteItem)
```

Methods map one-to-one to registrars: `Get/Head/Post/Put/Patch/Delete/Options`. Above, `listItems` etc. are `httpsrv.Handler` (i.e. `func(c httpsrv.Ctx) error`). **Methods are independent** — with only a GET route, a POST to the same path returns 404.

## Chaining

Registrars return the `Router`, so you can chain:

```go
app.Get("/a", hA).
	Get("/b", hB).
	Post("/c", hC)
```

## Path parameters `{name}`

Define a parameter segment with braces and read it in the handler via `c.Params(name)` (backed by the stdlib `r.PathValue`):

```go
app.Get("/users/{id}", func(c httpsrv.Ctx) error {
	return c.SendString("id=" + c.Params("id"))
})
```

- A parameter matches `[^/]+`, i.e. a single path segment (no `/`).
- Multiple parameters are supported:

```go
app.Get("/posts/{post_id}/comments/{comment_id}", func(c httpsrv.Ctx) error {
	return c.SendString(c.Params("post_id") + " " + c.Params("comment_id"))
})
// GET /posts/5/comments/22  ->  5 22
```

- The parameter value keeps its original content, e.g. for `GET /users/news.html` the `{id}` is `news.html`.

> ⚠️ Only the `{name}` syntax is supported; `:name` is **literal** and not treated as a parameter.

## catch-all parameter `{*name}`

`{*name}` matches the **rest of the path** (including `/`) and **must be the last segment** of the pattern:

```go
app.Get("/files/{*path}", func(c httpsrv.Ctx) error {
	return c.SendString("file=" + c.Params("path"))
})
// GET /files/a.txt          ->  path = "a.txt"
// GET /files/css/main.css   ->  path = "css/main.css"
// GET /files/deep/nested/x  ->  path = "deep/nested/x"
```

- The catch-all value has no leading `/` (e.g. `css/main.css`).
- Besides its named key, a catch-all is also exposed under the conventional key `"*"`, readable via `c.Params("*")` (mirroring fiber's `c.Params("*")` convention).
- Priority: **static segment > `{name}` parameter > `{*name}` catch-all**, independent of registration order.
- Nothing may follow `{*name}`; e.g. `/files/{*path}/extra` panics at registration.
- Two differently named catch-alls at the same position conflict (e.g. `/files/{*a}` vs `/files/{*b}`); re-registering the same name overwrites the handler.

## Static routes beat parameter routes

A static segment always beats a parameter or catch-all at the same position, **independent of registration order**:

```go
app.Get("/users/profile", showProfile) // static
app.Get("/users/{id}", showUser)       // parameter
app.Get("/users/{*rest}", showRest)    // catch-all
```

```
GET /users/profile  ->  showProfile   (static hit)
GET /users/42       ->  showUser      (parameter hit)
GET /users/a/b/c    ->  showRest      (catch-all hit)
```

## Case sensitivity

Matching is **case-sensitive**:

```go
app.Get("/Users", h)
// GET /Users   ->  match
// GET /users   ->  404
```

## Path normalization

Both registration and matching normalize the path with `path.Clean`: collapsing duplicate slashes, resolving `.`/`..`, and ensuring a leading `/`. So these are equivalent:

```go
app.Get("//api//users/", h)
// equivalent to app.Get("/api/users", h)
```

Request paths are normalized before matching as well.

## All: register for every method

```go
app.All("/healthz", func(c httpsrv.Ctx) error {
	return c.SendString("ok")
})
// GET/POST/PUT/... /healthz all match
```

## Invalid patterns panic at registration

To surface errors early, an invalid path pattern panics (already-registered routes are unaffected):

```go
app.Get("/users/{id", h)           // panic: unclosed '{'
app.Get("/users/{}", h)            // panic: empty parameter name
app.Get("/{a}{b}", h)              // panic: multiple parameters in one segment
app.Get("/files/{*path}/extra", h) // panic: catch-all must be the last segment
```
