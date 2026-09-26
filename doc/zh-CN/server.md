# 启动服务 Run

## 签名

```go
Run(args ...any) error
```

`Run` 会创建一个 `*http.Server`（Handler 为应用本身）并阻塞，直到服务停止。

## 监听地址

传入字符串作为监听地址：

```go
app.Run(":8080")          // 默认：0.0.0.0:8080
app.Run("127.0.0.1:3000") // 仅本机
app.Run(":0")             // 随机可用端口
```

不传任何参数时，使用默认地址 `:8080`：

```go
app.Run()
```

## 使用已有 Listener

传入 `net.Listener`，适合自定义网络类型或外部控制监听生命周期：

```go
ln, err := net.Listen("tcp", ":8080")
if err != nil {
	log.Fatal(err)
}
app.Run(ln)
```

也可用于测试或 Unix Socket：

```go
ln, _ := net.Listen("unix", "/tmp/app.sock")
app.Run(ln)
```

## 返回值

- 服务正常停止时，`http.ErrServerClosed` 会被吞掉，`Run` 返回 `nil`。
- 其它错误（如端口占用、监听失败）会原样返回。

## 注册时机

**路由与中间件必须在 `Run` 之前注册完毕。** 服务启动后，路由树进入只读服务状态；每个请求以 `RLock` 读取，可安全并发处理。

## 404 默认处理

未匹配到任何路由时，由内置的 `defaultNotFound` 处理，返回：

```
HTTP/1.1 404 Not Found
Content-Type: text/plain; charset=utf-8

404 page not found
```

## 优雅关闭 Shutdown

`app.Shutdown(ctx)` 优雅关闭服务（等活跃连接结束或 `ctx` 超时），需在 `Run` 阻塞期间从另一个 goroutine 调用：

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

`Run` 在优雅关闭后返回 `nil`（`http.ErrServerClosed` 已被吞掉）。

## Config（超时 / 地址 / 头限制）

`Run` 默认带上安全的服务端配置：读/写超时 60s、读头超时 10s、`MaxHeaderBytes` 1MiB。用 `New(WithConfig(...))` 覆盖（仅非零字段生效）：

```go
app := httpsrv.New(httpsrv.WithConfig(httpsrv.Config{
	Addr:              ":3000",
	ReadTimeout:       30 * time.Second,
	WriteTimeout:      30 * time.Second,
	ReadHeaderTimeout: 10 * time.Second,
	MaxHeaderBytes:    1 << 20,
}))
```

> `WriteTimeout` 会限制单次响应时长；做长连接/SSE/大文件下载时可设为 `0`。

## 响应压缩（可选）

`middleware/compress` 子包的 `New()` 按 `Accept-Encoding` 协商压缩编码，并设置 `Content-Encoding`/`Vary`、移除 `Content-Length`。默认仅内置 gzip（标准库实现，模块零第三方依赖）；brotli、zstd 等编码通过 `Register` 注入，压缩实现由应用侧自行依赖：

```go
import "github.com/hooto/httpsrv/v2/middleware/compress"

app.Use(compress.New())                       // 默认级别
app.Use(compress.New(compress.Config{         // 或指定级别
    Level: compress.LevelBestCompression,
}))
```

协商规则：已注册编码按注册顺序参与，内置 gzip 最后；客户端 `Accept-Encoding` 的权重（q 值）决定胜者，权重相同时靠前者优先，`q=0` 的编码不会被选用。`Config.Level` 仅作用于内置 gzip；`Register` 注入的编码在闭包中自带级别设置。`Register` 需在 `New` 之前调用，协商选项在中间件构造时固定。

注入 brotli（应用自行依赖 `github.com/andybalholm/brotli`）：

```go
import (
    "io"

    "github.com/andybalholm/brotli"
    "github.com/hooto/httpsrv/v2/middleware/compress"
)

func init() {
    compress.Register("br", func(w io.Writer) io.WriteCloser {
        return brotli.NewWriterLevel(w, 5) // quality 5: 压缩比与速度均衡
    })
}
```

注册后，`Accept-Encoding: gzip, br` 会选择 `br`。

## 自定义错误处理（可选）

Handler 返回非 nil error 时，默认写 500。可用 `WithErrorHandler` 自定义：

```go
app := httpsrv.New(httpsrv.WithErrorHandler(func(c httpsrv.Ctx, err error) {
	_ = c.Status(500).JSON(map[string]string{"error": err.Error()})
}))
```

## 并发安全

- 路由匹配为只读操作，天然支持并发。
- `Run` 期间应避免再注册路由 / 中间件；注册请集中在启动前完成。
- 单元测试已通过 `-race`：`go test -race ./...`。
