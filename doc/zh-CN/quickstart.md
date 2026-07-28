# 快速开始

## 创建应用

```go
app := httpsrv.New()
```

## 注册路由

每个方法对应一个注册函数，返回 `Router` 以支持链式调用。典型用法是传入 `Handler`（`func(c Ctx) error`）：

```go
app.Get("/hello", func(c httpsrv.Ctx) error {
	return c.SendString("hello")
})

app.Post("/echo", func(c httpsrv.Ctx) error {
	// ...
	return nil
})
```

路由注册方法同时接受标准库 `http.Handler`（如 `http.HandlerFunc`），详见 [Ctx 与 Handler](./ctx.md)。

## 启动服务

```go
err := app.Run(":8080") // 监听 0.0.0.0:8080
```

`Run` 会阻塞，直到服务停止。详见 [启动服务 Run](./server.md)。

## 完整最小程序

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

运行后：

```bash
$ go run .
$ curl http://localhost:8080/
Hello, httpsrv!
```

## 可用的注册方法

| 方法 | 说明 |
|---|---|
| `Get/Head/Post/Put/Patch/Delete/Options` | 注册对应 HTTP 方法的路由（接受 `Handler` 或 `http.Handler`，见 [Ctx 与 Handler](./ctx.md)） |
| `All(path, handler)` | 为**所有** HTTP 方法注册同一路由（含 GET/POST/PUT/PATCH/DELETE/HEAD/OPTIONS/CONNECT/TRACE） |
| `Group(prefix)` | 创建带前缀的子路由（见 [路由分组](./groups.md)） |
| `Use(args...)` | 注册中间件（见 [中间件](./middleware.md)） |
| `app.Get("/p/{*path}", static.New(...))` | 静态文件服务（`middleware/static` 子包的 `New`/`FS` 返回 `httpsrv.Handler`，见 [静态文件](./static.md)）；用 `All` 替代 `Get` 可对所有方法生效 |
