## Service Component

The Service component is the outermost object instance that provides HTTP services in httpsrv. Other components are registered directly or indirectly to the Service, and finally HTTP services are provided to the outside through the Service.Start() method.

> Hint: From outside to inside, the main component logical layers of httpsrv are: HTTP Request -> Service -> Module -> Controller -> Action

``` go
type Service struct {
	Config         Config
	Filters        []Filter
	TemplateLoader *TemplateLoader
}
```

Description

| Item | Description |
|----|----|
| Config | Basic configuration component that defines dependency parameters when HTTP service starts. See [Config Details](config.md) |
| Filter | Filter sequence configuration for the entire execution lifecycle of HTTP Request/Response. httpsrv executes core logic such as Router, Params, Action in this order. This is an abstract interface definition that can be customized, but in most cases does not need to be configured. The system default settings already meet most usage scenarios. For default configuration, refer to [file filter.go](https://github.com/hooto/httpsrv/blob/master/filter.go) |
| TemplateLoader | View loading and management component. When developing V (View) in Web MVC, this component will be automatically activated. For details, refer to [Template Details](template.md) |

## Quick Use of Service

httpsrv creates a global Service instance `httpsrv.GlobalService` by default. In most scenarios, you can use it directly. The simplest example only needs to register a module/controller and set the TCP port to start the service, such as:

``` go
package main

import (
	"github.com/hooto/httpsrv"
)

type Account struct {
	*httpsrv.Controller
}

func (c Account) LoginAction() {
	c.RenderString("hello world")
}

// Build a module example
func NewUserModule() *httpsrv.Module {

	// Initialize an empty module
	mod := httpsrv.NewModule()
    
	// Register a controller to module
	mod.RegisterController(new(Account))

	return mod
}

func main() {

	// Register this module to service (mounted to URL path /user to provide services externally, configurable)
	httpsrv.GlobalService.HandleModule("/user", NewUserModule())

	// Set service port
	httpsrv.GlobalService.Config.HttpPort = 8080

	// Start service
	httpsrv.GlobalService.Start()
}
```

Compile and start service

``` shell
go build -o demo-server main.go
./demo-server
```

According to the global naming convention of `/{module-path}/{controller}/{action}`, the above service can be accessed via http://localhost:8080/user/account/login.

## Multiple Services Coexist

In some scenarios, multiple sets of service instances need to be provided on different ports externally. For example: 80 port for frontend business (enterprise firewall only opens 80 port), 8080 port for API business and only opened internally. This can be achieved as follows:

``` go
package main

import (
	"github.com/hooto/httpsrv"
)

type ApiDemo struct {
	*httpsrv.Controller
}

func (c ApiDemo) ExampleAction() {
	jsonStruct := struct {
		Name string `json:"name"`
	} {
		Name: "robot",
	}
	c.RenderJson(jsonStruct)
}

// Build API module
func NewApiModule() *httpsrv.Module {
	mod := httpsrv.NewModule()
	mod.RegisterController(new(ApiDemo))
	return mod
}

type Index struct {
	*httpsrv.Controller
}

func (c Index) IndexAction() {
	c.RenderString("hello world")
}

// Build frontend module
func NewFrontendModule() *httpsrv.Module {
	mod := httpsrv.NewModule()
	mod.RegisterController(new(Index))
	return mod
}

func main() {

	serviceFrontend := httpsrv.NewService()
	serviceFrontend.Config.HttpPort = 80
	serviceFrontend.HandleModule("/", NewFrontendModule())

	serviceApi := httpsrv.NewService()
	serviceApi.Config.HttpPort = 8080
	serviceApi.HandleModule("/api/v1", NewApiModule())


	// Start frontend service
	go serviceFrontend.Start()

	// Start backend service
	serviceApi.Start()
}
```

## Service Main Interface Methods

On the basis of `type Service struct` data definition, some dynamic interface methods are extended to customize configuration items.

### Core Method ModuleRegister

``` go
// Interface definition
func (s *Service) HandlerRegister(baseuri string, h http.Handler)
```

This is a required method. All modules need to be registered to the Service to provide services externally. For specific examples, please refer to the above code.

It needs special explanation that the baseuri string parameter. In most enterprise applications, due to engineering and business requirements, there will be multiple modules coexisting scenarios (i.e., multiple business systems under the same domain). When different modules are registered to the Service, this module's mount directory in the URL needs to be specified. The baseuri path name generally corresponds to the business system function, such as /user, /cms, /mail, etc.

### HandlerRegister, HandlerFuncRegister

``` go
// Interface definition http.Handler
func (s *Service) HandlerRegister(baseuri string, h http.Handler)

// Interface definition http.HandlerFunc
func (s *Service) HandlerFuncRegister(baseuri string, h http.HandlerFunc)
```

These two interfaces are used to register native go/net/http handler functions to the Service, mainly used for:

* RPC type handler functions
* WebSocket type handler functions