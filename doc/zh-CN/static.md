# 静态文件 Static

httpsrv 通过独立的 `middleware/static` 子包提供静态文件服务。`static.New` 返回一个 `httpsrv.Handler`（参考 gofiber v3 的 `static.New`），注册到 catch-all 路由上即可服务整个文件树。它从文件系统目录或嵌入 FS 读取并返回文件。

```go
import (
	"github.com/hooto/httpsrv/v2"
	"github.com/hooto/httpsrv/v2/middleware/static"
)
```

## 构造函数

| 函数 / 类型 | 说明 |
|---|---|
| `static.New(root string, config ...Config) httpsrv.Handler` | `root` 为目录路径（经 `http.Dir` 防穿越）；设置 `Config.FS` 后，`root` 改为该 `fs.FS` 内的子路径（`.`/`""` 表示整个 FS） |
| `static.Config` | 配置类型，字段 `FS fs.FS`（`io/fs` 文件系统，如 `embed.FS`） |

## 注册：catch-all 路由

注册到 catch-all 路由上以服务整个文件树。gofiber v3 风格的无名通配符 `/*` 是推荐写法（无需 `{*path}` 命名参数）：

```go
app.Get("/*", static.New("./public"))
// GET /css/main.css  ->  ./public/css/main.css

app.Get("/static/*", static.New("./assets"))
// GET /static/css/main.css  ->  ./assets/css/main.css
```

也可用具名 catch-all `{*path}`：

```go
app.Get("/static/{*path}", static.New("./assets"))
// GET /static/css/main.css  ->  ./assets/css/main.css
```

catch-all 值（前缀之后的部分）以约定键 `"*"` 暴露（`r.PathValue("*")` / `c.Params("*")`），与参数名无关。详见 [路由 / catch-all 参数](./routing.md#catch-all-参数-name)。

- `app.Get` 只响应 GET；`app.All` 对所有方法生效。
- 也可注册到 `Group`，前缀自动拼接（见下）。

### 从 embed 提供服务

```go
//go:embed assets/*
var assets embed.FS

// root 为 FS 内子路径，"." 表示整个 FS
app.Get("/*", static.New(".", static.Config{FS: assets}))
app.Get("/static/*", static.New("dist", static.Config{FS: assets})) // 只暴露 dist 子树
```

## 行为

- **显式路由优先**：catch-all（`/*` 与 `{*name}`）是最弱的匹配，静态段与普通参数段都优先于它，因此显式路由天然胜出。
- **目录不列出**：请求命中目录时返回 404（不做目录索引）。
- **路径穿越防护**：基于 `http.Dir` 防止逃逸根目录，例如 `/static/../secret` 返回 404。
- **裸前缀不命中**：`/*` 不会匹配根路径 `/` 本身（与具名 catch-all 一致），需 `/...` 形式。
- 底层使用标准库 `http.ServeContent`，自动支持 GET/HEAD、Range 请求、`Last-Modified`/`ETag`。

## 在分组上注册

`Group` 的路由会自动拼接组前缀：

```go
assets := app.Group("/assets")
assets.Get("/img/*", static.New("./images"))
// GET /assets/img/logo.png  ->  ./images/logo.png
```

## 完整示例

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

	// 文件系统目录
	app.Get("/files/*", static.New("./local-files"))

	// 嵌入资源（整个 FS）
	app.Get("/public/*", static.New(".", static.Config{FS: public}))

	_ = app.Run(":8080")
}
```
