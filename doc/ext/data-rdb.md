## Model, ORM, Relational Database

httpsrv does not have built-in Model and ORM interfaces for relational databases. If you need such functionality, you can use the following dependency libraries:

* [https://github.com/lynkdb/mysqlgo](https://github.com/lynkdb/mysqlgo) - Go client for MySQL
* [https://github.com/lynkdb/pgsqlgo](https://github.com/lynkdb/pgsqlgo) - Go client for PostgreSQL

## MySQL Usage Example

### Install mysqlgo

```bash
go get -u github.com/lynkdb/mysqlgo
```

### Initialize Database Connection

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
	
	if err := DB.Ping(); err != nil {
		return err
	}
	
	return nil
}
```

### Using in Controller

```go
package controller

import (
	"github.com/hooto/httpsrv"
	"yourapp/data"
)

type User struct {
	*httpsrv.Controller
}

// Get user list
func (c User) ListAction() {
	var users []map[string]interface{}
	
	err := data.DB.Table("users").
		Select("id, username, email, created_at").
		Where("status = ?", 1).
		OrderBy("created_at DESC").
		Limit(20).
		Find(&users)
	
	if err != nil {
		c.RenderError(500, "Query failed: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"data":   users,
	})
}

// Get single user
func (c User) GetAction() {
	id := c.Params.IntValue("id")
	
	var user map[string]interface{}
	
	err := data.DB.Table("users").
		Where("id = ?", id).
		First(&user)
	
	if err != nil {
		c.RenderError(404, "User not found")
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"data":   user,
	})
}

// Create user
func (c User) CreateAction() {
	username := c.Params.Value("username")
	email := c.Params.Value("email")
	password := c.Params.Value("password")
	
	result, err := data.DB.Table("users").Insert(map[string]interface{}{
		"username":  username,
		"email":     email,
		"password":  hashPassword(password),
		"status":    1,
		"created_at": time.Now(),
	})
	
	if err != nil {
		c.RenderError(500, "Create failed: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status":  "success",
		"user_id": result.LastInsertId,
	})
}
```

### Transaction Handling

```go
type Order struct {
	*httpsrv.Controller
}

func (c Order) CreateAction() {
	userID := c.Params.IntValue("user_id")
	productID := c.Params.IntValue("product_id")
	
	// Start transaction
	tx := data.DB.Begin()
	
	// Create order
	orderResult, err := tx.Table("orders").Insert(map[string]interface{}{
		"user_id":    userID,
		"product_id": productID,
		"amount":     100.0,
		"status":     1,
		"created_at": time.Now(),
	})
	
	if err != nil {
		tx.Rollback()
		c.RenderError(500, "Create order failed: "+err.Error())
		return
	}
	
	// Deduct inventory
	_, err = tx.Table("products").
		Where("id = ? AND stock >= ?", productID, 1).
		Decrement("stock", 1)
	
	if err != nil {
		tx.Rollback()
		c.RenderError(500, "Deduct inventory failed: "+err.Error())
		return
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		c.RenderError(500, "Transaction commit failed: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status":    "success",
		"order_id":  orderResult.LastInsertId,
		"message":   "Order created successfully",
	})
}
```

## PostgreSQL Usage Example

### Install pgsqlgo

```bash
go get -u github.com/lynkdb/pgsqlgo
```

### Initialize Database Connection

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
	
	if err := PG.Ping(); err != nil {
		return err
	}
	
	return nil
}
```

### Using in Controller

```go
type Article struct {
	*httpsrv.Controller
}

// Get article list
func (c Article) ListAction() {
	var articles []map[string]interface{}
	
	err := data.PG.Table("articles").
		Select("id, title, summary, author, published_at").
		Where("status = $1", "published").
		OrderBy("published_at DESC").
		Limit(20).
		Find(&articles)
	
	if err != nil {
		c.RenderError(500, "Query failed: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"data":   articles,
	})
}
```

## Best Practices

### 1. Database Configuration Management

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

### 2. Data Access Layer Encapsulation

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

func (r *UserRepository) Create(user map[string]interface{}) (int64, error) {
	result, err := r.db.Table("users").Insert(user)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId, nil
}
```

## Other Recommendations

The Go programming language ecosystem contains a large number of excellent projects. Here is a recommended project navigation list for reference:

* Go Third-party Libraries Directory [https://github.com/avelino/awesome-go](https://github.com/avelino/awesome-go)

### Other Common Go Database Libraries

* GORM - Full-featured ORM [https://gorm.io](https://gorm.io)
* sqlx - Extension to standard library database/sql [https://github.com/jmoiron/sqlx](https://github.com/jmoiron/sqlx)
* ent - Entity framework open-sourced by Facebook [https://entgo.io](https://entgo.io)