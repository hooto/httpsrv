# Quick Start

## Create an app

```go
app := httpsrv.New()
```

## Register routes

Each HTTP method has a registrar that returns the `Router`, so calls chain. The typical form passes a `Handler` (`func(c Ctx) error`):

```go
app.Get("/hello", func(c httpsrv.Ctx) error {
	return c.SendString("hello")
})

app.Post("/echo", func(c httpsrv.Ctx) error {
	// ...
	return nil
})
```

Registrars also accept a stdlib `http.Handler` (e.g. `http.HandlerFunc`). See [Ctx & Handler](./ctx.md).

## Start the server

```go
err := app.Run(":8080") // listen on 0.0.0.0:8080
```

`Run` blocks until the server stops. See [Running the Server](./server.md).

## Complete minimal program

```go
package main

import (
	"github.com/hooto/httpsrv/v2"
)

func main() {
	app := httpsrv.New()

	app.Get("/", func(c httpsrv.Ctx) error {
		return c.SendString("Hello, httpsrv!")
	})

	if err := app.Run(":8080"); err != nil {
		panic(err)
	}
}
```

Run it:

```bash
$ go run .
$ curl http://localhost:8080/
Hello, httpsrv!
```

## Registration methods

| Method | Description |
|---|---|
| `Get/Head/Post/Put/Patch/Delete/Options` | Register a route for the given HTTP method (accepts a `Handler` or `http.Handler`, see [Ctx & Handler](./ctx.md)) |
| `All(path, handler)` | Register the same route for **all** HTTP methods (GET/POST/PUT/PATCH/DELETE/HEAD/OPTIONS/CONNECT/TRACE) |
| `Group(prefix)` | Create a prefixed sub-router (see [Route Groups](./groups.md)) |
| `Use(args...)` | Register middleware (see [Middleware](./middleware.md)) |
| `app.Get("/p/{*path}", static.New(...))` | Static file serving (`New`/`FS` from the `middleware/static` subpackage return an `httpsrv.Handler`, see [Static Files](./static.md)); use `All` instead of `Get` to cover every method |
