package kcrawl

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kevin-zx/kbase/kcache"
)

type Crawler interface {
	Get(url string, data interface{}) error
	DeleteCache(url string) error
	Post(url string, body string, data interface{}) error
	GetTryCache(url []string, data interface{}) (bool, error)
	PostTryCache(urls []string, payloads []string, data interface{}) (bool, error)
}

type cacheCrawler struct {
	cacheDir          kcache.KCache
	header            http.Header
	preHandles        []func([]byte) []byte
	reCombineCacheKey func(string) string
}

func NewCacheCrawler(cacheDir string, header map[string]string, cos ...CrawlerOption) Crawler {
	httpHeader := http.Header{}
	for k, v := range header {
		httpHeader.Add(k, v)
	}
	c := &cacheCrawler{
		cacheDir: kcache.NewFileCache(cacheDir),
		header:   httpHeader,
	}
	for _, co := range cos {
		co(c)
	}
	return c
}

func (c *cacheCrawler) DeleteCache(url string) error {
	return c.cacheDir.Delete(url)
}

func (c *cacheCrawler) Post(url string, payload string, data interface{}) error {
	raw, err := c.cacheDir.Get(url + payload)
	if err != nil {
		return err
	}
	if raw != nil {
		err = json.Unmarshal(raw, data)
		if err != nil {
			return err
		}
		return nil
	}
	method := "POST"
	payloadr := strings.NewReader(payload)
	client := &http.Client{}
	req, err := http.NewRequest(method, url, payloadr)
	if err != nil {
		return err
	}
	req.Header = c.header
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	time.Sleep(5 * time.Second)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("status code error: %d %s and body is: %s", res.StatusCode, res.Status, string(body))
	}
	for _, preHandle := range c.preHandles {
		body = preHandle(body)
	}
	err = json.Unmarshal(body, data)
	if err != nil {
		return fmt.Errorf("unmarshal error: %s and body is: %s", err.Error(), string(body))
	}
	return c.cacheDir.Save(c.generateCacheKey(url, payload), body)
}

func (c *cacheCrawler) PostTryCache(urls []string, payloads []string, data interface{}) (bool, error) {
	for i, url := range urls {
		raw, err := c.cacheDir.Get(c.generateCacheKey(url, payloads[i]))
		if err != nil {
			return false, err
		}
		if raw != nil {
			err = json.Unmarshal(raw, data)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

// 如果payload为空，那么就是get请求
func (c *cacheCrawler) generateCacheKey(url string, payload string) string {
	cacheKey := url + payload
	if c.reCombineCacheKey != nil {
		cacheKey = c.reCombineCacheKey(cacheKey)
	}
	return cacheKey
}

func (c *cacheCrawler) GetTryCache(url []string, data interface{}) (bool, error) {
	for _, u := range url {
		raw, err := c.cacheDir.Get(c.generateCacheKey(u, ""))
		if err != nil {
			return false, err
		}
		if raw != nil {
			err = json.Unmarshal(raw, data)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

func (c *cacheCrawler) Get(url string, data interface{}) error {
	raw, err := c.cacheDir.Get(url)
	if err != nil {
		return err
	}
	if raw != nil {
		err = json.Unmarshal(raw, data)
		if err != nil {
			return err
		}
		return nil
	}

	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return err
	}
	req.Header = c.header
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	time.Sleep(5 * time.Second)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("status code error: %d %s and body is: %s", res.StatusCode, res.Status, string(body))
	}

	for _, preHandle := range c.preHandles {
		body = preHandle(body)
	}
	err = json.Unmarshal(body, data)
	if err != nil {
		fmt.Println(string(body))
		return err
	}
	return c.cacheDir.Save(c.generateCacheKey(url, ""), body)
}
