# 完整示例

一个最小的 RESTful 用户资源服务，演示：分组、全局中间件、路径参数、JSON 响应（采用 `Ctx`/`Handler` 风格）。

```go
package main

import (
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/hooto/httpsrv/v2"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

var (
	mu     sync.Mutex
	users  = map[int]User{1: {ID: 1, Name: "alice"}, 2: {ID: 2, Name: "bob"}}
	nextID = 3
)

func main() {
	app := httpsrv.New()

	// 全局日志中间件（gofiber v3 风格：Handler + c.Next()）
	app.Use(func(c httpsrv.Ctx) error {
		err := c.Next()
		log.Printf("%s %s", c.Method(), c.Path())
		return err
	})

	api := app.Group("/api")

	// 列表
	api.Get("/users", func(c httpsrv.Ctx) error {
		mu.Lock()
		defer mu.Unlock()
		list := make([]User, 0, len(users))
		for _, u := range users {
			list = append(list, u)
		}
		return c.JSON(list)
	})

	// 详情（路径参数）
	api.Get("/users/{id}", func(c httpsrv.Ctx) error {
		id, err := strconv.Atoi(c.Params("id"))
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(map[string]string{"error": "invalid id"})
		}
		mu.Lock()
		defer mu.Unlock()
		u, ok := users[id]
		if !ok {
			return c.Status(http.StatusNotFound).JSON(map[string]string{"error": "not found"})
		}
		return c.JSON(u)
	})

	// 创建
	api.Post("/users", func(c httpsrv.Ctx) error {
		var u User
		if err := c.Bind(&u); err != nil {
			return c.Status(http.StatusBadRequest).JSON(map[string]string{"error": "invalid body"})
		}
		mu.Lock()
		u.ID = nextID
		nextID++
		users[u.ID] = u
		mu.Unlock()
		return c.Status(http.StatusCreated).JSON(u)
	})

	log.Println("listening on :8080")
	log.Fatal(app.Run(":8080"))
}
```

试运行：

```bash
$ go run .

$ curl http://localhost:8080/api/users
[{"id":1,"name":"alice"},{"id":2,"name":"bob"}]

$ curl http://localhost:8080/api/users/1
{"id":1,"name":"alice"}

$ curl -X POST http://localhost:8080/api/users \
    -H 'Content-Type: application/json' \
    -d '{"name":"carol"}'
{"id":3,"name":"carol"}
```

