# 路由分组

使用 `Group(prefix)` 创建子路由，其下注册的路由会自动带上前缀。`Group` 也返回 `Router`，可继续链式注册或再次分组。

## 基本用法

```go
api := app.Group("/api")

api.Get("/users", listUsers)        // -> GET  /api/users
api.Get("/users/{id}", getUser)     // -> GET  /api/users/{id}
api.Post("/users", createUser)      // -> POST /api/users
```

上例中的 `listUsers`、`getUser` 等为 `httpsrv.Handler`（`func(c httpsrv.Ctx) error`）。

## 嵌套分组

分组可以嵌套，前缀逐级拼接：

```go
api := app.Group("/api")
v1 := api.Group("/v1") // 前缀 /api/v1

v1.Get("/items", listItems)         // -> GET /api/v1/items
v1.Get("/items/{id}", getItem)      // -> GET /api/v1/items/{id}
```

## 链式

```go
app.Group("/api").
	Get("/users", listUsers).
	Post("/users", createUser)
```

## 与参数、中间件组合

分组内可正常使用路径参数，并可通过 `Use` 注册组级中间件（见 [中间件](./middleware.md)）：

```go
admin := app.Group("/admin")
admin.Use(authMiddleware)               // 仅对 /admin/* 生效
admin.Get("/users/{id}", adminGetUser)  // 带参数，且经过 authMiddleware
```
