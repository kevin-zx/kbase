package kcrawl

type CrawlerOption func(c *cacheCrawler)

func WithBodyPreHandles(preHandles ...func([]byte) []byte) CrawlerOption {
	return func(c *cacheCrawler) {
		c.preHandles = preHandles
	}
}
