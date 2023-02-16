package kcrawl

import (
	"encoding/json"
	"time"

	"github.com/kevin-zx/kbase/kcache"
)

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
	cd := combineCacheData(url, "", "GET", data)
	d, _ := json.Marshal(&cd)
	err = rcc.cache.Save(key, d)
	return data, err
}

func combineCacheData(url string, payload string, method string, data []byte) cacheData {
	cd := cacheData{
		Data: data,
		Request: request{
			Method:  method,
			URL:     url,
			Payload: payload,
		},
		CreatedAt: time.Now(),
	}
	return cd
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
	cd := combineCacheData(url, payload, "POST", data)
	d, _ := json.Marshal(&cd)
	err = rcc.cache.Save(key, d)
	return data, err
}

func (rcc *rawCacheCrawler) GetCache(url string) ([]byte, error, bool) {
	return rcc.getCache(url, "")
}

func (rcc *rawCacheCrawler) CacheKey(url string, payload string) string {
	if rcc.recombineCacheKey != nil {
		return rcc.recombineCacheKey(url + payload)
	}
	return url + payload
}

func (rcc *rawCacheCrawler) PostCache(url string, payload string) ([]byte, error, bool) {
	return rcc.getCache(url, payload)
}

func (rcc *rawCacheCrawler) getCache(url string, payload string) ([]byte, error, bool) {
	key := rcc.CacheKey(url, payload)
	data, err := rcc.cache.Get(key)
	if err != nil {
		return nil, err, false
	}
	cd := cacheData{}
	err = json.Unmarshal(data, &cd)
	if err != nil {
		return data, nil, data != nil
	}
	if cd.Data == nil {
		return data, nil, data != nil
	}
	return cd.Data, nil, cd.Data != nil
}

func (rcc *rawCacheCrawler) DeleteCache(url string, payload string) error {
	key := rcc.CacheKey(url, payload)
	return rcc.cache.Delete(key)
}
