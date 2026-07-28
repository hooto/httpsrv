# Middleware (Use)

## Middleware type

A middleware is just a [`Handler`](./ctx.md) (`func(Ctx) error`) — like gofiber v3, there is no separate middleware type. A middleware does its work, then calls `c.Next()` to continue the chain:

```go
func logging(c httpsrv.Ctx) error {
	start := time.Now()
	if err := c.Next(); err != nil { // continue to the next middleware / route handler
		return err
	}
	log.Printf("%s %s %v", c.Method(), c.Path(), time.Since(start))
	return nil
}
```

## Registration: Use

`Use` signature:

```go
Use(args ...any) Router
```

- If the first argument is a `string`, it is a **path prefix**; the rest are middleware.
- If the first argument is not a string, all arguments are **global** middleware.

### Global middleware

```go
app.Use(func(c httpsrv.Ctx) error {
	c.SetHeader("X-Frame-Options", "DENY")
	return c.Next()
})
```

Applies to **every** request and **every** HTTP method (including unmatched 404s, matching gofiber v3's default behavior).

### Prefix-scoped middleware

```go
app.Use("/api", func(c httpsrv.Ctx) error {
	// scoped to /api/*
	return c.Next()
})
```

The prefix matches on **`/` segment boundaries**, so `/api` matches `/api/users` but **not** `/apifoo`.

A single `Use` may take multiple middleware:

```go
app.Use("/api", logging, recoverer, cors)
```

## Registration order

Middleware run in **registration order** — the first registered runs first and is the outermost:

```go
app.Use(mwA) // runs first
app.Use(mwB) // runs second
// order: mwA -> mwB -> route handler
```

## Short-circuiting (not calling Next)

A middleware that does not call `c.Next()` terminates the chain; the route handler will not run:

```go
app.Use(func(c httpsrv.Ctx) error {
	if !authorized(c) {
		c.Status(http.StatusUnauthorized)
		return c.SendString("forbidden") // no Next, ends here
	}
	return c.Next()
})
```

## Error propagation

Any non-`nil` error returned by a `Handler` (middleware or route handler) bubbles up along `c.Next()` to the top of the chain, where the `ErrorHandler` handles it (default 500; customize with `WithErrorHandler`). A middleware may also intercept downstream errors:

```go
app.Use(func(c httpsrv.Ctx) error {
	err := c.Next()
	if err != nil {
		log.Printf("handler error: %v", err) // log, transform, or swallow
	}
	return err
})
```

## Group-scoped middleware

`Group.Use` automatically joins the group prefix onto the middleware path:

```go
api := app.Group("/api")
api.Use(rateLimit) // scoped to /api/*
api.Get("/users", listUsers)
```

## Argument types

`Use` accepts both a named `Handler` type and an inline `func(Ctx) error`:

```go
var myMw httpsrv.Handler = func(c httpsrv.Ctx) error { /* ... */ }

app.Use(myMw, func(c httpsrv.Ctx) error { /* ... */ })
```

## Built-in middleware

- `compress.New()` (the `middleware/compress` subpackage): gzip/brotli compression by `Accept-Encoding` (brotli preferred), mirroring gofiber v3's `compress.New`.
- `httpsrv.AcceptLanguage(def, others...)`: detect the request language from `Accept-Language` (BCP-47, via `x/text`).

```go
import "github.com/hooto/httpsrv/v2/middleware/compress"

app.Use(compress.New())                       // default level
app.Use(compress.New(compress.Config{         // optional config
    Level: compress.LevelBestSpeed,
}))
app.Use(httpsrv.AcceptLanguage("en", "zh"))
```

## Limitations (basic implementation)

- Middleware path prefixes support **static prefixes only** — no `/users/{id}` parameter prefixes.
- Path parameters are **not available before the middleware calls `c.Next()`** (routing happens in the terminal `dispatch`); they are readable via `c.Params()` after `c.Next()` returns.
