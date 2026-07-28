# Running the Server (Run)

## Signature

```go
Run(args ...any) error
```

`Run` creates an `*http.Server` (with the app itself as handler) and blocks until the server stops.

## Listen address

Pass a string as the listen address:

```go
app.Run(":8080")            // default: 0.0.0.0:8080
app.Run("127.0.0.1:3000")   // localhost only
app.Run(":0")               // random available port
```

With no arguments, the default address `:8080` is used:

```go
app.Run()
```

## Using an existing Listener

Pass a `net.Listener` — useful for custom network types or externally controlled lifecycles:

```go
ln, err := net.Listen("tcp", ":8080")
if err != nil {
	log.Fatal(err)
}
app.Run(ln)
```

Also works for testing or Unix sockets:

```go
ln, _ := net.Listen("unix", "/tmp/app.sock")
app.Run(ln)
```

## Return value

- On a normal stop, `http.ErrServerClosed` is swallowed and `Run` returns `nil`.
- Other errors (port in use, listen failure) are returned as-is.

## Registration timing

**Routes and middleware must be fully registered before `Run`.** Once serving, the route trees are read-only; each request acquires an `RLock`, so concurrent serving is safe.

## Default 404 handling

When no route matches, the built-in `defaultNotFound` responds:

```
HTTP/1.1 404 Not Found
Content-Type: text/plain; charset=utf-8

404 page not found
```

## Graceful shutdown

`app.Shutdown(ctx)` gracefully stops the server (waiting for active connections to finish or `ctx` to expire). Call it from another goroutine while `Run` is blocking:

```go
go func() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = app.Shutdown(ctx)
}()
if err := app.Run(":8080"); err != nil { log.Fatal(err) }
```

`Run` returns `nil` after a graceful shutdown (`http.ErrServerClosed` is swallowed).

## Config (timeouts / address / header limit)

`Run` applies safe server defaults: read/write timeouts 60s, read-header timeout 10s, `MaxHeaderBytes` 1 MiB. Override with `New(WithConfig(...))` (only non-zero fields take effect):

```go
app := httpsrv.New(httpsrv.WithConfig(httpsrv.Config{
	Addr:              ":3000",
	ReadTimeout:       30 * time.Second,
	WriteTimeout:      30 * time.Second,
	ReadHeaderTimeout: 10 * time.Second,
	MaxHeaderBytes:    1 << 20,
}))
```

> `WriteTimeout` caps a single response's duration; set it to `0` for long-lived connections / SSE / large downloads.

## Response compression (optional)

`middleware/compress`'s `New()` (mirroring gofiber v3's `compress.New`) gzip/brotli-compresses by `Accept-Encoding` (brotli preferred), sets `Content-Encoding`/`Vary`, and drops `Content-Length`:

```go
import "github.com/hooto/httpsrv/v2/middleware/compress"

app.Use(compress.New())                       // default level
app.Use(compress.New(compress.Config{         // or pick a level
    Level: compress.LevelBestCompression,
}))
```

## Custom error handling (optional)

When a Handler returns a non-nil error, the default writes 500. Customize with `WithErrorHandler`:

```go
app := httpsrv.New(httpsrv.WithErrorHandler(func(c httpsrv.Ctx, err error) {
	_ = c.Status(500).JSON(map[string]string{"error": err.Error()})
}))
```

## Concurrency safety

- Route matching is a read-only operation, naturally concurrency-safe.
- Avoid registering routes / middleware during `Run`; do all registration before startup.
- Unit tests pass under `-race`: `go test -race ./...`.
