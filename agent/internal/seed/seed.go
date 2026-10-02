// Package seed 统一库引导: 确保 roots 目录存在, 首次为空时写入骨架内容。
// 铁律: 已存在且非空的库零覆盖; 链接/junction 一律不写穿; 失败不留 stamp 可重试。
package seed

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed all:pack
var packFS embed.FS

// SeedVersion 当前种子包版本; 升级内容时 bump, 已有库不受影响(零覆盖)。
const SeedVersion = "0.1.0"

// stampName 种子标记文件名, 写在库根内。
const stampName = ".fenjue-seed"

// packSubdir root 键 -> pack 内子目录; 未登记的键只建目录不播种。
var packSubdir = map[string]string{
	"memory": "pack/memory",
	"skills": "pack/skills",
}

// Bootstrap 确保 roots 全部存在, 并对空库写入骨架。返回变更描述列表。
func Bootstrap(roots map[string]string) ([]string, error) {
	var changes []string
	for _, key := range []string{"memory", "skills"} {
		root, ok := roots[key]
		if !ok || strings.TrimSpace(root) == "" {
			continue
		}
		ch, err := bootstrapRoot(key, root)
		if err != nil {
			return changes, fmt.Errorf("seed root %s (%s): %w", key, root, err)
		}
		changes = append(changes, ch...)
	}
	return changes, nil
}

func bootstrapRoot(key, root string) ([]string, error) {
	var changes []string
	if isLink(root) {
		return []string{"root is a link, left untouched: " + root}, nil
	}
	empty := false
	fi, err := os.Lstat(root)
	switch {
	case os.IsNotExist(err):
		if err := os.MkdirAll(root, 0o755); err != nil {
			return nil, fmt.Errorf("create root: %w", err)
		}
		changes = append(changes, "created root "+root)
		empty = true
	case err != nil:
		return nil, fmt.Errorf("stat root: %w", err)
	case !fi.IsDir():
		return []string{"root is a file, left untouched: " + root}, nil
	default:
		empty = dirEmpty(root)
	}
	sub, ok := packSubdir[key]
	if !ok || !empty {
		if !empty {
			changes = append(changes, "root not empty, seed skipped (zero-overwrite): "+root)
		}
		return changes, nil
	}
	n, err := copyPackDir(sub, root)
	if err != nil {
		return nil, fmt.Errorf("copy seed: %w", err)
	}
	if err := os.WriteFile(filepath.Join(root, stampName), []byte(SeedVersion+"\n"), 0o644); err != nil {
		return nil, fmt.Errorf("write seed stamp: %w", err)
	}
	changes = append(changes, fmt.Sprintf("seeded %d items into %s (v%s)", n, root, SeedVersion))
	return changes, nil
}

// isLink 链接判定: symlink 与 Windows junction 的 os.Readlink 都会成功。
func isLink(path string) bool {
	_, err := os.Readlink(path)
	return err == nil
}

func dirEmpty(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	return len(entries) == 0
}

// copyPackDir 把 pack 内子目录整体复制到目标目录, 返回写入文件数。
func copyPackDir(sub, dest string) (int, error) {
	n := 0
	err := fs.WalkDir(packFS, sub, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sub, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		src, err := packFS.Open(p)
		if err != nil {
			return err
		}
		defer src.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		if _, err := io.Copy(out, src); err != nil {
			return err
		}
		n++
		return nil
	})
	return n, err
}
