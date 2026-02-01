## Cache, Key-Value Database

httpsrv does not have built-in cache or Key-Value database interfaces. If you need such functionality, you can use following dependency libraries:

* [https://github.com/lynkdb/redisgo](https://github.com/lynkdb/redisgo) - Go client for Redis
* [https://github.com/lynkdb/ssdbgo](https://github.com/lynkdb/ssdbgo) - Go client for SSDB
* [https://github.com/lynkdb/kvgo](https://github.com/lynkdb/kvgo) - An embedded Key-Value database library for Go

## Redis Usage Example

### Install redisgo

```bash
go get -u github.com/lynkdb/redisgo
```

### Initialize Redis Connection

```go
package cache

import (
	"github.com/lynkdb/redisgo"
)

var Redis *redisgo.RedisClient

func InitRedis() error {
	Redis = redisgo.NewClient(&redisgo.Config{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
		PoolSize: 100,
	})
	
	if err := Redis.Ping().Err(); err != nil {
		return err
	}
	
	return nil
}
```

### Basic Operations

```go
package controller

import (
	"time"
	"github.com/hooto/httpsrv"
	"yourapp/cache"
)

type CacheDemo struct {
	*httpsrv.Controller
}

// Set cache
func (c CacheDemo) SetAction() {
	key := c.Params.Value("key")
	value := c.Params.Value("value")
	
	err := cache.Redis.Set(key, value, 5*time.Minute).Err()
	if err != nil {
		c.RenderError(500, "Set cache failed: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"message": "Cache set successfully",
	})
}

// Get cache
func (c CacheDemo) GetAction() {
	key := c.Params.Value("key")
	
	value, err := cache.Redis.Get(key).Result()
	if err != nil {
		c.RenderError(404, "Cache not found: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"value":  value,
	})
}
```

### Cache Aside Pattern

```go
type Product struct {
	*httpsrv.Controller
}

func (c Product) GetAction() {
	productID := c.Params.Value("id")
	cacheKey := "product:" + productID
	
	// Check cache first
	product, err := cache.Redis.Get(cacheKey).Result()
	if err == nil && product != "" {
		c.Response.Out.Header().Set("X-Cache", "HIT")
		c.RenderJson(map[string]interface{}{
			"status": "success",
			"data":   product,
		})
		return
	}
	
	// Cache miss, query database
	productData := fetchProductFromDB(productID)
	if productData == nil {
		c.RenderError(404, "Product not found")
		return
	}
	
	// Write to cache
	cache.Redis.Set(cacheKey, productData, 10*time.Minute)
	
	c.Response.Out.Header().Set("X-Cache", "MISS")
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"data":   productData,
	})
}
```

### Distributed Lock

```go
type OrderLock struct {
	*httpsrv.Controller
}

func (c OrderLock) CreateAction() {
	userID := c.Params.Value("user_id")
	productID := c.Params.Value("product_id")
	
	// Generate lock key
	lockKey := "lock:order:" + productID
	
	// Try to acquire lock
	locked, err := cache.Redis.SetNX(lockKey, userID, 10*time.Second).Result()
	if err != nil {
		c.RenderError(500, "Acquire lock failed: "+err.Error())
		return
	}
	
	if !locked {
		c.RenderError(429, "System busy, please try again later")
		return
	}
	
	// Ensure release lock
	defer cache.Redis.Del(lockKey)
	
	// Execute order logic
	orderID := createOrder(userID, productID)
	
	c.RenderJson(map[string]interface{}{
		"status":   "success",
		"order_id": orderID,
	})
}
```

## Embedded KV Database (kvgo)

### Install kvgo

```bash
go get -u github.com/lynkdb/kvgo
```

### Initialize kvgo

```go
package storage

import (
	"github.com/lynkdb/kvgo"
)

var KV *kvgo.KVDB

func InitKV() error {
	var err error
	KV, err = kvgo.Open("data/kvdb")
	if err != nil {
		return err
	}
	return nil
}

func CloseKV() {
	if KV != nil {
		KV.Close()
	}
}
```

### Basic Operations

```go
type LocalStorage struct {
	*httpsrv.Controller
}

// Set key-value
func (c LocalStorage) SetAction() {
	key := c.Params.Value("key")
	value := c.Params.Value("value")
	
	err := storage.KV.Put([]byte(key), []byte(value))
	if err != nil {
		c.RenderError(500, "Set failed: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"message": "Set successfully",
	})
}

// Get key-value
func (c LocalStorage) GetAction() {
	key := c.Params.Value("key")
	
	value, err := storage.KV.Get([]byte(key))
	if err != nil {
		c.RenderError(404, "Key not found: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"value":  string(value),
	})
}
```

## Cache Best Practices

### 1. Cache Key Naming

```go
// Recommended naming convention
func CacheKey(category, identifier string) string {
	return fmt.Sprintf("%s:%s", category, identifier)
}

// Examples
userCacheKey := CacheKey("user", "12345")         // user:12345
productCacheKey := CacheKey("product", "67890")    // product:67890
```

### 2. Cache Expiration Strategy

```go
const (
	// Short-term cache (5 minutes)
	CacheShort = 5 * time.Minute
	// Medium-term cache (1 hour)
	CacheMedium = 1 * time.Hour
	// Long-term cache (1 day)
	CacheLong = 24 * time.Hour
)

// Choose appropriate expiration based on data update frequency
func SetUserCache(key, value string) {
	cache.Redis.Set(key, value, CacheMedium)
}

func SetConfigCache(key, value string) {
	cache.Redis.Set(key, value, CacheLong)
}
```

### 3. Cache Warm-up

```go
// Warm-up common caches when application starts
func WarmUpCache() {
	// Pre-load hot products
	popularProducts := getPopularProductsFromDB()
	for _, product := range popularProducts {
		cacheKey := fmt.Sprintf("product:%s", product.ID)
		cache.Redis.Set(cacheKey, product, 10*time.Minute)
	}
	
	// Pre-load system configs
	configs := loadSystemConfigs()
	cache.Redis.Set("system:config", configs, 1*time.Hour)
}
```

### 4. Cache Penetration Protection

```go
func GetUser(id string) (map[string]interface{}, error) {
	cacheKey := "user:" + id
	
	// Check cache first
	user, err := cache.Redis.Get(cacheKey).Result()
	if err == nil && user != "" {
		// Cache hit
		var userMap map[string]interface{}
		json.Unmarshal([]byte(user), &userMap)
		return userMap, nil
	}
	
	// Query database
	userData, err := fetchUserFromDB(id)
	if err != nil {
		// Cache empty value to prevent cache penetration
		if err.Error() == "user not found" {
			cache.Redis.Set(cacheKey, "", 5*time.Minute)
		}
		return nil, err
	}
	
	// Write to cache
	userJSON, _ := json.Marshal(userData)
	cache.Redis.Set(cacheKey, userJSON, 10*time.Minute)
	
	return userData, nil
}
```

## Other Recommendations

The Go programming language ecosystem contains a large number of excellent projects. Here is a recommended project navigation list for reference:

* Go Third-party Libraries Directory [https://github.com/avelino/awesome-go](https://github.com/avelino/awesome-go)

### Other Common Go Cache Libraries

* bigcache - High-performance memory cache [https://github.com/allegro/bigcache](https://github.com/allegro/bigcache)
* ristretto - High-performance cache library [https://github.com/dgraph-io/ristretto](https://github.com/dgraph-io/ristretto)
* go-cache - Memory cache library [https://github.com/patrickmn/go-cache](https://github.com/patrickmn/go-cache)