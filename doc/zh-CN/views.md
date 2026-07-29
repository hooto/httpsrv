# 模板渲染

httpsrv 提供基于 `html/template` 的模板引擎，通过 `Ctx.Render(name, bind, layouts...)` 在 Handler 中渲染页面。

## Views 接口

模板引擎实现 `Views` 接口：

```go
type Views interface {
    Load() error
    Render(w io.Writer, name string, bind any, layout ...string) error
}
```

内置引擎即满足该接口；你也可以实现自己的引擎并接入。

## 构造引擎

两种来源（均返回 `Views, error`，模板在构造时**一次性解析**，因此 `{{template "x"}}` 包含可用）：

```go
// 从文件系统目录
r, err := httpsrv.TemplatesDir("./views", nil)
```

```go
// 从 embed.FS（推荐用于单二进制部署）
//go:embed views/*
var views embed.FS
sub, _ := fs.Sub(views, "views")
r, err := httpsrv.TemplatesFS(sub, nil)
```

第二个参数 `extraFuncs template.FuncMap` 用于追加自定义模板函数（传 `nil` 即只用内建函数）。

## 挂载到 App

通过 `WithViews` 挂载（接受任意 `Views` 实现）：

```go
app := httpsrv.New(httpsrv.WithViews(r))
```

也接受自定义引擎：

```go
app := httpsrv.New(httpsrv.WithViews(myEngine{}))
```

未挂载时调用 `c.Render(...)` 会返回错误（默认 → 500）。

## 渲染

```go
app.Get("/users/{id}", func(c httpsrv.Ctx) error {
    return c.Render("users/show.html", map[string]any{
        "Name": "alice",
        "Lang": "zh",
    })
})
```

- `name`：模板文件相对路径（含扩展名，如 `users/show.html`）。
- `bind`：绑定到模板的数据。
- `layouts...`：可选布局模板，见下。
- 响应自动写为 `text/html; charset=utf-8`。

## 布局（layout）

```go
return c.Render("page.html", data, "layout.html")
```

- 先渲染 `page.html` 得到内容，再渲染 `layout.html`，把内容作为 `.Content` 传入。
- 布局可用 `{{.Content}}` 嵌入内层 HTML（已标记为安全，不会被二次转义）。
- 当 `bind` 是 `map[string]any` 时，其字段也会合并进布局数据，故布局可用 `{{.Title}}` 等。
- 多个 layout 按顺序层层包裹，最后一个最外层。

```html
<!-- layout.html -->
<html><head><title>{{.Title}}</title></head><body>{{.Content}}</body></html>
```

## 内建模板函数

| 函数 | 说明 |
|---|---|
| `raw s` | 输出不转义的 HTML（仅用于可信内容，禁止用于用户可控数据，见下方「安全须知」） |
| `replace s old new` | 字符串替换 |
| `upper s` / `lower s` | 转大写 / 小写 |
| `date t` / `datetime t` | 格式化时间（`2006-01-02` / `2006-01-02 15:04`） |
| `T locale key args...` | i18n 翻译 |

追加自定义函数：

```go
r, _ := httpsrv.TemplatesFS(sub, template.FuncMap{
    "exclaim": func(s string) string { return s + "!" },
})
```

## 安全须知

**模板名 `name` 必须可信。** `Render` 按 `name` 在已解析的模板集合中查找并渲染；任意一个已加载的模板都能按名被渲染，包括你不打算对外暴露的页面（如后台模板）。框架会折叠 `..` 等路径片段，因此不存在目录穿越或任意文件读取，但**不会**限制可选模板的范围。所以切勿把用户输入直接作为 `name`（例如 `c.Render(c.Params("page")+".html", ...)`）；确需动态选择时请用白名单校验。

**默认 HTML 转义。** 所有插值默认经 `html/template` 上下文感知转义，普通数据是安全的。

**`raw` 会跳过转义。** `{{raw .X}}` 把内容原样作为 HTML 输出。仅当内容完全可信时使用；**绝不**对用户可控的数据使用，否则会引入 XSS。

## i18n

```go
i := httpsrv.NewI18n("en")            // 默认语言
i.Add("zh", map[string]string{"hi": "你好"})
i.Add("en", map[string]string{"hi": "Hello"})

r, _ := httpsrv.TemplatesFS(sub, i.Funcs()) // 把 T 函数注入模板
```

模板中：`{{T .Lang "hi"}}`，按 locale 取值，找不到回退默认语言，再回退 key 本身；带 args 时按 `fmt.Sprintf` 格式化。

> 若还要在 Handler 中使用 `c.Translate` / `c.Locale`，需用 `httpsrv.AcceptLanguage` 中间件探测语言，并把 store 挂到 App：`httpsrv.New(httpsrv.WithI18n(i))`（详见 [i18n 国际化](./i18n.md)）。

## 错误处理

- 未配置 Views 引擎、模板名不存在、模板执行出错 → `Render` 返回 error → 默认 **500**。
- 默认 500 响应体为固定文案（`Internal Server Error`），原始 error 通过 `slog` 记录到服务端日志，**不会**回写到客户端，避免泄露模板路径、表达式等内部信息。如需自定义错误响应，用 `WithErrorHandler`。
- 模板**解析**错误在 `TemplatesDir`/`TemplatesFS` 构造时即返回（fail-fast）。

## 完整示例

```go
package main

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/hooto/httpsrv/v2"
)

//go:embed views/*
var views embed.FS

func main() {
	sub, _ := fs.Sub(views, "views")
	i := httpsrv.NewI18n("en")
	i.Add("zh", map[string]string{"welcome": "欢迎"})
	i.Add("en", map[string]string{"welcome": "Welcome"})

	r, err := httpsrv.TemplatesFS(sub, i.Funcs()) // 注入模板函数 T
	if err != nil {
		panic(err)
	}

	app := httpsrv.New(httpsrv.WithViews(r))
	app.Get("/", func(c httpsrv.Ctx) error {
		return c.Render("home.html",
			map[string]any{"Lang": "zh", "Title": "Home"},
			"layout.html")
	})

	app.Run(":8080")
}
```
