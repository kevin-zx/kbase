package kcrawl

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type RawCrawler interface {
	Get(url string) ([]byte, error)
	Post(url string, payload string) ([]byte, error)
}

type rawCrawler struct {
	header http.Header
	// 每次请求之间的间隔时间，单位秒
	intervalSeconds int
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

func (c *rawCrawler) Get(url string) ([]byte, error) {
	return c.Request(url, "", "GET")
}

func (c *rawCrawler) Post(url string, payload string) ([]byte, error) {
	return c.Request(url, payload, "POST")
}

func (c *rawCrawler) Request(url string, payload string, method string) ([]byte, error) {
	var payloadr io.Reader
	if payload != "" {
		payloadr = strings.NewReader(payload)
	}
	client := &http.Client{}
	req, err := http.NewRequest(method, url, payloadr)
	if err != nil {
		return nil, err
	}
	req.Header = c.header
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	time.Sleep(time.Duration(c.intervalSeconds) * time.Second)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s and body is: %s", res.StatusCode, res.Status, string(body))
	}
	return body, nil
}
