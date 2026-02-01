# QA 问答

## 基础问题

#### 如何支持 SSL (https)

**私有服务器**：管理 SSL 证书可以通过前端 Nginx 等负载均衡实现，这种方式性能上更高效。

**云服务商**：多数云服务商提供的负载均衡服务内置了 SSL 证书管理。

示例 Nginx 配置：

```nginx
server {
    listen 443 ssl;
    server_name your-domain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

#### 如何支持数据压缩、HTTP/2.0

通过前端 nginx 等负载均衡实现，这种方式比 httpsrv 内置实现肯定更高效，同时也符合运维管理的便利性；更重要的是保持 httpsrv 定位的纯粹精简。

Nginx 开启 gzip 压缩示例：

```nginx
http {
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript
               application/x-javascript application/xml+rss
               application/json application/javascript;
}
```

#### 如何获取客户端真实 IP

当使用 Nginx 等反向代理时，需要从 HTTP 头中获取真实 IP：

```go
func (c BaseController) GetClientIP() string {
	// 检查 X-Real-IP 头
	if ip := c.Request.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	// 检查 X-Forwarded-For 头
	if ip := c.Request.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	// 默认使用 RemoteAddr
	return c.Request.RemoteAddr
}
```

## 开发问题

#### 如何处理跨域请求 (CORS)

httpsrv 没有内置 CORS 支持，可以通过 Filter 实现：

```go
package main

import (
	"github.com/hooto/httpsrv"
)

func CorsFilter(c *httpsrv.Controller, fc []httpsrv.Filter) {
	// 设置 CORS 头
	c.Response.Out.Header().Set("Access-Control-Allow-Origin", "*")
	c.Response.Out.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	c.Response.Out.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	c.Response.Out.Header().Set("Access-Control-Max-Age", "86400")
	
	// 处理 OPTIONS 预检请求
	if c.Request.Method == "OPTIONS" {
		c.Response.Out.WriteHeader(200)
		return
	}
	
	// 继续执行后续 Filter
	fc[0](c, fc[1:])
}

func main() {
	// 注册 CORS Filter
	httpsrv.GlobalService.Filters = []httpsrv.Filter{
		CorsFilter,
	}
	
	// 注册模块
	httpsrv.GlobalService.HandleModule("/", NewModule())
	httpsrv.GlobalService.Start()
}
```

#### 如何实现文件上传

httpsrv 支持标准的多部分表单数据上传：

```go
type FileUpload struct {
	*httpsrv.Controller
}

func (c FileUpload) UploadAction() {
	// 解析表单
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.RenderError(400, "解析表单失败")
		return
	}
	
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.RenderError(400, "获取文件失败")
		return
	}
	defer file.Close()
	
	// 保存文件
	// 这里使用示例逻辑，实际项目中应该保存到指定目录
	// 文件信息可以通过 header 获取
	filename := header.Filename
	filesize := header.Size
	contentType := header.Header.Get("Content-Type")
	
	c.RenderJson(map[string]interface{}{
		"status":  "success",
		"filename": filename,
		"size":     filesize,
		"type":     contentType,
	})
}
```

前端 HTML 表单：

```html
<form action="/file-upload/upload" method="POST" enctype="multipart/form-data">
    <input type="file" name="file" />
    <button type="submit">上传</button>
</form>
```

#### 如何处理 Session

httpsrv 内置了 Session 支持：

```go
type Auth struct {
	*httpsrv.Controller
}

func (c Auth) LoginAction() {
	username := c.Params.Value("username")
	password := c.Params.Value("password")
	
	// 验证用户名密码
	if username == "admin" && password == "password" {
		// 设置 Session
		c.Session.Set("user_id", "12345")
		c.Session.Set("username", username)
		
		c.RenderJson(map[string]string{
			"status": "success",
			"message": "登录成功",
		})
	} else {
		c.RenderError(401, "用户名或密码错误")
	}
}

func (c Auth) ProfileAction() {
	// 获取 Session
	userId := c.Session.Get("user_id")
	username := c.Session.Get("username")
	
	if userId == "" {
		c.RenderError(401, "未登录")
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"user_id":  userId,
		"username": username,
	})
}

