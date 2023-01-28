package kcrawl

type CrawlerOption func(c *cacheCrawler)

// 有时候需要对body进行一些预处理
func WithBodyPreHandles(preHandles ...func([]byte) []byte) CrawlerOption {
	return func(c *cacheCrawler) {
		c.preHandles = preHandles
	}
}

func WithRecombineCacheKey(handleKey func(string) string) CrawlerOption {
	return func(c *cacheCrawler) {
		c.reCombineCacheKey = handleKey
	}
}
