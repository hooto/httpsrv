# Ctx 与 Handler

`Ctx`（请求上下文）与 `Handler`（`func(c Ctx) error`）是 v2 的**典型处理器写法**：接收 Ctx、返回 error，路径参数、查询、请求头、JSON、表单、模板渲染等都通过 Ctx 完成。

httpsrv 的 `Ctx` 基于 `*http.Request`/`http.ResponseWriter`，路径参数来自标准库 `r.PathValue`。


## 注册

路由注册方法（`Get/Post/.../All`）接受 `Handler`（典型用法），也接受标准库 `http.Handler`：

```go
app := httpsrv.New()

// Handler (Ctx)：典型用法
app.Get("/users/{id}", func(c httpsrv.Ctx) error {
	return c.SendString("id=" + c.Params("id"))
})

// http.Handler：直接复用标准库处理器（用 http.HandlerFunc 显式包装 func）
app.Get("/basic", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "basic")
}))
```

## Ctx 核心方法

| 分类 | 方法 | 说明 |
|---|---|---|
| 请求/响应 | `Request() *http.Request`、`Response() http.ResponseWriter` | 原始 stdlib 对象 |
| 标识 | `Method()`、`Path()`、`IP()`、`BaseURL()` | `BaseURL` 返回「协议 + 主机 + base path」 |
| 输入 | `Params(key)`、`Query(key, def...)`、`Header(key, def...)`、`Body()`、`FormValue(key, def...)`、`Bind(out)` | 路径参数 / query / 请求头 / body / 表单字段 / JSON→结构体 |
| 输出 | `Status(code)`、`SetHeader(k,v)`、`JSON(v)`、`Send(b)`、`SendString(s)`、`Redirect(status, url)` | `Status`/`SetHeader` 链式 |

> `Status`/`SetHeader` 需在写入响应体之前调用（与标准库一致）。

## 示例

```go
type Resp struct {
	OK bool   `json:"ok"`
	ID string `json:"id"`
}

app.Get("/users/{id}", func(c httpsrv.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).SendString("missing id") // 自行写响应，返回 nil
	}
	return c.JSON(&Resp{OK: true, ID: id})
})
```

查询参数、请求头与 body：

```go
app.Post("/echo", func(c httpsrv.Ctx) error {
	name := c.Query("name", "anon")        // 带默认值
	tok := c.Header("X-Token")             // 请求头
	return c.SendString(name + ":" + tok + ":" + string(c.Body()))
})
```

## JSON 解析（Bind）

`Bind(out any) error` 把 JSON 请求体反序列化到结构体（`json.Unmarshal`），返回解析错误：

```go
type Login struct {
	User string `json:"user"`
	Pass string `json:"pass"`
}

app.Post("/login", func(c httpsrv.Ctx) error {
	var in Login
	if err := c.Bind(&in); err != nil {
		return c.Status(400).SendString("invalid json: " + err.Error())
	}
	return c.SendString(in.User + ":" + in.Pass)
})
```

- 空请求体视为无操作（返回 `nil`）。
- `Bind` 读取的是已缓存的请求体，与 `Body()`/`FormValue()` 调用顺序无关（先 `Bind` 再 `Body()` 仍能取到完整内容）。

## 表单（Form）

`FormValue` 取请求体里的表单字段，支持 `application/x-www-form-urlencoded` 与 `multipart/form-data`，并带默认值：

```go
app.Post("/login", func(c httpsrv.Ctx) error {
	user := c.FormValue("username")
	pass := c.FormValue("password")
	if user == "" {
		return c.Status(400).SendString("missing username")
	}
	_ = pass
	return c.SendString("ok")
})
```

文件上传经标准库 `c.Request()`：`req.FormFile("file")`、`req.ParseMultipartForm(maxBytes)`。

> `Body()` 与 `FormValue()` 已内部处理互斥：调用顺序任意，先 `Body()` 再 `FormValue()`（或反之）都能正常取值，不会因请求体流被读空而失败。

## 错误处理

`Handler` 返回**非 nil error** 时，框架默认返回 **500**（写入 error 文本）。若 handler 已自行写响应（如上面的 400），返回 `nil` 即可：

```go
app.Get("/boom", func(c httpsrv.Ctx) error {
	return errors.New("something failed") // -> 500
})
```

## BaseURL / IP

- `BaseURL()` 返回 `协议://主机`（尊崇 `X-Forwarded-Proto`/`X-Forwarded-Host`、TLS；目前无 base-path 配置，base path 为空）。
- `IP()` 返回客户端地址（尊崇 `X-Forwarded-For` 第一跳、`X-Real-Ip`，回退 `RemoteAddr`）。