func (c Auth) LogoutAction() {
	// 清除 Session
	c.Session.Clear()
	
	c.RenderJson(map[string]string{
		"status": "success",
		"message": "退出成功",
	})
}
```

#### 如何实现 i18n 国际化

httpsrv 支持多语言，可以通过设置 Locale 实现：

```go
type I18nDemo struct {
	*httpsrv.Controller
}

func (c I18nDemo) IndexAction() {
	// 从请求中获取语言设置
	// 可以从 URL 参数、Cookie 或 HTTP 头中获取
	lang := c.Params.Value("lang")
	if lang == "" {
		lang = c.Request.Header.Get("Accept-Language")
	}
	
	messages := map[string]map[string]string{
		"zh": {
			"title": "欢迎",
			"hello": "你好，世界",
		},
		"en": {
			"title": "Welcome",
			"hello": "Hello, World",
		},
	}
	
	if msg, ok := messages[lang]; ok {
		c.RenderJson(msg)
	} else {
		c.RenderJson(messages["zh"])
	}
}
```

#### 如何实现 API 版本控制

推荐通过 Module 路径实现版本控制：

```go
func main() {
	// v1 API
	httpsrv.GlobalService.HandleModule("/api/v1", NewApiV1Module())
	
	// v2 API
	httpsrv.GlobalService.HandleModule("/api/v2", NewApiV2Module())
	
	httpsrv.GlobalService.Start()
}

// v1 模块
func NewApiV1Module() *httpsrv.Module {
	mod := httpsrv.NewModule()
	mod.RegisterController(new(ApiV1Controller))
	return mod
}

// v2 模块
func NewApiV2Module() *httpsrv.Module {
	mod := httpsrv.NewModule()
	mod.RegisterController(new(ApiV2Controller))
	return mod
}
```

访问路径：
- v1 API: `/api/v1/controller/action`
- v2 API: `/api/v2/controller/action`

## 部署问题

#### 如何配置优雅关闭

使用信号处理实现优雅关闭：

```go
package main

import (
	"os"
	"os/signal"
	"syscall"
	"github.com/hooto/httpsrv"
)

func main() {
	// 启动服务
	go func() {
		httpsrv.GlobalService.Start()
	}()
	
	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	// 执行清理操作
	// 关闭数据库连接
	// 保存缓存等
	
	// httpsrv 会自动处理关闭
}
```

#### 如何配置生产环境配置文件

httpsrv 支持多种配置文件格式，推荐使用 JSON 或 YAML：

```go
package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"github.com/hooto/httpsrv"
)

type Config struct {
	HttpPort       int    `json:"http_port"`
	DatabaseURL    string `json:"database_url"`
	RedisURL       string `json:"redis_url"`
	LogLevel       string `json:"log_level"`
}

func loadConfig() (*Config, error) {
	configFile := "config.json"
	if os.Getenv("APP_ENV") == "production" {
		configFile = "config.prod.json"
	}
	
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	
	return &config, nil
}

func main() {
	config, err := loadConfig()
	if err != nil {
		panic(err)
	}
	
	httpsrv.GlobalService.Config.HttpPort = uint16(config.HttpPort)
	
	// 使用其他配置项连接数据库、Redis 等
	
	httpsrv.GlobalService.Start()
}
```

#### 如何实现健康检查端点

创建专门的健康检查模块：

```go
type HealthCheck struct {
	*httpsrv.Controller
}

func (c HealthCheck) IndexAction() {
	// 检查数据库连接
	dbOK := checkDatabase()
	
	// 检查 Redis 连接
	redisOK := checkRedis()
	
	status := "ok"
	if !dbOK || !redisOK {
		status = "error"
		c.Response.Out.WriteHeader(503)
	}
	
	c.RenderJson(map[string]interface{}{
		"status":   status,
		"database": dbOK,
		"redis":    redisOK,
	})
}

func NewHealthCheckModule() *httpsrv.Module {
	mod := httpsrv.NewModule()
	mod.RegisterController(new(HealthCheck))
	return mod
}

