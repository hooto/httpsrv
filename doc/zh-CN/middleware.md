# 中间件 Use

## 中间件类型

中间件就是一个 [`Handler`](./ctx.md)（`func(Ctx) error`），与 gofiber v3 一致，没有单独的中间件类型。中间件做完自己的事，调用 `c.Next()` 继续链路：

```go
func logging(c httpsrv.Ctx) error {
	start := time.Now()
	if err := c.Next(); err != nil { // 继续执行后续中间件 / 路由处理器
		return err
	}
	log.Printf("%s %s %v", c.Method(), c.Path(), time.Since(start))
	return nil
}
```

## 注册：Use

`Use` 签名：

```go
Use(args ...any) Router
```

- 若第一个参数为 `string`，则它是**路径前缀**；其余参数为中间件。
- 若第一个参数不是字符串，则全部参数为**全局**中间件。

### 全局中间件

```go
app.Use(func(c httpsrv.Ctx) error {
	c.SetHeader("X-Frame-Options", "DENY")
	return c.Next()
})
```

对**所有**请求、**所有** HTTP 方法生效（包括未命中的 404，与 gofiber v3 默认行为一致）。

### 前缀作用域中间件

```go
app.Use("/api", func(c httpsrv.Ctx) error {
	// 仅对 /api/* 生效
	return c.Next()
})
```

前缀按 **`/` 段边界**匹配，因此 `/api` 会命中 `/api/users`，但**不会**命中 `/apifoo`。

一次 `Use` 可传入多个中间件：

```go
app.Use("/api", logging, recoverer, cors)
```

## 注册顺序

中间件按**注册顺序**执行，先注册的先运行、最外层：

```go
app.Use(mwA) // 先执行
app.Use(mwB) // 后执行
// 执行顺序：mwA -> mwB -> 路由处理器
```

## 短路（不调用 Next）

中间件如果不调用 `c.Next()`，链路终止，路由处理器不会执行：

```go
app.Use(func(c httpsrv.Ctx) error {
	if !authorized(c) {
		c.Status(http.StatusUnauthorized)
		return c.SendString("forbidden") // 不调用 Next，直接结束
	}
	return c.Next()
})
```

## 错误冒泡

任意 `Handler`（中间件或路由处理器）返回的非 `nil` 错误会沿 `c.Next()` 一路冒泡到链路顶端，交由 `ErrorHandler` 处理（默认返回 500；可用 `WithErrorHandler` 自定义）。中间件也可拦截并处理下游错误：

```go
app.Use(func(c httpsrv.Ctx) error {
	err := c.Next()
	if err != nil {
		log.Printf("handler error: %v", err) // 记录、转换或吞掉错误
	}
	return err
})
```

## 分组级中间件

`Group.Use` 自动把组前缀拼到中间件路径上：

```go
api := app.Group("/api")
api.Use(rateLimit) // 仅对 /api/* 生效
api.Get("/users", listUsers)
```

## 参数类型

`Use` 同时接受具名 `Handler` 类型与内联 `func(Ctx) error`：

```go
var myMw httpsrv.Handler = func(c httpsrv.Ctx) error { /* ... */ }

app.Use(myMw, func(c httpsrv.Ctx) error { /* ... */ })
```

## 内置中间件

- `compress.New()`（`middleware/compress` 子包）：按 `Accept-Encoding` 协商压缩编码。默认仅内置 gzip（标准库实现，零第三方依赖）；brotli、zstd 等编码通过 `compress.Register` 注入，见 [server](server.md) 的响应压缩一节。
- `httpsrv.AcceptLanguage(def, others...)`：按 `Accept-Language` 探测请求语言（框架内实现的 RFC 5646 子集，见 [i18n](i18n.md)）。

```go
import "github.com/hooto/httpsrv/v2/middleware/compress"

app.Use(compress.New())                       // 默认压缩级别
app.Use(compress.New(compress.Config{         // 可选配置
    Level: compress.LevelBestSpeed,
}))
app.Use(httpsrv.AcceptLanguage("en", "zh"))
```

## 限制（基础实现）

- 中间件的路径前缀仅支持**静态前缀**，不支持 `/users/{id}` 形式的参数前缀。
- 路径参数在中间件调用 `c.Next()` **之前不可用**（路由匹配发生在链路终点 `dispatch` 内）；`c.Next()` 返回后可通过 `c.Params()` 读取。
