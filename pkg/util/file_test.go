package util_test

import (
	"bou.ke/monkey"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"numaadj.huawei.com/pkg/util"
	"os"
	"testing"
)

func TestReadFile(t *testing.T) {
	t.Run("should read file successfully", func(t *testing.T) {

		path := "test.txt"
		expectedData := []byte("Hello, World!")
		monkey.Patch(os.Lstat, func(string) (os.FileInfo, error) {
			return nil, nil
		})
		monkey.Patch(os.ReadFile, func(string) ([]byte, error) {
			return expectedData, nil
		})
		monkey.Patch(util.IsDir, func(string) bool {
			return false
		})
		monkey.Patch(util.PathExist, func(string) bool {
			return true
		})
		defer monkey.UnpatchAll()

		data, err := util.ReadFile(path)

		assert.NoError(t, err)
		assert.Equal(t, expectedData, data)
	})

	t.Run("should return error when path is directory", func(t *testing.T) {

		path := "test_dir"
		monkey.Patch(util.IsDir, func(string) bool {
			return true
		})
		defer monkey.UnpatchAll()

		_, err := util.ReadFile(path)

		assert.Error(t, err)
		assert.Equal(t, fmt.Errorf("%s is not a file", path), err)
	})

	t.Run("should return error when path does not exist", func(t *testing.T) {

		path := "non_existent.txt"
		monkey.Patch(util.IsDir, func(string) bool {
			return false
		})
		monkey.Patch(util.PathExist, func(string) bool {
			return false
		})
		defer monkey.UnpatchAll()

		_, err := util.ReadFile(path)

		assert.Error(t, err)
		assert.Equal(t, fmt.Errorf("%s is not exist", path), err)
	})
}

func TestWriteFile(t *testing.T) {
	t.Run("should write file successfully", func(t *testing.T) {

		path := "test.txt"
		data := "Hello, World!"
		monkey.Patch(os.Lstat, func(string) (os.FileInfo, error) {
			return nil, nil
		})
		monkey.Patch(os.WriteFile, func(string, []byte, os.FileMode) error {
			return nil
		})
		monkey.Patch(util.IsDir, func(string) bool {
			return false
		})
		monkey.Patch(util.PathExist, func(string) bool {
			return true
		})
		defer monkey.UnpatchAll()

		err := util.WriteFile(path, data)

		assert.NoError(t, err)
	})

	t.Run("should return error when path is directory", func(t *testing.T) {

		path := "test_dir"
		monkey.Patch(util.IsDir, func(string) bool {
			return true
		})
		defer monkey.UnpatchAll()

		err := util.WriteFile(path, "data")

		assert.Error(t, err)
		assert.Equal(t, fmt.Errorf("%s is not a file", path), err)
	})

	t.Run("should return error when path does not exist", func(t *testing.T) {

		path := "non_existent.txt"
		monkey.Patch(util.IsDir, func(string) bool {
			return false
		})
		monkey.Patch(util.PathExist, func(string) bool {
			return false
		})
		defer monkey.UnpatchAll()

		err := util.WriteFile(path, "data")

		assert.Error(t, err)
		assert.Equal(t, fmt.Errorf("%s is not exist", path), err)
	})
}

func TestIsDir(t *testing.T) {
	t.Run("should return false when path does not exist", func(t *testing.T) {

		path := "non_existent"
		monkey.Patch(os.Lstat, func(string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		})
		defer monkey.UnpatchAll()

		result := util.IsDir(path)

		assert.False(t, result)
	})
}

func TestMkdir(t *testing.T) {
	t.Run("should create directory successfully", func(t *testing.T) {

		name := "test_dir"
		perm := os.FileMode(0755)
		monkey.Patch(os.Mkdir, func(string, os.FileMode) error {
			return nil
		})
		defer monkey.UnpatchAll()

		err := util.Mkdir(name, perm)

		assert.NoError(t, err)
	})

	t.Run("should return error when creation fails", func(t *testing.T) {

		name := "test_dir"
		perm := os.FileMode(0755)
		expectedError := errors.New("mkdir failed")
		monkey.Patch(os.Mkdir, func(string, os.FileMode) error {
			return expectedError
		})
		defer monkey.UnpatchAll()

		err := util.Mkdir(name, perm)

		assert.Error(t, err)
		assert.Equal(t, fmt.Errorf("failed to create directory: %s, err: %s", name, expectedError), err)
	})

	t.Run("should return error when directory already exists", func(t *testing.T) {

		name := "existing_dir"
		perm := os.FileMode(0755)
		monkey.Patch(os.Mkdir, func(string, os.FileMode) error {
			return os.ErrExist
		})
		defer monkey.UnpatchAll()

		err := util.Mkdir(name, perm)

		assert.NoError(t, err)
	})
}

func TestPathExist(t *testing.T) {

	t.Run("should return false when path does not exist", func(t *testing.T) {

		path := "non_existent"
		monkey.Patch(os.Lstat, func(string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		})
		defer monkey.UnpatchAll()

		result := util.PathExist(path)

		assert.False(t, result)
	})
}

func TestMain(m *testing.M) {
	// Initialize test environment if needed
	code := m.Run()
	// Cleanup test environment if needed
	os.Exit(code)
}
