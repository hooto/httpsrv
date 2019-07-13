## Router 组件

Router 用于解析 Http Request URL 规则，并选择匹配的 Module:Controller/Action 并执行。

与大多数 Web 框架不同, httpsrv 弱化了 router 功能，只包含两种最简单的路由类型: 固定路由 和 静态路由, 

* 固定路由(或称标准路由): 除了注册 Module 到 Service 时指定的 baseuri 参数可变以外，其后面的路径都是两级固定的 (controller/action 名) 
* 静态路由: 用于单独设置静态文件的访问规则 (参考 [Module 组件](module.md))

固定路由是系统默认，不需要单独设置, 其在 URL 对应的名称严格与 Controller/Action 的名称对应: 

* 控制器里面的 Action 方法必需以 *Action() 结尾才能被路由匹配并执行
* 访问服务时, URL 对应的名称统一转换为小写
* 名称包含驼峰写法, 用 "-" 符号连接，如 NameUpper 对应 URL 名称是 name-upper
* "Index" 是默认名，当 Action 名为 IndexAction() 时，在 URL 中可省略 Action 名; 当 Controller 和 Action 同时为 "type Index struct" 和 "IndexAction()" 时，可同时省略 URL 两级的名称.

