# Ctx & Handler

`Ctx` (request context) and `Handler` (`func(c Ctx) error`) are v2's **typical handler form**: receive a Ctx, return an error. Path parameters, query, headers, JSON, forms, template rendering — all go through Ctx.

httpsrv's `Ctx` is built on `*http.Request`/`http.ResponseWriter`; path parameters come from the stdlib `r.PathValue`.

## Registration

Registrars (`Get/Post/.../All`) accept a `Handler` (typical), and also a stdlib `http.Handler`:

```go
app := httpsrv.New()

// Handler (Ctx) — typical
app.Get("/users/{id}", func(c httpsrv.Ctx) error {
	return c.SendString("id=" + c.Params("id"))
})

// http.Handler — reuse a stdlib handler directly (wrap a func with http.HandlerFunc)
app.Get("/basic", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "basic")
}))
```

## Core Ctx methods

| Group | Method | Description |
|---|---|---|
| Request/Response | `Request() *http.Request`, `Response() http.ResponseWriter` | raw stdlib objects |
| Identity | `Method()`, `Path()`, `IP()`, `BaseURL()` | `BaseURL` returns "scheme + host + base path" |
| Input | `Params(key)`, `Query(key, def...)`, `Header(key, def...)`, `Body()`, `FormValue(key, def...)`, `Bind(out)` | path params / query / headers / body / form fields / JSON → struct |
| Output | `Status(code)`, `SetHeader(k,v)`, `JSON(v)`, `Send(b)`, `SendString(s)`, `Redirect(status, url)` | `Status`/`SetHeader` chain |

> `Status`/`SetHeader` must be called before writing the response body (same as the stdlib).

## Example

```go
type Resp struct {
	OK bool   `json:"ok"`
	ID string `json:"id"`
}

app.Get("/users/{id}", func(c httpsrv.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).SendString("missing id") // write the response yourself, return nil
	}
	return c.JSON(&Resp{OK: true, ID: id})
})
```

`JSON(v)` sets `Content-Type` to `application/json; charset=utf-8` only when not already set; to use a custom media type or charset (e.g. `application/problem+json`), set it beforehand via `SetHeader`. Likewise, `Render` sets `text/html; charset=utf-8` only when not set.

Query, headers, and body:

```go
app.Post("/echo", func(c httpsrv.Ctx) error {
	name := c.Query("name", "anon")        // with a default
	tok := c.Header("X-Token")             // request header
	return c.SendString(name + ":" + tok + ":" + string(c.Body()))
})
```

## JSON parsing (Bind)

`Bind(out any) error` unmarshals the JSON request body into a struct (`json.Unmarshal`), returning the parse error:

```go
type Login struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}

app.Post("/login", func(c httpsrv.Ctx) error {
	var in Login
	if err := c.Bind(&in); err != nil {
		return c.Status(400).SendString("invalid json: " + err.Error())
	}
	return c.SendString(in.User + ":" + in.Pass)
})
```

- An empty body is a no-op (returns `nil`).
- `Bind` reads the cached body, so it is independent of `Body()`/`FormValue()` call order (calling `Bind` then `Body()` still gets the full content).

## Forms

`FormValue` reads a body form field, supporting `application/x-www-form-urlencoded` and `multipart/form-data`, with a default:

```go
app.Post("/login", func(c httpsrv.Ctx) error {
	user := c.FormValue("username")
	pass := c.FormValue("password")
	if user == "" {
		return c.Status(400).SendString("missing username")
	}
	_ = pass
	return c.SendString("ok")
})
```

File uploads go through the stdlib `c.Request()`: `req.FormFile("file")`, `req.ParseMultipartForm(maxBytes)`.

> `Body()` and `FormValue()` are mutually safe: call them in any order — `Body()` then `FormValue()` (or vice versa) both work and never fail from a drained body stream.

## Error handling

When a `Handler` returns a **non-nil error**, the framework responds **500** by default (writing the error text). If the handler already wrote a response (like the 400 above), return `nil`:

```go
app.Get("/boom", func(c httpsrv.Ctx) error {
	return errors.New("something failed") // -> 500
})
```

## BaseURL / IP

- `BaseURL()` returns `scheme://host` (honoring `X-Forwarded-Proto`/`X-Forwarded-Host`, TLS; no base-path config yet, so the base path is empty).
- `IP()` returns the client address (honoring the first hop of `X-Forwarded-For`, then `X-Real-Ip`, falling back to `RemoteAddr`).
