## Controller 组件

Controller 是 Web MVC 中的 C, 它包含 http 逻辑最主要的两个对象 Request 和 Response, 所有新建控制器都需继承 httpsrv.Controller, 如:

``` go
type User struct {
	*httpsrv.Controller
}

func (c User) LoginAction() {
	// ...
}

func (c User) RegisterAction() {
	// ...
}
```

httpsrv.Controller 数据定义如下:

``` go
type Controller struct {
	Name          string // The controller name, e.g. "App"
	ActionName    string // The action name, e.g. "Index"
	Request       *Request
	Response      *Response
	Params        *Params  // Parameters from URL and form (including multipart).
	Session       *Session // Session, stored in cookie, signed.
	AutoRender    bool
	Data          map[string]interface{}
}
```

注:

* httpsrv 内核根据 URL/router 规则匹配 Module/Controller/Action 并运行时
* 生命周期是以 HTTP Request/Response 为单位创建和销毁，各个请求之间完全隔离并线程安全, *Action() 方法引用 Controller 内置变量时其对应值只在本地请求中有效. 

说明

| 项目 | 说明 |
|----|----|
| Name | 当前请求的控制器名 |
| ActionName | 当前请求的Action名 |
| Request | 当前请求时的 Request 对象实例 | 
| Response | 当前请求时的 Response 对象实例 | 
| Params | 基于 Request 封装的，获取请求参数的快捷对象 | 
| Session | 用于保存 Session 信息到浏览器，或者取得 Session 信息的对象实例 | 
| AutoRender | 系统默认会查找 View Template 模版文件并向 Response 对象输出返回数据，设置为 false 可以关闭此功能 | 
| Data | 用于向 View Template 注入模版内需要的结构化数据 |

## Controller 内置方法


### Request 对象实例

httpsrv.Request 对象基于 go/net/http.Request, 并提供了部分扩展字段和功能,定义如下:

``` go
type Request struct {
	*http.Request
	ContentType    string
	AcceptLanguage []AcceptLanguage
	Locale         string
	RequestPath    string
	UrlPathExtra   string
	RawBody        []byte
	WebSocket      *WebSocket
}
```

| 项目 | 说明 |
|----|----|
| ContentType | 当前请求的http/header `Content-Type` 值 |
| AcceptLanguage | 当前请求的 http/header `Accept-Language` 值 |
| Locale | 当启用 i18n 功能是, 当前值为 http 客户端指定的语言包名 |
| RequestPath | 当前请求时的 URL Path 值 | 
| UrlPathExtra | 当前请求时的 URL Path 截断前缀 `/basepath/{controller}/{action}` 后的值 | 
| RawBody | 当前请求为 POST, PUT 时原始的数据 | 
| WebSocket | 当前请求为 WebSocket 时所建立的连接对象实例 | 


#### Request 对象实例所扩展的方法

#### JsonDecode(obj interface{}) error

客户端 POST JsonObject 场景中反序列化接口

``` go
func (c User) LoginAction() {
	var jsonObject struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
    err := c.Request.JsonDecode(&jsonObject)
	// ...
}
```

注: httpsrv.Controller 基于 Request 封装了部分快捷接口，如 c.Params 等，详细请参考后续说明.

### Response 对象实例

httpsrv.Response 对象基于 go/net/http.ResponseWriter, 并提供了部分扩展字段和功能,定义如下:

``` go
type Response struct {
	Status      int
	ContentType string
	Out         http.ResponseWriter
}
```

| 项目 | 说明 |
|----|----|
| Status | 返回 Response 内容时的 HTTP 的标准状态码, 如 200, 404, .. |
| ContentType | 返回 Response 内容时的内容类型 |
| Out | 原始 IO 接口，返回 Response 内容时的原始数据写接口 |

注: httpsrv.Controller 基于 Response 封装了部分快捷接口，如 c.Render(), c.RenderJson() 等，详细请参考后续说明.

### Params 对象

可便捷获取 GET URL 中的请求参数，或者 POST, PUT 通过 ContentType == "application/x-www-form-urlencoded" 方式请求的参数，如:

``` go
func (c User) EntryAction() {
	id_string := c.Params.Get("id")
	id_int64 := c.Params.Int64("age")
	// ...
}
```

### Render(args ...interface{})

Render() 渲染模版视图的HTML数据并写入Response对象，传入模版视图相对路径(默认根路径由 [Module.RouteSet](module.md) 设置), 用法如下: 

* 当业务Action方法中传入 c.Render("user", "path/of/name.tpl") 时, 系统会在 模块名为 "user" 的模版路径下寻找 "path/of/name.tpl" 的模版并渲染到 Response 对象.
* 当业务Action方法中传入 c.Render("path/of/name.tpl") 时, 系统在当前模块下寻找 "path/of/name.tpl" 的模版并渲染到 Response 对象.
* 当业务Action方法中没有调用 c.Render() 同时 AutoRender==true 时，系统在当前模块下寻找 "Controller/Action.tpl" (注意模版名字大小需要和Controller/Action 名对应) 固定格式的模版并渲染到 Response 对象.

### RenderError(status int, msg string)

RenderError() 用于向 Response 输出异常 HTTP Status 状态的信息，比如:
``` go
func (c User) EntryAction() {
	if c.Params.Get("id") == "" {
		c.RenderError(400, "id not found")
	}
}
```

### RenderJson(...) 和 RenderJsonIndent(...)

用于向 Response 输出 JSON 格式文本信息, 多用于API/JSON场景，如:

``` go
func (c User) EntryAction() {
	jsonStruct := struct {
		Name string `json:"name"`
	} {
		Name: "robot"
	}
	c.RenderJson(jsonStruct)
	// c.RenderJsonIndent(jsonStruct, "\t")
}
```

注: 当 RenderJson*() 或者 RenderError() 被调用时, 自动设置 AutoRender=false, 系统不再执行其它默认的 Reander 操作.

