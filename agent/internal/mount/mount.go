// Package mount 平台挂载原语: 链接(junction/symlink)、物理镜像、逐技能树。
// 平台相关实现: junction_windows.go (Windows, mklink /J 免管理员) / symlink_unix.go (os.Symlink)。
package mount

import (
	"fmt"
	"os"
)

// Info 链接探活结果。
type Info struct {
	IsLink bool
	Target string
}

// 以下函数由平台实现文件提供(构建标签分离):
//
//	Create(from, to string) error    在 from 建链接指向 to
//	Remove(path string) error        摘除链接本身, 不动目标
//	Probe(path string) (Info, error) 不存在时返回零值 Info 与 nil error
//	SameTarget(a, b string) bool     链接目标等价比较(Windows 大小写不敏感)

// RemoveAny 删除 path: 链接只摘链不动目标; 普通文件/目录递归删除。
func RemoveAny(path string) error {
	info, err := Probe(path)
	if err != nil {
		return fmt.Errorf("mount: probe %q: %w", path, err)
	}
	if info.IsLink {
		return Remove(path)
	}
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("mount: lstat %q: %w", path, err)
	}
	return os.RemoveAll(path)
}
