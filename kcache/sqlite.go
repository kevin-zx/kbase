// 使用 sqlite 作为本地缓存

package kcache

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type sqliteCache struct {
	db         *sql.DB
	prefix     string
	tableName  string
	defaultTTL time.Duration // 默认 TTL，0 表示永不过期
}

func NewSqliteCache(dbfile string, prefix string) (*sqliteCache, error) {
	return NewSqliteCacheWithTTL(dbfile, prefix, 0)
}

// NewSqliteCacheWithTTL 创建带有 TTL 功能的 SQLite 缓存
func NewSqliteCacheWithTTL(dbfile string, prefix string, defaultTTL time.Duration) (*sqliteCache, error) {
	conectFmt := "file:%s?cache=shared&mode=rwc"
	db, err := sql.Open("sqlite3", fmt.Sprintf(conectFmt, dbfile))
	if err != nil {
		return nil, fmt.Errorf("ksqlite file: %s, got a error: %v", dbfile, err)
	}
	tableName := prefix + "_cache"

	sc := &sqliteCache{
		db:         db,
		tableName:  tableName,
		defaultTTL: defaultTTL,
	}
	err = sc.createCacheTable()
	if err != nil {
		return nil, fmt.Errorf("ksqlite file: %s, got a error: %v", dbfile, err)
	}
	return sc, nil
}

func (c *sqliteCache) createCacheTable() error {
	_, err := c.db.Exec(
		"CREATE TABLE IF NOT EXISTS " + c.tableName + " (key TEXT PRIMARY KEY, value BLOB, created_time DATETIME, expires_at DATETIME)",
	)
	if err != nil {
		return err
	}

	return err
}

func (c *sqliteCache) Get(key string) ([]byte, error) {
	key = md5String(key)
	var value []byte
	// 检查是否过期，只返回未过期的数据
	// 使用 CURRENT_TIMESTAMP 确保时间一致性
	// 条件: 1) expires_at IS NULL (永不过期) OR 2) expires_at > CURRENT_TIMESTAMP (未过期)
	err := c.db.QueryRow(
		"SELECT value FROM "+
			c.tableName+
			" WHERE key = ? AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)", key).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, nil // 键不存在或已过期
	}
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (s *sqliteCache) Save(key string, value []byte) error {
	// 使用默认 TTL
	return s.SaveWithTTL(key, value, s.defaultTTL)
}

// SaveWithTTL 保存数据并设置过期时间
func (s *sqliteCache) SaveWithTTL(key string, value []byte, ttl time.Duration) error {
	key = md5String(key)
	err := s.Delete(key)
	if err != nil {
		return err
	}

	var query string
	var args []interface{}

	if ttl > 0 {
		// 直接在 SQL 中计算过期时间
		query = "INSERT OR REPLACE INTO " + s.tableName +
			" (key, value, created_time, expires_at) VALUES (?, ?, CURRENT_TIMESTAMP, datetime(CURRENT_TIMESTAMP, ?))"
		args = []interface{}{key, value, fmt.Sprintf("+%d seconds", int(ttl.Seconds()))}
	} else {
		// 永不过期
		query = "INSERT OR REPLACE INTO " + s.tableName +
			" (key, value, created_time, expires_at) VALUES (?, ?, CURRENT_TIMESTAMP, NULL)"
		args = []interface{}{key, value}
	}

	_, err = s.db.Exec(query, args...)
	return err
}

func (s *sqliteCache) Delete(key string) error {
	key = md5String(key)
	_, err := s.db.Exec(
		"DELETE FROM "+
			s.tableName+
			" WHERE key = ?",
		key)
	return err
}

// GetWithExpiry 获取数据并返回过期信息
func (c *sqliteCache) GetWithExpiry(key string) ([]byte, *time.Time, error) {
	key = md5String(key)
	var value []byte
	var expiresAtStr sql.NullString

	err := c.db.QueryRow(
		"SELECT value, expires_at FROM "+
			c.tableName+
			" WHERE key = ? AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)",
		key).Scan(&value, &expiresAtStr)

	if err == sql.ErrNoRows {
		return nil, nil, nil // 键不存在或已过期
	}
	if err != nil {
		return nil, nil, err
	}

	if expiresAtStr.Valid && expiresAtStr.String != "" {
		// 尝试解析不同的时间格式
		var expiryTime time.Time
		var err error

		// 尝试 ISO 8601 格式: "2006-01-02T15:04:05Z"
		expiryTime, err = time.Parse(time.RFC3339, expiresAtStr.String)
		if err != nil {
			// 尝试 SQLite datetime 格式: "2006-01-02 15:04:05"
			expiryTime, err = time.Parse("2006-01-02 15:04:05", expiresAtStr.String)
			if err != nil {
				return nil, nil, fmt.Errorf("解析过期时间失败: %v, 原始字符串: %s", err, expiresAtStr.String)
			}
		}
		return value, &expiryTime, nil
	}
	return value, nil, nil // 永不过期 (expires_at IS NULL)
}

// CleanExpired 清理所有过期数据
func (c *sqliteCache) CleanExpired() error {
	_, err := c.db.Exec(
		"DELETE FROM " + c.tableName + " WHERE expires_at IS NOT NULL AND expires_at <= CURRENT_TIMESTAMP")
	return err
}

// SetConfig 设置缓存配置
func (c *sqliteCache) SetConfig(config TTLConfig) error {
	c.defaultTTL = config.DefaultTTL
	return nil
}

func (s *sqliteCache) Close() error {
	return s.db.Close()
}
