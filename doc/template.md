## Template

httpsrv provides view template rendering capabilities using Go's template engine. Templates are typically used in Web MVC's View (V) layer to render HTML pages.

## Setting Up Template Path

Set template path when initializing Module:

```go
func NewModule() *httpsrv.Module {
    mod := httpsrv.NewModule("ui")
    mod.SetTemplatePath("./views/ui")
    return mod
}
```

## Basic Usage

### Simple Template Rendering

```go
type User struct {
    *httpsrv.Controller
}

func (c User) ProfileAction() {
    user := map[string]interface{}{
        "name": "John Doe",
        "email": "john@example.com",
    }
    
    c.Data["user"] = user
    c.Render("user/profile.tpl")
}
```

Template file `user/profile.tpl`:

```html
<!DOCTYPE html>
<html>
<head>
    <title>User Profile</title>
</head>
<body>
    <h1>{{.user.name}}</h1>
    <p>Email: {{.user.email}}</p>
</body>
</html>
```

### Rendering with Specific Module

```go
// Render template from specific module
c.Render("user", "profile.tpl")
```

## Template Syntax

### Variables

```html
{{.variable}}
{{.user.name}}
{{index .users 0}}
```

### Conditional Statements

```html
{{if .user.is_admin}}
    <p>Admin User</p>
{{else}}
    <p>Regular User</p>
{{end}}
```

### Loops

```html
{{range .users}}
    <div>{{.name}}</div>
{{end}}
```

### With Context

```html
{{with .user}}
    <p>{{.name}} - {{.email}}</p>
{{end}}
```

## Template Inheritance

### Base Template

```html
<!-- layouts/base.tpl -->
<!DOCTYPE html>
<html>
<head>
    <title>{{block "title" .}}Default Title{{end}}</title>
</head>
<body>
    <header>{{block "header" .}}Default Header{{end}}</header>
    <main>
        {{block "content" .}}{{end}}
    </main>
    <footer>{{block "footer" .}}Default Footer{{end}}</footer>
</body>
</html>
```

### Child Template

```html
<!-- user/profile.tpl -->
{{template "layouts/base.tpl" .}}

{{define "title"}}User Profile{{end}}

{{define "header"}}
    <nav>My App Navigation</nav>
{{end}}

{{define "content"}}
    <h1>{{.user.name}}</h1>
    <p>{{.user.email}}</p>
{{end}}
```

## Built-in Functions

### String Functions

```html
{{.name | upper}}      <!-- Uppercase -->
{{.name | lower}}      <!-- Lowercase -->
{{.name | trim}}       <!-- Trim whitespace -->
{{.name | htmlEscape}} <!-- HTML escape -->
```

### Number Functions

```html
{{.price | format "%.2f"}} <!-- Format number -->
{{.count | add 10}}       <!-- Add -->
{{.count | sub 5}}        <!-- Subtract -->
```

### Date Functions

```html
{{.created_at | date "2006-01-02"}}
{{.updated_at | datetime}}
```

### Array Functions

```html
{{len .items}}         <!-- Length -->
{{index .items 0}}      <!-- Get by index -->
```

## Custom Functions

Register custom template functions:

```go
func init() {
    httpsrv.GlobalService.Config.RegisterTemplateFunc("truncate", Truncate)
}

func Truncate(s string, length int) string {
    if len(s) <= length {
        return s
    }
    return s[:length] + "..."
}
```

Use in template:

```html
{{.description | truncate 100}}
```

## Practical Examples

### User List

```go
type User struct {
    *httpsrv.Controller
}

func (c User) ListAction() {
    users := []map[string]interface{}{
        {"id": 1, "name": "Alice", "email": "alice@example.com"},
        {"id": 2, "name": "Bob", "email": "bob@example.com"},
    }
    
    c.Data["users"] = users
    c.Data["title"] = "User List"
    c.Render("user/list.tpl")
}
```

Template:

```html
{{template "layouts/base.tpl" .}}

{{define "title"}}{{.title}}{{end}}

{{define "content"}}
    <table>
        <thead>
            <tr>
                <th>ID</th>
                <th>Name</th>
                <th>Email</th>
            </tr>
        </thead>
        <tbody>
            {{range .users}}
            <tr>
                <td>{{.id}}</td>
                <td>{{.name}}</td>
                <td>{{.email}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
{{end}}
```

### Form with Errors

```go
func (c User) CreateAction() {
    if c.Request.Method == "GET" {
        c.Render("user/form.tpl")
        return
    }
    
    name := c.Params.Value("name")
    if name == "" {
        c.Data["error"] = "Name is required"
        c.Data["name"] = ""
        c.Render("user/form.tpl")
        return
    }
    
    // Create user...
}
```

Template:

```html
<form method="POST" action="/user/create">
    {{if .error}}
        <div class="error">{{.error}}</div>
    {{end}}
    
    <div>
        <label>Name:</label>
        <input type="text" name="name" value="{{.name}}">
    </div>
    
    <button type="submit">Submit</button>
</form>
```

## Auto Render

When `c.AutoRender` is `true` (default), system automatically searches for template:

```go
type User struct {
    *httpsrv.Controller
}

// System will look for: user/index.tpl
func (c User) IndexAction() {
    c.Data["users"] = getUsers()
    // No need to call c.Render()
}

// System will look for: user/profile.tpl
func (c User) ProfileAction() {
    c.Data["user"] = getUser()
    // No need to call c.Render()
}
```

## Template Path Rules

For Controller named `User` with Action `ProfileAction`:

1. If `c.Render("user/profile.tpl")` → Look for `user/profile.tpl` in all template paths
2. If `c.Render("user", "profile.tpl")` → Look for `user/profile.tpl` in all template paths
3. If AutoRender → Look for `user/profile.tpl` in all template paths

Template search order:
- First template path set in Module
- Second template path set in Module
- ...

## Notes

1. Template files use `.tpl` extension by default
2. Template variables are passed through `c.Data` map
3. Use `{{block}}` for template inheritance
4. Use `{{define}}` to define reusable template blocks
5. All built-in Go template functions are available