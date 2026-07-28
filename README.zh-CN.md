# httpsrv

httpsrv 是一个轻量级、net/http 原生的 Go Web 框架。它提供 Fiber v3 风格的路由接口（`App`/`Router`/`Ctx`/`Handler`），完全基于标准库从零实现。

**语言:** [English](README.md) | [中文](README.zh-CN.md)

## 特性

- **net/http 原生**：典型处理函数为 `func(Ctx) error`，同时也接受标准库的 `http.Handler`，可与 Go 生态直接互通
- **radix 树路由**：按方法分树、`{param}` 与 `{*catchAll}` 路径参数、大小写敏感、静态段优先于参数优先于 catch-all（与注册顺序无关）
- **中间件与分组**：`Use` 使用 fiber v3 风格中间件（`func(Ctx) error` + `c.Next()`），全局或按前缀作用域；`Group(prefix)` 共享前缀
- **静态文件**：`middleware/static` 子包提供 `New(root)` / `FS(http.FS(embed))`，返回 `httpsrv.Handler`：`app.Get("/static/{*path}", static.New(root))`；支持目录或嵌入式文件系统
- **模板渲染**：`html/template` 一次解析，`Ctx.Render(name, bind, layouts...)`
- **i18n（可选）**：扁平的 locale 消息存储，`AcceptLanguage` 中间件（经 `x/text` 做 BCP-47 匹配）
- **压缩（可选）**：`middleware/compress` 子包提供 `New(config...)`（按 `Accept-Encoding` 选 gzip/brotli，brotli 优先）
- **优雅服务**：安全的默认超时，`Shutdown(ctx)`

## 文档

完整指南（中文）：[`doc/zh-CN/`](doc/zh-CN/README.md)，[快速开始](doc/zh-CN/quickstart.md) · [路由](doc/zh-CN/routing.md) · [分组](doc/zh-CN/groups.md) · [中间件](doc/zh-CN/middleware.md) · [服务](doc/zh-CN/server.md) · [静态文件](doc/zh-CN/static.md) · [Ctx 与 Handler](doc/zh-CN/ctx.md) · [模板渲染](doc/zh-CN/views.md) · [i18n](doc/zh-CN/i18n.md) · [示例](doc/zh-CN/examples.md) · [目录](doc/zh-CN/SUMMARY.md)

英文文档：[`doc/`](doc/README.md)

## 安装

```bash
go get -u github.com/hooto/httpsrv/v2
```

## 快速开始

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

运行并访问：

```bash
$ go run .
$ curl http://localhost:8080/
hello httpsrv
```

可运行示例见 [`examples/`](examples)（hello、i18n、分组/静态/模板）。

## 推荐依赖库

httpsrv 保持核心简洁，以下是一些推荐使用的第三方库：

### 数据库
- [mysqlgo](https://github.com/lynkdb/mysqlgo) - MySQL 客户端
- [pgsqlgo](https://github.com/lynkdb/pgsqlgo) - PostgreSQL 客户端
- [redisgo](https://github.com/lynkdb/redisgo) - Redis 客户端
- [kvgo](https://github.com/lynkdb/kvgo) - 嵌入式 Key-Value 数据库

### 工具库
- [hlog4g](https://github.com/hooto/hlog4g) - 日志库
- [hini4g](https://github.com/hooto/hini4g) - INI 配置文件解析
- [hflag4g](https://github.com/hooto/hflag4g) - 命令行参数处理
- [hlang4g](https://github.com/hooto/hlang4g) - i18n 国际化
- [hcaptcha4g](https://github.com/hooto/hcaptcha4g) - 验证码生成

更多 Go 生态库可参考：[awesome-go](https://github.com/avelino/awesome-go)

## 系统要求

- **Go 版本**: 1.26 或更高
- **推荐系统**: Linux、Unix 或 macOS

## 参考项目

httpsrv 的 API 接口（`App`/`Router`/`Ctx`/`Handler`）参考了 [Fiber v3](https://github.com/gofiber/fiber/v3)，基于 net/http 的从零实现，不依赖 fiber。

## 许可证

[Apache License 2.0](LICENSE)
