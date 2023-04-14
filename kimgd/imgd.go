// download images from the web
package kimgd

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/kevin-zx/kbase/kfile"
)

type ImageDownloader interface {
	Download(url string, path string) error
}

type ImageDownloaderImpl struct {
	headers map[string]string
	// 是否覆盖已存在的文件
	isOverwrite bool
	// file prefix
	prefix string
}

type ImageDownloaderOption func(*ImageDownloaderImpl)

func WithHeader(header map[string]string) ImageDownloaderOption {
	return func(d *ImageDownloaderImpl) {
		for k, v := range header {
			d.headers[k] = v
		}
	}
}

func WithIsOverwrite(isOverwrite bool) ImageDownloaderOption {
	return func(d *ImageDownloaderImpl) {
		d.isOverwrite = isOverwrite
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
	fileNameWithoutExt := d.generateFileName(url)
	if !d.isOverwrite {
		if filePath, ok := d.isExistInDir(fileNameWithoutExt, dir); ok {
			return filePath, nil
		}
	}
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
	filename := fmt.Sprintf("%x%s", fileNameWithoutExt, ext)

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

func (d *ImageDownloaderImpl) isExistInDir(fileNameWithoutExt, dir string) (string, bool) {
	files, err := kfile.GetFilesFromDir(dir)
	if err != nil {
		return "", false
	}
	for _, f := range files {
		if kfile.GetPureFilename(f) == fileNameWithoutExt {
			return f, true
		}
	}
	return "", false
}

func (d *ImageDownloaderImpl) generateFileName(url string) string {
	urlHash := md5.Sum([]byte(url))
	if d.prefix == "" {
		return fmt.Sprintf("%x", urlHash)
	}
	return fmt.Sprintf("%s_%x", d.prefix, urlHash)
}
