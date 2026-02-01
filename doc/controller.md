## Controller Component

Controller is the C in Web MVC. It includes the two most important objects for HTTP logic: Request and Response. All newly created controllers need to inherit from httpsrv.Controller, such as:

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

httpsrv.Controller data definition is as follows:

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

Note:

* httpsrv kernel matches Module-Path/Controller/Action based on URL/router rules and runs
* Lifecycle is created and destroyed per HTTP Request/Response unit, completely isolated between different requests and thread-safe. When *Action() method references Controller built-in variables, its corresponding value is only valid in local request.

Description

| Item | Description |
|----|----|
| Name | Current request's controller name |
| ActionName | Current request's action name |
| Request | Request object instance when current request occurs | 
| Response | Response object instance when current request occurs | 
| Params | Shortcut object to get request parameters based on Request encapsulation | 
| Session | Object instance to save Session information to browser or obtain Session information | 
| AutoRender | By default, system will search for View Template template file and output return data to Response object. Setting to false can disable this feature | 
| Data | Structured data to inject into View Template |

## Controller Built-in Methods

### Request Object Instance

httpsrv.Request object is based on go/net/http.Request and provides some extended fields and functions, defined as follows:

``` go
type Request struct {
	*http.Request
	Time           time.Time
	ContentType    string
	Locale         string
}
```

| Item | Description |
|----|----|
| Time | Current request start time |
| ContentType | Current request's http/header `Content-Type` value |
| Locale | When i18n function is enabled, current value is language package name specified by http client |

#### Extended Methods of Request Object Instance

#### RawBody() []byte

Client POST/PUT raw body data

``` go
func (c File) UploadAction() {
    b := c.RawBody()
}
```

#### JsonDecode(obj interface{}) error

Deserialization interface for client POST JsonObject scenarios

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

Note: httpsrv.Controller encapsulates some shortcut interfaces based on Request, such as c.Params, etc. For details, please refer to subsequent instructions.

### Response Object Instance

httpsrv.Response object is based on go/net/http.ResponseWriter and provides some extended fields and functions, defined as follows:

``` go
type Response struct {
	Status      int
	Out         http.ResponseWriter
}
```

| Item | Description |
|----|----|
| Status | HTTP standard status code when returning Response content, such as 200, 404, ... |
| Out | Original IO interface, original data write interface when returning Response content |

Note: httpsrv.Controller encapsulates some shortcut interfaces based on Response, such as c.Render(), c.RenderJson(), etc. For details, please refer to subsequent instructions.

### Params Object

Can conveniently get request parameters in GET URL, or parameters requested via POST, PUT with ContentType == "application/x-www-form-urlencoded", such as:

``` go
func (c User) EntryAction() {
	id := c.Params.Value("id")
	age := c.Params.IntValue("age")
	// ...
}
```

### Render(args ...interface{})

Render() renders template view HTML data and writes to Response object. Pass template view relative path (default root path is set by [Module.RouteSet](module.md)), usage as follows:

* When c.Render("user", "path/of/name.tpl") is passed in business Action method, system will search for "path/of/name.tpl" template in module named "user" and render to Response object.
* When c.Render("path/of/name.tpl") is passed in business Action method, system will search for "path/of/name.tpl" template in current module and render to Response object.
* When c.Render() is not called in business Action method and AutoRender==true, system will search for "Controller/Action.tpl" (note that template name case needs to correspond to Controller/Action name) fixed format template in current module and render to Response object.

### RenderError(status int, msg string)

RenderError() is used to output exception HTTP Status status information to Response, such as:
``` go
func (c User) EntryAction() {
	if c.Params.Get("id") == "" {
		c.RenderError(400, "id not found")
	}
}
```

### RenderJson(...) and RenderJsonIndent(...)

Used to output JSON format text information to Response, mostly used for API/JSON scenarios, such as:

``` go
func (c User) EntryAction() {
	jsonStruct := struct {
		Name string `json:"name"`
	} {
		Name: "robot",
	}
	c.RenderJson(jsonStruct)
	// c.RenderJsonIndent(jsonStruct, "\t")
}
```

Note: When RenderJson*() or RenderError() is called, AutoRender is automatically set to false, and system no longer executes other default Render operations.