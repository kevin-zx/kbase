package kcache

import "time"

type KCache interface {
	Get(key string) ([]byte, error)
	Save(key string, value []byte) error
	Delete(key string) error
}

type KCloseCache interface {
	KCache
	Closer
}

// KCacheWithTTL 扩展 KCache 接口，添加 TTL 功能
type KCacheWithTTL interface {
	KCache
	// SaveWithTTL 保存数据并设置过期时间
	SaveWithTTL(key string, value []byte, ttl time.Duration) error
	// GetWithExpiry 获取数据并返回过期信息
	GetWithExpiry(key string) ([]byte, *time.Time, error)
	// CleanExpired 清理所有过期数据
	CleanExpired() error
}

// TTLConfig TTL 配置
type TTLConfig struct {
	DefaultTTL time.Duration
}

// ConfigurableCache 可配置缓存接口
type ConfigurableCache interface {
	KCache
	SetConfig(config TTLConfig) error
}
