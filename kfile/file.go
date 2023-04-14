package kfile

import (
	"os"
	"path"
)

func FileExist(filename string) bool {
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func Mkdir(dir string) error {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}
	return nil
}

func GetFilesFromDir(dir string) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var filenames []string
	for _, file := range files {
		filenames = append(filenames, file.Name())
	}
	return filenames, nil
}

// remove file path and extension
func GetPureFilename(filename string) string {
	fn := path.Base(filename)
	ext := path.Ext(fn)
	return fn[:len(fn)-len(ext)]
}
