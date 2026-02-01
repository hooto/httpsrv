## Template 组件

Template 是 Web MVC 中的 V (View)，它基于 go/html/template 并扩展了部分功能。

httpsrv.Template 多数场景下在 httpsrv.Controller 中通过 Render* 类接口调用，具体请参考 [Controller 组件](controller.md)。

注:

* View Template 内置的逻辑语法定义完全沿用了 go/html/template，所以相关用法请参考 [go/html/template](https://golang.org/pkg/html/template/)
* View Template 在 httpsrv 具有缓存机制，能提高性能效率；但如果修改了模版，则需重启 httpsrv 服务后才能生效。

## 模版路径设置

在使用模版之前，需要在 Module 中设置模版路径：

``` go
func NewModule() *httpsrv.Module {
	mod := httpsrv.NewModule("ui")
	// 设置模版文件路径，可以为一个模块设置 1 ~ N 个路径
	mod.SetTemplatePath("/path/of/module/ui/views")
	return mod
}
```

## 基本使用

### 自动渲染模版

当 Controller 的 Action 方法中没有调用 `c.Render()` 且 `AutoRender==true` 时，系统会自动在当前模块下寻找 `Controller/Action.tpl` 格式的模版并渲染到 Response 对象。

``` go
type User struct {
	*httpsrv.Controller
}

func (c User) IndexAction() {
	c.Data["title"] = "用户列表"
	c.Data["users"] = []string{"user1", "user2"}
	// 不调用 Render()，系统自动渲染 user/index.tpl
}
```

### 手动指定模版

``` go
func (c User) ProfileAction() {
	c.Data["title"] = "用户资料"
	// 渲染指定路径的模版
	c.Render("profile.tpl")
}
```

### 跨模块渲染

``` go
func (c User) DashboardAction() {
	// 渲染另一个模块 "admin" 下的 dashboard.tpl
	c.Render("admin", "dashboard.tpl")
}
```

## 模版数据注入

通过 `c.Data` 向模版注入数据：

``` go
func (c User) ShowAction() {
	id := c.Params.Value("id")
	
	user := map[string]interface{}{
		"Id":   id,
		"Name": "张三",
		"Age":  30,
	}
	
	c.Data["user"] = user
	c.Data["title"] = "用户详情"
	c.Data["isAdmin"] = true
	
	c.Render("show.tpl")
}
```

## 模版语法示例

### 变量输出

``` html
<!-- 简单变量 -->
<h1>{{ .title }}</h1>

<!-- 结构体字段 -->
<p>用户名: {{ .user.Name }}</p>
<p>年龄: {{ .user.Age }}</p>

<!-- 使用 with 缩短作用域 -->
{{ with .user }}
    <p>用户名: {{ .Name }}</p>
    <p>年龄: {{ .Age }}</p>
{{ end }}
```

### 条件判断

``` html
{{ if .isAdmin }}
    <button>管理员操作</button>
{{ else }}
    <button>普通操作</button>
{{ end }}

{{ if and .user .user.Id }}
    <p>用户ID: {{ .user.Id }}</p>
{{ end }}
```

### 循环

``` html
<ul>
{{ range .users }}
    <li>{{ . }}</li>
{{ end }}
</ul>

<!-- 带索引的循环 -->
{{ range $index, $user := .users }}
    <li>{{ $index }}: {{ $user }}</li>
{{ end }}

<!-- 循环结构体切片 -->
{{ range .users }}
    <div>
        <p>姓名: {{ .Name }}</p>
        <p>年龄: {{ .Age }}</p>
    </div>
{{ end }}
```

### 模版定义和引用

``` html
<!-- 定义模版块 -->
{{ define "header" }}
    <header>
        <h1>{{ .title }}</h1>
    </header>
{{ end }}

<!-- 引用模版块 -->
{{ template "header" . }}

<!-- 定义多个模版 -->
{{ define "content" }}
    <main>
        <p>主要内容</p>
    </main>
{{ end }}

{{ define "footer" }}
    <footer>
        <p>&copy; 2024</p>
    </footer>
{{ end }}

<!-- 组合使用 -->
{{ template "header" . }}
{{ template "content" . }}
{{ template "footer" . }}
```

## 内置函数

httpsrv 扩展了部分常用模版函数，可以在模版中直接使用。默认内置的函数参考 [template-func.go](https://github.com/hooto/httpsrv/blob/master/template-func.go)。

### 常用内置函数

``` html
<!-- 字符串处理 -->
{{ .title | upper }}      <!-- 转大写 -->
{{ .title | lower }}      <!-- 转小写 -->
{{ .title | title }}      <!-- 标题化 -->
{{ .text | trim }}        <!-- 去除首尾空格 -->

<!-- URL 编码 -->
{{ .url | urlquery }}

<!-- HTML 转义 -->
{{ .html | html }}
{{ .text | js }}

<!-- 安全输出（不转义）-->
{{ .html | safeHtml }}

<!-- 格式化 -->
{{ .time | date "2006-01-02" }}
{{ .number | printf "%.2f" }}

<!-- 长度 -->
{{ len .list }}
{{ len .string }}
```

## 自定义模版函数

可以通过 Config 注册自定义的模版函数：

``` go
package main

import (
	"strings"
	"github.com/hooto/httpsrv"
)

// 自定义函数：反转字符串
func ReverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// 自定义函数：截断字符串
func TruncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

func main() {
	// 注册自定义函数
	httpsrv.GlobalService.Config.RegisterTemplateFunc("reverse", ReverseString)
	httpsrv.GlobalService.Config.RegisterTemplateFunc("truncate", TruncateString)
	
	// ... 其他配置
	httpsrv.GlobalService.Start()
}
```

在模版中使用自定义函数：

``` html
<p>原字符串: {{ .text }}</p>
<p>反转后: {{ .text | reverse }}</p>
<p>截断后: {{ .text | truncate 10 }}</p>
```

## 实际应用示例

### 用户列表页面

Controller:
``` go
func (c User) ListAction() {
	users := []map[string]interface{}{
		{"Id": 1, "Name": "张三", "Email": "zhangsan@example.com", "Role": "admin"},
		{"Id": 2, "Name": "李四", "Email": "lisi@example.com", "Role": "user"},
		{"Id": 3, "Name": "王五", "Email": "wangwu@example.com", "Role": "user"},
	}
	
	c.Data["title"] = "用户列表"
	c.Data["users"] = users
	c.Data["currentUser"] = map[string]interface{}{
		"Id": 1,
		"Role": "admin",
	}
	
	c.Render("list.tpl")
}
```

Template (list.tpl):
``` html
<!DOCTYPE html>
<html>
<head>
    <title>{{ .title }}</title>
</head>
<body>
    <h1>{{ .title }}</h1>
    
    {{ if eq .currentUser.Role "admin" }}
    <button>添加用户</button>
    {{ end }}
    
    <table>
        <thead>
            <tr>
                <th>ID</th>
                <th>姓名</th>
                <th>邮箱</th>
                <th>角色</th>
                {{ if eq .currentUser.Role "admin" }}
                <th>操作</th>
                {{ end }}
            </tr>
        </thead>
        <tbody>
            {{ range .users }}
            <tr>
                <td>{{ .Id }}</td>
                <td>{{ .Name }}</td>
                <td>{{ .Email }}</td>
                <td>{{ .Role }}</td>
                {{ if eq $.currentUser.Role "admin" }}
                <td>
                    <a href="/user/edit/{{ .Id }}">编辑</a>
                    <a href="/user/delete/{{ .Id }}">删除</a>
                </td>
                {{ end }}
            </tr>
            {{ end }}
        </tbody>
    </table>
</body>
</html>
```

### 模版继承示例

base.tpl:
``` html
<!DOCTYPE html>
<html>
<head>
    <title>{{ block "title" . }}默认标题{{ end }}</title>
</head>
<body>
    <header>
        {{ block "header" . }}
        <h1>网站标题</h1>
        {{ end }}
    </header>
    
    <main>
        {{ block "content" . }}{{ end }}
    </main>
    
    <footer>
        {{ block "footer" . }}
        <p>&copy; 2024 My Website</p>
        {{ end }}
    </footer>
</body>
</html>
```

index.tpl:
``` html
{{ template "base.tpl" . }}

{{ define "title" }}首页{{ end }}

{{ define "content" }}
<div class="hero">
    <h2>欢迎访问</h2>
    <p>这是一个示例页面</p>
</div>

{{ if .features }}
<div class="features">
    <h3>特性</h3>
    <ul>
    {{ range .features }}
        <li>{{ .Name }}: {{ .Description }}</li>
    {{ end }}
    </ul>
</div>
{{ end }}
{{ end }}
```

## 注意事项

1. **模版缓存**：修改模版后需要重启服务才能生效
2. **自动转义**：go/html/template 默认会对 HTML 进行转义，如果需要输出原始 HTML，使用 `| safeHtml` 或自定义函数
3. **性能考虑**：模版编译需要时间，建议在应用启动时预加载所有模版
4. **错误处理**：模版语法错误会在运行时报错，开发时注意检查模版文件
5. **路径规范**：模版路径建议使用相对路径，便于项目迁移