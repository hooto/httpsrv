# httpsrv

httpsrv is a lightweight, net/http-native web framework for Go. It offers a Fiber v3-style routing surface (`App`/`Router`/`Ctx`/`Handler`) implemented from scratch over the standard library.

**Language:** [English](README.md) | [中文](README.zh-CN.md)

## Features

- **net/http-native** — the typical handler is `func(Ctx) error`; a stdlib `http.Handler` (e.g. `http.HandlerFunc`) is also accepted, so handlers interoperate directly with the Go ecosystem
- **Radix-tree routing** — per-method trees, `{param}` and `{*catchAll}` path values, case-sensitive, static segments beat params beat catch-alls (order-independent)
- **Middleware & Groups** — `Use` with fiber v3-style middleware (`func(Ctx) error` + `c.Next()`), global or prefix-scoped; `Group(prefix)` for prefix sharing
- **Static files** — `middleware/static` exposes `New(root, ...Config)` returning an `httpsrv.Handler`: `app.Get("/*", static.New("./public"))` or `app.Get("/*", static.New(".", static.Config{FS: embedFS}))`; directory or embedded filesystem
- **Template rendering** — `html/template` parsed once, `Ctx.Render(name, bind, layouts...)`
- **i18n (opt-in)** — flat locale message store, `AcceptLanguage` middleware (built-in RFC 5646 subset matching)
- **Compression (opt-in)** — `middleware/compress` exposes `New(config...)` (gzip/brotli by `Accept-Encoding`, brotli preferred)
- **Graceful server** — safe default timeouts, `Shutdown(ctx)`

## Documentation

Full guide (English): [`doc/`](doc/README.md) — [Quick Start](doc/quickstart.md) · [Routing](doc/routing.md) · [Groups](doc/groups.md) · [Middleware](doc/middleware.md) · [Server](doc/server.md) · [Static](doc/static.md) · [Ctx & Handler](doc/ctx.md) · [Render](doc/views.md) · [i18n](doc/i18n.md) · [Examples](doc/examples.md) · [Index](doc/SUMMARY.md)

中文文档：[`doc/zh-CN/`](doc/zh-CN/README.md)

## Installation

```bash
go get -u github.com/hooto/httpsrv/v2
```

## Quick Start

```go
package main

import (
    "github.com/hooto/httpsrv/v2"
)

func main() {
    app := httpsrv.New()

    app.Get("/", func(c httpsrv.Ctx) error {
        return c.SendString("hello httpsrv")
    })

    app.Run(":8080")
}
```

Run and try it:

```bash
$ go run .
$ curl http://localhost:8080/
hello httpsrv
```

See [`examples/`](examples) for runnable programs (hello, i18n, groups/static/render).

## Recommended Dependencies

httpsrv keeps the core concise. Some recommended third-party libraries:

### Database
- [mysqlgo](https://github.com/lynkdb/mysqlgo) - MySQL client
- [pgsqlgo](https://github.com/lynkdb/pgsqlgo) - PostgreSQL client
- [redisgo](https://github.com/lynkdb/redisgo) - Redis client
- [kvgo](https://github.com/lynkdb/kvgo) - Embedded Key-Value database

### Utility Libraries
- [hlog4g](https://github.com/hooto/hlog4g) - Logging library
- [hini4g](https://github.com/hooto/hini4g) - INI configuration file parsing
- [hflag4g](https://github.com/hooto/hflag4g) - Command line argument handling
- [hlang4g](https://github.com/hooto/hlang4g) - i18n internationalization
- [hcaptcha4g](https://github.com/hooto/hcaptcha4g) - CAPTCHA generation

More Go ecosystem libraries: [awesome-go](https://github.com/avelino/awesome-go)

## System Requirements

- **Go Version**: 1.26 or higher
- **Recommended Systems**: Linux, Unix, or macOS

## Reference Projects

httpsrv's API surface (`App`/`Router`/`Ctx`/`Handler`) is inspired by [Fiber v3](https://github.com/gofiber/fiber/v3) — a from-scratch implementation over net/http with no dependency on fiber.

## License

[Apache License 2.0](LICENSE)
