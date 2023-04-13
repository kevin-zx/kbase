// download images from the web
package kimgd

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
)

type ImageDownloader interface {
	Download(url string, path string) error
}

type ImageDownloaderImpl struct {
	headers map[string]string
}

type ImageDownloaderOption func(*ImageDownloaderImpl)

func WithHeader(header map[string]string) ImageDownloaderOption {
	return func(d *ImageDownloaderImpl) {
		for k, v := range header {
			d.headers[k] = v
		}
	}
}

func NewImageDownloader(opts ...ImageDownloaderOption) *ImageDownloaderImpl {
	d := &ImageDownloaderImpl{
		headers: make(map[string]string),
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *ImageDownloaderImpl) Download(url string, dir string) (string, error) {
	// download image
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	for k, v := range d.headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// save image to dir
	contentType := resp.Header.Get("Content-Type")
	ext := ".jpg"
	if contentType == "image/png" {
		ext = ".png"
	} else if contentType == "image/gif" {
		ext = ".gif"
	} else if contentType == "image/jpeg" {
		ext = ".jpeg"
	} else if contentType == "image/webp" {
		ext = ".webp"
	} else if contentType == "image/svg+xml" {
		ext = ".svg"
	}
	// 计算文件名
	urlHash := md5.Sum([]byte(url))
	filename := fmt.Sprintf("%x%s", urlHash, ext)

	filePath := filepath.Join(dir, filename)
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	err = ioutil.WriteFile(filePath, data, 0644)
	if err != nil {
		return "", err
	}

	return filePath, nil
}
