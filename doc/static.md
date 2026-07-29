# Static Files

httpsrv provides static file serving through the standalone `middleware/static` subpackage. `static.New` returns an `httpsrv.Handler` (mirroring gofiber v3's `static.New`) that you register on a catch-all route to serve a whole file tree. It reads from a filesystem directory or an embedded FS and serves the file.

```go
import (
	"github.com/hooto/httpsrv/v2"
	"github.com/hooto/httpsrv/v2/middleware/static"
)
```

## Constructors

| Function / type | Description |
|---|---|
| `static.New(root string, config ...Config) httpsrv.Handler` | `root` is a directory path (served via `http.Dir`, which prevents traversal); when `Config.FS` is set, `root` is instead a sub-path within that `fs.FS` (`.`/`""` means the whole FS) |
| `static.Config` | Config type with field `FS fs.FS` (an `io/fs` filesystem, e.g. `embed.FS`) |

## Registration: catch-all route

Register it on a catch-all route to serve a whole file tree. The gofiber-v3-style unnamed `/*` wildcard is the recommended form (no `{*path}` named parameter):

```go
app.Get("/*", static.New("./public"))
// GET /css/main.css  ->  ./public/css/main.css

app.Get("/static/*", static.New("./assets"))
// GET /static/css/main.css  ->  ./assets/css/main.css
```

You can also use a named catch-all `{*path}`:

```go
app.Get("/static/{*path}", static.New("./assets"))
// GET /static/css/main.css  ->  ./assets/css/main.css
```

The catch-all value (the part of the path after the prefix) is exposed under the conventional key `"*"` (`r.PathValue("*")` / `c.Params("*")`), regardless of the parameter name. See [Routing / catch-all](./routing.md#catch-all-parameter-name).

- `app.Get` responds to GET only; `app.All` covers every method.
- You can also register on a `Group`; the prefix is joined automatically (see below).

### Serve from an embed

```go
//go:embed assets/*
var assets embed.FS

// root is a sub-path within the FS; "." means the whole FS
app.Get("/*", static.New(".", static.Config{FS: assets}))
app.Get("/static/*", static.New("dist", static.Config{FS: assets})) // expose only the dist subtree
```

## Behavior

- **Explicit routes win**: the catch-all (`/*` and `{*name}`) is the weakest match — static and parameter segments both beat it, so explicit routes win naturally.
- **No directory listing**: a request hitting a directory returns 404 (no auto-index).
- **Path-traversal protection**: based on `http.Dir`, escaping the root is prevented, e.g. `/static/../secret` returns 404.
- **Bare root does not match**: `/*` does not match the root path `/` itself (consistent with named catch-alls); you need `/...`.
- Backed by the stdlib `http.ServeContent`: automatic GET/HEAD, Range requests, `Last-Modified`/`ETag`.

## Registering on a group

A `Group` joins its prefix automatically:

```go
assets := app.Group("/assets")
assets.Get("/img/*", static.New("./images"))
// GET /assets/img/logo.png  ->  ./images/logo.png
```

## Full example

```go
package main

import (
	"embed"

	"github.com/hooto/httpsrv/v2"
	"github.com/hooto/httpsrv/v2/middleware/static"
)

//go:embed public/*
var public embed.FS

func main() {
	app := httpsrv.New()

	// filesystem directory
	app.Get("/files/*", static.New("./local-files"))

	// embedded assets (whole FS)
	app.Get("/public/*", static.New(".", static.Config{FS: public}))

	_ = app.Run(":8080")
}
```
