//go:build windows

package mount

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func toBack(p string) string { return strings.ReplaceAll(p, "/", "\\") }

// Create 创建 junction: cmd /c mklink /J (免管理员权限)。
func Create(from, to string) error {
	if err := os.MkdirAll(filepath.Dir(from), 0o755); err != nil {
		return fmt.Errorf("mount: mkdir parent of %q: %w", from, err)
	}
	cmd := exec.Command("cmd", "/c", "mklink", "/J", toBack(from), toBack(to))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mount: mklink /J %q -> %q failed: %s", from, to, strings.TrimSpace(string(out)))
	}
	return nil
}

// Remove 摘除 junction (rmdir 只删链接本身, 不动目标; 对真实非空目录会失败, 天然防误删)。
func Remove(path string) error {
	cmd := exec.Command("cmd", "/c", "rmdir", toBack(path))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mount: rmdir junction %q failed: %s", path, strings.TrimSpace(string(out)))
	}
	return nil
}

// Probe 探测链接: 不存在返回零值。
// 实测 Go 1.27: junction 的 Lstat 既不设 ModeSymlink 也不算 IsDir(ModeIrregular),
// 但 os.Readlink 对 junction 可直接读出目标 —— 以 Readlink 成功为准。
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
	// junction (IO_REPARSE_TAG_MOUNT_POINT): Readlink 成功即链接。
	if target, err := os.Readlink(path); err == nil {
		return Info{IsLink: true, Target: target}, nil
	}
	return Info{}, nil
}

// SameTarget 链接目标等价比较: 去设备前缀 + 大小写不敏感 + 路径清洗。
func SameTarget(a, b string) bool {
	return normalizeTarget(a) == normalizeTarget(b)
}

func normalizeTarget(t string) string {
	t = strings.TrimPrefix(t, `\\?\`)
	t = strings.TrimPrefix(t, `\??\`)
	t = strings.TrimPrefix(t, `//?/`)
	return strings.ToLower(filepath.Clean(t))
}
