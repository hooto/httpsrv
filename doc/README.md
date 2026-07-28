# httpsrv Documentation

> **English** | [中文](./zh-CN/README.md)

httpsrv is a lightweight, net/http-native Go web framework.

## Features

- **net/http-native**: the typical handler is `func(c Ctx) error`, and a stdlib `http.Handler` is also accepted, so you can reuse the stdlib ecosystem and existing middleware directly.
- **Concise API**: per-method route registration (`Get/Post/Put/...`), `All`, nestable `Group`, `Use` middleware, with chaining.
- **High-performance radix-tree routing**: exact static segments, `{name}` parameter segments; **static routes take priority over parameter routes** (independent of registration order).
- **Path parameters**: `{name}` segments are read via `c.Params("name")` (backed by the stdlib `r.PathValue`); no custom Context needed.
- **Few dependencies**: stdlib only (plus `andybalholm/brotli` for compression and `golang.org/x/text` for locale matching); handlers can reuse the stdlib ecosystem and existing middleware directly.

## Contents

- [Quick Start](./quickstart.md)
- [Routing & Path Parameters](./routing.md)
- [Route Groups](./groups.md)
- [Middleware (Use)](./middleware.md)
- [Running the Server (Run)](./server.md)
- [Examples](./examples.md)
- [Static Files](./static.md)
- [Ctx & Handler](./ctx.md)
- [Template Rendering](./views.md)
- [i18n](./i18n.md)

## Minimal example

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

	app.Run(":8080") // open http://localhost:8080/
}
```

## Requirements

- Go **1.26+**
- Run the tests: `go test ./...`
