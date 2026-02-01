## Cache, Key-Value 数据库

httpsrv 没有内置缓存等 Key-Value 数据库接口，如果有这方面需求，可使用如下依赖库：

* [https://github.com/lynkdb/redisgo](https://github.com/lynkdb/redisgo) - Go client for Redis
* [https://github.com/lynkdb/ssdbgo](https://github.com/lynkdb/ssdbgo) - Go client for SSDB
* [https://github.com/lynkdb/kvgo](https://github.com/lynkdb/kvgo) - An embedded Key-Value database library for Go language

## Redis 使用示例

### 安装 redisgo

```bash
go get -u github.com/lynkdb/redisgo
```

### 初始化 Redis 连接

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
	
	// 测试连接
	if err := Redis.Ping().Err(); err != nil {
		return err
	}
	
	return nil
}
```

### 基本操作

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

// 设置缓存
func (c CacheDemo) SetAction() {
	key := c.Params.Value("key")
	value := c.Params.Value("value")
	
	err := cache.Redis.Set(key, value, 5*time.Minute).Err()
	if err != nil {
		c.RenderError(500, "设置缓存失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"message": "缓存设置成功",
	})
}

// 获取缓存
func (c CacheDemo) GetAction() {
	key := c.Params.Value("key")
	
	value, err := cache.Redis.Get(key).Result()
	if err != nil {
		c.RenderError(404, "缓存不存在: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"value":  value,
	})
}

// 删除缓存
func (c CacheDemo) DeleteAction() {
	key := c.Params.Value("key")
	
	err := cache.Redis.Del(key).Err()
	if err != nil {
		c.RenderError(500, "删除缓存失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"message": "缓存删除成功",
	})
}

// 批量设置
func (c CacheDemo) MSetAction() {
	err := cache.Redis.MSet("key1", "value1", "key2", "value2", "key3", "value3").Err()
	if err != nil {
		c.RenderError(500, "批量设置失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"message": "批量设置成功",
	})
}

// 批量获取
func (c CacheDemo) MGetAction() {
	values, err := cache.Redis.MGet("key1", "key2", "key3").Result()
	if err != nil {
		c.RenderError(500, "批量获取失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"values": values,
	})
}
```

### 哈希表操作

```go
type UserCache struct {
	*httpsrv.Controller
}

// 设置用户信息（Hash）
func (c UserCache) SetUserAction() {
	userID := c.Params.Value("user_id")
	username := c.Params.Value("username")
	email := c.Params.Value("email")
	
	// 使用 Hash 存储用户信息
	err := cache.Redis.HSet("user:"+userID, map[string]interface{}{
		"username":  username,
		"email":     email,
		"updated_at": time.Now().Unix(),
	}).Err()
	
	if err != nil {
		c.RenderError(500, "设置失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"message": "用户信息缓存成功",
	})
}

// 获取用户信息
func (c UserCache) GetUserAction() {
	userID := c.Params.Value("user_id")
	
	// 获取整个 Hash
	userInfo, err := cache.Redis.HGetAll("user:" + userID).Result()
	if err != nil {
		c.RenderError(500, "获取失败: "+err.Error())
		return
	}
	
	if len(userInfo) == 0 {
		c.RenderError(404, "用户不存在")
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"user":   userInfo,
	})
}

// 获取用户单个字段
func (c UserCache) GetUserFieldAction() {
	userID := c.Params.Value("user_id")
	field := c.Params.Value("field")
	
	value, err := cache.Redis.HGet("user:"+userID, field).Result()
	if err != nil {
		c.RenderError(404, "字段不存在: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"field":  field,
		"value":  value,
	})
}
```

### 列表操作

```go
type ListCache struct {
	*httpsrv.Controller
}

// 添加到列表左侧
func (c ListCache) LPushAction() {
	key := c.Params.Value("key")
	value := c.Params.Value("value")
	
	err := cache.Redis.LPush(key, value).Err()
	if err != nil {
		c.RenderError(500, "添加失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"message": "添加成功",
	})
}

// 添加到列表右侧
func (c ListCache) RPushAction() {
	key := c.Params.Value("key")
	value := c.Params.Value("value")
	
	err := cache.Redis.RPush(key, value).Err()
	if err != nil {
		c.RenderError(500, "添加失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"message": "添加成功",
	})
}

// 获取列表范围
func (c ListCache) LRangeAction() {
	key := c.Params.Value("key")
	start := c.Params.IntValue("start", 0)
	stop := c.Params.IntValue("stop", -1)
	
	values, err := cache.Redis.LRange(key, int64(start), int64(stop)).Result()
	if err != nil {
		c.RenderError(500, "获取失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"values": values,
	})
}

// 获取列表长度
func (c ListCache) LLenAction() {
	key := c.Params.Value("key")
	
	length, err := cache.Redis.LLen(key).Result()
	if err != nil {
		c.RenderError(500, "获取失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"length": length,
	})
}
```

### 集合操作

```go
type SetCache struct {
	*httpsrv.Controller
}

// 添加到集合
func (c SetCache) SAddAction() {
	key := c.Params.Value("key")
	value := c.Params.Value("value")
	
	err := cache.Redis.SAdd(key, value).Err()
	if err != nil {
		c.RenderError(500, "添加失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"message": "添加成功",
	})
}

// 获取集合所有成员
func (c SetCache) SMembersAction() {
	key := c.Params.Value("key")
	
	members, err := cache.Redis.SMembers(key).Result()
	if err != nil {
		c.RenderError(500, "获取失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status":  "success",
		"members": members,
	})
}

// 判断元素是否在集合中
func (c SetCache) SIsMemberAction() {
	key := c.Params.Value("key")
	value := c.Params.Value("value")
	
	exists, err := cache.Redis.SIsMember(key, value).Result()
	if err != nil {
		c.RenderError(500, "查询失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"exists": exists,
	})
}
```

### 有序集合操作

```go
type ZSetCache struct {
	*httpsrv.Controller
}

// 添加到有序集合
func (c ZSetCache) ZAddAction() {
	key := c.Params.Value("key")
	member := c.Params.Value("member")
	score := c.Params.Float64Value("score")
	
	err := cache.Redis.ZAdd(key, redisgo.Z{
		Score:  score,
		Member: member,
	}).Err()
	
	if err != nil {
		c.RenderError(500, "添加失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"message": "添加成功",
	})
}

// 获取排行榜（分数从高到低）
func (c ZSetCache) ZRevRangeAction() {
	key := c.Params.Value("key")
	start := c.Params.IntValue("start", 0)
	stop := c.Params.IntValue("stop", 9)
	
	members, err := cache.Redis.ZRevRange(key, int64(start), int64(stop)).Result()
	if err != nil {
		c.RenderError(500, "获取失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status":  "success",
		"members": members,
	})
}

// 获取带分数的排行榜
func (c ZSetCache) ZRevRangeWithScoresAction() {
	key := c.Params.Value("key")
	start := c.Params.IntValue("start", 0)
	stop := c.Params.IntValue("stop", 9)
	
	results, err := cache.Redis.ZRevRangeWithScores(key, int64(start), int64(stop)).Result()
	if err != nil {
		c.RenderError(500, "获取失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status":  "success",
		"results": results,
	})
}
```

### 缓存模式

#### 1. Cache Aside 模式（旁路缓存）

```go
type Product struct {
	*httpsrv.Controller
}

func (c Product) GetAction() {
	productID := c.Params.Value("id")
	cacheKey := "product:" + productID
	
	// 先查缓存
	product, err := cache.Redis.Get(cacheKey).Result()
	if err == nil && product != "" {
		c.Response.Out.Header().Set("X-Cache", "HIT")
		c.RenderJson(map[string]interface{}{
			"status": "success",
			"data":   product,
		})
		return
	}
	
	// 缓存未命中，查数据库
	productData := fetchProductFromDB(productID)
	if productData == nil {
		c.RenderError(404, "产品不存在")
		return
	}
	
	// 写入缓存
	cache.Redis.Set(cacheKey, productData, 10*time.Minute)
	
	c.Response.Out.Header().Set("X-Cache", "MISS")
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"data":   productData,
	})
}
```

#### 2. Write Through 模式（写穿透）

```go
func (c Product) UpdateAction() {
	productID := c.Params.Value("id")
	cacheKey := "product:" + productID
	
	// 更新数据库
	productData := updateProductInDB(productID, c.Params)
	
	// 同时更新缓存
	cache.Redis.Set(cacheKey, productData, 10*time.Minute)
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"message": "更新成功",
	})
}
```

#### 3. 分布式锁

```go
type OrderLock struct {
	*httpsrv.Controller
}

func (c OrderLock) CreateAction() {
	userID := c.Params.Value("user_id")
	productID := c.Params.Value("product_id")
	
	// 生成锁的 key
	lockKey := "lock:order:" + productID
	
	// 尝试获取锁
	locked, err := cache.Redis.SetNX(lockKey, userID, 10*time.Second).Result()
	if err != nil {
		c.RenderError(500, "获取锁失败: "+err.Error())
		return
	}
	
	if !locked {
		c.RenderError(429, "系统繁忙，请稍后重试")
		return
	}
	
	// 确保释放锁
	defer cache.Redis.Del(lockKey)
	
	// 执行下单逻辑
	orderID := createOrder(userID, productID)
	
	c.RenderJson(map[string]interface{}{
		"status":   "success",
		"order_id": orderID,
	})
}
```

## SSDB 使用示例

SSDB 是一个高性能的支持丰富数据结构的 NoSQL 数据库，兼容 Redis 协议。

### 初始化 SSDB 连接

```go
package cache

import (
	"github.com/lynkdb/ssdbgo"
)

var SSDB *ssdbgo.Client

func InitSSDB() error {
	SSDB = ssdbgo.NewClient(&ssdbgo.Config{
		Addr:     "localhost:8888",
		Password: "",
		PoolSize: 100,
	})
	
	// 测试连接
	if err := SSDB.Ping().Err(); err != nil {
		return err
	}
	
	return nil
}
```

SSDB 的 API 与 Redis 类似，使用方法基本相同，可以参考上面的 Redis 示例。

## 嵌入式 KV 数据库 (kvgo)

### 安装 kvgo

```bash
go get -u github.com/lynkdb/kvgo
```

### 初始化 kvgo

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

### 基本操作

```go
type LocalStorage struct {
	*httpsrv.Controller
}

// 设置键值
func (c LocalStorage) SetAction() {
	key := c.Params.Value("key")
	value := c.Params.Value("value")
	
	err := storage.KV.Put([]byte(key), []byte(value))
	if err != nil {
		c.RenderError(500, "设置失败: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"message": "设置成功",
	})
}

// 获取键值
func (c LocalStorage) GetAction() {
	key := c.Params.Value("key")
	
	value, err := storage.KV.Get([]byte(key))
	if err != nil {
		c.RenderError(404, "键不存在: "+err.Error())
		return
	}
	
	c.RenderJson(map[string]interface{}{
		"status": "success",
		"value":  string(value),
	})
}

// 删除键
func (c LocalStorage) DeleteAction() {
	key := c.Params.Value("key")
	
	err := storage.KV.Delete([]byte(key))
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

## 缓存最佳实践

### 1. 缓存键命名规范

```go
// 推荐的命名规范
func CacheKey(category, identifier string) string {
	return fmt.Sprintf("%s:%s", category, identifier)
}

// 示例
userCacheKey := CacheKey("user", "12345")          // user:12345
productCacheKey := CacheKey("product", "67890")  // product:67890
```

### 2. 缓存过期时间策略

```go
const (
	// 短期缓存（5分钟）
	CacheShort = 5 * time.Minute
	// 中期缓存（1小时）
	CacheMedium = 1 * time.Hour
	// 长期缓存（1天）
	CacheLong = 24 * time.Hour
)

// 根据数据更新频率选择不同的过期时间
func SetUserCache(key, value string) {
	cache.Redis.Set(key, value, CacheMedium)
}

func SetConfigCache(key, value string) {
	cache.Redis.Set(key, value, CacheLong)
}
```

### 3. 缓存预热

```go
// 应用启动时预热常用缓存
func WarmUpCache() {
	// 预加载热门商品
	popularProducts := getPopularProductsFromDB()
	for _, product := range popularProducts {
		cacheKey := fmt.Sprintf("product:%s", product.ID)
		cache.Redis.Set(cacheKey, product, 10*time.Minute)
	}
	
	// 预加载系统配置
	configs := loadSystemConfigs()
	cache.Redis.Set("system:config", configs, 1*time.Hour)
}
```

### 4. 缓存穿透保护

```go
func GetUser(id string) (map[string]interface{}, error) {
	cacheKey := "user:" + id
	
	// 先查缓存
	user, err := cache.Redis.Get(cacheKey).Result()
	if err == nil && user != "" {
		// 缓存命中
		var userMap map[string]interface{}
		json.Unmarshal([]byte(user), &userMap)
		return userMap, nil
	}
	
	// 查数据库
	userData, err := fetchUserFromDB(id)
	if err != nil {
		// 缓存空值，防止缓存穿透
		if err.Error() == "user not found" {
			cache.Redis.Set(cacheKey, "", 5*time.Minute)
		}
		return nil, err
	}
	
	// 写入缓存
	userJSON, _ := json.Marshal(userData)
	cache.Redis.Set(cacheKey, userJSON, 10*time.Minute)
	
	return userData, nil
}
```

## 其他推荐

Go 编程语言生态系统里包含大量优秀项目，推荐一个项目导航清单供参考：

* Go 第三方库导航 [https://github.com/avelino/awesome-go](https://github.com/avelino/awesome-go)

### 其他常用的 Go 缓存库

* bigcache - 高性能内存缓存 [https://github.com/allegro/bigcache](https://github.com/allegro/bigcache)
* ristretto - 高性能缓存库 [https://github.com/dgraph-io/ristretto](https://github.com/dgraph-io/ristretto)
* go-cache - 内存缓存库 [https://github.com/patrickmn/go-cache](https://github.com/patrickmn/go-cache)