package managedfs

import (
	"errors"
	"os"
	"path/filepath"
)

func AtomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".atomic.*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err = temp.Write(data); err == nil {
		err = temp.Chmod(mode)
	}
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func EnsureSymlink(path, target string) error {
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			return errors.New("目标存在且不是受管符号链接")
		}
		current, err := os.Readlink(path)
		if err != nil {
			return err
		}
		if current != target {
			return errors.New("符号链接目标冲突")
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	return os.Symlink(target, path)
}

func AtomicSymlink(path, target string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".link.*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	_ = temp.Close()
	_ = os.Remove(tempPath)
	defer os.Remove(tempPath)
	if err := os.Symlink(target, tempPath); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}
