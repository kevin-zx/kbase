package kcrawl

import (
	"errors"
	"io"
	"net/http"
	"time"
)

// Deprecated: 还是使用封装好的 crawler吧
func Crawl(url string, header map[string]string, sleep time.Duration) ([]byte, error) {
	// sleep need be preposed
	time.Sleep(sleep)
	method := "GET"
	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range header {
		req.Header.Add(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errors.New("status code is not 200")
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}
