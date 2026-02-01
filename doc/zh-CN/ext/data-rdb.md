## Model, ORM, 关系数据库

httpsrv 没有内置 Model 和 ORM 等关系数据库接口，如果有这方面需求，可使用如下依赖库：

* [https://github.com/lynkdb/mysqlgo](https://github.com/lynkdb/mysqlgo) - Go client for MySQL
* [https://github.com/lynkdb/pgsqlgo](https://github.com/lynkdb/pgsqlgo) - Go client for PostgreSQL

## MySQL 使用示例

### 安装 mysqlgo

```bash
go get -u github.com/lynkdb/mysqlgo
```

### 初始化数据库连接

```go
package data

import (
	"github.com/lynkdb/mysqlgo"
)

var DB *mysqlgo.MySQLClient

func InitDB() error {
	DB = mysqlgo.NewClient(&mysqlgo.Config{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "password",
		Database: "myapp",
		Charset:  "utf8mb4",
		MaxOpenConns: 100,
		MaxIdleConns: 10,
	})
	
	// 测试连接
	if err := DB.Ping(); err != nil {
		return err
	}
	
	return nil
}
```

### 在 Controller 中使用

```go
package controller

import (
	"github.com/hooto/httpsrv"
	"yourapp/data"
)

type User struct {
	*httpsrv.Controller
}

// 获取用户列表
func (c User) ListAction() {
	// 查询数据库
	var users []map[string]interface{}
	
	err := data.DB.Table("users").
		Select("id, username, email, created_at").
		Where("status = ?", 1).
		OrderBy("created_at DESC").
		Limit(20).
		Find(&users)
	
	if err != nil {
		c.RenderError(500, "查询失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"data":   users,
	})
}

// 获取单个用户
func (c User) GetAction() {
	id := c.Params.IntValue("id")
	
	var user map[string]interface{}
	
	err := data.DB.Table("users").
		Where("id = ?", id).
		First(&user)
	
	if err != nil {
		c.RenderError(404, "用户不存在")
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"data":   user,
	})
}

// 创建用户
func (c User) CreateAction() {
	type CreateUserRequest struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	
	var req CreateUserRequest
	if err := c.Request.JsonDecode(&req); err != nil {
		c.RenderError(400, "参数错误")
		return
	}
	
	// 插入数据
	result, err := data.DB.Table("users").Insert(map[string]interface{}{
		"username":  req.Username,
		"email":     req.Email,
		"password":  hashPassword(req.Password),
		"status":    1,
		"created_at": time.Now(),
	})
	
	if err != nil {
		c.RenderError(500, "创建失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status":  "success",
		"user_id": result.LastInsertId,
	})
}

// 更新用户
func (c User) UpdateAction() {
	id := c.Params.IntValue("id")
	
	type UpdateUserRequest struct {
		Email string `json:"email"`
	}
	
	var req UpdateUserRequest
	if err := c.Request.JsonDecode(&req); err != nil {
		c.RenderError(400, "参数错误")
		return
	}
	
	// 更新数据
	_, err := data.DB.Table("users").
		Where("id = ?", id).
		Update(map[string]interface{}{
			"email":      req.Email,
			"updated_at": time.Now(),
		})
	
	if err != nil {
		c.RenderError(500, "更新失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"message": "更新成功",
	})
}

// 删除用户
func (c User) DeleteAction() {
	id := c.Params.IntValue("id")
	
	// 软删除
	_, err := data.DB.Table("users").
		Where("id = ?", id).
		Update(map[string]interface{}{
			"status":     0,
			"deleted_at": time.Now(),
		})
	
	if err != nil {
		c.RenderError(500, "删除失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"message": "删除成功",
	})
}
```

### 事务处理

```go
type Order struct {
	*httpsrv.Controller
}

func (c Order) CreateAction() {
	type CreateOrderRequest struct {
		UserID    int64   `json:"user_id"`
		ProductID int64   `json:"product_id"`
		Amount    float64 `json:"amount"`
	}
	
	var req CreateOrderRequest
	if err := c.Request.JsonDecode(&req); err != nil {
		c.RenderError(400, "参数错误")
		return
	}
	
	// 开始事务
	tx := data.DB.Begin()
	
	// 创建订单
	orderResult, err := tx.Table("orders").Insert(map[string]interface{}{
		"user_id":    req.UserID,
		"product_id": req.ProductID,
		"amount":     req.Amount,
		"status":     1,
		"created_at": time.Now(),
	})
	
	if err != nil {
		tx.Rollback()
		c.RenderError(500, "创建订单失败: "+err.Error())
		return
	}
	
	// 扣减库存
	_, err = tx.Table("products").
		Where("id = ? AND stock >= ?", req.ProductID, 1).
		Decrement("stock", 1)
	
	if err != nil {
		tx.Rollback()
		c.RenderError(500, "扣减库存失败: "+err.Error())
		return
	}
	
	// 提交事务
	if err := tx.Commit(); err != nil {
		c.RenderError(500, "事务提交失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status":    "success",
		"order_id":  orderResult.LastInsertId,
		"message":   "下单成功",
	})
}
```

## PostgreSQL 使用示例

### 安装 pgsqlgo

```bash
go get -u github.com/lynkdb/pgsqlgo
```

### 初始化数据库连接

```go
package data

import (
	"github.com/lynkdb/pgsqlgo"
)

var PG *pgsqlgo.PgSQLClient

func InitPG() error {
	PG = pgsqlgo.NewClient(&pgsqlgo.Config{
		Host:     "localhost",
		Port:     5432,
		Username: "postgres",
		Password: "password",
		Database: "myapp",
		SSLMode:  "disable",
		MaxOpenConns: 100,
		MaxIdleConns: 10,
	})
	
	// 测试连接
	if err := PG.Ping(); err != nil {
		return err
	}
	
	return nil
}
```

### 在 Controller 中使用

```go
type Article struct {
	*httpsrv.Controller
}

// 获取文章列表
func (c Article) ListAction() {
	var articles []map[string]interface{}
	
	err := data.PG.Table("articles").
		Select("id, title, summary, author, published_at").
		Where("status = $1", "published").
		OrderBy("published_at DESC").
		Limit(20).
		Find(&articles)
	
	if err != nil {
		c.RenderError(500, "查询失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"data":   articles,
	})
}

// 搜索文章
func (c Article) SearchAction() {
	keyword := c.Params.Value("keyword")
	
	var articles []map[string]interface{}
	
	err := data.PG.Table("articles").
		Select("id, title, summary").
		Where("title ILIKE $1 OR content ILIKE $1", "%"+keyword+"%").
		OrderBy("published_at DESC").
		Limit(50).
		Find(&articles)
	
	if err != nil {
		c.RenderError(500, "搜索失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"data":   articles,
		"count":  len(articles),
	})
}
```

## 最佳实践

### 1. 数据库配置管理

```go
// config/database.go
package config

type DatabaseConfig struct {
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	Charset  string `json:"charset"`
}

func LoadDatabaseConfig() (*DatabaseConfig, error) {
	// 从配置文件或环境变量加载
	config := &DatabaseConfig{
		Driver:   "mysql",
		Host:     os.Getenv("DB_HOST"),
		Port:     3306,
		Username: os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Database: os.Getenv("DB_NAME"),
		Charset:  "utf8mb4",
	}
	return config, nil
}
```

### 2. 数据访问层封装

```go
// repository/user.go
package repository

import (
	"github.com/lynkdb/mysqlgo"
)

type UserRepository struct {
	db *mysqlgo.MySQLClient
}

func NewUserRepository(db *mysqlgo.MySQLClient) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(id int64) (map[string]interface{}, error) {
	var user map[string]interface{}
	err := r.db.Table("users").Where("id = ?", id).First(&user)
	return user, err
}

func (r *UserRepository) FindByEmail(email string) (map[string]interface{}, error) {
	var user map[string]interface{}
	err := r.db.Table("users").Where("email = ?", email).First(&user)
	return user, err
}

func (r *UserRepository) Create(user map[string]interface{}) (int64, error) {
	result, err := r.db.Table("users").Insert(user)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId, nil
}

func (r *UserRepository) Update(id int64, data map[string]interface{}) error {
	_, err := r.db.Table("users").Where("id = ?", id).Update(data)
	return err
}

func (r *UserRepository) Delete(id int64) error {
	_, err := r.db.Table("users").Where("id = ?", id).Delete()
	return err
}
```

### 3. 在 Controller 中使用 Repository

```go
type User struct {
	*httpsrv.Controller
	userRepo *repository.UserRepository
}

func NewController(userRepo *repository.UserRepository) *User {
	return &User{
		userRepo: userRepo,
	}
}

func (c User) GetAction() {
	id := c.Params.IntValue("id")
	
	user, err := c.userRepo.FindByID(id)
	if err != nil {
		c.RenderError(404, "用户不存在")
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"data":   user,
	})
}
```

### 4. 数据库迁移

```go
// migrate/migrate.go
package migrate

import (
	"github.com/lynkdb/mysqlgo"
)

type Migration struct {
	Name string
	Up   func(*mysqlgo.MySQLClient) error
	Down func(*mysqlgo.MySQLClient) error
}

var migrations = []Migration{
	{
		Name: "create_users_table",
		Up: func(db *mysqlgo.MySQLClient) error {
			_, err := db.Exec(`
				CREATE TABLE IF NOT EXISTS users (
					id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
					username VARCHAR(50) NOT NULL UNIQUE,
					email VARCHAR(100) NOT NULL UNIQUE,
					password VARCHAR(255) NOT NULL,
					status TINYINT DEFAULT 1,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
					INDEX idx_email (email)
				) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
			`)
			return err
		},
		Down: func(db *mysqlgo.MySQLClient) error {
			_, err := db.Exec("DROP TABLE IF EXISTS users")
			return err
		},
	},
}

func Run(db *mysqlgo.MySQLClient) error {
	for _, migration := range migrations {
		// 检查迁移是否已执行
		// 这里简化处理，实际应该记录迁移历史
		
		// 执行迁移
		if err := migration.Up(db); err != nil {
			return err
		}
	}
	return nil
}
```

## 其他推荐

Go 编程语言生态系统里包含大量优秀项目，推荐一个项目导航清单供参考：

* Go 第三方库导航 [https://github.com/avelino/awesome-go](https://github.com/avelino/awesome-go)

### 其他常用的 Go 数据库库

* GORM - 全功能 ORM [https://gorm.io](https://gorm.io)
* sqlx - 扩展标准库 database/sql [https://github.com/jmoiron/sqlx](https://github.com/jmoiron/sqlx)
* ent - Facebook 开源的实体框架 [https://entgo.io](https://entgo.io)