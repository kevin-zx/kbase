package kcache

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path"
)

type FileCache struct {
	dir string
}

func NewFileCache(dir string) *FileCache {
	return &FileCache{dir}
}

func (c *FileCache) Save(key string, data []byte) error {
	key = md5String(key)
	if data == nil {
		return nil
	}
	file, err := os.Create(path.Join(c.dir, key))
	if err != nil {
		return err
	}
	defer file.Close()

	buw := bufio.NewWriter(file)
	defer buw.Flush()
	_, err = buw.Write(data)
	return err
}

func (c *FileCache) Get(key string) ([]byte, error) {
	key = md5String(key)
	if !exists(path.Join(c.dir, key)) {
		return nil, nil
	}
	file, err := os.Open(path.Join(c.dir, key))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func exists(fileName string) bool {
	_, err := os.Stat(fileName)
	return err == nil
}

func md5String(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}

func (c *FileCache) DeleteCache(key string) error {
	key = md5String(key)
	if !exists(path.Join(c.dir, key)) {
		return nil
	}
	return os.Remove(path.Join(c.dir, key))
}
