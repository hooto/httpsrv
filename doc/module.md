## Module Component

Module is used to combine code with similar business functions, including Controller, Router, Template and other related components.

### Typical Use of Module

Assuming an API service, module name is github.com/hooto/httpsrv-demo/websrv/v1, it includes User, Role two Controllers. Generally, we create a setup.go file in the root directory of this module, and import the two controllers involved, such as:

``` go
// vim setup.go
package v1

import (
	"github.com/hooto/httpsrv"
)

func NewModule() *httpsrv.Module {

	mod := httpsrv.NewModule()
    
	mod.RegisterController(new(User), new(Role))

	return mod
}

// User, Role specific code implementation omitted...
```

In main.go entry file, register above module to the service (Service) and access modules' various methods via URL /api/v1 prefix:

``` go
package main

import (
	"github.com/hooto/httpsrv"
	"github.com/hooto/httpsrv-demo/websrv/v1"
)

func main() {

	httpsrv.DefaultService.HandleModule("/api/v1", v1.NewModule())

	httpsrv.DefaultService.Config.HttpPort = 8080

	httpsrv.DefaultService.Start()
}
```

### Set View Template Path

If current Module is a frontend UI type business module that needs view templates, set template path via following interface:

``` go
func NewModule() *httpsrv.Module {
	mod := httpsrv.NewModule("ui")
	mod.SetTemplatePath("/path/of/module/ui/views") // Local file path of templates, can set 1 ~ N paths for a module
	return mod
}
```

### Set Static File Path

If current Module depends on static files, such as js, css, img, etc., set path via following interface

``` go
func NewModule() *httpsrv.Module {
	mod := httpsrv.NewModule()

	mod.RegisterFileServer("/assets", "/path/of/static/files", nil)

	return module
}
```

Note:

Route{Path: "assets"} parameter indicates that frontend access URL path is baseuri + "/assets/静态文件相对路径" set by Service.HandleModule(baseuri,...), such as:

``` go
httpsrv.DefaultService.HandleModule("/cms", ui.NewModule())
```

Static file relative path is js/main.js, then final URL static file access path is:

``` shell
# baseuri + route.path + static.file
/cms/assets/js/main.js