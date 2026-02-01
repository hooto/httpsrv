## Router

Router is a rule engine for matching HTTP request URL to specific Controller/Action.

## URL Naming Conventions

httpsrv uses fixed routing rules:

```
/{module-path}/{controller}/{action}
```

Where:
- `module-path`: Mounting path specified when calling `Service.HandleModule(baseuri, module)`
- `controller`: Controller name (lowercase, file name)
- `action`: Action name (lowercase, method name ending with `Action`)

## Example

```go
// main.go
httpsrv.GlobalService.HandleModule("/api/v1", NewApiV1Module())
```

If there is a Controller named `User` with an Action named `LoginAction`, then:

| URL | Description |
|-----|-----------|
| `/api/v1/user/login` | Match User Controller's LoginAction |
| `/api/v1/user/register` | Match User Controller's RegisterAction |
| `/api/v1/user` | Match User Controller's IndexAction (default) |

## Multiple Routing Matches

### Basic Routing

```go
// Mount module at root path
httpsrv.GlobalService.HandleModule("/", NewModule())
```

Access path: `/user/login`

### Module Prefix Routing

```go
// Mount module with prefix
httpsrv.GlobalService.HandleModule("/api/v1", NewModule())
```

Access path: `/api/v1/user/login`

### Index Default Routing

```go
type User struct {
    *httpsrv.Controller
}

// This Action is default for /user or /user/ requests
func (c User) IndexAction() {
    c.RenderString("User Index")
}
```

Access paths:
- `/user` → IndexAction
- `/user/` → IndexAction
- `/user/list` → ListAction

### Camel Case Routing

httpsrv automatically handles CamelCase to lowercase:

```go
type UserProfile struct {
    *httpsrv.Controller
}

func (c UserProfile) UpdateInfoAction() {
    c.RenderString("Update Info")
}
```

Access path: `/user-profile/update-info`

## Module Mounting and Nesting

```go
func main() {
    // Mount multiple modules
    httpsrv.GlobalService.HandleModule("/api/v1", NewApiV1Module())
    httpsrv.GlobalService.HandleModule("/api/v2", NewApiV2Module())
    httpsrv.GlobalService.HandleModule("/admin", NewAdminModule())
    httpsrv.GlobalService.HandleModule("/", NewFrontendModule())
    
    httpsrv.GlobalService.Start()
}
```

## Static File Routing

```go
func NewModule() *httpsrv.Module {
    mod := httpsrv.NewModule()
    
    // Register static file server
    mod.RegisterFileServer("/assets", "./static", nil)
    
    return mod
}
```

Access path:
- `/assets/css/style.css` → `./static/css/style.css`
- `/assets/js/app.js` → `./static/js/app.js`

## Routing Priority

httpsrv matches routes in the following order:

1. Exact match
2. Static files
3. Controller/Action match

## Complete Example

```go
package main

import "github.com/hooto/httpsrv"

type User struct {
    *httpsrv.Controller
}

func (c User) IndexAction() {
    c.RenderString("User List")
}

func (c User) ShowAction() {
    id := c.Params.Value("id")
    c.RenderString("Show User: " + id)
}

type Product struct {
    *httpsrv.Controller
}

func (c Product) ListAction() {
    c.RenderString("Product List")
}

func NewModule() *httpsrv.Module {
    mod := httpsrv.NewModule()
    mod.RegisterController(new(User), new(Product))
    mod.RegisterFileServer("/static", "./public", nil)
    return mod
}

func main() {
    httpsrv.GlobalService.HandleModule("/api", NewModule())
    httpsrv.GlobalService.Config.HttpPort = 8080
    httpsrv.GlobalService.Start()
}
```

Access paths:
- `/api/user` → User.IndexAction
- `/api/user/show` → User.ShowAction
- `/api/product/list` → Product.ListAction
- `/api/static/logo.png` → Serve static file

## Notes

1. Controller and Action names are case-insensitive in URLs
2. Controller names ending with `Action` in method names are automatically recognized
3. Missing action defaults to `IndexAction`
4. Missing controller defaults to module's `IndexController`'s `IndexAction`

## Comparison with Other Frameworks

| Framework | Routing Style |
|-----------|--------------|
| httpsrv | /{module}/{controller}/{action} |
| Express.js | app.get('/path', handler) |
| Spring MVC | @RequestMapping("/path") |
| Django | urlpatterns = [...] |

httpsrv's routing is more explicit and consistent, suitable for rapid development and team collaboration.