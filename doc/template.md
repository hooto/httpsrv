## Template 组件

Template 是 Web MVC 中的 V (View), 它基于 go/html/template 并扩展了部分功能.

httpsrv.Template 多数场景下在 httpsrv.Controller 中通过 Render* 类接口调用, 具体请参考 [Controller 组件](controller.md).

注:

* View Template 内置的逻辑语法定义完全沿用了 go/html/template, 所以相关用法请参考 [go/html/template](https://golang.org/pkg/html/template/)
* View Template 在 httpsrv 具有缓存机制，能提高性能效率； 但如果修改了模版，则需重启 httpsrv 服务后才能生效.

