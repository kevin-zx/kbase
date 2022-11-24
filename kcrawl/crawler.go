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
}

type cacheCrawler struct {
	cacheDir   *kcache.FileCache
	header     http.Header
	preHandles []func([]byte) []byte
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
	return c.cacheDir.DeleteCache(url)
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
	for _, preHandle := range c.preHandles {
		body = preHandle(body)
	}
	err = json.Unmarshal(body, data)
	if err != nil {
		return err
	}
	return c.cacheDir.Save(url+payload, body)
}

func (c *cacheCrawler) GetTryCache(url []string, data interface{}) (bool, error) {
	for _, u := range url {
		raw, err := c.cacheDir.Get(u)
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
	for _, preHandle := range c.preHandles {
		body = preHandle(body)
	}
	err = json.Unmarshal(body, data)
	if err != nil {
		fmt.Println(string(body))
		return err
	}

	return c.cacheDir.Save(url, body)
}
