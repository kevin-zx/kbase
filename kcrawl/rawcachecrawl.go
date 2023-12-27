package kcrawl

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/kevin-zx/kbase/kcache"
)

// raw cache crawl
type RawCacheCrawler interface {
	// RawCrawler
	Get(url string, keys ...string) ([]byte, error)
	Post(url string, payload string, keys ...string) ([]byte, error)

	GetWithHeader(url string, header map[string]string, keys ...string) ([]byte, error)
	PostWithHeader(url string, payload string, header map[string]string, keys ...string) ([]byte, error)

	GetCache(url string, keys ...string) ([]byte, error, bool)
	PostCache(url string, payload string, keys ...string) ([]byte, error, bool)
	DeleteCache(url string, payload string, keys ...string) error
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

func (rcc *rawCacheCrawler) GetWithHeader(url string, header map[string]string, keys ...string) ([]byte, error) {
	if data, err, ok := rcc.GetCache(url, keys...); ok {
		return data, err
	}
	data, err := rcc.rawCrawler.GetWithHeader(url, header)
	if err != nil {
		return nil, err
	}
	key := rcc.CacheKey(url, "")
	cd := combineCacheData(url, "", "GET", data)
	d, _ := json.Marshal(&cd)
	err = rcc.cache.Save(key, d)
	return data, err
}

func (rcc *rawCacheCrawler) Get(url string, keys ...string) ([]byte, error) {
	return rcc.GetWithHeader(url, nil, keys...)
}

func combineCacheData(url string, payload string, method string, data []byte) cacheData {
	cd := cacheData{
		Data: string(data),
		Request: request{
			Method:  method,
			URL:     url,
			Payload: payload,
		},
		CreatedAt: time.Now(),
	}
	return cd
}

func (rcc *rawCacheCrawler) PostWithHeader(url string, payload string, header map[string]string, keys ...string) ([]byte, error) {
	if data, err, ok := rcc.PostCache(url, payload, keys...); ok {
		return data, err
	}
	data, err := rcc.rawCrawler.PostWithHeader(url, payload, header)
	if err != nil {
		return nil, err
	}
	key := rcc.CacheKey(url, payload)
	cd := combineCacheData(url, payload, "POST", data)
	d, _ := json.Marshal(&cd)
	err = rcc.cache.Save(key, d)
	return data, err
}

func (rcc *rawCacheCrawler) Post(url string, payload string, keys ...string) ([]byte, error) {
	return rcc.PostWithHeader(url, payload, nil, keys...)
}

func (rcc *rawCacheCrawler) GetCache(url string, keys ...string) ([]byte, error, bool) {
	return rcc.getCache(url, "", keys...)
}

func (rcc *rawCacheCrawler) CacheKey(url string, payload string, keys ...string) string {
	keyseed := url + payload
	if len(keys) > 0 {
		keyseed = ""
		sortedKeys := sort.StringSlice(keys)
		sortedKeys.Sort()
		for _, k := range sortedKeys {
			keyseed += k
		}
	}

	if rcc.recombineCacheKey != nil {
		return rcc.recombineCacheKey(keyseed)
	}
	return keyseed
}

func (rcc *rawCacheCrawler) PostCache(url string, payload string, keys ...string) ([]byte, error, bool) {
	return rcc.getCache(url, payload, keys...)
}

func (rcc *rawCacheCrawler) getCache(url string, payload string, keys ...string) ([]byte, error, bool) {
	key := rcc.CacheKey(url, payload, keys...)
	data, err := rcc.cache.Get(key)
	if err != nil {
		return nil, err, false
	}
	cd := cacheData{}
	err = json.Unmarshal(data, &cd)
	if err != nil {
		return data, nil, data != nil
	}
	if cd.Data == "" {
		return data, nil, data != nil
	}
	return []byte(cd.Data), nil, cd.Data != ""
}

func (rcc *rawCacheCrawler) DeleteCache(url string, payload string, keys ...string) error {
	key := rcc.CacheKey(url, payload, keys...)
	return rcc.cache.Delete(key)
}
