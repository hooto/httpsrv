# Route Groups

Use `Group(prefix)` to create a sub-router; routes registered on it are automatically prefixed. `Group` also returns a `Router`, so you can keep chaining or nest further groups.

## Basic usage

```go
api := app.Group("/api")

api.Get("/users", listUsers)        // -> GET  /api/users
api.Get("/users/{id}", getUser)     // -> GET  /api/users/{id}
api.Post("/users", createUser)      // -> POST /api/users
```

Above, `listUsers`, `getUser`, etc. are `httpsrv.Handler` (`func(c httpsrv.Ctx) error`).

## Nested groups

Groups nest; prefixes are concatenated level by level:

```go
api := app.Group("/api")
v1 := api.Group("/v1") // prefix /api/v1

v1.Get("/items", listItems)         // -> GET /api/v1/items
v1.Get("/items/{id}", getItem)      // -> GET /api/v1/items/{id}
```

## Chaining

```go
app.Group("/api").
	Get("/users", listUsers).
	Post("/users", createUser)
```

## Combining with parameters and middleware

Path parameters work normally inside a group, and `Use` registers group-scoped middleware (see [Middleware](./middleware.md)):

```go
admin := app.Group("/admin")
admin.Use(authMiddleware)               // scoped to /admin/*
admin.Get("/users/{id}", adminGetUser)  // with a parameter, after authMiddleware
```
