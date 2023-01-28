package kcrawl

import "github.com/kevin-zx/kbase/kcache"

// raw cache crawl
type RawCacheCrawler interface {
	RawCrawler
	GetCache(url string) ([]byte, error, bool)
	PostCache(url string, payload string) ([]byte, error, bool)
	DeleteCache(url string, payload string) error
}

type rawCacheCrawler struct {
	rawCrawler
	cache             kcache.KCloseCache
	recombineCacheKey func(key string) string
}

func NewRawCacheCrawler(cache kcache.KCloseCache, opts ...RawCacheCrawlerOption) RawCacheCrawler {
	rcc := &rawCacheCrawler{
		cache:      cache,
		rawCrawler: rawCrawler{},
	}
	for _, opt := range opts {
		opt(rcc)
	}

	return rcc
}

func (rcc *rawCacheCrawler) Get(url string) ([]byte, error) {
	if data, err, ok := rcc.GetCache(url); ok {
		return data, err
	}
	data, err := rcc.rawCrawler.Get(url)
	if err != nil {
		return nil, err
	}
	key := rcc.CacheKey(url, "")
	err = rcc.cache.Save(key, data)
	return data, err
}

func (rcc *rawCacheCrawler) Post(url string, payload string) ([]byte, error) {
	if data, err, ok := rcc.PostCache(url, payload); ok {
		return data, err
	}
	data, err := rcc.rawCrawler.Post(url, payload)
	if err != nil {
		return nil, err
	}
	key := rcc.CacheKey(url, payload)
	err = rcc.cache.Save(key, data)
	return data, err
}

func (rcc *rawCacheCrawler) GetCache(url string) ([]byte, error, bool) {
	key := rcc.CacheKey(url, "")
	data, err := rcc.cache.Get(key)
	if err != nil {
		return nil, err, false
	}
	return data, nil, data != nil
}

func (rcc *rawCacheCrawler) CacheKey(url string, payload string) string {
	if rcc.recombineCacheKey != nil {
		return rcc.recombineCacheKey(url + payload)
	}
	return url + payload
}

func (rcc *rawCacheCrawler) PostCache(url string, payload string) ([]byte, error, bool) {
	key := rcc.CacheKey(url, payload)
	data, err := rcc.cache.Get(key)
	if err != nil {
		return nil, err, false
	}
	return data, nil, data != nil
}

func (rcc *rawCacheCrawler) DeleteCache(url string, payload string) error {
	key := rcc.CacheKey(url, payload)
	return rcc.cache.Delete(key)
}
