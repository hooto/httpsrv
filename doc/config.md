## Config 组件

配置模块用于定义 HTTP 服务启动时的依赖参数，定义如下:

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


参数说明

| 字段 | 类型 | 默认值 | 必需 | 说明 |
|----|----|----|----|----|
| HttpAddr | string | 否 | 空 | 设置以 Unix domain socket 方式发布服务 |
| HttpPort | int | 否 | 8080 | 设置以 TCP 端口发布服务 |
| HttpTimeout | int | 否 |  30 | 设置 http 连接超时时间, 单位秒 |
| UrlBasePath | string | 否 | / | 设置 http 服务访问的URL根路径，默认为 / |
| CookieKeyLocale | string | 否 | lang | 当启用 i18n 时，httpsrv 会在cookie中以默认字段名 `lang` 设置语言包参数，这个值可自定义 cookie 保存的字段名 |
| CookieKeySession | string | 否 | access_token | 当启用 Session 时，httpsrv 会在cookie中以默认字段名 `access_token` 设置用户状态的 Session 值信息，这个值可自定义 cookie 保存的字段名 |

Config 是 [Service](service.md) 的一个内置项，通过 Service 引用, 如:

``` go
package main

import ( 
	"github.com/hooto/httpsrv"
)

func main() {
	// 通过全局 Service 实例引用 
	httpsrv.GlobalService.Config.HttpPort = 8080

	// 新建 Service 实例引用
	server := httpsrv.NewService()
	server.Config.HttpPort = 8081
}
```

## 扩展配置项

在 `type Config struct` 这个数据定义基础之上，扩展了部分动态接口用于扩展配置项

### 注册新的模版内置调用函数 (可选)

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
	conf.TemplateFuncRegister("upper", ExampleUpper)
}
```

> 注: 系统默认内置了常用视图模版(View Template)函数，具体可参考代码文件 [template-func.go](https://github.com/hooto/httpsrv/blob/master/template-func.go)