func main() {
	// 注册健康检查模块
	httpsrv.GlobalService.HandleModule("/health", NewHealthCheckModule())
	
	// 注册其他模块
	httpsrv.GlobalService.HandleModule("/", NewMainModule())
	
	httpsrv.GlobalService.Start()
}
```

## 性能问题

#### 如何优化静态文件服务

建议使用专门的静态文件服务器：

```go
func main() {
	// 使用 CDN 或专门的静态文件服务
	// httpsrv 只负责动态内容
	
	httpsrv.GlobalService.HandleModule("/api", NewApiModule())
	httpsrv.GlobalService.Start()
}
```

或者使用 Nginx 提供静态文件：

```nginx
server {
    location /static/ {
        root /path/to/static/files;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
    
    location / {
        proxy_pass http://localhost:8080;
    }
}
```

#### 如何实现缓存

httpsrv 没有内置缓存，推荐使用 Redis 或内存缓存：

```go
import "github.com/lynkdb/redisgo"

var redisClient *redisgo.RedisClient

func main() {
	// 初始化 Redis 客户端
	redisClient = redisgo.NewClient(&redisgo.Config{
		Addr: "localhost:6379",
	})
	
	httpsrv.GlobalService.Start()
}

type CacheDemo struct {
	*httpsrv.Controller
}

func (c CacheDemo) GetDataAction() {
	key := "data_key"
	
	// 尝试从缓存获取
	data, err := redisClient.Get(key).Result()
	if err == nil && data != "" {
		c.Response.Out.Header().Set("X-Cache", "HIT")
		c.RenderString(data)
		return
	}
	
	// 缓存未命中，从数据库获取
	dbData := fetchDataFromDB()
	
	// 设置缓存
	redisClient.Set(key, dbData, 5*time.Minute)
	
	c.Response.Out.Header().Set("X-Cache", "MISS")
	c.RenderString(dbData)
}
```

## 常见错误

#### Action 方法没有被路由识别

确保 Action 方法以 `Action()` 结尾：

```go
// ❌ 错误
func (c User) Login() {
	// ...
}

// ✅ 正确
func (c User) LoginAction() {
	// ...
}
```

#### 静态文件 404 错误

检查静态文件路径配置是否正确：

```go
func NewModule() *httpsrv.Module {
	mod := httpsrv.NewModule()
	
	// 确保本地文件路径存在
	mod.RegisterFileServer("/assets", "./static", nil)
	
	return mod
}

// 访问 URL: /assets/css/style.css
// 对应本地文件: ./static/css/style.css
```

#### 模版文件找不到

检查模版路径设置是否正确：

```go
func NewModule() *httpsrv.Module {
	mod := httpsrv.NewModule("ui")
	
	// 确保模版路径设置正确
	mod.SetTemplatePath("./views/ui")
	
	return mod
}

// 如果 Controller 是 User，Action 是 Index
// 系统会查找: ./views/ui/user/index.tpl
```

## 应用案例

#### 是否有应用案例

httpsrv 作为个人项目最早从 2013 年开始应用于某些行业软件中：

* [SysInner.com](https://www.sysinner.com/) 使用 httpsrv 实现所有 API 和 Web 模块
* 在某千亿级广告系统中提供 API 实现，稳定高效处理数据收集和索引查询
* 在某PB级对象存储服务中提供 API 实现，大流量IO读写场景下的稳定性和内存使用均稳定可靠
* 在多个卫视晚会线上互动活动中提供包括用户系统，消息系统，互动系统等 API 实现，在亿级PV，数百万UV 场景中高效运行无异常，综合响应时间 50 毫秒以内（环境备注：60台接口服务器2核云主机，6台数据库4核高速SSD云主机，集群负载峰值小于 5%，流量通过云服务商负载均衡出口实现）

## 其他问题

#### 没有解决问题？

如果上述信息没有帮助，可将问题发邮件到 evorui at gmail dot com。

#### 如何贡献代码

欢迎提交 Pull Request 或报告 Bug，请访问：https://github.com/hooto/httpsrv

#### 如何获取最新更新

```bash
go get -u github.com/hooto/httpsrv