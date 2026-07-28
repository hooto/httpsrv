# 静态文件 Static

httpsrv 通过独立的 `middleware/static` 子包提供静态文件服务。`New` / `FS` 返回一个 `httpsrv.Handler`（参考 gofiber v3 的 `static.New(root) -> Handler` 风格），可直接注册到任意路由，它从文件系统（或嵌入 FS）读取并返回文件。

```go
import (
	"github.com/hooto/httpsrv/v2"
	"github.com/hooto/httpsrv/v2/middleware/static"
)
```

## 构造函数

| 函数 | 说明 |
|---|---|
| `static.New(root string) httpsrv.Handler` | 从文件系统目录 `root` 提供服务 |
| `static.FS(fs http.FileSystem) httpsrv.Handler` | 从嵌入文件系统提供服务，如 `http.FS(embed.FS)` |

## 注册：catch-all 路由

通常配合 catch-all 路由 `{*path}` 服务整个文件树：

```go
app.Get("/static/{*path}", static.New("./assets"))
// GET /static/css/main.css  ->  ./assets/css/main.css
```

catch-all 值（路径中前缀之后的部分）会以约定键 `"*"` 暴露，handler 通过 `r.PathValue("*")` 读取，与你给参数起的名字无关。也可用具名键读取：`c.Params("path")`。详见 [路由 / catch-all 参数](./routing.md#catch-all-参数-name)。

- 用 `app.Get` 只响应 GET；用 `app.All` 则对所有 HTTP 方法生效（含 POST/PUT/...）。
- 也可注册到 `Group` 上，前缀自动拼接（见下）。

### 从目录提供服务

```go
app.Get("/static/{*path}", static.New("./assets"))
// GET /static/css/main.css  ->  ./assets/css/main.css
```

### 从 embed 提供服务

```go
//go:embed assets/*
var assets embed.FS

app.Get("/static/{*path}", static.FS(http.FS(assets)))
```

## 行为

- **显式路由优先**：catch-all 是最弱的匹配，静态段与普通参数段都优先于它，因此显式路由天然胜出。
- **目录不列出**：请求命中目录时返回 404（不做目录索引）。
- **路径穿越防护**：基于 `http.Dir` 防止逃逸根目录，例如 `/static/../secret` 返回 404。
- **前缀按段匹配**：`/static` 不会匹配 `/staticfoo`。
- **裸前缀不命中**：`/static`（无尾随段）不会命中 catch-all，需 `/static/...` 形式。
- **经过中间件链**：静态请求同样会经过 `Use` 注册的中间件（`dispatch` 是链的终点）。
- 底层使用标准库 `http.ServeContent`，自动支持 GET/HEAD、Range 请求、`Last-Modified`/`ETag`。

## 在分组上注册

`Group` 的路由会自动拼接组前缀：

```go
assets := app.Group("/assets")
assets.Get("/img/{*path}", static.New("./images"))
// GET /assets/img/logo.png  ->  ./images/logo.png
```

## 完整示例

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

	// 文件系统目录
	app.Get("/files/{*path}", static.New("./local-files"))

	// 嵌入资源
	app.Get("/public/{*path}", static.FS(http.FS(public)))

	_ = app.Run(":8080")
}
```
