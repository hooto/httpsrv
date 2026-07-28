# httpsrv 使用

> **中文** | [English](../README.md)

httpsrv 是一个 net/http 原生的轻量级 Go Web 框架。

## 特性

- **net/http 原生**：典型处理器为 `func(c Ctx) error`，同时也接受标准库 `http.Handler`，可直接复用标准库生态与现成中间件。
- **简洁 API**：按方法注册路由（`Get/Post/Put/...`）、`All`、可嵌套 `Group`、`Use` 中间件，方法调用支持链式。
- **高性能基数树（radix tree）路由**：静态段精确匹配、`{name}` 参数段；**静态路由优先于参数路由**（与注册顺序无关）。
- **路径参数**：`{name}` 段经 `c.Params("name")` 读取（底层即标准库 `r.PathValue`），无需自定义 Context。
- **零依赖**：仅依赖标准库（外加 `andybalholm/brotli` 压缩、`golang.org/x/text` 语言匹配），处理器可直接复用标准库生态与现成中间件。

## 目录

- [快速开始](./quickstart.md)
- [路由注册与路径参数](./routing.md)
- [路由分组](./groups.md)
- [中间件 Use](./middleware.md)
- [启动服务 Run](./server.md)
- [完整示例](./examples.md)
- [静态文件 Static](./static.md)
- [Ctx 与 Handler](./ctx.md)
- [模板渲染 Render](./views.md)
- [i18n 国际化](./i18n.md)

## 最小示例

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

	app.Run(":8080") // 访问 http://localhost:8080/
}
```

## 环境要求

- Go **1.26+**
- 运行测试：`go test ./...`
