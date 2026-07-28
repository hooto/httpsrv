# Template Rendering

httpsrv provides an `html/template`-based engine, `Renderer`, used from a Handler via `Ctx.Render(name, bind, layouts...)`.

## The Views interface

A template engine implements the `Views` interface:

```go
type Views interface {
    Load() error
    Render(w io.Writer, name string, bind any, layout ...string) error
}
```

The built-in `*Renderer` satisfies it; you may also implement your own engine and plug it in.

## Build a Renderer

Two sources (both return `*Renderer, error`; templates are **parsed once** at construction, so `{{template "x"}}` includes work):

```go
// from a filesystem directory
r, err := httpsrv.TemplatesDir("./views", nil)

// from an embed.FS (recommended for single-binary deploys)
//go:embed views/*
var views embed.FS
sub, _ := fs.Sub(views, "views")
r, err := httpsrv.TemplatesFS(sub, nil)
```

The second argument, `extraFuncs template.FuncMap`, adds custom template functions (pass `nil` for built-ins only).

## Attach to the App

Two equivalent forms:

```go
// 1) via WithViews (accepts any Views implementation)
app := httpsrv.New(httpsrv.WithViews(r))

// 2) via Config.Views (same thing, inline with other server settings)
app := httpsrv.New(httpsrv.WithConfig(httpsrv.Config{Views: r}))
```

Both accept any `Views` engine, e.g. a custom one:

```go
app := httpsrv.New(httpsrv.WithViews(myEngine{}))
```

Without an engine, `c.Render(...)` returns an error (default → 500).

## Rendering

```go
app.Get("/users/{id}", func(c httpsrv.Ctx) error {
    return c.Render("users/show.html", map[string]any{
        "Name": "alice",
        "Lang": "zh",
    })
})
```

- `name`: the template file's relative path (with extension, e.g. `users/show.html`).
- `bind`: data bound to the template.
- `layouts...`: optional layout templates, see below.
- The response is written as `text/html; charset=utf-8`.

## Layouts

```go
return c.Render("page.html", data, "layout.html")
```

- `page.html` is rendered first to produce content, then `layout.html` is rendered with the content passed as `.Content`.
- A layout embeds the inner HTML via `{{.Content}}` (marked safe, not double-escaped).
- When `bind` is a `map[string]any`, its fields are merged into the layout data, so the layout can use `{{.Title}}` etc.
- Multiple layouts wrap in order; the last one is the outermost.

```html
<!-- layout.html -->
<html><head><title>{{.Title}}</title></head><body>{{.Content}}</body></html>
```

## Built-in template functions

| Function | Description |
|---|---|
| `raw s` | output unescaped HTML (only for trusted content) |
| `replace s old new` | string replacement |
| `upper s` / `lower s` | uppercase / lowercase |
| `date t` / `datetime t` | format time (`2006-01-02` / `2006-01-02 15:04`) |
| `T locale key args...` | i18n translation |

Add custom functions:

```go
r, _ := httpsrv.TemplatesFS(sub, template.FuncMap{
    "exclaim": func(s string) string { return s + "!" },
})
```

## i18n

```go
i := httpsrv.NewI18n("en")            // default language
i.Add("zh", map[string]string{"hi": "你好"})
i.Add("en", map[string]string{"hi": "Hello"})

r, _ := httpsrv.TemplatesFS(sub, i.Funcs()) // inject the T function into templates
```

In a template: `{{T .Lang "hi"}}` — looks up by locale, falling back to the default language, then the key itself; with args it formats via `fmt.Sprintf`.

> To also use `c.Translate` / `c.Locale` in a Handler, detect the locale with the `httpsrv.AcceptLanguage` middleware and attach the store to the App: `httpsrv.New(httpsrv.WithI18n(i))` (see [i18n](./i18n.md)).

## Error handling

- No Views engine configured, missing template name, or template execution error → `Render` returns an error → default **500**.
- Template **parse** errors surface from `TemplatesDir`/`TemplatesFS` at construction (fail-fast).

## Full example

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

	r, err := httpsrv.TemplatesFS(sub, i.Funcs()) // inject T
	if err != nil {
		panic(err)
	}

	app := httpsrv.New(httpsrv.WithViews(r))
	app.Get("/", func(c httpsrv.Ctx) error {
		return c.Render("home.html",
			map[string]any{"Lang": "zh", "Title": "Home"},
			"layout.html")
	})

	_ = app.Run(":8080")
}
```
