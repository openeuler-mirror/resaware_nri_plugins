package util

import (
	"fmt"
	"os"
)

func ReadFile(path string) ([]byte, error) {
	if IsDir(path) {
		return nil, fmt.Errorf("%s is not a file", path)
	}

	if !PathExist(path) {
		return nil, fmt.Errorf("%s is not exist", path)
	}

	return os.ReadFile(path)
}

func WriteFile(path string, data string) error {
	if IsDir(path) {
		return fmt.Errorf("%s is not a file", path)
	}

	if !PathExist(path) {
		return fmt.Errorf("%s is not exist", path)
	}

	return os.WriteFile(path, []byte(data), 0644)
}

func IsDir(path string) bool {
	file, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return file.IsDir()
}

func Mkdir(name string, perm os.FileMode) error {
	if err := os.Mkdir(name, perm); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create directory: %s, err: %s", name, err)
	}

	return nil
}

// PathExist returns true if the path exists
func PathExist(path string) bool {
	if _, err := os.Lstat(path); err != nil {
		return false
	}

	return true
}
