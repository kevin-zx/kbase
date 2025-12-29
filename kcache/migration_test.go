package kcache

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestSqliteCacheMigration(t *testing.T) {
	// 创建临时数据库文件
	tmpfile, err := os.CreateTemp("", "migrationdb-*.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// 第一步：创建一个旧版本的表（没有 expires_at 列）
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?cache=shared&mode=rwc", tmpfile.Name()))
	if err != nil {
		t.Fatalf("打开数据库失败: %v", err)
	}
	defer db.Close()

	// 创建旧版本的表结构
	tableName := "migration_cache"
	_, err = db.Exec(
		"CREATE TABLE IF NOT EXISTS " + tableName + " (key TEXT PRIMARY KEY, value BLOB, created_time DATETIME)",
	)
	if err != nil {
		t.Fatalf("创建旧版本表失败: %v", err)
	}

	// 插入一些测试数据
	testKey := "test-key-old"
	testValue := []byte("test-value-old")
	_, err = db.Exec(
		"INSERT OR REPLACE INTO "+tableName+" (key, value, created_time) VALUES (?, ?, datetime('now'))",
		testKey, testValue,
	)
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	// 第二步：使用新的构造函数，应该自动迁移表结构
	cache, err := NewSqliteCacheWithTTL(tmpfile.Name(), "migration", 0)
	if err != nil {
		t.Fatalf("创建缓存失败: %v", err)
	}
	defer cache.Close()

	// 验证旧数据仍然可以访问
	retrieved, err := cache.Get(testKey)
	if err != nil {
		t.Fatalf("获取旧数据失败: %v", err)
	}
	if string(retrieved) != string(testValue) {
		t.Fatalf("旧数据不匹配: 期望 %s, 得到 %s", testValue, retrieved)
	}

	// 验证新功能可以正常工作
	newKey := "test-key-new"
	newValue := []byte("test-value-new")

	// 测试 Save（使用默认 TTL）
	err = cache.Save(newKey, newValue)
	if err != nil {
		t.Fatalf("保存新数据失败: %v", err)
	}

	// 测试 Get
	retrieved2, err := cache.Get(newKey)
	if err != nil {
		t.Fatalf("获取新数据失败: %v", err)
	}
	if string(retrieved2) != string(newValue) {
		t.Fatalf("新数据不匹配: 期望 %s, 得到 %s", newValue, retrieved2)
	}

	// 测试 SaveWithTTL
	ttlKey := "test-key-ttl"
	ttlValue := []byte("test-value-ttl")
	err = cache.SaveWithTTL(ttlKey, ttlValue, 1*time.Second)
	if err != nil {
		t.Fatalf("SaveWithTTL 失败: %v", err)
	}

	// 立即获取应该能获取到
	retrieved3, err := cache.Get(ttlKey)
	if err != nil {
		t.Fatalf("获取 TTL 数据失败: %v", err)
	}
	if string(retrieved3) != string(ttlValue) {
		t.Fatalf("TTL 数据不匹配: 期望 %s, 得到 %s", ttlValue, retrieved3)
	}

	// 验证表结构已迁移
	var columnCount int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info(?) WHERE name='expires_at'",
		tableName,
	).Scan(&columnCount)
	if err != nil {
		t.Fatalf("检查列失败: %v", err)
	}
	if columnCount != 1 {
		t.Fatalf("expires_at 列应该存在，但找到 %d 个", columnCount)
	}

	// 验证 expires_at 列可以为 NULL（旧数据）
	var expiresAt sql.NullString
	err = db.QueryRow(
		"SELECT expires_at FROM "+tableName+" WHERE key = ?",
		testKey,
	).Scan(&expiresAt)
	if err != nil {
		t.Fatalf("查询旧数据的 expires_at 失败: %v", err)
	}
	if !expiresAt.Valid {
		t.Log("旧数据的 expires_at 为 NULL，符合预期")
	}

	t.Log("迁移测试通过：旧数据保留，新功能正常，表结构已自动升级")
}

func TestSqliteCacheMultipleMigration(t *testing.T) {
	// 测试多次调用不会出错
	tmpfile, err := os.CreateTemp("", "multimigration-*.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// 第一次创建
	cache1, err := NewSqliteCache(tmpfile.Name(), "multi")
	if err != nil {
		t.Fatalf("第一次创建缓存失败: %v", err)
	}
	cache1.Close()

	// 第二次创建（应该不会出错）
	cache2, err := NewSqliteCache(tmpfile.Name(), "multi")
	if err != nil {
		t.Fatalf("第二次创建缓存失败: %v", err)
	}
	cache2.Close()

	// 第三次创建带 TTL
	cache3, err := NewSqliteCacheWithTTL(tmpfile.Name(), "multi", 5*time.Second)
	if err != nil {
		t.Fatalf("第三次创建缓存失败: %v", err)
	}
	cache3.Close()

	t.Log("多次迁移测试通过：重复创建不会出错")
}
