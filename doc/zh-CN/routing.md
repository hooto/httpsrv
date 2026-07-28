# 路由注册与路径参数

## 按方法注册

```go
app.Get("/items", listItems)
app.Post("/items", createItem)
app.Put("/items/{id}", updateItem)
app.Delete("/items/{id}", deleteItem)
```

方法与注册函数一一对应：`Get/Head/Post/Put/Patch/Delete/Options`。上例中的 `listItems` 等为 `httpsrv.Handler`（即 `func(c httpsrv.Ctx) error`）。**不同方法互不影响**：只有 GET 路由时，对同一路径发起 POST 会得到 404。

## 链式注册

注册函数返回 `Router`，可链式书写：

```go
app.Get("/a", hA).
	Get("/b", hB).
	Post("/c", hC)
```

## 路径参数 `{name}`

使用花括号定义参数段，处理器内通过 `c.Params(name)` 读取（底层即标准库 `r.PathValue`）：

```go
app.Get("/users/{id}", func(c httpsrv.Ctx) error {
	return c.SendString("id=" + c.Params("id"))
})
```

- 参数匹配 `[^/]+`，即单个路径段（不含 `/`）。
- 支持多参数：

```go
app.Get("/posts/{post_id}/comments/{comment_id}", func(c httpsrv.Ctx) error {
	return c.SendString(c.Params("post_id") + " " + c.Params("comment_id"))
})
// GET /posts/5/comments/22  ->  5 22
```

- 参数值会保留原始内容，例如 `GET /users/news.html` 中 `{id}` = `news.html`。

> ⚠️ 仅支持 `{name}` 语法；`:name` 是**字面量**，不会被当作参数。

## catch-all 参数 `{*name}`

`{*name}` 匹配路径的**剩余全部内容**（含 `/`），且**必须是模式的最后一段**：

```go
app.Get("/files/{*path}", func(c httpsrv.Ctx) error {
	return c.SendString("file=" + c.Params("path"))
})
// GET /files/a.txt          ->  path = "a.txt"
// GET /files/css/main.css   ->  path = "css/main.css"
// GET /files/deep/nested/x  ->  path = "deep/nested/x"
```

- catch-all 的值不含前导 `/`（如 `css/main.css`）。
- 除具名键外，catch-all 还会以约定键 `"*"` 暴露，可用 `c.Params("*")` 读取（对应 fiber 的 `c.Params("*")` 约定）。
- 优先级：**静态段 > 普通参数 `{name}` > catch-all `{*name}`**，与注册顺序无关。
- `{*name}` 之后不能再有内容，例如 `/files/{*path}/extra` 注册时会 panic。
- 同一位置不能有两个不同名的 catch-all，例如 `/files/{*a}` 与 `/files/{*b}` 冲突；同名重复注册则覆盖 handler。

## 静态路由优先于参数路由

静态段总是优先于同位置的参数段与 catch-all，**与注册顺序无关**：

```go
app.Get("/users/profile", showProfile) // 静态
app.Get("/users/{id}", showUser)       // 参数
app.Get("/users/{*rest}", showRest)    // catch-all
```

```
GET /users/profile  ->  showProfile   （静态命中）
GET /users/42       ->  showUser      （参数命中）
GET /users/a/b/c    ->  showRest      （catch-all 命中）
```

## 大小写敏感

匹配**大小写敏感**：

```go
app.Get("/Users", h)
// GET /Users   ->  命中
// GET /users   ->  404
```

## 路径归一化

注册与匹配都会对路径做 `path.Clean` 归一化：合并重复斜杠、解析 `.`/`..`、保证以 `/` 开头。因此下面的写法等价：

```go
app.Get("//api//users/", h)
// 等价于 app.Get("/api/users", h)
```

请求路径同样会被归一化后再匹配。

## All：注册到所有方法

```go
app.All("/healthz", func(c httpsrv.Ctx) error {
	return c.SendString("ok")
})
// GET/POST/PUT/... /healthz 都会命中
```

## 非法模式会在注册时 panic

为尽早暴露错误，非法的路径模式会触发 panic（已注册的路由不受影响）：

```go
app.Get("/users/{id", h)           // panic: unclosed '{'
app.Get("/users/{}", h)            // panic: empty parameter name
app.Get("/{a}{b}", h)              // panic: 同一段内多个参数
app.Get("/files/{*path}/extra", h) // panic: catch-all 必须是最后一段
```
