## httpsrv

httpsrv is a Lightweight, Modular, High Performance MVC web framework for the Go language.

Documents:
* 中文文档 [https://www.hooto.com/gdoc/view/hooto-httpsrv/](https://www.hooto.com/gdoc/view/hooto-httpsrv/)

## Quick Start

first hello world demo

```go
package main

import (
    "github.com/hooto/httpsrv"
)

type Hello struct {
    *httpsrv.Controller
}

func (c Hello) WorldAction() {
    c.RenderString("hello world")
}

func main() {

	mod := httpsrv.NewModule()

	mod.RegisterController(new(Hello))

	srv := httpsrv.NewService()

	srv.SetLogger(httpsrv.NewRawLogger())

	srv.HandleModule("/demo", mod)

	srv.Start(":3000")
}
```

```shell
go run hello.go
```

```shell
curl http://localhost:3000/demo/hello/world

hello world
```

## Licensing
Licensed under the Apache License, Version 2.0

