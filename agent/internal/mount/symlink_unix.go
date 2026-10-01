//go:build !windows

package mount

import (
	"fmt"
	"os"
	"path/filepath"
)

// Create 创建符号链接: os.Symlink。
func Create(from, to string) error {
	if err := os.MkdirAll(filepath.Dir(from), 0o755); err != nil {
		return fmt.Errorf("mount: mkdir parent of %q: %w", from, err)
	}
	if err := os.Symlink(to, from); err != nil {
		return fmt.Errorf("mount: symlink %q -> %q: %w", from, to, err)
	}
	return nil
}

// Remove 摘除链接本身。
func Remove(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("mount: remove link %q: %w", path, err)
	}
	return nil
}

// Probe 探测链接: 不存在返回零值。
func Probe(path string) (Info, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Info{}, nil
		}
		return Info{}, fmt.Errorf("mount: lstat %q: %w", path, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return Info{}, fmt.Errorf("mount: readlink %q: %w", path, err)
		}
		return Info{IsLink: true, Target: target}, nil
	}
	return Info{}, nil
}

// SameTarget 链接目标等价比较。
func SameTarget(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}
