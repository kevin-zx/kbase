package kcrawl

import (
	"compress/flate"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type RawCrawler interface {
	Get(url string) ([]byte, error)
	Post(url string, payload string) ([]byte, error)
	Put(url string, payload string) ([]byte, error)

	GetWithHeader(url string, header map[string]string) ([]byte, error)
	PostWithHeader(url string, payload string, header map[string]string) ([]byte, error)
	PutWithHeader(url string, payload string, header map[string]string) ([]byte, error)
}

type rawCrawler struct {
	header http.Header
	// 每次请求之间的间隔时间，单位秒
	intervalSeconds      int
	proxyPool            ProxyPool
	hasDeflateCompressed bool
}

func NewRawCrawler(intervalSeconds int, header map[string]string) RawCrawler {
	httpHeader := http.Header{}
	for k, v := range header {
		httpHeader.Add(k, v)
	}
	return &rawCrawler{
		intervalSeconds: intervalSeconds,
		header:          httpHeader,
	}
}

func (c *rawCrawler) GetWithHeader(url string, header map[string]string) ([]byte, error) {
	return c.Request(url, "", "GET", header)
}

func (c *rawCrawler) PostWithHeader(url string, payload string, header map[string]string) ([]byte, error) {
	return c.Request(url, payload, "POST", header)
}

func (c *rawCrawler) Get(url string) ([]byte, error) {
	return c.Request(url, "", "GET", nil)
}

func (c *rawCrawler) Post(url string, payload string) ([]byte, error) {
	return c.Request(url, payload, "POST", nil)
}

// put
func (c *rawCrawler) Put(url string, payload string) ([]byte, error) {
	return c.Request(url, payload, "PUT", nil)
}

func (c *rawCrawler) PutWithHeader(url string, payload string, header map[string]string) ([]byte, error) {
	return c.Request(url, payload, "PUT", header)
}

func (c *rawCrawler) Request(url string, payload string, method string, header map[string]string) ([]byte, error) {
	var payloadr io.Reader
	if payload != "" {
		payloadr = strings.NewReader(payload)
	}
	client := &http.Client{}
	if c.proxyPool != nil {
		proxy := c.proxyPool.GetHttpProxy()
		if proxy != nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxy),
			}
		}
	}

	req, err := http.NewRequest(method, url, payloadr)
	if err != nil {
		return nil, err
	}
	req.Header = c.header
	for k, v := range header {
		req.Header.Set(k, v)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	time.Sleep(time.Duration(c.intervalSeconds) * time.Second)
	var body []byte
	// 如果有 deflate 压缩，需要解压
	if c.hasDeflateCompressed {
		flateReader := flate.NewReader(res.Body)
		defer flateReader.Close()
		body, err = io.ReadAll(flateReader)
		if err != nil {
			return nil, err
		}
	} else {
		body, err = io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}
	}

	if res.StatusCode >= 300 || res.StatusCode < 200 {
		return nil, fmt.Errorf("status code error: %d %s and body is: %s", res.StatusCode, res.Status, string(body))
	}
	return body, nil
}
