# Static Files

httpsrv provides static file serving through the standalone `middleware/static` subpackage. `New` / `FS` return an `httpsrv.Handler` (mirroring gofiber v3's `static.New(root) -> Handler` style) that you can register on any route — it reads from the filesystem (or an embedded FS) and serves the file.

```go
import (
	"github.com/hooto/httpsrv/v2"
	"github.com/hooto/httpsrv/v2/middleware/static"
)
```

## Constructors

| Function | Description |
|---|---|
| `static.New(root string) httpsrv.Handler` | Serve from the filesystem directory `root` |
| `static.FS(fs http.FileSystem) httpsrv.Handler` | Serve from an embedded filesystem, e.g. `http.FS(embed.FS)` |

## Registration: catch-all route

Typically combined with a catch-all route `{*path}` to serve an entire file tree:

```go
app.Get("/static/{*path}", static.New("./assets"))
// GET /static/css/main.css  ->  ./assets/css/main.css
```

The catch-all value (the part of the path after the prefix) is exposed under the conventional key `"*"`, which the handler reads via `r.PathValue("*")` regardless of the parameter name you declared. You can also read it by its named key: `c.Params("path")`. See [Routing / catch-all](./routing.md#catch-all-parameter-name).

- `app.Get` responds to GET only; `app.All` covers every HTTP method (POST/PUT/...).
- You can also register on a `Group`; the prefix is joined automatically (see below).

### Serve from a directory

```go
app.Get("/static/{*path}", static.New("./assets"))
// GET /static/css/main.css  ->  ./assets/css/main.css
```

### Serve from an embed

```go
//go:embed assets/*
var assets embed.FS

app.Get("/static/{*path}", static.FS(http.FS(assets)))
```

## Behavior

- **Explicit routes win**: the catch-all is the weakest match — static and parameter segments both beat it, so explicit routes win naturally.
- **No directory listing**: a request hitting a directory returns 404 (no auto-index).
- **Path-traversal protection**: based on `http.Dir`, escaping the root is prevented, e.g. `/static/../secret` returns 404.
- **Segment-boundary prefix**: `/static` does not match `/staticfoo`.
- **Bare prefix does not match**: `/static` (no trailing segment) does not match the catch-all; you need `/static/...`.
- **Goes through the middleware chain**: static requests pass through `Use`-registered middleware (`dispatch` is the chain's terminus).
- Backed by the stdlib `http.ServeContent`: automatic GET/HEAD, Range requests, `Last-Modified`/`ETag`.

## Registering on a group

A `Group` automatically joins its prefix:

```go
assets := app.Group("/assets")
assets.Get("/img/{*path}", static.New("./images"))
// GET /assets/img/logo.png  ->  ./images/logo.png
```

## Full example

```go
package main

import (
	"embed"
	"net/http"

	"github.com/hooto/httpsrv/v2"
	"github.com/hooto/httpsrv/v2/middleware/static"
)

//go:embed public/*
var public embed.FS

func main() {
	app := httpsrv.New()

	// filesystem directory
	app.Get("/files/{*path}", static.New("./local-files"))

	// embedded assets
	app.Get("/public/{*path}", static.FS(http.FS(public)))

	_ = app.Run(":8080")
}
```
