package kcrawl

import "net/http"

type RawCacheCrawlerOption func(c *rawCacheCrawler)

func RawCacheCrawlerWithRecombineCacheKey(handleKey func(string) string) RawCacheCrawlerOption {
	return func(c *rawCacheCrawler) {
		c.recombineCacheKey = handleKey
	}
}

func RawCacheWithIntervalSeconds(intervalSeconds int) RawCacheCrawlerOption {
	return func(c *rawCacheCrawler) {
		c.intervalSeconds = intervalSeconds
	}
}

func RawCacheWithHeader(header map[string]string) RawCacheCrawlerOption {
	return func(c *rawCacheCrawler) {
		httpHeader := http.Header{}
		for k, v := range header {
			httpHeader.Set(k, v)
		}
		c.header = httpHeader
	}
}

func RawCacheWithProxyPool(proxyPool ProxyPool) RawCacheCrawlerOption {
	return func(c *rawCacheCrawler) {
		c.proxyPool = proxyPool
	}
}

func RawCacheWithHasDeflatCompressed(has bool) RawCacheCrawlerOption {
	return func(c *rawCacheCrawler) {
		c.hasDeflateCompressed = has
	}
}
