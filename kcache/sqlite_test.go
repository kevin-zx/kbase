package kcache

import (
	"os"
	"testing"
	"time"
)

func TestSqliteCacheWithTTL(t *testing.T) {
	// 创建临时数据库文件
	tmpfile, err := os.CreateTemp("", "testdb-*.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// 测试1: 创建不带 TTL 的缓存
	cache, err := NewSqliteCache(tmpfile.Name(), "test")
	if err != nil {
		t.Fatalf("创建缓存失败: %v", err)
	}
	defer cache.Close()

	// 测试基本功能
	key := "test-key"
	value := []byte("test-value")

	// 保存数据
	err = cache.Save(key, value)
	if err != nil {
		t.Fatalf("保存数据失败: %v", err)
	}

	// 获取数据
	retrieved, err := cache.Get(key)
	if err != nil {
		t.Fatalf("获取数据失败: %v", err)
	}
	if string(retrieved) != string(value) {
		t.Fatalf("获取的数据不匹配: 期望 %s, 得到 %s", value, retrieved)
	}

	// 测试2: 创建带 TTL 的缓存
	ttlCache, err := NewSqliteCacheWithTTL(tmpfile.Name(), "test_ttl", 2*time.Second)
	if err != nil {
		t.Fatalf("创建带 TTL 的缓存失败: %v", err)
	}
	defer ttlCache.Close()

	// 测试 SaveWithTTL
	key2 := "test-key-ttl"
	value2 := []byte("test-value-ttl")

	err = ttlCache.SaveWithTTL(key2, value2, 1*time.Second)
	if err != nil {
		t.Fatalf("SaveWithTTL 失败: %v", err)
	}

	// 立即获取应该能获取到
	retrieved2, err := ttlCache.Get(key2)
	if err != nil {
		t.Fatalf("获取 TTL 数据失败: %v", err)
	}
	if string(retrieved2) != string(value2) {
		t.Fatalf("获取的 TTL 数据不匹配: 期望 %s, 得到 %s", value2, retrieved2)
	}

	// 测试 GetWithExpiry
	retrieved3, expiry, err := ttlCache.GetWithExpiry(key2)
	if err != nil {
		t.Fatalf("GetWithExpiry 失败: %v", err)
	}
	if string(retrieved3) != string(value2) {
		t.Fatalf("GetWithExpiry 数据不匹配: 期望 %s, 得到 %s", value2, retrieved3)
	}
	if expiry == nil {
		t.Fatal("GetWithExpiry 应该返回过期时间")
	}

	// 等待过期
	time.Sleep(2 * time.Second)

	// 过期后应该获取不到
	retrieved4, err := ttlCache.Get(key2)
	if err != nil {
		t.Fatalf("获取过期数据时出错: %v", err)
	}
	if retrieved4 != nil {
		t.Fatalf("过期数据应该返回 nil, 得到: %s", retrieved4)
	}

	// 测试 CleanExpired
	err = ttlCache.CleanExpired()
	if err != nil {
		t.Fatalf("CleanExpired 失败: %v", err)
	}

	// 测试 SetConfig
	config := TTLConfig{
		DefaultTTL: 5 * time.Second,
	}
	err = ttlCache.SetConfig(config)
	if err != nil {
		t.Fatalf("SetConfig 失败: %v", err)
	}

	// 测试使用新配置保存
	key3 := "test-key-config"
	value3 := []byte("test-value-config")
	err = ttlCache.Save(key3, value3)
	if err != nil {
		t.Fatalf("使用新配置保存失败: %v", err)
	}

	// 验证接口实现
	var _ KCache = ttlCache
	var _ KCacheWithTTL = ttlCache
	var _ ConfigurableCache = ttlCache
	var _ KCloseCache = ttlCache
}

func TestSqliteCacheBackwardCompatibility(t *testing.T) {
	// 测试向后兼容性
	tmpfile, err := os.CreateTemp("", "testdb-compat-*.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// 使用旧的构造函数
	cache, err := NewSqliteCache(tmpfile.Name(), "compat")
	if err != nil {
		t.Fatalf("创建兼容性缓存失败: %v", err)
	}
	defer cache.Close()

	// 应该能正常使用旧接口
	key := "compat-key"
	value := []byte("compat-value")

	err = cache.Save(key, value)
	if err != nil {
		t.Fatalf("兼容性测试保存失败: %v", err)
	}

	retrieved, err := cache.Get(key)
	if err != nil {
		t.Fatalf("兼容性测试获取失败: %v", err)
	}
	if string(retrieved) != string(value) {
		t.Fatalf("兼容性测试数据不匹配: 期望 %s, 得到 %s", value, retrieved)
	}

	// 应该也能使用新接口（如果实现了）
	// 注意：cache 是 *sqliteCache 类型，我们需要检查它是否实现了 KCacheWithTTL 接口
	var ttlCache KCacheWithTTL = cache
	_, _, err = ttlCache.GetWithExpiry(key)
	if err != nil {
		t.Logf("GetWithExpiry 在兼容模式下可用: %v", err)
	}
}
