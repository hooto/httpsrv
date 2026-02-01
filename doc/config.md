## Config Component

The configuration module is used to define dependency parameters when HTTP service starts, defined as follows:

``` go
type Config struct {
	HttpAddr         string `json:"http_addr,omitempty"` // e.g. "127.0.0.1", "unix:/tmp/app.sock"
	HttpPort         uint16 `json:"http_port,omitempty"` // e.g. 8080
	HttpTimeout      uint16 `json:"http_timeout,omitempty"`
	UrlBasePath      string `json:"url_base_path,omitempty"`
	CookieKeyLocale  string `json:"cookie_key_locale,omitempty"`
	CookieKeySession string `json:"cookie_key_session,omitempty"`
}
```

Parameter Description

| Field | Type | Default | Required | Description |
|----|----|----|----|----|
| HttpAddr | string | No | Empty | Set to publish services via Unix domain socket |
| HttpPort | int | No | 8080 | Set to publish services via TCP port |
| HttpTimeout | int | No | 30 | Set HTTP connection timeout in seconds |
| UrlBasePath | string | No | / | Set root URL path for HTTP service access, default is / |
| CookieKeyLocale | string | No | lang | When i18n is enabled, httpsrv will set language package parameters in cookie with default field name `lang`. This value can customize cookie field name for saving |
| CookieKeySession | string | No | access_token | When Session is enabled, httpsrv will set user status Session value information in cookie with default field name `access_token`. This value can customize cookie field name for saving |

Config is a built-in item of [Service](service.md) and can be referenced via Service, such as:

``` go
package main

import ( 
	"github.com/hooto/httpsrv"
)

func main() {
	// Reference via global Service instance 
	httpsrv.DefaultService.Config.HttpPort = 8080

	// Reference via new Service instance
	srv := httpsrv.NewService()
	srv.Config.HttpPort = 8081
}
```

## Extended Configuration Items

On the basis of `type Config struct` data definition, some dynamic interfaces are extended to extend configuration items.

### Register New Template Built-in Function (Optional)

``` go
package main

import ( 
	"strings"
	"github.com/hooto/httpsrv"
)

func ExampleUpper(v string) string {
	return strings.ToUpper(v)
}

func main() {
	var conf httpsrv.Config
	// ...
	conf.RegisterTemplateFunc("upper", ExampleUpper)
}
```

> Note: System has built-in commonly used view template (View Template) functions by default. For details, refer to code file [template-func.go](https://github.com/hooto/httpsrv/blob/master/template-func.go)